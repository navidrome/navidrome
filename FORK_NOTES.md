# Fork customizations

This fork keeps its changes deliberately small so updates from the Navidrome
`upstream` remote remain straightforward to merge.

## Permanent music-file deletion

Administrators can permanently delete an individual song from its context
menu. The capability is disabled by default. Enable it explicitly with either:

```toml
EnableMediaFileDeletion = true
```

or the Docker environment variable:

```text
ND_ENABLEMEDIAFILEDELETION=true
```

The music volume must be mounted read-write. Only administrators can call the
endpoint, and the server checks that permission independently of the UI. Only
local storage opts into mutation. Removal uses Go's rooted filesystem API to
reject path traversal and symlink escapes, refuses directories and special
files, and records the administrator, media ID, and relative path in the log.

Deletion is permanent. It removes the audio file and Navidrome's associated
database references, including playlist entries, ratings, bookmarks, and play
history. Artwork and lyric sidecar files are not removed.

## Updating from Navidrome

```bash
git fetch upstream
git merge upstream/master
```

Resolve conflicts if Git reports any, run the backend and UI test suites, then
build and deploy a new image from this fork. Pulling an official Navidrome image
will not contain these customizations.

Pushes to `master` automatically build the existing `Dockerfile` for
`linux/amd64` and publish two tags to GitHub Container Registry:

```text
ghcr.io/<github-owner>/<repository>:latest
ghcr.io/<github-owner>/<repository>:sha-<short-commit>
```

The workflow uses the repository-scoped `GITHUB_TOKEN`; no Docker Hub password
or additional repository secret is required. GitHub Packages may initially mark
the image private. Either make the package public or configure the Proxmox
container host to authenticate to GHCR before pulling it.

## Local development dependencies

- `ui/node_modules` is required for frontend tests and builds.
- Go 1.26 is required for backend builds.
- A C compiler is required by SQLite/CGO. The Windows setup uses a portable Zig
  toolchain instead of installing GCC/MinGW system-wide.
- Go's module cache avoids repeatedly downloading locked dependencies.
- `ui/build` and compiler/test caches are generated output and can be removed
  after verification.
