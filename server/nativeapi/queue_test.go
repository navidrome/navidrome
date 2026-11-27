package nativeapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/persistence/persistencetest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// queueFaults makes the play queue repository fail on demand, and records the columns of the last Store.
type queueFaults struct {
	lastCols          []string
	storeErr          error
	retrieveErr       error
	withMediaFilesErr error
	clearErr          error
}

type faultyPlayQueue struct {
	model.PlayQueueRepository
	*queueFaults
}

func (q faultyPlayQueue) Store(ctx context.Context, pq *model.PlayQueue, cols ...string) error {
	if q.storeErr != nil {
		return q.storeErr
	}
	q.lastCols = cols
	return q.PlayQueueRepository.Store(ctx, pq, cols...)
}

func (q faultyPlayQueue) RetrieveWithMediaFiles(ctx context.Context, userID string) (*model.PlayQueue, error) {
	if q.retrieveErr != nil {
		return nil, q.retrieveErr
	}
	pq, err := q.PlayQueueRepository.RetrieveWithMediaFiles(ctx, userID)
	if err == nil {
		err = q.withMediaFilesErr
	}
	return pq, err
}

func (q faultyPlayQueue) Clear(ctx context.Context, userID string) error {
	if q.clearErr != nil {
		return q.clearErr
	}
	return q.PlayQueueRepository.Clear(ctx, userID)
}

