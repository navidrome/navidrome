# Lab 2 — Integration Test Plan: BE-INT

**Integration level:** 1
**Boundary tested:** Backend ↔ Database (real persistence layer ↔ real migrated SQLite)
**Scenario:** Star an existing song successfully.

## Steps (scenario)

1. Send an authenticated star request for an existing song ID.
2. Read favorites back via the backend API / DB verification.

**Expected result:** the song is starred and appears in the favorites/starred list.

> **Constraint:** create new test files only — do not modify any existing test. This plan adds a
> brand-new file and only *reads* the suite's already-seeded fixtures; nothing existing is changed.

---

## For readers new to Go / Go testing

If you know software development in general but have never touched Go, Go's testing tools, or
this codebase, this section gives you enough mental model to read the test. The ideas are the
same ones you already know from other languages — only the names differ.

### The big picture in one paragraph

"Star a song" in Navidrome means *mark it as a favorite*. The backend stores that fact in a
database table called `annotation` (one row per user + item, with a `starred` boolean and a
`starred_at` timestamp). Our test asks a simple question: **if I tell the real backend code to
star a real song in a real database, and then ask the backend to list my favorites, does the
song show up?** That round-trip — write through real code, read back through real code — is what
makes it an *integration* test rather than a unit test.

### Go testing & Ginkgo, mapped to tools you already know

Go has a built-in test runner (`go test`). On top of it, Navidrome uses **Ginkgo**, a
behavior-style framework. If you've used JUnit, Jest, RSpec, or Mocha, this table will feel
familiar:

| In the test you'll see | What it means | Equivalent you may know |
|------------------------|---------------|-------------------------|
| `Describe("...", func(){...})` | Groups related test cases | `describe()` (Jest), test class (JUnit) |
| `It("...", func(){...})` | One individual test case | `it()` / `test()` / `@Test` |
| `BeforeEach(func(){...})` | Runs before *each* case — setup | `beforeEach` / `@BeforeEach` |
| `AfterEach(func(){...})` | Runs after *each* case — teardown/cleanup | `afterEach` / `@AfterEach` |
| `Expect(x).To(BeTrue())` | An assertion | `expect(x).toBe(true)` / `assertTrue(x)` |
| `Expect(err).ToNot(HaveOccurred())` | Assert "no error happened" | `assertDoesNotThrow` |
| `BeforeSuite` | One-time setup for the whole file/suite | `@BeforeAll` |

`func(){...}` is just Go's syntax for an anonymous function (a lambda/closure). Ginkgo tests are
written by passing these little functions into `Describe`/`It`/etc.

In Go, **any file ending in `_test.go` is test code** and is compiled only when testing — it never
ships in the product binary. That's why adding our file changes nothing about the running app.

### The building blocks this test uses

- **Repository** (`repo` in the code): an object that reads/writes one kind of data in the
  database. Think DAO, or a single-table ORM repository. `MediaFileRepository` is "the songs
  table, plus the favorites/ratings attached to songs." Its methods (`Get`, `GetAll`, `SetStar`)
  are the *real backend code* — the same methods the web API calls in production.
- **`SetStar(true, id)` / `SetStar(false, id)`**: the backend method that marks/unmarks a favorite.
  It does a real SQL "update the row, or insert it if it doesn't exist yet" (an *upsert*) against
  the `annotation` table.
- **`Get(id)`**: fetch one song *with its favorite/rating info merged in* (via a SQL `JOIN`). So
  `song.Starred` tells you whether the current user has favorited it.
- **`GetAll(... starred = true ...)`**: fetch every song the current user has favorited — i.e. read
  the favorites list. This is literally the query behind the app's "Starred / Favorites" screen.
- **Fixtures / seed data**: before the tests run, the suite inserts a fixed set of sample songs and
  users into the database (e.g. song `"1003"` = "Radioactivity", user `adminUser`). These are
  pre-existing rows we can rely on — like a test database snapshot. We *read* them; we don't change
  the setup.
