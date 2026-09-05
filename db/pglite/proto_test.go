package pglite_test

import (
	"context"
	"net"
	"path/filepath"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgproto3"
	"github.com/navidrome/navidrome/db/pglite"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("PGlite wire protocol", func() {
	var pg *pglite.PGlite

	BeforeEach(func() {
		var err error
		pg, err = pglite.New(context.Background(), pglite.Config{DataDir: GinkgoT().TempDir(), Stderr: GinkgoWriter})
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(pg.Close)
	})

	It("pgconn survives a SQL error", func() {
		ctx := context.Background()
		conn, err := pgconn.Connect(ctx, pg.DSN())
		Expect(err).ToNot(HaveOccurred())
		results, err := conn.Exec(ctx, "SELECT * FROM nope").ReadAll()
		GinkgoWriter.Printf("results=%d err=%v closed=%v txStatus=%c\n", len(results), err, conn.IsClosed(), conn.TxStatus())
		Expect(err).To(MatchError(ContainSubstring("does not exist")))
		Expect(conn.IsClosed()).To(BeFalse())
	})

	It("pgproto3 decodes every message of the error reply", func() {
		cfg, err := pgconn.ParseConfig(pg.DSN())
		Expect(err).ToNot(HaveOccurred())
		sock, err := net.Dial("unix", filepath.Join(cfg.Host, ".s.PGSQL.5432"))
		Expect(err).ToNot(HaveOccurred())
		defer sock.Close()
		fe := pgproto3.NewFrontend(sock, sock)
		fe.Send(&pgproto3.StartupMessage{ProtocolVersion: pgproto3.ProtocolVersionNumber, Parameters: map[string]string{"user": "postgres", "database": "postgres"}})
		Expect(fe.Flush()).To(Succeed())
		readUntilReady := func() {
			for {
				msg, err := fe.Receive()
				Expect(err).ToNot(HaveOccurred())
				GinkgoWriter.Printf("  <- %T\n", msg)
				switch m := msg.(type) {
				case *pgproto3.AuthenticationMD5Password:
					fe.Send(&pgproto3.PasswordMessage{Password: "md5" + "x"})
					Expect(fe.Flush()).To(Succeed())
				case *pgproto3.ErrorResponse:
					GinkgoWriter.Printf("     error: %s\n", m.Message)
				case *pgproto3.ReadyForQuery:
					return
				}
			}
		}
		readUntilReady()
		fe.Send(&pgproto3.Query{String: "SELECT * FROM nope"})
		Expect(fe.Flush()).To(Succeed())
		readUntilReady()
		fe.Send(&pgproto3.Query{String: "SELECT 1"})
		Expect(fe.Flush()).To(Succeed())
		readUntilReady()
	})
})
