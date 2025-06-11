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
			req = req.WithContext(request.WithUser(req.Context(), user))
			w := httptest.NewRecorder()

			saveQueue(ds)(w, req)
			Expect(w.Code).To(Equal(http.StatusNoContent))
			Expect(repo.Queue).ToNot(BeNil())
			Expect(repo.Queue.Current).To(Equal(1))
			Expect(repo.Queue.Items).To(HaveLen(2))
			Expect(repo.Queue.Items[1].ID).To(Equal("s2"))
		})
	})

	Describe("GET /queue", func() {
		It("returns the queue", func() {
			repo.Queue = &model.PlayQueue{UserID: user.ID, Current: 2, Items: model.MediaFiles{{ID: "s1"}}}
			req := httptest.NewRequest("GET", "/queue", nil)
			req = req.WithContext(request.WithUser(req.Context(), user))
			w := httptest.NewRecorder()

			getQueue(ds)(w, req)
			Expect(w.Code).To(Equal(http.StatusOK))
			var resp model.PlayQueue
			Expect(json.Unmarshal(w.Body.Bytes(), &resp)).To(Succeed())
			Expect(resp.Current).To(Equal(2))
			Expect(resp.Items).To(HaveLen(1))
			Expect(resp.Items[0].ID).To(Equal("s1"))
		})
	})
})
