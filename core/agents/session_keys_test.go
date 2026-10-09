package agents

import (
	"bytes"
	"context"
	"database/sql"
	"os"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/tests"
	"github.com/pocketbase/dbx"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SessionKeys", func() {
	var ctx context.Context
	user := model.User{ID: "u-1"}
	ds := &tests.MockDataStore{MockedUserProps: &tests.MockedUserPropsRepo{}}
	sk := SessionKeys{DataStore: ds, KeyName: "fakeSessionKey"}

	BeforeEach(func() {
		ctx = GinkgoT().Context()
	})

	It("uses the assigned key name", func() {
		Expect(sk.KeyName).To(Equal("fakeSessionKey"))
	})
	It("stores a value in the DB", func() {
		Expect(sk.Put(ctx, user.ID, "test-stored-value")).To(BeNil())
	})
	It("fetches the stored value", func() {
		value, err := sk.Get(ctx, user.ID)
		Expect(err).ToNot(HaveOccurred())
		Expect(value).To(Equal("test-stored-value"))
	})
	It("deletes the stored value", func() {
		Expect(sk.Delete(ctx, user.ID)).To(BeNil())
	})
	It("handles a not found value", func() {
		_, err := sk.Get(ctx, "u-2")
		Expect(err).To(MatchError(model.ErrNotFound))
	})

	It("never logs the session key, but still logs the user id and key name", func() {
		conn, err := sql.Open("sqlite3", ":memory:")
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(conn.Close)
		conn.SetMaxOpenConns(1)
		_, err = conn.ExecContext(ctx, "create table user_props (user_id varchar, key varchar, value varchar)")
		Expect(err).ToNot(HaveOccurred())
		props := persistence.NewUserPropsRepository(dbx.NewFromDB(conn, "sqlite3"))
		dbKeys := SessionKeys{DataStore: &tests.MockDataStore{MockedUserProps: props}, KeyName: "LastFMSessionKey"}

		logs := &bytes.Buffer{}
		log.SetOutput(logs)
		log.SetLevel(log.LevelTrace)
		DeferCleanup(func() {
			log.SetOutput(os.Stderr)
			log.SetLevel(log.LevelFatal)
		})

		Expect(dbKeys.Put(ctx, "logged-user-id", "inserted-session-key")).To(Succeed())
		Expect(dbKeys.Put(ctx, "logged-user-id", "updated-session-key")).To(Succeed())

		Expect(dbKeys.Get(ctx, "logged-user-id")).To(Equal("updated-session-key"))
		Expect(logs.String()).To(ContainSubstring("INSERT INTO user_props"))
		Expect(logs.String()).To(ContainSubstring("UPDATE user_props"))
		Expect(logs.String()).To(ContainSubstring("logged-user-id"))
		Expect(logs.String()).To(ContainSubstring("LastFMSessionKey"))
		Expect(logs.String()).ToNot(ContainSubstring("inserted-session-key"))
		Expect(logs.String()).ToNot(ContainSubstring("updated-session-key"))
	})
})
