package engine

import (
	"context"
	"time"

	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
	"github.com/deluan/navidrome/persistence"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Players", func() {
	var players Players
	var repo *mockPlayerRepository
	ctx := context.WithValue(log.NewContext(nil), "user", model.User{ID: "userid", UserName: "johndoe"})
	ctx = context.WithValue(ctx, "username", "johndoe")
	var beforeRegister time.Time

	BeforeEach(func() {
		repo = &mockPlayerRepository{}
		ds := &persistence.MockDataStore{MockedPlayer: repo}
		players = NewPlayers(ds)
		beforeRegister = time.Now()
	})

	Describe("Register", func() {
		It("creates a new player when no ID is specified", func() {
			p, err := players.Register(ctx, "", "client", "chrome", "1.2.3.4")
			Expect(err).ToNot(HaveOccurred())
			Expect(p.ID).ToNot(BeEmpty())
			Expect(p.LastSeen).To(BeTemporally(">=", beforeRegister))
			Expect(p.Client).To(Equal("client"))
			Expect(p.UserName).To(Equal("johndoe"))
			Expect(p.Type).To(Equal("chrome"))
			Expect(repo.lastSaved).To(Equal(p))
		})

		It("creates a new player if it cannot find any matching player", func() {
			p, err := players.Register(ctx, "123", "client", "chrome", "1.2.3.4")
			Expect(err).ToNot(HaveOccurred())
			Expect(p.ID).ToNot(BeEmpty())
			Expect(p.LastSeen).To(BeTemporally(">=", beforeRegister))
			Expect(repo.lastSaved).To(Equal(p))
		})

		It("finds players by ID", func() {
			plr := &model.Player{ID: "123", Name: "A Player", LastSeen: time.Time{}}
			repo.add(plr)
			p, err := players.Register(ctx, "123", "client", "chrome", "1.2.3.4")
			Expect(err).ToNot(HaveOccurred())
			Expect(p.ID).To(Equal("123"))
			Expect(p.LastSeen).To(BeTemporally(">=", beforeRegister))
			Expect(repo.lastSaved).To(Equal(p))
		})

		It("finds player by client and user names when ID is not found", func() {
			plr := &model.Player{ID: "123", Name: "A Player", Client: "client", UserName: "johndoe", LastSeen: time.Time{}}
			repo.add(plr)
			p, err := players.Register(ctx, "999", "client", "chrome", "1.2.3.4")
			Expect(err).ToNot(HaveOccurred())
			Expect(p.ID).To(Equal("123"))
			Expect(p.LastSeen).To(BeTemporally(">=", beforeRegister))
			Expect(repo.lastSaved).To(Equal(p))
		})

		It("finds player by client and user names when not ID is provided", func() {
			plr := &model.Player{ID: "123", Name: "A Player", Client: "client", UserName: "johndoe", LastSeen: time.Time{}}
			repo.add(plr)
			p, err := players.Register(ctx, "", "client", "chrome", "1.2.3.4")
			Expect(err).ToNot(HaveOccurred())
			Expect(p.ID).To(Equal("123"))
			Expect(p.LastSeen).To(BeTemporally(">=", beforeRegister))
			Expect(repo.lastSaved).To(Equal(p))
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

func (m *mockPlayerRepository) Get(id string) (*model.Player, error) {
	if p, ok := m.data[id]; ok {
		return &p, nil
	}
	return nil, model.ErrNotFound
}

func (m *mockPlayerRepository) FindByName(client, userName string) (*model.Player, error) {
	for _, p := range m.data {
		if p.Client == client && p.UserName == userName {
			return &p, nil
		}
	}
	return nil, model.ErrNotFound
}

func (m *mockPlayerRepository) Put(p *model.Player) error {
	m.lastSaved = p
	return nil
}
