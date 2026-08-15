# Navidrome fork

[![Build custom Docker image](https://github.com/Kinanqaz/navidrome/actions/workflows/custom-image.yml/badge.svg)](https://github.com/Kinanqaz/navidrome/actions/workflows/custom-image.yml)

This repository is a small, maintenance-friendly fork of
[Navidrome](https://github.com/navidrome/navidrome). It adds selected features
while keeping the codebase close to upstream so future Navidrome updates remain
straightforward to merge.

For standard installation, configuration, clients, and usage documentation,
visit the [original Navidrome repository](https://github.com/navidrome/navidrome)
or the [official documentation](https://www.navidrome.org/docs/).

## Changes in this fork

### Delete music from the web interface

Administrators can permanently delete individual songs from the song context
menu. The action requires explicit confirmation and is disabled by default.

The server applies the following safeguards:

- administrator authorization is checked independently of the UI;
- only local, writable music storage supports deletion;
- deletion is confined to the configured library root;
- path traversal, symlink escapes, directories, and special files are rejected;
- successful deletion is recorded in the server log;
- related Navidrome database records are cleaned after the file is removed.

Audio-file deletion is permanent. Artwork and lyric sidecar files are not
removed.

Enable the feature with:

```yaml
environment:
  ND_ENABLEMEDIAFILEDELETION: "true"
```

The music library must be mounted read-write and the container user must have
permission to modify it:

```yaml
volumes:
  - /path/to/music:/music:rw
```

Keep backups of irreplaceable media before enabling permanent deletion.

## Container images

Pushes to `master` automatically build the existing Navidrome Dockerfile for
`linux/amd64`. Images are published to GitHub Container Registry as:

```text
ghcr.io/kinanqaz/navidrome:latest
ghcr.io/kinanqaz/navidrome:sha-<short-commit>
```

A minimal Compose override for this fork is:

```yaml
services:
  navidrome:
    image: ghcr.io/kinanqaz/navidrome:latest
    environment:
      ND_ENABLEMEDIAFILEDELETION: "true"
    volumes:
      - /path/to/data:/data
      - /path/to/music:/music:rw
    ports:
      - "4533:4533"
```

All other Navidrome configuration options work as documented upstream.

## Updating from upstream

The original Navidrome repository can be configured as the `upstream` remote:

```bash
git remote add upstream https://github.com/navidrome/navidrome.git
git fetch upstream
git merge upstream/master
```

## Development

This fork uses the same Go, React, SQLite, and Docker toolchain as Navidrome.
Refer to the
[upstream development documentation](https://www.navidrome.org/docs/developers/)
for general setup and build instructions. Fork-specific implementation notes
are available in [FORK_NOTES.md](FORK_NOTES.md).

## License

This project remains licensed under the
[GNU General Public License v3](LICENSE), the same license as upstream
Navidrome.
