package persistence_test

import (
	"context"
	"database/sql"
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

	// ================== TEST 6: ON DELETE CASCADE ==================
	Describe("TEST 6: ON DELETE CASCADE - User Deletion", func() {
		BeforeEach(func() {
			// Setup
			_, err := database.ExecContext(ctx,
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

	// ================== TEST 7: Multi-usuario Aislamiento ==================
	Describe("TEST 7: Multi-usuario - Aislamiento de Datos", func() {
		BeforeEach(func() {
			// Setup: crear 2 usuarios
			_, err := database.ExecContext(ctx,
				`INSERT INTO user (id, user_name, password, is_admin, created_at, updated_at)
				 VALUES (?, ?, ?, ?, ?, ?)`,
				"admin", "admin", "hashedpwd", true, time.Now(), time.Now())
			Expect(err).NotTo(HaveOccurred())

			_, err = database.ExecContext(ctx,
				`INSERT INTO user (id, user_name, password, is_admin, created_at, updated_at)
				 VALUES (?, ?, ?, ?, ?, ?)`,
				"user2", "user2", "hashedpwd", false, time.Now(), time.Now())
			Expect(err).NotTo(HaveOccurred())

			// Ambos marcan la misma canción pero diferente estado
			_, err = database.ExecContext(ctx,
				`INSERT INTO annotation (user_id, item_id, item_type, starred)
				 VALUES (?, ?, ?, ?)`,
				"admin", "song123", "song", true)
			Expect(err).NotTo(HaveOccurred())

			_, err = database.ExecContext(ctx,
				`INSERT INTO annotation (user_id, item_id, item_type, starred)
				 VALUES (?, ?, ?, ?)`,
				"user2", "song123", "song", false)
			Expect(err).NotTo(HaveOccurred())
		})

		It("admin debe ver starred=true", func() {
			var starred bool
			err := database.QueryRowContext(ctx,
				`SELECT starred FROM annotation WHERE user_id = ? AND item_id = ?`,
				"admin", "song123").Scan(&starred)
			Expect(err).NotTo(HaveOccurred())
			Expect(starred).To(BeTrue())
		})

		It("user2 debe ver starred=false", func() {
			var starred bool
			err := database.QueryRowContext(ctx,
				`SELECT starred FROM annotation WHERE user_id = ? AND item_id = ?`,
				"user2", "song123").Scan(&starred)
			Expect(err).NotTo(HaveOccurred())
			Expect(starred).To(BeFalse())
		})
	})

	// ================== TEST 8: Timestamps Accuracy ==================
	Describe("TEST 8: Timestamps - starred_at Precision", func() {
		BeforeEach(func() {
			_, err := database.ExecContext(ctx,
				`INSERT INTO user (id, user_name, password, is_admin, created_at, updated_at)
				 VALUES (?, ?, ?, ?, ?, ?)`,
				"admin", "admin", "hashedpwd", true, time.Now(), time.Now())
			Expect(err).NotTo(HaveOccurred())
		})

		It("debe actualizar starred_at al marcar favorita", func() {
			t1 := time.Now()
			_, err := database.ExecContext(ctx,
				`INSERT INTO annotation (user_id, item_id, item_type, starred, starred_at)
				 VALUES (?, ?, ?, ?, ?)`,
				"admin", "song123", "song", true, t1)
			Expect(err).NotTo(HaveOccurred())

			// Esperar un poco
			time.Sleep(100 * time.Millisecond)

			// Update starred nuevamente
			t2 := time.Now()
			_, err = database.ExecContext(ctx,
				`UPDATE annotation SET starred = ?, starred_at = ?
				 WHERE user_id = ? AND item_id = ?`,
				true, t2, "admin", "song123")
			Expect(err).NotTo(HaveOccurred())

			// Verify que el timestamp es más reciente
			var starredAt time.Time
			err = database.QueryRowContext(ctx,
				`SELECT starred_at FROM annotation WHERE user_id = ? AND item_id = ?`,
				"admin", "song123").Scan(&starredAt)
			Expect(err).NotTo(HaveOccurred())
			Expect(starredAt.After(t1)).To(BeTrue())
		})

		It("debe permitir NULL en starred_at cuando no está starred", func() {
			_, err := database.ExecContext(ctx,
				`INSERT INTO annotation (user_id, item_id, item_type, starred, starred_at)
				 VALUES (?, ?, ?, ?, ?)`,
				"admin", "song123", "song", false, nil)
			Expect(err).NotTo(HaveOccurred())

			var starredAt sql.NullTime
			err = database.QueryRowContext(ctx,
				`SELECT starred_at FROM annotation WHERE user_id = ? AND item_id = ?`,
				"admin", "song123").Scan(&starredAt)
			Expect(err).NotTo(HaveOccurred())
			Expect(starredAt.Valid).To(BeFalse())
		})
	})

	// ================== TEST 9: Idempotencia - Upsert Pattern ==================
	Describe("TEST 9: Idempotencia - Star/Unstar Múltiples Veces", func() {
		BeforeEach(func() {
			_, err := database.ExecContext(ctx,
				`INSERT INTO user (id, user_name, password, is_admin, created_at, updated_at)
				 VALUES (?, ?, ?, ?, ?, ?)`,
				"admin", "admin", "hashedpwd", true, time.Now(), time.Now())
			Expect(err).NotTo(HaveOccurred())
		})

		It("debe ser idempotente al marcar favorita múltiples veces", func() {
			// First star
			_, err := database.ExecContext(ctx,
				`INSERT OR REPLACE INTO annotation (user_id, item_id, item_type, starred, starred_at)
				 VALUES (?, ?, ?, ?, ?)`,
				"admin", "song123", "song", true, time.Now())
			Expect(err).NotTo(HaveOccurred())

			// Star again (should succeed, not duplicate)
			_, err = database.ExecContext(ctx,
				`INSERT OR REPLACE INTO annotation (user_id, item_id, item_type, starred, starred_at)
				 VALUES (?, ?, ?, ?, ?)`,
				"admin", "song123", "song", true, time.Now())
			Expect(err).NotTo(HaveOccurred())

			// Verify solo una annotation existe
			var count int
			err = database.QueryRowContext(ctx,
				`SELECT COUNT(*) FROM annotation WHERE user_id = ? AND item_id = ?`,
				"admin", "song123").Scan(&count)
			Expect(err).NotTo(HaveOccurred())
			Expect(count).To(Equal(1))
		})
	})

	// ================== TEST 10: Polimorfismo - item_type ==================
	Describe("TEST 10: Polimorfismo - Diferentes item_type", func() {
		BeforeEach(func() {
			_, err := database.ExecContext(ctx,
				`INSERT INTO user (id, user_name, password, is_admin, created_at, updated_at)
				 VALUES (?, ?, ?, ?, ?, ?)`,
				"admin", "admin", "hashedpwd", true, time.Now(), time.Now())
			Expect(err).NotTo(HaveOccurred())
		})

		It("debe permitir marcar como favorita song, album, y artist", func() {
			types := []string{"song", "album", "artist", "folder"}
			for _, itemType := range types {
				_, err := database.ExecContext(ctx,
					`INSERT INTO annotation (user_id, item_id, item_type, starred)
					 VALUES (?, ?, ?, ?)`,
					"admin", "item"+itemType, itemType, true)
				Expect(err).NotTo(HaveOccurred())
			}

			// Verify cada uno existe
			var count int
			err := database.QueryRowContext(ctx,
				`SELECT COUNT(*) FROM annotation WHERE user_id = ?`,
				"admin").Scan(&count)
			Expect(err).NotTo(HaveOccurred())
			Expect(count).To(Equal(4))
		})
	})
})