var _ = Describe("Queue Endpoints", func() {
	var (
		ctx    context.Context
		realDS model.DataStore
		ds     model.DataStore
		faults *queueFaults
		user   model.User
	)

	seed := func(pq *model.PlayQueue) {
		GinkgoHelper()
		pq.UserID = user.ID
		Expect(realDS.PlayQueue().Store(ctx, pq, "current", "position", "items")).To(Succeed())
	}
	stored := func() *model.PlayQueue {
		GinkgoHelper()
		pq, err := realDS.PlayQueue().Retrieve(ctx, user.ID)
		Expect(err).ToNot(HaveOccurred())
		return pq
	}
	expectNoQueue := func() {
		GinkgoHelper()
		_, err := realDS.PlayQueue().Retrieve(ctx, user.ID)
		Expect(err).To(MatchError(model.ErrNotFound))
	}

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		realDS = persistence.New(db.Db())
		faults = &queueFaults{}
		ds = persistencetest.Stub(realDS, persistencetest.Repos{PlayQueue: func(r model.PlayQueueRepository) model.PlayQueueRepository {
			return faultyPlayQueue{r, faults}
		}})
		user = model.User{ID: "u1", UserName: "user"}
		putUsers(ctx, realDS, &user)
		for _, id := range []string{"s1", "s2", "s3", "s4", "track1", "track2", "track3"} {
			mf := &model.MediaFile{ID: id, LibraryID: model.DefaultLibraryID, Path: id + ".mp3", Title: id}
			Expect(realDS.MediaFile().Put(ctx, mf)).To(Succeed())
		}
	})

	DescribeTable("rejects a body over the size limit with 413 without saving",
		func(method string, handler func(model.DataStore) http.HandlerFunc) {
			body := `{"ids":["` + strings.Repeat("x", int(maxQueueBody())) + `"]}`
			req := httptest.NewRequestWithContext(ctx, method, "/queue", strings.NewReader(body))
			req = req.WithContext(request.WithUser(req.Context(), user))
			w := httptest.NewRecorder()

			handler(ds)(w, req)
			Expect(w.Code).To(Equal(http.StatusRequestEntityTooLarge))
			expectNoQueue()
		},
		Entry("POST", http.MethodPost, saveQueue),
		Entry("PUT", http.MethodPut, updateQueue),
	)

	It("saves a body larger than 4 MiB when the track limit is raised", func() {
		conf.Update(func(c *conf.Options) { c.DevMaxPlaylistTracksPerRequest = 100_000 })
		body := `{"ids":["` + strings.Repeat("x", 4<<20) + `"],"current":0}`
		req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/queue", strings.NewReader(body))
		req = req.WithContext(request.WithUser(req.Context(), user))
		w := httptest.NewRecorder()

		saveQueue(ds)(w, req)
		Expect(w.Code).To(Equal(http.StatusNoContent))
		Expect(stored().Items).To(HaveLen(1))
	})

	Describe("POST /queue", func() {
		It("saves the queue", func() {
			payload := updateQueuePayload{Ids: new([]string{"s1", "s2"}), Current: new(1), Position: new(int64(10))}
			body, _ := json.Marshal(payload)
			req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/queue", bytes.NewReader(body))
			ctx := request.WithUser(req.Context(), user)
			ctx = request.WithClient(ctx, "TestClient")
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			saveQueue(ds)(w, req)
			Expect(w.Code).To(Equal(http.StatusNoContent))
			pq := stored()
			Expect(pq.Current).To(Equal(1))
			Expect(pq.Items).To(HaveLen(2))
			Expect(pq.Items[1].ID).To(Equal("s2"))
			Expect(pq.ChangedBy).To(Equal("TestClient"))
		})

		It("saves an empty queue", func() {
			seed(&model.PlayQueue{Items: model.MediaFiles{{ID: "s1"}}})
			payload := updateQueuePayload{Ids: new([]string{}), Current: new(0), Position: new(int64(0))}
			body, _ := json.Marshal(payload)
			req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/queue", bytes.NewReader(body))
			req = req.WithContext(request.WithUser(req.Context(), user))
			w := httptest.NewRecorder()

			saveQueue(ds)(w, req)
			Expect(w.Code).To(Equal(http.StatusNoContent))
			expectNoQueue()
		})

		It("returns bad request for invalid current index (negative)", func() {
			payload := updateQueuePayload{Ids: new([]string{"s1", "s2"}), Current: new(-1), Position: new(int64(10))}
			body, _ := json.Marshal(payload)
			req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/queue", bytes.NewReader(body))
			req = req.WithContext(request.WithUser(req.Context(), user))
			w := httptest.NewRecorder()

			saveQueue(ds)(w, req)
			Expect(w.Code).To(Equal(http.StatusBadRequest))
			Expect(w.Body.String()).To(ContainSubstring("current index out of bounds"))
		})

		It("returns bad request for invalid current index (too large)", func() {
			payload := updateQueuePayload{Ids: new([]string{"s1", "s2"}), Current: new(2), Position: new(int64(10))}
			body, _ := json.Marshal(payload)
			req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/queue", bytes.NewReader(body))
			req = req.WithContext(request.WithUser(req.Context(), user))
			w := httptest.NewRecorder()

			saveQueue(ds)(w, req)
			Expect(w.Code).To(Equal(http.StatusBadRequest))
			Expect(w.Body.String()).To(ContainSubstring("current index out of bounds"))
		})

		It("returns bad request for malformed JSON", func() {
			req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/queue", bytes.NewReader([]byte("invalid json")))
			req = req.WithContext(request.WithUser(req.Context(), user))
			w := httptest.NewRecorder()

			saveQueue(ds)(w, req)
			Expect(w.Code).To(Equal(http.StatusBadRequest))
		})

		It("returns internal server error when store fails", func() {
			faults.storeErr = errors.New("db unavailable")
			payload := updateQueuePayload{Ids: new([]string{"s1"}), Current: new(0), Position: new(int64(10))}
			body, _ := json.Marshal(payload)
			req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/queue", bytes.NewReader(body))
			req = req.WithContext(request.WithUser(req.Context(), user))
			w := httptest.NewRecorder()

			saveQueue(ds)(w, req)
			Expect(w.Code).To(Equal(http.StatusInternalServerError))
		})
	})

	Describe("GET /queue", func() {
		It("returns the queue", func() {
			seed(&model.PlayQueue{
				Current:  1,
				Position: 55,
				Items:    model.MediaFiles{{ID: "track1"}, {ID: "track2"}, {ID: "track3"}},
			})
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/queue", nil)
			req = req.WithContext(request.WithUser(req.Context(), user))
			w := httptest.NewRecorder()

			getQueue(ds)(w, req)
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Header().Get("Content-Type")).To(Equal("application/json"))
			var resp model.PlayQueue
			Expect(json.Unmarshal(w.Body.Bytes(), &resp)).To(Succeed())
			Expect(resp.Current).To(Equal(1))
			Expect(resp.Position).To(Equal(int64(55)))
			Expect(resp.Items).To(HaveLen(3))
			Expect(resp.Items[0].ID).To(Equal("track1"))
			Expect(resp.Items[0].Title).To(Equal("track1"))
			Expect(resp.Items[1].ID).To(Equal("track2"))
			Expect(resp.Items[2].ID).To(Equal("track3"))
		})

		It("returns empty queue when user has no queue", func() {
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/queue", nil)
			req = req.WithContext(request.WithUser(req.Context(), user))
			w := httptest.NewRecorder()

			getQueue(ds)(w, req)
			Expect(w.Code).To(Equal(http.StatusOK))
			var resp model.PlayQueue
			Expect(json.Unmarshal(w.Body.Bytes(), &resp)).To(Succeed())
			Expect(resp.Items).To(BeEmpty())
			Expect(resp.Current).To(Equal(0))
			Expect(resp.Position).To(Equal(int64(0)))
		})

		It("returns the loaded tracks when some of them fail to load", func() {
			seed(&model.PlayQueue{Items: model.MediaFiles{{ID: "track1"}}})
			faults.withMediaFilesErr = fmt.Errorf("%w: chunk failed", model.ErrIncompleteQueue)
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/queue", nil)
			req = req.WithContext(request.WithUser(req.Context(), user))
			w := httptest.NewRecorder()

			getQueue(ds)(w, req)
			Expect(w.Code).To(Equal(http.StatusOK))
			var resp model.PlayQueue
			Expect(json.Unmarshal(w.Body.Bytes(), &resp)).To(Succeed())
			Expect(resp.Items).To(HaveLen(1))
		})

		It("returns internal server error when retrieve fails", func() {
			faults.retrieveErr = errors.New("db unavailable")
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/queue", nil)
			req = req.WithContext(request.WithUser(req.Context(), user))
			w := httptest.NewRecorder()

			getQueue(ds)(w, req)
			Expect(w.Code).To(Equal(http.StatusInternalServerError))
		})
	})

	Describe("PUT /queue", func() {
		updateFieldsAsTestClient := func() {
			GinkgoHelper()
			seed(&model.PlayQueue{Items: model.MediaFiles{{ID: "s1"}, {ID: "s2"}, {ID: "s3"}}})
			payload := updateQueuePayload{Current: new(2), Position: new(int64(20))}
			body, _ := json.Marshal(payload)
			req := httptest.NewRequestWithContext(ctx, http.MethodPut, "/queue", bytes.NewReader(body))
			ctx := request.WithUser(req.Context(), user)
			ctx = request.WithClient(ctx, "TestClient")
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			updateQueue(ds)(w, req)
			Expect(w.Code).To(Equal(http.StatusNoContent))
		}

		It("updates the queue fields", func() {
			updateFieldsAsTestClient()
			Expect(stored().Current).To(Equal(2))
			Expect(stored().Position).To(Equal(int64(20)))
		})

		PIt("records the client that updated the queue", func() {
			updateFieldsAsTestClient()
			Expect(stored().ChangedBy).To(Equal("TestClient"))
		})

		It("updates only ids", func() {
			seed(&model.PlayQueue{Current: 1})
			payload := updateQueuePayload{Ids: new([]string{"s1", "s2"})}
			body, _ := json.Marshal(payload)
			req := httptest.NewRequestWithContext(ctx, http.MethodPut, "/queue", bytes.NewReader(body))
			req = req.WithContext(request.WithUser(req.Context(), user))
			w := httptest.NewRecorder()

			updateQueue(ds)(w, req)
			Expect(w.Code).To(Equal(http.StatusNoContent))
			Expect(stored().Items).To(HaveLen(2))
			Expect(faults.lastCols).To(ConsistOf("items"))
		})

		It("updates only ids when some tracks fail to load", func() {
			seed(&model.PlayQueue{Current: 1, Position: 45,
				Items: model.MediaFiles{{ID: "s1"}, {ID: "s2"}, {ID: "s3"}}})
			faults.withMediaFilesErr = fmt.Errorf("%w: chunk failed", model.ErrIncompleteQueue)
			body, _ := json.Marshal(updateQueuePayload{Ids: new([]string{"s1", "s2", "s3", "s4"})})
			req := httptest.NewRequestWithContext(ctx, http.MethodPut, "/queue", bytes.NewReader(body))
			req = req.WithContext(request.WithUser(req.Context(), user))
			w := httptest.NewRecorder()

			updateQueue(ds)(w, req)
			Expect(w.Code).To(Equal(http.StatusNoContent))
			Expect(stored().Items).To(HaveLen(4))
			Expect(faults.lastCols).To(ConsistOf("items"))
		})

		Context("when the stored queue has tracks the user can't see", func() {
			storedItems := model.MediaFiles{{ID: "gone1"}, {ID: "s1"}, {ID: "gone2"}, {ID: "s2"}}

			BeforeEach(func() {
				seed(&model.PlayQueue{Current: 2, Position: 45, Items: storedItems})
			})

			put := func(payload updateQueuePayload) int {
				body, _ := json.Marshal(payload)
				req := httptest.NewRequestWithContext(ctx, http.MethodPut, "/queue", bytes.NewReader(body))
				req = req.WithContext(request.WithUser(req.Context(), user))
				w := httptest.NewRecorder()
				updateQueue(ds)(w, req)
				return w.Code
			}

			It("stores current as the index of the same track in the stored queue", func() {
				Expect(put(updateQueuePayload{Current: new(1)})).To(Equal(http.StatusNoContent))
				Expect(stored().Current).To(Equal(3))
				Expect(faults.lastCols).To(ConsistOf("current"))
			})

			It("rejects a current that is only valid for the stored queue", func() {
				Expect(put(updateQueuePayload{Current: new(2)})).To(Equal(http.StatusBadRequest))
			})

			It("moves current to the track the client saw when only position changes", func() {
				Expect(put(updateQueuePayload{Position: new(int64(30))})).To(Equal(http.StatusNoContent))
				Expect(stored().Current).To(Equal(3))
				Expect(stored().Position).To(Equal(int64(30)))
				Expect(faults.lastCols).To(ConsistOf("current", "position"))
			})

			It("writes only position when the stored current track is visible", func() {
				seed(&model.PlayQueue{Current: 1, Position: 45, Items: storedItems})
				Expect(put(updateQueuePayload{Position: new(int64(30))})).To(Equal(http.StatusNoContent))
				Expect(stored().Current).To(Equal(1))
				Expect(faults.lastCols).To(ConsistOf("position"))
			})

			It("refuses to update current when the tracks fail to load", func() {
				faults.withMediaFilesErr = fmt.Errorf("%w: chunk failed", model.ErrIncompleteQueue)
				Expect(put(updateQueuePayload{Current: new(0)})).To(Equal(http.StatusInternalServerError))
				Expect(faults.lastCols).To(BeNil())
				Expect(stored().Current).To(Equal(2))
			})

			It("keeps the current track and position the client saw when only ids change", func() {
				Expect(put(updateQueuePayload{Ids: new([]string{"s1", "s2", "s3"})})).To(Equal(http.StatusNoContent))
				Expect(stored().Current).To(Equal(1))
				Expect(stored().Position).To(BeZero())
				Expect(faults.lastCols).To(ConsistOf("items", "current", "position"))
			})

			It("writes only the current index when the current track is visible and only ids change", func() {
				seed(&model.PlayQueue{Current: 1, Position: 45, Items: storedItems})
				Expect(put(updateQueuePayload{Ids: new([]string{"s1", "s2"})})).To(Equal(http.StatusNoContent))
				Expect(stored().Current).To(BeZero())
				Expect(stored().Position).To(Equal(int64(45)))
				Expect(faults.lastCols).To(ConsistOf("items", "current"))
			})

			It("uses the position from the request when ids and position change", func() {
				Expect(put(updateQueuePayload{Ids: new([]string{"s1", "s2"}), Position: new(int64(10))})).To(Equal(http.StatusNoContent))
				Expect(stored().Current).To(Equal(1))
				Expect(stored().Position).To(Equal(int64(10)))
			})
		})

		It("updates ids and current", func() {
			seed(&model.PlayQueue{})
			payload := updateQueuePayload{Ids: new([]string{"s1", "s2"}), Current: new(1)}
			body, _ := json.Marshal(payload)
			req := httptest.NewRequestWithContext(ctx, http.MethodPut, "/queue", bytes.NewReader(body))
			req = req.WithContext(request.WithUser(req.Context(), user))
			w := httptest.NewRecorder()

			updateQueue(ds)(w, req)
			Expect(w.Code).To(Equal(http.StatusNoContent))
			Expect(stored().Items).To(HaveLen(2))
			Expect(stored().Current).To(Equal(1))
			Expect(faults.lastCols).To(ConsistOf("items", "current"))
		})

		It("returns bad request when new ids invalidate current", func() {
			seed(&model.PlayQueue{Current: 2})
			payload := updateQueuePayload{Ids: new([]string{"s1", "s2"})}
			body, _ := json.Marshal(payload)
			req := httptest.NewRequestWithContext(ctx, http.MethodPut, "/queue", bytes.NewReader(body))
			req = req.WithContext(request.WithUser(req.Context(), user))
			w := httptest.NewRecorder()

			updateQueue(ds)(w, req)
			Expect(w.Code).To(Equal(http.StatusBadRequest))
		})

		It("returns bad request when current out of bounds", func() {
			seed(&model.PlayQueue{Items: model.MediaFiles{{ID: "s1"}}})
			payload := updateQueuePayload{Current: new(3)}
			body, _ := json.Marshal(payload)
			req := httptest.NewRequestWithContext(ctx, http.MethodPut, "/queue", bytes.NewReader(body))
			req = req.WithContext(request.WithUser(req.Context(), user))
			w := httptest.NewRecorder()

			updateQueue(ds)(w, req)
			Expect(w.Code).To(Equal(http.StatusBadRequest))
		})

		It("returns bad request for malformed JSON", func() {
			req := httptest.NewRequestWithContext(ctx, http.MethodPut, "/queue", bytes.NewReader([]byte("{")))
			req = req.WithContext(request.WithUser(req.Context(), user))
			w := httptest.NewRecorder()

			updateQueue(ds)(w, req)
			Expect(w.Code).To(Equal(http.StatusBadRequest))
		})

		It("returns internal server error when store fails", func() {
			seed(&model.PlayQueue{Items: model.MediaFiles{{ID: "s1"}}})
			faults.storeErr = errors.New("db unavailable")
			payload := updateQueuePayload{Position: new(int64(10))}
			body, _ := json.Marshal(payload)
			req := httptest.NewRequestWithContext(ctx, http.MethodPut, "/queue", bytes.NewReader(body))
			req = req.WithContext(request.WithUser(req.Context(), user))
			w := httptest.NewRecorder()

			updateQueue(ds)(w, req)
			Expect(w.Code).To(Equal(http.StatusInternalServerError))
		})
	})

	Describe("DELETE /queue", func() {
		It("clears the queue", func() {
			seed(&model.PlayQueue{Items: model.MediaFiles{{ID: "s1"}}})
			req := httptest.NewRequestWithContext(ctx, http.MethodDelete, "/queue", nil)
			req = req.WithContext(request.WithUser(req.Context(), user))
			w := httptest.NewRecorder()

			clearQueue(ds)(w, req)
			Expect(w.Code).To(Equal(http.StatusNoContent))
			expectNoQueue()
		})

		It("returns internal server error when clear fails", func() {
			faults.clearErr = errors.New("db unavailable")
			req := httptest.NewRequestWithContext(ctx, http.MethodDelete, "/queue", nil)
			req = req.WithContext(request.WithUser(req.Context(), user))
			w := httptest.NewRecorder()

			clearQueue(ds)(w, req)
			Expect(w.Code).To(Equal(http.StatusInternalServerError))
		})
	})
})
