# Navidrome — Development Notes

## Running with Docker Compose

The dev stack splits the backend and frontend into separate services with hot-reload.

```bash
# First run (builds Go image, downloads all deps — takes a few minutes)
docker compose -f docker-compose.dev.yml up --build

# Subsequent runs
docker compose -f docker-compose.dev.yml up
```

| Service  | Port | Description                          |
|----------|------|--------------------------------------|
| backend  | 4633 | Go server with reflex hot-reload     |
| frontend | 4533 | Vite dev server — open this in browser |

The frontend proxies `/auth`, `/api`, `/rest`, and `/backgrounds` requests to the backend automatically.

```bash
docker compose -f docker-compose.dev.yml logs -f backend   # backend logs
docker compose -f docker-compose.dev.yml logs -f frontend  # frontend logs
docker compose -f docker-compose.dev.yml down              # stop everything
```

## Running without Docker

```bash
make setup   # one-time: installs Go and Node dependencies
make dev     # starts both services with hot-reload
```

## Database

Navidrome uses **SQLite only** — there is no Postgres or MySQL support. The database file is persisted in `./data/navidrome.db` (bind-mounted into the container at `/data`).

### Inspecting the database

From your host (requires `sqlite3`):

```bash
sqlite3 ./data/navidrome.db
```

From inside the running backend container:

```bash
docker compose -f docker-compose.dev.yml exec backend sqlite3 /data/navidrome.db
```

Useful SQLite commands:

```sql
.tables               -- list all tables
.schema media_file    -- show a table's schema
SELECT * FROM user;   -- query data
.quit                 -- exit
```

## Configuration

App config lives in `navidrome.toml`. Environment variables (prefixed `ND_`) override it. The Docker Compose file sets:

| Variable                    | Value             |
|-----------------------------|-------------------|
| `ND_PORT`                   | `4633`            |
| `ND_MUSICFOLDER`            | `/music`          |
| `ND_DATAFOLDER`             | `/data`           |
| `ND_LOGLEVEL`               | `info`            |
| `ND_ENABLEINSIGHTSCOLLECTOR`| `false`           |
| `ND_DEVAUTOCREATEADMINPASSWORD` | `admin`      |

The default dev credentials are **`admin` / `admin`** (set via `ND_DEVAUTOCREATEADMINPASSWORD`).

The music folder maps to `./music` (project root) inside the container. Drop audio files there and the scanner will pick them up.
