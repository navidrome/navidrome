package core

import (
	"context"
	"time"

	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/persistence"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Players", func() {
	var players Players
	var ds model.DataStore
	var ctx context.Context
	var beforeRegister time.Time
	var trcID string

	add := func(p *model.Player) {
		GinkgoHelper()
		Expect(ds.Player().Put(ctx, p)).To(Succeed())
		if p.HasAPIKey {
			_, err := db.Db().Exec("UPDATE player SET api_key_hash = 'hash' WHERE id = ?", p.ID)
			Expect(err).ToNot(HaveOccurred())
		}
	}

	expectStored := func(p *model.Player) {
		GinkgoHelper()
		stored, err := ds.Player().Get(ctx, p.ID)
		Expect(err).ToNot(HaveOccurred())
		Expect(stored.UserId).To(Equal(p.UserId))
		Expect(stored.Client).To(Equal(p.Client))
		Expect(stored.Name).To(Equal(p.Name))
		Expect(stored.UserAgent).To(Equal(p.UserAgent))
		Expect(stored.IP).To(Equal(p.IP))
		Expect(stored.LastSeen).To(BeTemporally("~", p.LastSeen, time.Millisecond))
	}

	BeforeEach(func() {
		ctx = log.NewContext(GinkgoT().Context())
		ds = persistence.New(db.Db())
		for _, u := range []model.User{{ID: "userid", UserName: "johndoe"}, {ID: "otheruser", UserName: "other"}} {
			Expect(ds.User().Put(ctx, &u)).To(Succeed())
		}
		ctx = request.WithUser(ctx, model.User{ID: "userid", UserName: "johndoe"})
		ctx = request.WithUsername(ctx, "johndoe")
		trc, err := ds.Transcoding().FindByFormat(ctx, "mp3")
		Expect(err).ToNot(HaveOccurred())
		trcID = trc.ID
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
			expectStored(p)
			Expect(trc).To(BeNil())
		})

		It("creates a new player if it cannot find any matching player", func() {
			p, trc, err := players.Register(ctx, "123", "client", "chrome", "1.2.3.4")
			Expect(err).ToNot(HaveOccurred())
			Expect(p.ID).ToNot(BeEmpty())
			Expect(p.LastSeen).To(BeTemporally(">=", beforeRegister))
			expectStored(p)
			Expect(trc).To(BeNil())
		})

		It("creates a new player if client does not match the one in DB", func() {
			plr := &model.Player{ID: "123", Name: "A Player", Client: "client1111", UserId: "userid", LastSeen: time.Time{}}
			add(plr)
			p, trc, err := players.Register(ctx, "123", "client2222", "chrome", "1.2.3.4")
			Expect(err).ToNot(HaveOccurred())
			Expect(p.ID).ToNot(BeEmpty())
			Expect(p.ID).ToNot(Equal("123"))
			Expect(p.LastSeen).To(BeTemporally(">=", beforeRegister))
			Expect(p.Client).To(Equal("client2222"))
			Expect(trc).To(BeNil())
		})

		It("does not reuse another user's player by ID", func() {
			plr := &model.Player{ID: "123", Name: "A Player", Client: "client", UserId: "otheruser", UserAgent: "Pixel", TranscodingId: trcID}
			add(plr)
			p, trc, err := players.Register(ctx, "123", "client", "chrome", "1.2.3.4")
			Expect(err).ToNot(HaveOccurred())
			Expect(p.ID).ToNot(Equal("123"))
			Expect(p.UserId).To(Equal("userid"))
			expectStored(p)
			Expect(trc).To(BeNil())
		})

		It("finds players by ID", func() {
			plr := &model.Player{ID: "123", Name: "A Player", Client: "client", UserId: "userid", LastSeen: time.Time{}}
			add(plr)
			p, trc, err := players.Register(ctx, "123", "client", "chrome", "1.2.3.4")
			Expect(err).ToNot(HaveOccurred())
			Expect(p.ID).To(Equal("123"))
			Expect(p.LastSeen).To(BeTemporally(">=", beforeRegister))
			expectStored(p)
			Expect(trc).To(BeNil())
		})

		It("finds player by client and user names when ID is not found", func() {
			plr := &model.Player{ID: "123", Name: "A Player", Client: "client", UserId: "userid", UserAgent: "chrome", LastSeen: time.Time{}}
			add(plr)
			p, _, err := players.Register(ctx, "999", "client", "chrome", "1.2.3.4")
			Expect(err).ToNot(HaveOccurred())
			Expect(p.ID).To(Equal("123"))
			Expect(p.LastSeen).To(BeTemporally(">=", beforeRegister))
			expectStored(p)
		})

		It("finds player by client and user names when not ID is provided", func() {
			plr := &model.Player{ID: "123", Name: "A Player", Client: "client", UserId: "userid", UserAgent: "chrome", LastSeen: time.Time{}}
			add(plr)
			p, _, err := players.Register(ctx, "", "client", "chrome", "1.2.3.4")
			Expect(err).ToNot(HaveOccurred())
			Expect(p.ID).To(Equal("123"))
			Expect(p.LastSeen).To(BeTemporally(">=", beforeRegister))
			expectStored(p)
		})

		It("finds player by ID and return its transcoding", func() {
			plr := &model.Player{ID: "123", Name: "A Player", Client: "client", UserId: "userid", LastSeen: time.Time{}, TranscodingId: trcID}
			add(plr)
			p, trc, err := players.Register(ctx, "123", "client", "chrome", "1.2.3.4")
			Expect(err).ToNot(HaveOccurred())
			Expect(p.ID).To(Equal("123"))
			Expect(p.LastSeen).To(BeTemporally(">=", beforeRegister))
			expectStored(p)
			Expect(trc.ID).To(Equal(trcID))
		})

		It("does not rename a player that has an API key", func() {
			plr := &model.Player{ID: "123", Name: "My Phone", Client: "client", UserId: "userid", HasAPIKey: true}
			add(plr)
			p, _, err := players.Register(ctx, "123", "client", "chrome", "1.2.3.4")
			Expect(err).ToNot(HaveOccurred())
			Expect(p.ID).To(Equal("123"))
			Expect(p.Name).To(Equal("My Phone"))
		})

		Context("bad username casing", func() {
			BeforeEach(func() {
				ctx = request.WithUser(ctx, model.User{ID: "userid", UserName: "Johndoe"})
				ctx = request.WithUsername(ctx, "Johndoe")
			})

			It("finds player by client and user names when not ID is provided", func() {
				plr := &model.Player{ID: "123", Name: "A Player", Client: "client", UserId: "userid", UserAgent: "chrome", LastSeen: time.Time{}}
				add(plr)
				p, _, err := players.Register(ctx, "", "client", "chrome", "1.2.3.4")
				Expect(err).ToNot(HaveOccurred())
				Expect(p.ID).To(Equal("123"))
				Expect(p.LastSeen).To(BeTemporally(">=", beforeRegister))
				expectStored(p)
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
			expectStored(p)
			Expect(trc).To(BeNil())
		})

		It("fills in the client on first use", func() {
			p, _, err := players.Touch(ctx, model.Player{ID: "123", Name: "Manual", UserId: "userid"}, "Symfonium", "android", "1.2.3.4")
			Expect(err).ToNot(HaveOccurred())
			Expect(p.Client).To(Equal("Symfonium"))
		})

		It("returns the player's transcoding", func() {
			p, trc, err := players.Touch(ctx, model.Player{ID: "123", UserId: "userid", TranscodingId: trcID}, "c", "ua", "1.2.3.4")
			Expect(err).ToNot(HaveOccurred())
			Expect(p.ID).To(Equal("123"))
			Expect(trc.ID).To(Equal(trcID))
		})
	})
})
