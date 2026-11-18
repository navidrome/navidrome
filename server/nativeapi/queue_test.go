package nativeapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Queue Endpoints", func() {
	var (
		ds       *tests.MockDataStore
		repo     *tests.MockPlayQueueRepo
		user     model.User
		userRepo *tests.MockedUserRepo
	)

	BeforeEach(func() {
		repo = &tests.MockPlayQueueRepo{}
		user = model.User{ID: "u1", UserName: "user"}
		userRepo = tests.CreateMockUserRepo()
		_ = userRepo.Put(GinkgoT().Context(), &user)
		ds = &tests.MockDataStore{MockedPlayQueue: repo, MockedUser: userRepo, MockedProperty: &tests.MockedPropertyRepo{}}
	})

	Describe("POST /queue", func() {
		It("saves the queue", func() {
			payload := updateQueuePayload{Ids: new([]string{"s1", "s2"}), Current: new(1), Position: new(int64(10))}
			body, _ := json.Marshal(payload)
			req := httptest.NewRequest(http.MethodPost, "/queue", bytes.NewReader(body))
			ctx := request.WithUser(req.Context(), user)
			ctx = request.WithClient(ctx, "TestClient")
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			saveQueue(ds)(w, req)
			Expect(w.Code).To(Equal(http.StatusNoContent))
			Expect(repo.Queue).ToNot(BeNil())
			Expect(repo.Queue.Current).To(Equal(1))
			Expect(repo.Queue.Items).To(HaveLen(2))
			Expect(repo.Queue.Items[1].ID).To(Equal("s2"))
			Expect(repo.Queue.ChangedBy).To(Equal("TestClient"))
		})

		It("saves an empty queue", func() {
			payload := updateQueuePayload{Ids: new([]string{}), Current: new(0), Position: new(int64(0))}
			body, _ := json.Marshal(payload)
			req := httptest.NewRequest(http.MethodPost, "/queue", bytes.NewReader(body))
			req = req.WithContext(request.WithUser(req.Context(), user))
			w := httptest.NewRecorder()

			saveQueue(ds)(w, req)
			Expect(w.Code).To(Equal(http.StatusNoContent))
			Expect(repo.Queue).ToNot(BeNil())
			Expect(repo.Queue.Items).To(BeEmpty())
		})

		It("returns bad request for invalid current index (negative)", func() {
			payload := updateQueuePayload{Ids: new([]string{"s1", "s2"}), Current: new(-1), Position: new(int64(10))}
			body, _ := json.Marshal(payload)
			req := httptest.NewRequest(http.MethodPost, "/queue", bytes.NewReader(body))
			req = req.WithContext(request.WithUser(req.Context(), user))
			w := httptest.NewRecorder()

			saveQueue(ds)(w, req)
			Expect(w.Code).To(Equal(http.StatusBadRequest))
			Expect(w.Body.String()).To(ContainSubstring("current index out of bounds"))
		})

		It("returns bad request for invalid current index (too large)", func() {
			payload := updateQueuePayload{Ids: new([]string{"s1", "s2"}), Current: new(2), Position: new(int64(10))}
			body, _ := json.Marshal(payload)
			req := httptest.NewRequest(http.MethodPost, "/queue", bytes.NewReader(body))
			req = req.WithContext(request.WithUser(req.Context(), user))
			w := httptest.NewRecorder()

			saveQueue(ds)(w, req)
			Expect(w.Code).To(Equal(http.StatusBadRequest))
			Expect(w.Body.String()).To(ContainSubstring("current index out of bounds"))
		})

		It("returns bad request for malformed JSON", func() {
			req := httptest.NewRequest(http.MethodPost, "/queue", bytes.NewReader([]byte("invalid json")))
			req = req.WithContext(request.WithUser(req.Context(), user))
			w := httptest.NewRecorder()

			saveQueue(ds)(w, req)
			Expect(w.Code).To(Equal(http.StatusBadRequest))
		})

		It("returns internal server error when store fails", func() {
			repo.Err = true
			payload := updateQueuePayload{Ids: new([]string{"s1"}), Current: new(0), Position: new(int64(10))}
			body, _ := json.Marshal(payload)
			req := httptest.NewRequest(http.MethodPost, "/queue", bytes.NewReader(body))
			req = req.WithContext(request.WithUser(req.Context(), user))
			w := httptest.NewRecorder()

			saveQueue(ds)(w, req)
			Expect(w.Code).To(Equal(http.StatusInternalServerError))
		})
	})

	Describe("GET /queue", func() {
		It("returns the queue", func() {
			queue := &model.PlayQueue{
				UserID:   user.ID,
				Current:  1,
				Position: 55,
				Items: model.MediaFiles{
					{ID: "track1", Title: "Song 1"},
					{ID: "track2", Title: "Song 2"},
					{ID: "track3", Title: "Song 3"},
				},
			}
			repo.Queue = queue
			req := httptest.NewRequest(http.MethodGet, "/queue", nil)
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
			Expect(resp.Items[1].ID).To(Equal("track2"))
			Expect(resp.Items[2].ID).To(Equal("track3"))
		})

		It("returns empty queue when user has no queue", func() {
			req := httptest.NewRequest(http.MethodGet, "/queue", nil)
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
			repo.Queue = &model.PlayQueue{UserID: user.ID, Items: model.MediaFiles{{ID: "track1"}}}
			repo.WithMediaFilesErr = fmt.Errorf("%w: chunk failed", model.ErrIncompleteQueue)
			req := httptest.NewRequest(http.MethodGet, "/queue", nil)
			req = req.WithContext(request.WithUser(req.Context(), user))
			w := httptest.NewRecorder()

			getQueue(ds)(w, req)
			Expect(w.Code).To(Equal(http.StatusOK))
			var resp model.PlayQueue
			Expect(json.Unmarshal(w.Body.Bytes(), &resp)).To(Succeed())
			Expect(resp.Items).To(HaveLen(1))
		})

		It("returns internal server error when retrieve fails", func() {
			repo.Err = true
			req := httptest.NewRequest(http.MethodGet, "/queue", nil)
			req = req.WithContext(request.WithUser(req.Context(), user))
			w := httptest.NewRecorder()

			getQueue(ds)(w, req)
			Expect(w.Code).To(Equal(http.StatusInternalServerError))
		})
	})

	Describe("PUT /queue", func() {
		It("updates the queue fields", func() {
			repo.Queue = &model.PlayQueue{UserID: user.ID, Items: model.MediaFiles{{ID: "s1"}, {ID: "s2"}, {ID: "s3"}}}
			payload := updateQueuePayload{Current: new(2), Position: new(int64(20))}
			body, _ := json.Marshal(payload)
			req := httptest.NewRequest(http.MethodPut, "/queue", bytes.NewReader(body))
			ctx := request.WithUser(req.Context(), user)
			ctx = request.WithClient(ctx, "TestClient")
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			updateQueue(ds)(w, req)
			Expect(w.Code).To(Equal(http.StatusNoContent))
			Expect(repo.Queue).ToNot(BeNil())
			Expect(repo.Queue.Current).To(Equal(2))
			Expect(repo.Queue.Position).To(Equal(int64(20)))
			Expect(repo.Queue.ChangedBy).To(Equal("TestClient"))
		})

		It("updates only ids", func() {
			repo.Queue = &model.PlayQueue{UserID: user.ID, Current: 1}
			payload := updateQueuePayload{Ids: new([]string{"s1", "s2"})}
			body, _ := json.Marshal(payload)
			req := httptest.NewRequest(http.MethodPut, "/queue", bytes.NewReader(body))
			req = req.WithContext(request.WithUser(req.Context(), user))
			w := httptest.NewRecorder()

			updateQueue(ds)(w, req)
			Expect(w.Code).To(Equal(http.StatusNoContent))
			Expect(repo.Queue.Items).To(HaveLen(2))
			Expect(repo.LastCols).To(ConsistOf("items"))
		})

		It("updates only ids when some tracks fail to load", func() {
			repo.Queue = &model.PlayQueue{UserID: user.ID, Current: 1, Position: 45,
				Items: model.MediaFiles{{ID: "s1"}, {ID: "s2"}, {ID: "s3"}}}
			repo.QueueWithMediaFiles = &model.PlayQueue{UserID: user.ID, Current: 1, Position: 45,
				Items: model.MediaFiles{{ID: "s1"}, {ID: "s2"}}}
			repo.WithMediaFilesErr = fmt.Errorf("%w: chunk failed", model.ErrIncompleteQueue)
			body, _ := json.Marshal(updateQueuePayload{Ids: new([]string{"s1", "s2", "s3", "s4"})})
			req := httptest.NewRequest(http.MethodPut, "/queue", bytes.NewReader(body))
			req = req.WithContext(request.WithUser(req.Context(), user))
			w := httptest.NewRecorder()

			updateQueue(ds)(w, req)
			Expect(w.Code).To(Equal(http.StatusNoContent))
			Expect(repo.Queue.Items).To(HaveLen(4))
			Expect(repo.LastCols).To(ConsistOf("items"))
		})

		Context("when the stored queue has tracks the user can't see", func() {
			BeforeEach(func() {
				repo.Queue = &model.PlayQueue{UserID: user.ID, Current: 2, Position: 45,
					Items: model.MediaFiles{{ID: "gone1"}, {ID: "s1"}, {ID: "gone2"}, {ID: "s2"}}}
				repo.QueueWithMediaFiles = &model.PlayQueue{UserID: user.ID, Current: 1, Position: 0,
					Items: model.MediaFiles{{ID: "s1"}, {ID: "s2"}}}
				ds.MediaFile().(*tests.MockMediaFileRepo).SetData(repo.QueueWithMediaFiles.Items)
			})

			put := func(payload updateQueuePayload) int {
				body, _ := json.Marshal(payload)
				req := httptest.NewRequest(http.MethodPut, "/queue", bytes.NewReader(body))
				req = req.WithContext(request.WithUser(req.Context(), user))
				w := httptest.NewRecorder()
				updateQueue(ds)(w, req)
				return w.Code
			}

			It("stores current as the index of the same track in the stored queue", func() {
				Expect(put(updateQueuePayload{Current: new(1)})).To(Equal(http.StatusNoContent))
				Expect(repo.Queue.Current).To(Equal(3))
				Expect(repo.LastCols).To(ConsistOf("current"))
			})

			It("rejects a current that is only valid for the stored queue", func() {
				Expect(put(updateQueuePayload{Current: new(2)})).To(Equal(http.StatusBadRequest))
			})

			It("moves current to the track the client saw when only position changes", func() {
				Expect(put(updateQueuePayload{Position: new(int64(30))})).To(Equal(http.StatusNoContent))
				Expect(repo.Queue.Current).To(Equal(3))
				Expect(repo.Queue.Position).To(Equal(int64(30)))
				Expect(repo.LastCols).To(ConsistOf("current", "position"))
			})

			It("writes only position when the stored current track is visible", func() {
				repo.Queue.Current = 1
				repo.QueueWithMediaFiles.Current = 0
				Expect(put(updateQueuePayload{Position: new(int64(30))})).To(Equal(http.StatusNoContent))
				Expect(repo.LastCols).To(ConsistOf("position"))
			})

			It("refuses to update current when the tracks fail to load", func() {
				repo.WithMediaFilesErr = fmt.Errorf("%w: chunk failed", model.ErrIncompleteQueue)
				Expect(put(updateQueuePayload{Current: new(0)})).To(Equal(http.StatusInternalServerError))
				Expect(repo.LastCols).To(BeNil())
				Expect(repo.Queue.Current).To(Equal(2))
			})

			It("keeps the current track and position the client saw when only ids change", func() {
				Expect(put(updateQueuePayload{Ids: new([]string{"s1", "s2", "s3"})})).To(Equal(http.StatusNoContent))
				Expect(repo.Queue.Current).To(Equal(1))
				Expect(repo.Queue.Position).To(BeZero())
				Expect(repo.LastCols).To(ConsistOf("items", "current", "position"))
			})

			It("writes only the current index when the current track is visible and only ids change", func() {
				repo.Queue.Current = 1
				repo.QueueWithMediaFiles.Current = 0
				repo.QueueWithMediaFiles.Position = 45
				Expect(put(updateQueuePayload{Ids: new([]string{"s1", "s2"})})).To(Equal(http.StatusNoContent))
				Expect(repo.Queue.Current).To(BeZero())
				Expect(repo.LastCols).To(ConsistOf("items", "current"))
			})

			It("uses the position from the request when ids and position change", func() {
				Expect(put(updateQueuePayload{Ids: new([]string{"s1", "s2"}), Position: new(int64(10))})).To(Equal(http.StatusNoContent))
				Expect(repo.Queue.Current).To(Equal(1))
				Expect(repo.Queue.Position).To(Equal(int64(10)))
			})
		})

		It("updates ids and current", func() {
			repo.Queue = &model.PlayQueue{UserID: user.ID}
			payload := updateQueuePayload{Ids: new([]string{"s1", "s2"}), Current: new(1)}
			body, _ := json.Marshal(payload)
			req := httptest.NewRequest(http.MethodPut, "/queue", bytes.NewReader(body))
			req = req.WithContext(request.WithUser(req.Context(), user))
			w := httptest.NewRecorder()

			updateQueue(ds)(w, req)
			Expect(w.Code).To(Equal(http.StatusNoContent))
			Expect(repo.Queue.Items).To(HaveLen(2))
			Expect(repo.Queue.Current).To(Equal(1))
			Expect(repo.LastCols).To(ConsistOf("items", "current"))
		})

		It("returns bad request when new ids invalidate current", func() {
			repo.Queue = &model.PlayQueue{UserID: user.ID, Current: 2}
			payload := updateQueuePayload{Ids: new([]string{"s1", "s2"})}
			body, _ := json.Marshal(payload)
			req := httptest.NewRequest(http.MethodPut, "/queue", bytes.NewReader(body))
			req = req.WithContext(request.WithUser(req.Context(), user))
			w := httptest.NewRecorder()

			updateQueue(ds)(w, req)
			Expect(w.Code).To(Equal(http.StatusBadRequest))
		})

		It("returns bad request when current out of bounds", func() {
			repo.Queue = &model.PlayQueue{UserID: user.ID, Items: model.MediaFiles{{ID: "s1"}}}
			payload := updateQueuePayload{Current: new(3)}
			body, _ := json.Marshal(payload)
			req := httptest.NewRequest(http.MethodPut, "/queue", bytes.NewReader(body))
			req = req.WithContext(request.WithUser(req.Context(), user))
			w := httptest.NewRecorder()

			updateQueue(ds)(w, req)
			Expect(w.Code).To(Equal(http.StatusBadRequest))
		})

		It("returns bad request for malformed JSON", func() {
			req := httptest.NewRequest(http.MethodPut, "/queue", bytes.NewReader([]byte("{")))
			req = req.WithContext(request.WithUser(req.Context(), user))
			w := httptest.NewRecorder()

			updateQueue(ds)(w, req)
			Expect(w.Code).To(Equal(http.StatusBadRequest))
		})

		It("returns internal server error when store fails", func() {
			repo.Err = true
			payload := updateQueuePayload{Position: new(int64(10))}
			body, _ := json.Marshal(payload)
			req := httptest.NewRequest(http.MethodPut, "/queue", bytes.NewReader(body))
			req = req.WithContext(request.WithUser(req.Context(), user))
			w := httptest.NewRecorder()

			updateQueue(ds)(w, req)
			Expect(w.Code).To(Equal(http.StatusInternalServerError))
		})
	})

	Describe("DELETE /queue", func() {
		It("clears the queue", func() {
			repo.Queue = &model.PlayQueue{UserID: user.ID, Items: model.MediaFiles{{ID: "s1"}}}
			req := httptest.NewRequest(http.MethodDelete, "/queue", nil)
			req = req.WithContext(request.WithUser(req.Context(), user))
			w := httptest.NewRecorder()

			clearQueue(ds)(w, req)
			Expect(w.Code).To(Equal(http.StatusNoContent))
			Expect(repo.Queue).To(BeNil())
		})

		It("returns internal server error when clear fails", func() {
			repo.Err = true
			req := httptest.NewRequest(http.MethodDelete, "/queue", nil)
			req = req.WithContext(request.WithUser(req.Context(), user))
			w := httptest.NewRecorder()

			clearQueue(ds)(w, req)
			Expect(w.Code).To(Equal(http.StatusInternalServerError))
		})
	})
})
