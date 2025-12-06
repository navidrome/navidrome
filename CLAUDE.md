# Navidrome AI Assistant Guide

This document provides comprehensive information about the Navidrome codebase for AI assistants working on development tasks.

## Table of Contents

- [Project Overview](#project-overview)
- [Technology Stack](#technology-stack)
- [Codebase Structure](#codebase-structure)
- [Architecture & Patterns](#architecture--patterns)
- [Development Workflow](#development-workflow)
- [Testing](#testing)
- [Code Conventions](#code-conventions)
- [Common Tasks](#common-tasks)
- [Database](#database)
- [Plugin System](#plugin-system)
- [Frontend (UI)](#frontend-ui)
- [Important Notes](#important-notes)

## Project Overview

Navidrome is an open-source web-based music collection server and streamer. It allows users to stream their personal music library from any browser or mobile device, similar to services like Spotify but self-hosted.

**Key Features:**
- Handles very large music collections
- Streams virtually any audio format
- Multi-user support with individual play counts, playlists, favorites
- Subsonic API compatibility
- Modern React-based web interface
- Transcoding on the fly with FFmpeg
- Multi-library support
- Plugin system (WASM-based)

**Repository:** https://github.com/navidrome/navidrome

## Technology Stack

### Backend (Go)
- **Language:** Go 1.25
- **HTTP Router:** go-chi/chi
- **Database:** SQLite with mattn/go-sqlite3
- **Query Builder:** Masterminds/squirrel
- **Dependency Injection:** Google Wire (compile-time)
- **Testing:** Ginkgo/Gomega (BDD framework)
- **Configuration:** spf13/viper
- **CLI:** spf13/cobra
- **Migrations:** pressly/goose
- **Audio Metadata:** dhowden/tag (forked)
- **Plugin Runtime:** tetratelabs/wazero (WebAssembly)

### Frontend (JavaScript/React)
- **Framework:** React 17
- **Admin Framework:** react-admin 3.19
- **State Management:** Redux + Redux Saga
- **Build Tool:** Vite
- **UI Library:** Material-UI v4
- **Testing:** Vitest
- **Node Version:** 24 (see .nvmrc)

### Build & Development
- **Build System:** Make
- **Package Manager (UI):** npm
- **Cross-compilation:** Docker with buildx
- **CI/CD:** GitHub Actions
- **Linting (Go):** golangci-lint
- **Linting (JS):** ESLint + Prettier

## Codebase Structure

```
navidrome/
├── adapters/          # External library adapters (e.g., taglib for audio metadata)
├── cmd/               # CLI commands and Wire dependency injection
├── conf/              # Configuration management (Viper-based)
├── consts/            # Constants and build-time variables
├── core/              # Core business logic services
│   ├── agents/        # External metadata providers
│   ├── artwork/       # Artwork extraction and caching
│   ├── auth/          # Authentication and authorization
│   ├── ffmpeg/        # FFmpeg wrapper for transcoding
│   ├── metrics/       # Prometheus metrics
│   ├── playback/      # Remote playback control (MPV)
│   ├── scrobbler/     # Scrobbling to LastFM, ListenBrainz
│   └── lyrics/        # Lyrics management
├── db/                # Database initialization and migrations
├── git/               # Git hooks (pre-commit, pre-push)
├── log/               # Structured logging
├── model/             # Data models and repository interfaces
├── persistence/       # SQL repository implementations
├── plugins/           # WASM-based plugin system
│   ├── api/           # Plugin API protobuf definitions
│   ├── host/          # Host services for plugins
│   └── examples/      # Example plugin implementations
├── resources/         # Embedded static assets and translations
├── scanner/           # Music library scanning
├── scheduler/         # Cron-based task scheduler
├── server/            # HTTP server and API routers
│   ├── subsonic/      # Subsonic API implementation
│   ├── nativeapi/     # Native REST API
│   ├── public/        # Public endpoints
│   └── events/        # WebSocket event broker
├── tests/             # Shared test utilities
├── ui/                # React frontend
│   └── src/
│       ├── actions/       # Redux actions
│       ├── reducers/      # Redux reducers
│       ├── album/         # Album-related components
│       ├── artist/        # Artist-related components
│       ├── audioplayer/   # Audio player component
│       ├── playlist/      # Playlist components
│       └── ...
└── utils/             # Shared utilities
```

### Key Directory Purposes

**Core Domains:**
- **model/** - Defines domain entities (Album, Artist, MediaFile, User) and repository interfaces
- **persistence/** - Implements data access with SQL repositories (CRUD operations)
- **scanner/** - Handles music library scanning in multiple phases
- **server/** - HTTP routing, API handlers, WebSocket events
- **core/** - Business logic services (streaming, archiving, playlists, etc.)

**Infrastructure:**
- **db/** - Database setup, migrations (90+ migration files)
- **conf/** - Configuration with 80+ settings
- **cmd/** - Entry points and Wire DI setup
- **log/** - Context-aware logging
- **scheduler/** - Task scheduling wrapper

**Extensions:**
- **plugins/** - WASM plugin system with permission-based security
- **adapters/** - External library wrappers

## Architecture & Patterns

### Layered Architecture

```
┌─────────────────────────────────────┐
│   HTTP Layer (Chi Router)          │
├─────────────────────────────────────┤
│   Subsonic/Native API Routers      │
│   (Authentication, Validation)      │
├─────────────────────────────────────┤
│   Core Services (Business Logic)   │
├─────────────────────────────────────┤
│   Persistence (Repository Pattern)  │
├─────────────────────────────────────┤
│   Database (SQLite)                 │
└─────────────────────────────────────┘
```

### Key Architectural Patterns

#### 1. Dependency Injection (Wire)

**ALL dependency injection is done via Google Wire** - a compile-time DI framework:

- Providers defined in `wire_providers.go` files
- Bindings use `wire.NewSet()` for grouping
- Interfaces bound with `wire.Bind()`
- Code generation in `cmd/wire_gen.go`
- **Never use manual DI** - always use Wire

Example:
```go
// In some_service.go
func NewSomeService(repo Repository) Service {
    return &someService{repo: repo}
}

// In wire_providers.go
var Set = wire.NewSet(
    NewSomeService,
    wire.Bind(new(Service), new(*someService)),
)
```

#### 2. Repository Pattern

All data access uses the repository pattern:

- **Interface:** model.DataStore provides access to all repositories
- **Implementation:** Each entity has a repository (e.g., AlbumRepository, ArtistRepository)
- **Base Class:** sqlRepository provides common CRUD methods
- **Query Building:** Masterminds Squirrel for type-safe SQL construction
- **Transactions:** WithTx() and WithTxImmediate() for transaction support

Example:
```go
// Get repository from datastore
ds := ctx.Value("datastore").(model.DataStore)
repo := ds.Album(ctx)

// Use repository methods
album, err := repo.Get("album-id")
albums, err := repo.GetAll(model.QueryOptions{
    Sort:   "name",
    Order:  "asc",
    Offset: 0,
    Max:    50,
})
```

#### 3. Context-Based Architecture

Context is used extensively throughout the codebase:

- **User Injection:** request.WithUser() adds user to context
- **Authorization:** Repositories filter data based on user's library access
- **Cancellation:** All long-running operations respect context cancellation
- **Logging:** log.NewContext() creates context-aware loggers

#### 4. Service Pattern

Business logic is encapsulated in service interfaces:

```go
type MediaStreamer interface {
    Stream(ctx context.Context, id string, maxBitRate int) (io.ReadCloser, error)
}
```

Services are injected via Wire and implement core functionality.

#### 5. Event-Driven Communication

- **Event Broker:** events.Broker for real-time updates
- **WebSocket Integration:** Server-Sent Events via events package
- **Plugin Integration:** Plugins can subscribe to events

### Request Flow

1. **HTTP Request** → Chi Router
2. **Middleware** → Authentication, user injection, logging
3. **Handler** → Validation, parameter extraction
4. **Service** → Business logic execution
5. **Repository** → Database operations
6. **Response** → JSON/XML serialization

### Multi-Library Support

Navidrome supports multiple music libraries:

- Every entity has a `LibraryID` field
- Users are associated with libraries via `user_libraries` table
- Repositories automatically filter by user's accessible libraries
- Scanner operates per-library with selective scanning support

## Development Workflow

### Initial Setup

```bash
# Install dependencies (Go modules + Node packages)
make setup

# This will:
# - Download Go dependencies
# - Install golangci-lint
# - Setup git hooks (pre-commit, pre-push)
# - Install npm dependencies in ui/
```

### Development Mode

```bash
# Start both backend and frontend with hot-reload
make dev

# Or start them separately:
make server   # Backend only (with reflex for auto-reload)
cd ui && npm start  # Frontend only
```

The development server runs on http://localhost:4533

### Building

```bash
# Build both frontend and backend
make build

# Build frontend only
make buildjs

# Build backend only (requires frontend to be built first)
go build -tags=netgo
```

### Testing

```bash
# Run all tests (Go + JS)
make testall

# Run Go tests only
make test

# Run Go tests with race detector
make test-race

# Run specific package tests
make test PKG=./server/subsonic

# Run tests in watch mode
make watch

# Run JS tests
make test-js

# Validate i18n translations
make test-i18n
```

### Linting & Formatting

```bash
# Lint all code (Go + JS)
make lintall

# Lint Go code only
make lint

# Format code automatically
make format
```

### Git Hooks

Git hooks are set up automatically via `make setup`:

- **pre-commit:** Checks Go formatting with goimports (rejects unformatted code)
- **pre-push:** Runs `make pre-push` which executes lintall + testall

**Important:** The pre-push hook will prevent pushes if tests or linting fail!

### Wire Dependency Injection

When adding new dependencies or services:

```bash
# Regenerate Wire code
make wire

# This runs: go tool wire gen -tags=netgo ./...
```

**Always regenerate Wire code** after modifying:
- Provider functions
- Wire bindings
- Injector signatures

### Database Migrations

```bash
# Create a new SQL migration
make migration-sql name=add_some_feature

# Create a new Go migration
make migration-go name=migrate_complex_data

# Migrations are in db/migrations/
```

### Plugin Development

```bash
# Generate plugin code from protobuf
make plugin-gen

# Build example plugins
make plugin-examples

# Build test plugins
make plugin-tests

# Clean plugin builds
make plugin-clean
```

### Cross-compilation

```bash
# List supported platforms
make docker-platforms

# Build for specific platform
make docker-build PLATFORMS=linux/amd64

# Build Docker image
make docker-image

# Build MSI installers for Windows
make docker-msi
```

## Testing

### Testing Framework

Navidrome uses **Ginkgo/Gomega** for BDD-style testing:

```go
package mypackage_test

import (
    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
    "testing"
)

func TestMyPackage(t *testing.T) {
    RegisterFailHandler(Fail)
    RunSpecs(t, "MyPackage Suite")
}

var _ = Describe("MyService", func() {
    var (
        service *MyService
        repo    *mockRepository
    )

    BeforeEach(func() {
        repo = &mockRepository{}
        service = NewMyService(repo)
    })

    Describe("DoSomething", func() {
        It("should succeed with valid input", func() {
            result, err := service.DoSomething("valid")

            Expect(err).ToNot(HaveOccurred())
            Expect(result).To(Equal("expected"))
        })

        It("should fail with invalid input", func() {
            _, err := service.DoSomething("invalid")

            Expect(err).To(HaveOccurred())
        })
    })
})
```

### Test Patterns

#### 1. Suite Tests

Use `*_suite_test.go` for test setup:

```go
// persistence_suite_test.go
func TestPersistence(t *testing.T) {
    tests.Init(t, true)
    conf.Server.DbPath = "file::memory:?cache=shared&_foreign_keys=on"
    defer db.Init(context.Background())()

    RegisterFailHandler(Fail)
    RunSpecs(t, "Persistence Suite")
}
```

#### 2. Test Data

Define test data at package level:

```go
var (
    testArtist = model.Artist{ID: "1", Name: "Test Artist"}
    testAlbum  = model.Album{ID: "101", Name: "Test Album", ArtistID: "1"}
    testSong   = model.MediaFile{ID: "1001", Title: "Test Song"}
)
```

#### 3. BeforeSuite Setup

Use `BeforeSuite` for data initialization:

```go
var _ = BeforeSuite(func() {
    ctx := context.Background()
    repo := NewRepository(ctx, GetDBXBuilder())

    err := repo.Put(&testArtist)
    Expect(err).ToNot(HaveOccurred())
})
```

#### 4. Table-Driven Tests

Use `DescribeTable` for testing multiple scenarios:

```go
DescribeTable("Format validation",
    func(input string, expected bool) {
        result := IsValidFormat(input)
        Expect(result).To(Equal(expected))
    },
    Entry("valid mp3", "song.mp3", true),
    Entry("valid flac", "song.flac", true),
    Entry("invalid exe", "virus.exe", false),
)
```

### Test File Naming

- `*_test.go` - Regular tests
- `*_suite_test.go` - Test suite setup
- `*_internal_test.go` - White-box tests (same package)

### Running Tests with Tags

All Go tests must be run with the `netgo` tag:

```bash
go test -tags netgo ./...
```

This is required for proper SQLite compilation.

### Snapshot Testing

For subsonic API responses:

```bash
# Update snapshots
make snapshots

# This runs: UPDATE_SNAPSHOTS=true go tool ginkgo ./server/subsonic/responses/...
```

## Code Conventions

### Go Code Style

#### File Naming

- **Models:** `{entity}.go` (e.g., `album.go`, `artist.go`)
- **Repositories:** `{entity}_repository.go` (e.g., `album_repository.go`)
- **Services:** `{service_name}.go` (e.g., `media_streamer.go`)
- **Tests:** `{file}_test.go`
- **Wire Providers:** `wire_providers.go` in each package

#### Package Organization

- One package per directory
- Package name = directory name
- Internal packages for package-private code
- No circular dependencies

#### Naming Conventions

- **Interfaces:** Descriptive names without "Interface" suffix (e.g., `Repository`, `Service`)
- **Implementations:** Lowercase struct names (e.g., `albumRepository`, `mediaStreamer`)
- **DB Structs:** Prefix with `db` (e.g., `dbAlbum`, `dbMediaFile`) for scanning
- **Constants:** Use const groups with iota or explicit values
- **Errors:** Define as package-level vars (e.g., `var ErrNotFound = errors.New("not found")`)

#### Code Structure

```go
// Package comment
package mypackage

// Imports in groups: stdlib, external, internal
import (
    "context"
    "fmt"

    "github.com/external/package"

    "github.com/navidrome/navidrome/model"
)

// Constants
const (
    DefaultTimeout = 30
)

// Package-level vars
var (
    ErrInvalid = errors.New("invalid")
)

// Interface definitions
type Service interface {
    DoSomething(ctx context.Context) error
}

// Struct definitions
type service struct {
    repo Repository
}

// Constructor (for Wire)
func NewService(repo Repository) Service {
    return &service{repo: repo}
}

// Methods
func (s *service) DoSomething(ctx context.Context) error {
    // Implementation
}
```

#### Error Handling

```go
// Return errors, don't panic (except in init/setup)
if err != nil {
    return fmt.Errorf("failed to do something: %w", err)
}

// Use custom errors from model/errors.go
if notFound {
    return model.ErrNotFound
}

// Log errors with context
log.Error(ctx, "Operation failed", "error", err, "id", id)
```

#### Context Usage

```go
// Always accept context as first parameter
func DoSomething(ctx context.Context, param string) error

// Extract user from context
user := request.UserFrom(ctx)

// Check cancellation
select {
case <-ctx.Done():
    return ctx.Err()
default:
    // Continue
}
```

### JavaScript/React Code Style

#### File Structure (UI)

```
src/
├── {feature}/
│   ├── index.js          # Re-exports
│   ├── {Feature}List.js  # List view component
│   ├── {Feature}Show.js  # Detail view component
│   └── {Feature}Edit.js  # Edit form component
```

#### Component Naming

- **Components:** PascalCase (e.g., `AlbumList`, `PlayerToolbar`)
- **Files:** Match component name
- **Utilities:** camelCase (e.g., `dataProvider.js`, `authProvider.js`)

#### Formatting

Code must be formatted with Prettier:

```bash
# Format code
cd ui && npm run prettier

# Check formatting
cd ui && npm run check-formatting
```

### Commit Conventions

Commits must follow this format:

```
<type>(scope): <description> (#issue)

[optional body]
```

**Types:**
- `feat` - New feature
- `fix` - Bug fix
- `sec` - Security fix
- `docs` - Documentation
- `style` - Code style changes
- `refactor` - Code refactoring
- `perf` - Performance improvement
- `test` - Test changes
- `build` - Build system changes
- `revert` - Revert previous commit
- `chore` - Maintenance tasks

**Examples:**
```
feat(scanner): add support for multi-disc albums (#1234)

fix(api): correct album list pagination offset (#5678)

refactor(persistence): use squirrel for query building
```

**Important:** All commits must include DCO sign-off:
```bash
git commit --signoff -m "feat(ui): add dark mode support (#910)"
```

### Code Quality

#### Linting Rules (Go)

Enabled linters (see `.golangci.yml`):
- asasalint, asciicheck, bidichk
- bodyclose, copyloopvar, dogsled
- durationcheck, errorlint
- gocritic, gocyclo, goprintffuncname
- gosec (security), misspell
- nakedret, nilerr, rowserrcheck
- unconvert, whitespace

**Build tags:** Always use `-tags=netgo` for consistency

#### Generated Code

**Never edit generated files:**
- `*_gen.go` - Wire generated code
- `*.pb.go` - Protobuf generated code
- `ui/build/*` - Frontend build output

These files are excluded from linting and formatting.

## Common Tasks

### Adding a New Entity

1. **Define Model** (`model/entity.go`):
```go
type MyEntity struct {
    ID        string
    Name      string
    CreatedAt time.Time
    LibraryID int
}

type MyEntityRepository interface {
    Get(id string) (*MyEntity, error)
    Put(entity *MyEntity) error
    // ... other methods
}
```

2. **Create Repository** (`persistence/myentity_repository.go`):
```go
type myEntityRepository struct {
    sqlRepository
}

func NewMyEntityRepository(ctx context.Context, db *sql.DB) model.MyEntityRepository {
    return &myEntityRepository{
        sqlRepository: sqlRepository{
            ctx:       ctx,
            db:        db,
            tableName: "my_entity",
        },
    }
}

func (r *myEntityRepository) Get(id string) (*model.MyEntity, error) {
    // Implementation
}
```

3. **Add to DataStore** (`model/datastore.go`):
```go
type DataStore interface {
    // ... existing methods
    MyEntity(ctx context.Context) MyEntityRepository
}
```

4. **Implement in DataStore** (`persistence/datastore.go`):
```go
func (ds *dataStore) MyEntity(ctx context.Context) model.MyEntityRepository {
    return NewMyEntityRepository(ctx, ds.db)
}
```

5. **Create Migration** (`db/migrations/`):
```bash
make migration-sql name=add_my_entity_table
```

6. **Add Wire Providers** (`persistence/wire_providers.go`):
```go
var Set = wire.NewSet(
    // ... existing providers
    NewMyEntityRepository,
    wire.Bind(new(model.MyEntityRepository), new(*myEntityRepository)),
)
```

7. **Regenerate Wire**:
```bash
make wire
```

8. **Write Tests** (`persistence/myentity_repository_test.go`)

### Adding a New API Endpoint

1. **Create Handler** (`server/subsonic/myhandler.go`):
```go
func (api *Router) handleMyEndpoint(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
    service := api.ds.MyService(r.Context())

    result, err := service.DoSomething(r.Context())
    if err != nil {
        return nil, err
    }

    return newResponse(), nil
}
```

2. **Register Route** (`server/subsonic/api.go`):
```go
func (api *Router) Routes() http.Handler {
    r := chi.NewRouter()
    // ... existing routes
    r.Get("/myEndpoint", api.handleMyEndpoint)
    return r
}
```

3. **Add Response Type** (if needed in `server/subsonic/responses/`):
```go
type MyResponse struct {
    // Fields
}
```

4. **Write Tests** (handler tests + snapshot tests if Subsonic API)

### Adding Configuration Options

1. **Add to Config Struct** (`conf/configuration.go`):
```go
type Server struct {
    // ... existing fields
    MyNewOption string
}
```

2. **Set Default** (in `init()` or `InitConfig()`):
```go
viper.SetDefault("Server.MyNewOption", "default-value")
```

3. **Document** (update README or docs)

4. **Use in Code**:
```go
option := conf.Server.MyNewOption
```

### Adding a Migration

```bash
# SQL migration
make migration-sql name=add_column_to_table

# Go migration for complex data transformations
make migration-go name=migrate_old_to_new_format
```

Edit the created file in `db/migrations/`:

```sql
-- +goose Up
ALTER TABLE albums ADD COLUMN new_field TEXT;

-- +goose Down
ALTER TABLE albums DROP COLUMN new_field;
```

### Working with Translations

Translation files are in `resources/i18n/*.json`

- Use POEditor for managing translations
- Don't edit translation files directly (they're auto-updated)
- Validate with: `make test-i18n`

## Database

### Schema

Navidrome uses SQLite with the following main tables:

- **user** - User accounts
- **library** - Music libraries
- **artist** - Artists (album artists and track artists)
- **album** - Albums
- **media_file** - Individual tracks
- **playlist** - User playlists
- **playlist_tracks** - Playlist contents
- **annotation** - User-specific data (favorites, ratings, play counts)
- **radio** - Internet radio stations
- **share** - Shared links
- **player** - Player/client registrations
- **transcoding** - Transcoding configurations
- **property** - Key-value store for app state

### Migrations

- **Location:** `db/migrations/`
- **Tool:** Goose
- **Naming:** `YYYYMMDDHHMMSS_description.sql` or `.go`
- **90+ migrations** tracking schema evolution from v0.1.0

### Query Building

Use Squirrel for building SQL queries:

```go
sql, args, err := squirrel.Select("*").
    From("album").
    Where(squirrel.Eq{"artist_id": artistID}).
    OrderBy("year DESC").
    Limit(10).
    ToSql()
```

### Repository Base Class

The `sqlRepository` provides common functionality:

```go
type sqlRepository struct {
    ctx       context.Context
    db        *sql.DB
    tableName string
}

// Common methods:
// - Get(id string, entity interface{}) error
// - GetAll(entities interface{}, options QueryOptions) error
// - Put(entity interface{}) error
// - Delete(id string) error
// - Count(options QueryOptions) (int64, error)
```

### Transactions

```go
// Create transaction
err := ds.WithTx(func(tx model.DataStore) error {
    // Use tx.Album(ctx), tx.Artist(ctx), etc.
    // Changes are automatically committed or rolled back
    return nil
})

// Immediate transaction (for writes to prevent lock contention)
err := ds.WithTxImmediate(func(tx model.DataStore) error {
    // ...
})
```

### Custom SQLite Functions

Navidrome registers custom SQLite functions (see `db/db.go`):

- Date/time functions
- String manipulation
- JSON operations
- Custom sorting

## Plugin System

Navidrome supports WebAssembly (WASM) plugins for extending functionality.

### Architecture

- **Runtime:** Wazero (pure Go WASM runtime)
- **Protocol:** Protocol Buffers (gRPC-style)
- **Isolation:** Each plugin runs in sandboxed WASM environment
- **Security:** Permission-based system (secure by default)
- **Stateless:** Plugin instances created per-call and destroyed after

### Plugin Capabilities

Plugins can implement:

1. **MetadataAgent** - Fetch artist/album info, images
2. **Scrobbler** - Scrobble to external services
3. **SchedulerCallback** - Handle scheduled tasks
4. **WebSocketCallback** - WebSocket communication
5. **LifecycleManagement** - Plugin initialization

### Host Services (Available to Plugins)

- **HttpService** - Make HTTP requests (with URL whitelisting)
- **CacheService** - TTL-based caching
- **ConfigService** - Access plugin configuration
- **SchedulerService** - Schedule one-time/recurring tasks
- **WebSocketService** - WebSocket connections
- **ArtworkService** - Generate artwork URLs
- **SubsonicAPIService** - Access Navidrome's Subsonic API

### Permission System

Plugins must declare permissions in `manifest.json`:

```json
{
  "name": "my-plugin",
  "author": "Developer",
  "version": "1.0.0",
  "description": "Plugin description",
  "website": "https://github.com/user/plugin",
  "capabilities": ["MetadataAgent"],
  "permissions": {
    "http": {
      "reason": "To fetch metadata from MusicBrainz",
      "allowedUrls": {
        "https://musicbrainz.org/ws/2/*": ["GET"]
      },
      "allowLocalNetwork": false
    },
    "cache": {
      "reason": "To cache API responses"
    }
  }
}
```

**Security Model:**
- Secure by default (no permissions)
- Load-time enforcement
- Service isolation
- URL/method whitelisting for HTTP
- Username restrictions for SubsonicAPI

### Plugin Development

```bash
# Generate plugin code from protobuf
make plugin-gen

# Build example plugins
make plugin-examples

# Plugin CLI commands
navidrome plugin list
navidrome plugin install plugin.ndp
navidrome plugin remove plugin-folder
navidrome plugin refresh plugin-folder
navidrome plugin dev /path/to/dev/folder
```

### Plugin Directory Structure

```
plugins/
├── my-plugin/              # Folder name is the plugin ID
│   ├── plugin.wasm
│   └── manifest.json
```

**Note:** Plugins are identified by folder name, not manifest name!

For more details, see `plugins/README.md`

## Frontend (UI)

### Technology

- **React 17** with functional components
- **react-admin 3.19** for admin interface
- **Redux** for state management
- **Material-UI v4** for components
- **Vite** for building and dev server

### Structure

```
ui/src/
├── album/           # Album list/show/edit
├── artist/          # Artist list/show/edit
├── playlist/        # Playlist management
├── song/            # Song/track components
├── audioplayer/     # Audio player
├── player/          # Player controls
├── actions/         # Redux actions
├── reducers/        # Redux reducers
├── dataProvider/    # API data provider
├── authProvider.js  # Authentication
├── i18n/            # Translations
├── themes/          # UI themes
└── routes.jsx       # Routing configuration
```

### Key Patterns

#### React-Admin Resources

```jsx
<Resource
    name="album"
    list={AlbumList}
    show={AlbumShow}
    edit={AlbumEdit}
/>
```

#### Data Provider

Custom data provider in `dataProvider/` handles:
- REST API calls
- Subsonic API compatibility
- Response transformation
- Error handling

#### Audio Player

The audio player (`audioplayer/`) is a critical component:
- Uses `navidrome-music-player` package
- Integrates with queue management
- Handles transcoding
- Reports play counts

#### Theming

Themes in `src/themes/` define:
- Color schemes
- Component overrides
- Typography
- Custom styling

### Building

```bash
cd ui

# Install dependencies
npm ci

# Development server
npm start

# Production build
npm run build

# Tests
npm test

# Linting
npm run lint

# Formatting
npm run prettier
```

### i18n

Translations are in `resources/i18n/*.json` and managed via POEditor.

To use in components:
```jsx
import { useTranslate } from 'react-admin';

const translate = useTranslate();
translate('resources.album.name');
```

## Important Notes

### When Making Changes

1. **Always read files before editing** - Don't propose changes to code you haven't seen
2. **Use appropriate tools** - Don't use bash for file operations when dedicated tools exist
3. **Follow existing patterns** - Study similar code before implementing
4. **Run tests** - Use `make testall` before committing
5. **Format code** - Run `make format` to ensure proper formatting
6. **Regenerate Wire** - Run `make wire` after changing DI
7. **Check build** - Run `make build` to verify compilation

### Common Pitfalls

1. **Forgetting build tags** - Always use `-tags=netgo` for Go builds/tests
2. **Skipping Wire regeneration** - Changes to DI require `make wire`
3. **Direct file editing** - Never edit generated files (*_gen.go, *.pb.go)
4. **Missing context** - Always pass context.Context as first parameter
5. **Ignoring user context** - Respect user library permissions in repositories
6. **Circular dependencies** - Avoid importing packages that import you back
7. **Not using squirrel** - Build SQL queries with squirrel, not string concatenation
8. **Hardcoded LibraryID** - Always respect multi-library architecture

### Security Considerations

1. **Input Validation** - Validate all user inputs
2. **SQL Injection** - Use parameterized queries (squirrel handles this)
3. **Path Traversal** - Validate file paths in scanner and artwork
4. **Authentication** - Use existing auth middleware
5. **Authorization** - Respect user permissions and library access
6. **Secrets** - Never commit API keys or credentials
7. **CSRF** - Use proper CSRF protection in forms
8. **XSS** - Sanitize HTML output (use bluemonday)

### Performance Tips

1. **Database Indexes** - Check query plans for slow queries
2. **Caching** - Use ttlcache for expensive operations
3. **Lazy Loading** - Don't load all data upfront
4. **Pagination** - Always paginate large result sets
5. **N+1 Queries** - Use JOINs or batch loading
6. **File I/O** - Use buffers for large files
7. **Concurrency** - Use goroutines for parallel operations (carefully)

### Resources

- **Main Repo:** https://github.com/navidrome/navidrome
- **Documentation:** https://www.navidrome.org/docs/
- **API Docs:** https://www.navidrome.org/docs/developers/subsonic-api/
- **Discord:** https://discord.gg/xh7j7yF
- **Development Guide:** https://www.navidrome.org/docs/developers/

### Getting Help

1. Check existing code for similar implementations
2. Read the documentation at navidrome.org
3. Search GitHub issues
4. Ask in Discord #development channel
5. Review CONTRIBUTING.md for guidelines

---

**Last Updated:** 2025-12-06

This guide should be updated as the codebase evolves. When making significant architectural changes, update this document accordingly.
