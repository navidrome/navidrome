package nativeapi

import (
	"bytes"
	"encoding/json"
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
		_ = userRepo.Put(&user)
		ds = &tests.MockDataStore{MockedPlayQueue: repo, MockedUser: userRepo, MockedProperty: &tests.MockedPropertyRepo{}}
	})

	Describe("POST /queue", func() {
		It("saves the queue", func() {
			payload := queuePayload{Ids: []string{"s1", "s2"}, Current: 1, Position: 10}
			body, _ := json.Marshal(payload)
			req := httptest.NewRequest("POST", "/queue", bytes.NewReader(body))
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
			payload := queuePayload{Ids: []string{}, Current: 0, Position: 0}
			body, _ := json.Marshal(payload)
			req := httptest.NewRequest("POST", "/queue", bytes.NewReader(body))
			req = req.WithContext(request.WithUser(req.Context(), user))
			w := httptest.NewRecorder()

			saveQueue(ds)(w, req)
			Expect(w.Code).To(Equal(http.StatusNoContent))
			Expect(repo.Queue).ToNot(BeNil())
			Expect(repo.Queue.Items).To(HaveLen(0))
		})

		It("returns bad request for invalid current index (negative)", func() {
			payload := queuePayload{Ids: []string{"s1", "s2"}, Current: -1, Position: 10}
			body, _ := json.Marshal(payload)
			req := httptest.NewRequest("POST", "/queue", bytes.NewReader(body))
			req = req.WithContext(request.WithUser(req.Context(), user))
			w := httptest.NewRecorder()

			saveQueue(ds)(w, req)
			Expect(w.Code).To(Equal(http.StatusBadRequest))
			Expect(w.Body.String()).To(ContainSubstring("current index out of bounds"))
		})

		It("returns bad request for invalid current index (too large)", func() {
			payload := queuePayload{Ids: []string{"s1", "s2"}, Current: 2, Position: 10}
			body, _ := json.Marshal(payload)
			req := httptest.NewRequest("POST", "/queue", bytes.NewReader(body))
			req = req.WithContext(request.WithUser(req.Context(), user))
			w := httptest.NewRecorder()

			saveQueue(ds)(w, req)
			Expect(w.Code).To(Equal(http.StatusBadRequest))
			Expect(w.Body.String()).To(ContainSubstring("current index out of bounds"))
		})

		It("returns bad request for malformed JSON", func() {
			req := httptest.NewRequest("POST", "/queue", bytes.NewReader([]byte("invalid json")))
			req = req.WithContext(request.WithUser(req.Context(), user))
			w := httptest.NewRecorder()

			saveQueue(ds)(w, req)
			Expect(w.Code).To(Equal(http.StatusBadRequest))
		})

		It("returns internal server error when store fails", func() {
			repo.Err = true
			payload := queuePayload{Ids: []string{"s1"}, Current: 0, Position: 10}
			body, _ := json.Marshal(payload)
			req := httptest.NewRequest("POST", "/queue", bytes.NewReader(body))
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
			req := httptest.NewRequest("GET", "/queue", nil)
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
			req := httptest.NewRequest("GET", "/queue", nil)
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

		It("returns internal server error when retrieve fails", func() {
			repo.Err = true
			req := httptest.NewRequest("GET", "/queue", nil)
			req = req.WithContext(request.WithUser(req.Context(), user))
			w := httptest.NewRecorder()

			getQueue(ds)(w, req)
			Expect(w.Code).To(Equal(http.StatusInternalServerError))
		})
	})
})
