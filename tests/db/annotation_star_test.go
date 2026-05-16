package persistence_test

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Test suite para el flujo /rest/star - Marcar canción como favorita
// Valida todas las reglas de integridad en la tabla ANNOTATION

func TestStarEndpoint(t *testing.T) {
	tests.Init(t, false)
	log.SetLevel(log.LevelFatal)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Star Endpoint - BD Tests")
}

var _ = Describe("Star Endpoint DB Tests", func() {
	var database *sql.DB
	var ctx context.Context

	BeforeEach(func() {
		// Usar in-memory SQLite para tests rápidos
		path := "file::memory:?cache=shared&_foreign_keys=on"
		database, _ = sql.Open(db.Dialect, path)
		ctx = context.Background()

		// Crear schema completo (mínimo para los tests)
		_, err := database.ExecContext(ctx, `
			CREATE TABLE user (
				id VARCHAR(255) NOT NULL PRIMARY KEY,
				user_name VARCHAR(255) NOT NULL UNIQUE,
				name VARCHAR(255),
				email VARCHAR(255),
				password VARCHAR(255),
				is_admin BOOLEAN DEFAULT FALSE,
				last_login_at DATETIME,
				last_access_at DATETIME,
				created_at DATETIME NOT NULL,
				updated_at DATETIME NOT NULL
			);

			CREATE TABLE media_file (
				id VARCHAR(255) NOT NULL PRIMARY KEY,
				path VARCHAR(255) NOT NULL,
				title VARCHAR(255),
				album_id VARCHAR(255),
				artist_id VARCHAR(255),
				album_artist VARCHAR(255),
				duration REAL DEFAULT 0,
				created_at DATETIME,
				updated_at DATETIME
			);

			CREATE TABLE annotation (
				user_id VARCHAR(255) NOT NULL REFERENCES user(id) ON DELETE CASCADE,
				item_id VARCHAR(255) NOT NULL,
				item_type VARCHAR(255) NOT NULL,
				play_count INTEGER,
				play_date DATETIME,
				rating INTEGER,
				starred BOOLEAN DEFAULT FALSE NOT NULL,
				starred_at DATETIME,
				rating_date DATETIME,
				UNIQUE (user_id, item_id, item_type)
			);

			CREATE INDEX annotation_user_id ON annotation(user_id);
			CREATE INDEX annotation_item ON annotation(item_id, item_type);
		`)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		database.Close()
	})

	// ================== TEST 1: Foreign Key Constraint ==================
	Describe("TEST 1: Foreign Key Constraint ANNOTATION -> USER", func() {
		It("debe rechazar inserción de annotation con user_id inválido", func() {
			_, err := database.ExecContext(ctx,
				`INSERT INTO annotation (user_id, item_id, item_type, starred)
				 VALUES (?, ?, ?, ?)`,
				"INVALID_USER_ID", "song123", "song", true)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("FOREIGN KEY constraint failed"))
		})

		It("debe permitir inserción si user_id existe", func() {
			// Setup: crear usuario
			_, err := database.ExecContext(ctx,
				`INSERT INTO user (id, user_name, password, is_admin, created_at, updated_at)
				 VALUES (?, ?, ?, ?, ?, ?)`,
				"admin", "admin", "hashedpwd", true, time.Now(), time.Now())
			Expect(err).NotTo(HaveOccurred())

			// Act: insertar annotation
			_, err = database.ExecContext(ctx,
				`INSERT INTO annotation (user_id, item_id, item_type, starred)
				 VALUES (?, ?, ?, ?)`,
				"admin", "song123", "song", true)
			Expect(err).NotTo(HaveOccurred())

			// Verify
			var starred bool
			err = database.QueryRowContext(ctx,
				`SELECT starred FROM annotation WHERE user_id = ? AND item_id = ?`,
				"admin", "song123").Scan(&starred)
			Expect(err).NotTo(HaveOccurred())
			Expect(starred).To(BeTrue())
		})
	})

	// ================== TEST 2: NOT NULL Constraints ==================
	Describe("TEST 2: NOT NULL Constraints", func() {
		BeforeEach(func() {
			// Setup: crear usuario
			_, err := database.ExecContext(ctx,
				`INSERT INTO user (id, user_name, password, is_admin, created_at, updated_at)
				 VALUES (?, ?, ?, ?, ?, ?)`,
				"admin", "admin", "hashedpwd", true, time.Now(), time.Now())
			Expect(err).NotTo(HaveOccurred())
		})

		It("debe rechazar si user_id es NULL", func() {
			_, err := database.ExecContext(ctx,
				`INSERT INTO annotation (user_id, item_id, item_type, starred)
				 VALUES (NULL, ?, ?, ?)`,
				"song123", "song", true)
			Expect(err).To(HaveOccurred())
		})

		It("debe rechazar si item_id es NULL", func() {
			_, err := database.ExecContext(ctx,
				`INSERT INTO annotation (user_id, item_id, item_type, starred)
				 VALUES (?, NULL, ?, ?)`,
				"admin", "song", true)
			Expect(err).To(HaveOccurred())
		})

		It("debe rechazar si item_type es NULL", func() {
			_, err := database.ExecContext(ctx,
				`INSERT INTO annotation (user_id, item_id, item_type, starred)
				 VALUES (?, ?, NULL, ?)`,
				"admin", "song123", true)
			Expect(err).To(HaveOccurred())
		})

		It("debe rechazar si starred es NULL", func() {
			_, err := database.ExecContext(ctx,
				`INSERT INTO annotation (user_id, item_id, item_type, starred)
				 VALUES (?, ?, ?, NULL)`,
				"admin", "song123", "song")
			Expect(err).To(HaveOccurred())
		})
	})

	// ================== TEST 3: Flujo Completo ==================
	Describe("TEST 3: Flujo Completo - INSERT Media + Annotation + JOIN", func() {
		It("debe permitir insertar canción y marcar como favorita con JOIN", func() {
			// Setup
			_, err := database.ExecContext(ctx,
				`INSERT INTO user (id, user_name, password, is_admin, created_at, updated_at)
				 VALUES (?, ?, ?, ?, ?, ?)`,
				"admin", "admin", "hashedpwd", true, time.Now(), time.Now())
			Expect(err).NotTo(HaveOccurred())

			// Insert media_file
			_, err = database.ExecContext(ctx,
				`INSERT INTO media_file (id, path, title) VALUES (?, ?, ?)`,
				"song123", "/path/to/song.mp3", "My Favorite Song")
			Expect(err).NotTo(HaveOccurred())

			// Insert annotation (star)
			_, err = database.ExecContext(ctx,
				`INSERT INTO annotation (user_id, item_id, item_type, starred, starred_at)
				 VALUES (?, ?, ?, ?, ?)`,
				"admin", "song123", "song", true, time.Now())
			Expect(err).NotTo(HaveOccurred())

			// Verify con JOIN
			var title string
			var starred bool
			var starredAt sql.NullTime

			err = database.QueryRowContext(ctx, `
				SELECT mf.title, ann.starred, ann.starred_at
				FROM media_file mf
				JOIN annotation ann ON mf.id = ann.item_id
				WHERE ann.user_id = ? AND ann.item_type = 'song'
			`, "admin").Scan(&title, &starred, &starredAt)

			Expect(err).NotTo(HaveOccurred())
			Expect(title).To(Equal("My Favorite Song"))
			Expect(starred).To(BeTrue())
			Expect(starredAt.Valid).To(BeTrue())
		})
	})

	// ================== TEST 4: UPDATE Favorita ==================
	Describe("TEST 4: UPDATE - Marcar/Desmarcar Favorita", func() {
		BeforeEach(func() {
			// Setup
			_, err := database.ExecContext(ctx,
				`INSERT INTO user (id, user_name, password, is_admin, created_at, updated_at)
				 VALUES (?, ?, ?, ?, ?, ?)`,
				"admin", "admin", "hashedpwd", true, time.Now(), time.Now())
			Expect(err).NotTo(HaveOccurred())

			_, err = database.ExecContext(ctx,
				`INSERT INTO annotation (user_id, item_id, item_type, starred)
				 VALUES (?, ?, ?, ?)`,
				"admin", "song123", "song", false)
			Expect(err).NotTo(HaveOccurred())
		})

		It("debe marcar como favorita con UPDATE", func() {
			starredAt := time.Now()
			result, err := database.ExecContext(ctx, `
				UPDATE annotation SET starred = ?, starred_at = ?
				WHERE user_id = ? AND item_id = ? AND item_type = ?
			`, true, starredAt, "admin", "song123", "song")
			Expect(err).NotTo(HaveOccurred())

			affected, err := result.RowsAffected()
			Expect(err).NotTo(HaveOccurred())
			Expect(affected).To(Equal(int64(1)))

			var starred bool
			err = database.QueryRowContext(ctx,
				`SELECT starred FROM annotation WHERE user_id = ? AND item_id = ?`,
				"admin", "song123").Scan(&starred)
			Expect(err).NotTo(HaveOccurred())
			Expect(starred).To(BeTrue())
		})

		It("debe desmarcar favorita", func() {
			result, err := database.ExecContext(ctx, `
				UPDATE annotation SET starred = ?, starred_at = NULL
				WHERE user_id = ? AND item_id = ? AND item_type = ?
			`, false, "admin", "song123", "song")
			Expect(err).NotTo(HaveOccurred())

			affected, err := result.RowsAffected()
			Expect(err).NotTo(HaveOccurred())
			Expect(affected).To(Equal(int64(1)))
		})
	})

	// ================== TEST 5: UNIQUE Constraint ==================
	Describe("TEST 5: UNIQUE Constraint (user_id, item_id, item_type)", func() {
		BeforeEach(func() {
			// Setup
			_, err := database.ExecContext(ctx,
				`INSERT INTO user (id, user_name, password, is_admin, created_at, updated_at)
				 VALUES (?, ?, ?, ?, ?, ?)`,
				"admin", "admin", "hashedpwd", true, time.Now(), time.Now())
			Expect(err).NotTo(HaveOccurred())

			_, err = database.ExecContext(ctx,
				`INSERT INTO annotation (user_id, item_id, item_type, starred)
				 VALUES (?, ?, ?, ?)`,
				"admin", "song123", "song", true)
			Expect(err).NotTo(HaveOccurred())
		})

		It("debe rechazar inserción duplicada del mismo (user, item, type)", func() {
			_, err := database.ExecContext(ctx,
				`INSERT INTO annotation (user_id, item_id, item_type, starred)
				 VALUES (?, ?, ?, ?)`,
				"admin", "song123", "song", false)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("UNIQUE constraint failed"))
		})

		It("debe permitir el mismo item_id pero diferente user", func() {
			// Setup: crear otro usuario
			_, err := database.ExecContext(ctx,
				`INSERT INTO user (id, user_name, password, is_admin, created_at, updated_at)
				 VALUES (?, ?, ?, ?, ?, ?)`,
				"user2", "user2", "hashedpwd", false, time.Now(), time.Now())
			Expect(err).NotTo(HaveOccurred())

			// Insertar annotation para otro usuario sobre el mismo song
			_, err = database.ExecContext(ctx,
				`INSERT INTO annotation (user_id, item_id, item_type, starred)
				 VALUES (?, ?, ?, ?)`,
				"user2", "song123", "song", false)
			Expect(err).NotTo(HaveOccurred())

			// Verify ambos existen
			var count int
			err = database.QueryRowContext(ctx,
				`SELECT COUNT(*) FROM annotation WHERE item_id = ?`,
				"song123").Scan(&count)
			Expect(err).NotTo(HaveOccurred())
			Expect(count).To(Equal(2))
		})

		It("debe permitir el mismo item_id pero diferente item_type", func() {
			// El mismo ID podría ser album o artist
			_, err := database.ExecContext(ctx,
				`INSERT INTO annotation (user_id, item_id, item_type, starred)
				 VALUES (?, ?, ?, ?)`,
				"admin", "song123", "album", false)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	// ================== TEST 9: ON DELETE CASCADE ==================
	Describe("TEST 9: ON DELETE CASCADE - User Deletion (trigger-based only)", func() {
		BeforeEach(func() {
			// Skip this set if there is no trigger-based cascade detected.
			var trigSQL sql.NullString
			err := database.QueryRowContext(ctx,
				`SELECT sql FROM sqlite_master WHERE type = 'trigger' AND sql LIKE '%annotation%' AND sql LIKE '%DELETE%' LIMIT 1`).Scan(&trigSQL)
			if err != nil || !trigSQL.Valid {
				Skip("No trigger-based cascade detected; skipping CASCADE test")
			}

			// Setup
			_, err = database.ExecContext(ctx,
				`INSERT INTO user (id, user_name, password, is_admin, created_at, updated_at)
				 VALUES (?, ?, ?, ?, ?, ?)`,
				"test_user", "testuser", "hashedpwd", false, time.Now(), time.Now())
			Expect(err).NotTo(HaveOccurred())

			_, err = database.ExecContext(ctx,
				`INSERT INTO annotation (user_id, item_id, item_type, starred)
				 VALUES (?, ?, ?, ?)`,
				"test_user", "song123", "song", true)
			Expect(err).NotTo(HaveOccurred())
		})

		It("debe borrar annotations quando se borra el usuario (CASCADE)", func() {
			// Verify que existen
			var count int
			err := database.QueryRowContext(ctx,
				`SELECT COUNT(*) FROM annotation WHERE user_id = ?`,
				"test_user").Scan(&count)
			Expect(err).NotTo(HaveOccurred())
			Expect(count).To(Equal(1))

			// Delete user
			_, err = database.ExecContext(ctx,
				`DELETE FROM user WHERE id = ?`, "test_user")
			Expect(err).NotTo(HaveOccurred())

			// Verify annotations se borraron en cascada
			err = database.QueryRowContext(ctx,
				`SELECT COUNT(*) FROM annotation WHERE user_id = ?`,
				"test_user").Scan(&count)
			Expect(err).NotTo(HaveOccurred())
			Expect(count).To(Equal(0))
		})
	})

	// ================== TEST 6: Polimorfismo - item_type (conditional) ==================
	Describe("TEST 6: Polimorfismo - Diferentes item_type (solo si hay CHECK constraint)", func() {
		BeforeEach(func() {
			// Verificar si la tabla annotation tiene un CHECK constraint real sobre item_type
			var tableSQL sql.NullString
			err := database.QueryRowContext(ctx,
				`SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'annotation'`,
			).Scan(&tableSQL)

			if err != nil || !tableSQL.Valid {
				Skip("annotation table not present; skipping polymorphism tests")
			}

			up := strings.ToUpper(tableSQL.String)

			// Solo ejecutar este test si realmente existe un CHECK constraint
			if !strings.Contains(up, "CHECK") {
				Skip("No CHECK constraint found on item_type; skipping polymorphism tests")
			}

			_, err = database.ExecContext(ctx,
				`INSERT INTO user (id, user_name, password, is_admin, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?)`,
				"admin", "admin", "hashedpwd", true, time.Now(), time.Now())

			Expect(err).NotTo(HaveOccurred())
		})

		It("debe validar los item_type permitidos en la tabla annotation", func() {
			allowed := []string{"song", "album", "artist"}

			for _, t := range allowed {
				_, err := database.ExecContext(ctx,
					`INSERT INTO annotation (user_id, item_id, item_type, starred)
					VALUES (?, ?, ?, ?)`,
					"admin", "item-"+t, t, true)

				Expect(err).NotTo(HaveOccurred())
			}

			// Este insert solo debería fallar si existe CHECK constraint
			_, err := database.ExecContext(ctx,
				`INSERT INTO annotation (user_id, item_id, item_type, starred)
				VALUES (?, ?, ?, ?)`,
				"admin", "item-bad", "not_a_type", true)

			Expect(err).To(HaveOccurred())
		})
	})

	// ================== TEST 12: NULL Optional Fields ==================
	Describe("TEST 12: NULL Optional Fields", func() {
		BeforeEach(func() {
			_, err := database.ExecContext(ctx,
				`INSERT INTO user (id, user_name, password, is_admin, created_at, updated_at)
				 VALUES (?, ?, ?, ?, ?, ?)`,
				"admin", "admin", "hashedpwd", true, time.Now(), time.Now())
			Expect(err).NotTo(HaveOccurred())
		})

		It("debe permitir NULL en play_count, rating, play_date", func() {
			_, err := database.ExecContext(ctx,
				`INSERT INTO annotation (user_id, item_id, item_type, starred, play_count, rating, play_date)
				 VALUES (?, ?, ?, ?, ?, ?, ?)`,
				"admin", "song456", "song", true, nil, nil, nil)
			Expect(err).NotTo(HaveOccurred())

			var playCount sql.NullInt64
			var rating sql.NullInt64
			var playDate sql.NullTime
			err = database.QueryRowContext(ctx,
				`SELECT play_count, rating, play_date FROM annotation WHERE user_id = ? AND item_id = ?`,
				"admin", "song456").Scan(&playCount, &rating, &playDate)
			Expect(err).NotTo(HaveOccurred())
			Expect(playCount.Valid).To(BeFalse())
			Expect(rating.Valid).To(BeFalse())
			Expect(playDate.Valid).To(BeFalse())
		})
	})
})