- **"The authenticated user"**: favorites are per-user. In the running app, login middleware figures
  out *who* you are and attaches that to the request. In the test there's no HTTP login step;
  instead we attach a known seeded user to a Go **`context`** (a request-scoped bag of values that
  Go passes around). `request.WithUser(ctx, adminUser)` is the test's stand-in for "logged in as
  this user." The backend code reads the user from that context exactly as it would in production.
- **Real migrated database**: the test boots an actual SQLite database and runs Navidrome's real
  schema migrations against it (in memory, so it's fast and disposable). No fake/hand-written
  schema — the same tables the real app uses.

### How to read the test, in plain steps

Each test case follows the classic **Arrange → Act → Assert** shape:

1. **Arrange** — get a repository wired to the real DB, acting as the seeded user; force a known
   starting state (for the star case, make sure the song is *not* yet starred).
2. **Act** — call the real backend method: `SetStar(true, songID)` to favorite it (or
   `SetStar(false, songID)` to remove it).
3. **Assert** — read it back two ways and check both agree: `Get(songID).Starred` is now `true`,
   and the song appears in the favorites list returned by `GetAll(... starred=true ...)`.
4. **Cleanup** — `AfterEach` un-stars the song again so the shared in-memory database is left clean
   for the next test (tests must not leak state into each other).

The unstar test (BE-INT-02) is the mirror image: arrange the song as *starred*, act with
`SetStar(false, ...)`, then assert it's no longer starred and no longer in the favorites list.

---

## Context: what already exists vs. the real gap

Three pieces of star/favorites test code exist today. None of them exercises the real
backend persistence code against a real database, which is exactly the gap this test fills.

| File | What it actually is | Real backend code? | Real DB? |
|------|--------------------|--------------------|----------|
| `server/subsonic/media_annotation_test.go` | **Unit** test — `tests.MockDataStore` + spy repo | Handler only | No (fully mocked) |
| `tests/db/annotation_star_test.go` | **DB-only** test — raw SQL against a *hand-written* schema | No | A throwaway schema, not Navidrome's migrations |
| `persistence/sql_annotations_test.go` | Real-DB test, but only covers the **filter SQL** + album rating | Filter helpers | Yes |

So despite the `tests/db` file being *labeled* "integration", nothing currently exercises
**real backend persistence code → real migrated SQLite, then reads it back**. That round-trip
is the BE-INT-01 scenario and the legitimate value-add.

---

## Recommended approach — repository-level round-trip

The `persistence` package test suite already boots a **real migrated DB** via `db.Init()` and
seeds real fixtures in `BeforeSuite`: known songs (`"1001"` A Day In A Life, `"1003"`
Radioactivity, etc.), albums, and users (`adminUser`, ID `"userid"`). We reuse that instead of
hand-rolling a schema.

**New file:** `persistence/star_song_integration_test.go` (lives in the `persistence` suite so it
inherits the migrated DB + fixtures).

```go
ctx := request.WithUser(context.Background(), adminUser)   // the authenticated user
repo := NewMediaFileRepository(ctx, GetDBXBuilder())
```

Steps mapped to the scenario:

1. **Pre-condition** — pick a seeded existing song (e.g. ID `"1001"`); assert
   `repo.Get("1001").Starred == false`.
2. **Act (the "star request")** — `Expect(repo.SetStar(true, "1001")).To(Succeed())` → real
   upsert into the `annotation` table.
3. **Verify via DB read-back** (two complementary assertions):
   - `repo.Get("1001")` → `.Starred == true` **and** `.StarredAt` is non-zero (proves the row +
     timestamp persisted through the `LeftJoin annotation` query path).
   - `repo.GetAll(model.QueryOptions{Filters: squirrel.Eq{"starred": true}})` — confirms `"1001"`
     **appears in the starred/favorites list**. This is the *same query the `getStarred` Subsonic
     endpoint runs* (`server/subsonic/album_lists.go:120` `getStarredItems` → `MediaFile.GetAll`
     with the starred filter), so it is a faithful "read favorites" check.
4. **Cleanup** — `AfterEach`: `repo.SetStar(false, "1001")` (or delete the annotation row) so the
   shared in-memory DB stays clean for other specs.

### Why this is valid / what it proves

It exercises the real `sqlRepository.SetStar` → real squirrel SQL → real migrated `annotation`
table, then reads back through the real annotated-query join — proving the backend's write and
the backend's read agree against an actual database. The per-user scoping
(`loggedUser(ctx).ID`) stands in for the authenticated user, which is exactly what the auth
middleware produces downstream.

### Value for the product

Catches regressions that unit tests cannot: a broken migration, a wrong column name, a faulty
upsert, or a starred-filter query that drifts from the write path. If this round-trip fails, the
favorites feature is broken in production even if every unit test passes.

---

## Stronger (heavier) variant — full HTTP + auth

For the literal "authenticated HTTP star request", wire a real `Router` with a real
`persistence.New(...)` DataStore, seed a user *with a password*, then `router.ServeHTTP` a
`GET /rest/star?u=...&t=...&s=...&id=1001` followed by `GET /rest/getStarred`. This adds the auth
middleware + handler layers but costs noticeably more setup (real Router deps, password/token
computation). Good as one "showcase" backend test; overkill for all five lab tests.

---

## How to run

From the project root. Both BE-INT-01 (star) and BE-INT-02 (unstar) live in the same file,
`persistence/star_song_integration_test.go`, under the `Star/Unstar an existing song` describe:

```bash
# Run both star and unstar specs
go test -tags netgo,sqlite_fts5 -run TestPersistence ./persistence/ \
  --ginkgo.focus="Star/Unstar an existing song"

# Run only BE-INT-01 (star)
go test -tags netgo,sqlite_fts5 -run TestPersistence ./persistence/ \
  --ginkgo.focus="Star an existing song"

# Run only BE-INT-02 (unstar)
go test -tags netgo,sqlite_fts5 -run TestPersistence ./persistence/ \
  --ginkgo.focus="Unstar a previously starred song"
```

Expected output: `ok  github.com/navidrome/navidrome/persistence`.

### What each flag does

- `-tags netgo,sqlite_fts5` — **required.** The DB migrations create an FTS5 virtual table; without
  this build tag the SQLite driver lacks the `fts5` module and the schema fails to build
  (`no such module: fts5`). These are the project's standard build tags (`GO_BUILD_TAGS` in the
  Makefile).
- `-run TestPersistence` — scopes the run to the real suite entry point, sidestepping the
  pre-existing duplicate `RunSpecs` in `persistence/annotation_star_test.go` that otherwise fails
  the whole package with "Rerunning Suite" (see the note below).
- `--ginkgo.focus="..."` — runs only the specs whose description text matches the given pattern
  (e.g. one of the three focuses shown above). Omit it to run every spec in the suite.

To run this spec alongside the rest of the persistence suite, drop the focus flag:

```bash
go test -tags netgo,sqlite_fts5 -run TestPersistence ./persistence/
```

> **Pre-existing issue (not introduced by this test):** the file
> `persistence/annotation_star_test.go` (committed in Lab 1, a stray duplicate of
> `tests/db/annotation_star_test.go`) declares a *second* `RunSpecs` inside the `persistence`
> package. Ginkgo allows only one `RunSpecs` per test binary, so a bare `go test ./persistence/`
> fails with "Rerunning Suite". Scoping the run to `-run TestPersistence` (the real suite entry
> point) sidesteps the duplicate and runs this spec cleanly. The proper fix is to remove/relocate
> that stray file, but it is left untouched here per the "do not modify existing tests" constraint.
