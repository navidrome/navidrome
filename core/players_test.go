package core

import (
	"context"
	"time"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Players", func() {
	var players Players
	var repo *mockPlayerRepository
	ctx := log.NewContext(context.TODO())
	ctx = request.WithUser(ctx, model.User{ID: "userid", UserName: "johndoe"})
	ctx = request.WithUsername(ctx, "johndoe")
	var beforeRegister time.Time

	BeforeEach(func() {
		repo = &mockPlayerRepository{}
		ds := &tests.MockDataStore{MockedPlayer: repo, MockedTranscoding: &tests.MockTranscodingRepo{}}
		players = NewPlayers(ds)
		beforeRegister = time.Now()
	})

	Describe("Register", func() {
		It("creates a new player when no ID is specified", func() {
			p, trc, err := players.Register(ctx, "", "client", "chrome", "1.2.3.4")
			Expect(err).ToNot(HaveOccurred())
			Expect(p.ID).ToNot(BeEmpty())
			Expect(p.LastSeen).To(BeTemporally(">=", beforeRegister))
			Expect(p.Client).To(Equal("client"))
			Expect(p.UserId).To(Equal("userid"))
			Expect(p.UserAgent).To(Equal("chrome"))
			Expect(repo.lastSaved).To(Equal(p))
			Expect(trc).To(BeNil())
		})

		It("creates a new player if it cannot find any matching player", func() {
			p, trc, err := players.Register(ctx, "123", "client", "chrome", "1.2.3.4")
			Expect(err).ToNot(HaveOccurred())
			Expect(p.ID).ToNot(BeEmpty())
			Expect(p.LastSeen).To(BeTemporally(">=", beforeRegister))
			Expect(repo.lastSaved).To(Equal(p))
			Expect(trc).To(BeNil())
		})

		It("creates a new player if client does not match the one in DB", func() {
			plr := &model.Player{ID: "123", Name: "A Player", Client: "client1111", LastSeen: time.Time{}}
			repo.add(plr)
			p, trc, err := players.Register(ctx, "123", "client2222", "chrome", "1.2.3.4")
			Expect(err).ToNot(HaveOccurred())
			Expect(p.ID).ToNot(BeEmpty())
			Expect(p.ID).ToNot(Equal("123"))
			Expect(p.LastSeen).To(BeTemporally(">=", beforeRegister))
			Expect(p.Client).To(Equal("client2222"))
			Expect(trc).To(BeNil())
		})

		It("does not reuse another user's player by ID", func() {
			plr := &model.Player{ID: "123", Name: "A Player", Client: "client", UserId: "otheruser", UserAgent: "Pixel", TranscodingId: "1"}
			repo.add(plr)
			p, trc, err := players.Register(ctx, "123", "client", "chrome", "1.2.3.4")
			Expect(err).ToNot(HaveOccurred())
			Expect(p.ID).ToNot(Equal("123"))
			Expect(p.UserId).To(Equal("userid"))
			Expect(repo.lastSaved).To(Equal(p))
			Expect(trc).To(BeNil())
		})

		It("finds players by ID", func() {
			plr := &model.Player{ID: "123", Name: "A Player", Client: "client", UserId: "userid", LastSeen: time.Time{}}
			repo.add(plr)
			p, trc, err := players.Register(ctx, "123", "client", "chrome", "1.2.3.4")
			Expect(err).ToNot(HaveOccurred())
			Expect(p.ID).To(Equal("123"))
			Expect(p.LastSeen).To(BeTemporally(">=", beforeRegister))
			Expect(repo.lastSaved).To(Equal(p))
			Expect(trc).To(BeNil())
		})

		It("finds player by client and user names when ID is not found", func() {
			plr := &model.Player{ID: "123", Name: "A Player", Client: "client", UserId: "userid", UserAgent: "chrome", LastSeen: time.Time{}}
			repo.add(plr)
			p, _, err := players.Register(ctx, "999", "client", "chrome", "1.2.3.4")
			Expect(err).ToNot(HaveOccurred())
			Expect(p.ID).To(Equal("123"))
			Expect(p.LastSeen).To(BeTemporally(">=", beforeRegister))
			Expect(repo.lastSaved).To(Equal(p))
		})

		It("finds player by client and user names when not ID is provided", func() {
			plr := &model.Player{ID: "123", Name: "A Player", Client: "client", UserId: "userid", UserAgent: "chrome", LastSeen: time.Time{}}
			repo.add(plr)
			p, _, err := players.Register(ctx, "", "client", "chrome", "1.2.3.4")
			Expect(err).ToNot(HaveOccurred())
			Expect(p.ID).To(Equal("123"))
			Expect(p.LastSeen).To(BeTemporally(">=", beforeRegister))
			Expect(repo.lastSaved).To(Equal(p))
		})

		It("finds player by ID and return its transcoding", func() {
			plr := &model.Player{ID: "123", Name: "A Player", Client: "client", UserId: "userid", LastSeen: time.Time{}, TranscodingId: "1"}
			repo.add(plr)
			p, trc, err := players.Register(ctx, "123", "client", "chrome", "1.2.3.4")
			Expect(err).ToNot(HaveOccurred())
			Expect(p.ID).To(Equal("123"))
			Expect(p.LastSeen).To(BeTemporally(">=", beforeRegister))
			Expect(repo.lastSaved).To(Equal(p))
			Expect(trc.ID).To(Equal("1"))
		})

		It("does not rename a player that has an API key", func() {
			plr := &model.Player{ID: "123", Name: "My Phone", Client: "client", UserId: "userid", HasAPIKey: true}
			repo.add(plr)
			p, _, err := players.Register(ctx, "123", "client", "chrome", "1.2.3.4")
			Expect(err).ToNot(HaveOccurred())
			Expect(p.ID).To(Equal("123"))
			Expect(p.Name).To(Equal("My Phone"))
		})

		Context("bad username casing", func() {
			ctx := log.NewContext(context.TODO())
			ctx = request.WithUser(ctx, model.User{ID: "userid", UserName: "Johndoe"})
			ctx = request.WithUsername(ctx, "Johndoe")

			It("finds player by client and user names when not ID is provided", func() {
				plr := &model.Player{ID: "123", Name: "A Player", Client: "client", UserId: "userid", UserAgent: "chrome", LastSeen: time.Time{}}
				repo.add(plr)
				p, _, err := players.Register(ctx, "", "client", "chrome", "1.2.3.4")
				Expect(err).ToNot(HaveOccurred())
				Expect(p.ID).To(Equal("123"))
				Expect(p.LastSeen).To(BeTemporally(">=", beforeRegister))
				Expect(repo.lastSaved).To(Equal(p))
			})
		})
	})

	Describe("Touch", func() {
		It("records usage but keeps the name and client", func() {
			plr := model.Player{ID: "123", Name: "My Phone", Client: "Symfonium", UserId: "userid", HasAPIKey: true}
			p, trc, err := players.Touch(ctx, plr, "OtherClient", "android", "1.2.3.4")
			Expect(err).ToNot(HaveOccurred())
			Expect(p.Name).To(Equal("My Phone"))
			Expect(p.Client).To(Equal("Symfonium"))
			Expect(p.UserAgent).To(Equal("android"))
			Expect(p.IP).To(Equal("1.2.3.4"))
			Expect(p.LastSeen).To(BeTemporally(">=", beforeRegister))
			Expect(repo.lastSaved).To(Equal(p))
			Expect(trc).To(BeNil())
		})

		It("fills in the client on first use", func() {
			p, _, err := players.Touch(ctx, model.Player{ID: "123", Name: "Manual", UserId: "userid"}, "Symfonium", "android", "1.2.3.4")
			Expect(err).ToNot(HaveOccurred())
			Expect(p.Client).To(Equal("Symfonium"))
		})

		It("returns the player's transcoding", func() {
			p, trc, err := players.Touch(ctx, model.Player{ID: "123", UserId: "userid", TranscodingId: "1"}, "c", "ua", "1.2.3.4")
			Expect(err).ToNot(HaveOccurred())
			Expect(p.ID).To(Equal("123"))
			Expect(trc.ID).To(Equal("1"))
		})
	})
})

type mockPlayerRepository struct {
	model.PlayerRepository
	lastSaved *model.Player
	data      map[string]model.Player
}

func (m *mockPlayerRepository) add(p *model.Player) {
	if m.data == nil {
		m.data = make(map[string]model.Player)
	}
	m.data[p.ID] = *p
}

func (m *mockPlayerRepository) Get(_ context.Context, id string) (*model.Player, error) {
	if p, ok := m.data[id]; ok {
		return &p, nil
	}
	return nil, model.ErrNotFound
}

func (m *mockPlayerRepository) FindMatch(_ context.Context, userId, client, userAgent string) (*model.Player, error) {
	for _, p := range m.data {
		if p.Client == client && p.UserId == userId && p.UserAgent == userAgent {
			return &p, nil
		}
	}
	return nil, model.ErrNotFound
}

func (m *mockPlayerRepository) Put(_ context.Context, p *model.Player) error {
	m.lastSaved = p
	return nil
}
