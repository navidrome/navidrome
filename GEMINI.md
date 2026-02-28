# Task: Fix Issue #5112

## Issue to Solve
**Title:** [Bug]: /share/img/ returns 404 for artists without artwork, triggers security tool false positives (CrowdSec, fail2ban)
**Number:** #5112

### I confirm that:

- [x] I have searched the existing [open AND closed issues](https://github.com/navidrome/navidrome/issues?q=is%3Aissue) to see if an issue already exists for the bug I've encountered
- [x] I'm using the latest version (your issue may have been fixed already)

### Version

0.60.3

### Current Behavior

 When an artist has no artwork, the /share/img/{jwt} endpoint returns 404 Not Found. When a client displays a folder/artist list containing multiple artists without artwork, this generates a burst of concurrent 404 responses. Security tools running on the same host (CrowdSec http-probing scenario, fail2ban, WAFs) interpret this burst as an attack and ban the client's IP address — including legitimate users on the same network as the server.

### Expected Behavior

/share/img/{jwt} should return either:
  - 204 No Content when no artwork is available for the resource, or
  - a generic placeholder image (200 OK)

A 404 is semantically valid but operationally harmful: it is indistinguishable from probing/scanning at the HTTP layer.

### Steps To Reproduce

 1. Set up Navidrome behind a reverse proxy with CrowdSec (or any security tool monitoring 404 rates)
  2. Have a music library where some artists have no embedded artwork
  3. Connect with a client that uses /share/img/ URLs to display artist images (e.g. Substreamer, "Folders" view)
  4. Open a view that loads multiple artist images simultaneously (Substreamer's "Folders" view)

### Environment

```markdown
- Navidrome version: 0.60.3 (34c6f12a)
- OS: Debian GNU/Linux 13 (trixie), aarch64 (Raspberry Pi 4)
- Client: Substreamer (Android WebView), "Folders" view
- Reverse proxy: Nginx with CrowdSec firewall bouncer
```

### How Navidrome is installed?

Binary (from downloads page)

### Configuration

```toml

```

### Relevant log output

```shell

```

### Anything else?

 My temporary workaround:

  Intercept 404s at the Nginx reverse proxy level:

  location /share/img/ {
      proxy_pass http://127.0.0.1:4533;
      proxy_set_header Host $host;
      proxy_set_header X-Real-IP $remote_addr;
      proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
      proxy_set_header X-Forwarded-Proto $scheme;
      proxy_intercept_errors on;
      error_page 404 = @noimage;
  }

  location @noimage {
      return 204;
  }


### Code of Conduct

- [x] I agree to follow Navidrome's Code of Conduct

## Repository Info
**Repository:** navidrome/navidrome

### Key Files
- .devcontainer/devcontainer.json
- .github/FUNDING.yml
- .github/ISSUE_TEMPLATE/bug_report.yml
- .github/ISSUE_TEMPLATE/config.yml
- .github/actions/download-taglib/action.yml
- .github/actions/prepare-docker/action.yml
- .github/dependabot.yml
- .github/workflows/download-link-on-pr.yml
- .github/workflows/pipeline.yml
- .github/workflows/push-translations.yml
- .github/workflows/stale.yml
- .github/workflows/update-translations.yml
- .golangci.yml
- adapters/deezer/client.go
- adapters/deezer/client_auth.go
- adapters/deezer/client_auth_test.go
- adapters/deezer/client_test.go
- adapters/deezer/deezer.go
- adapters/deezer/deezer_suite_test.go
- adapters/deezer/deezer_test.go
- adapters/deezer/responses.go
- adapters/deezer/responses_test.go
- adapters/gotaglib/end_to_end_test.go
- adapters/gotaglib/gotaglib.go
- adapters/gotaglib/gotaglib_suite_test.go
- adapters/gotaglib/gotaglib_test.go
- adapters/lastfm/agent.go
- adapters/lastfm/agent_test.go
- adapters/lastfm/auth_router.go
- adapters/lastfm/client.go
- adapters/lastfm/client_test.go
- adapters/lastfm/lastfm_suite_test.go
- adapters/lastfm/responses.go
- adapters/lastfm/responses_test.go
- adapters/listenbrainz/agent.go
- adapters/listenbrainz/agent_test.go
- adapters/listenbrainz/auth_router.go
- adapters/listenbrainz/auth_router_test.go
- adapters/listenbrainz/client.go
- adapters/listenbrainz/client_test.go
- adapters/listenbrainz/listenbrainz_suite_test.go
- adapters/spotify/client.go
- adapters/spotify/client_test.go
- adapters/spotify/responses.go
- adapters/spotify/responses_test.go
- adapters/spotify/spotify.go
- adapters/spotify/spotify_suite_test.go
- adapters/taglib/end_to_end_test.go
- adapters/taglib/get_filename.go
- adapters/taglib/get_filename_win.go

## How to Build & Test
**Setup:** `go mod tidy`
**Build:** `go build -v ./...`
**Test:** `go test -v ./...`

## CI Configuration
### `.github/workflows/download-link-on-pr.yml`
```
name: Add download link to PR
on:
  workflow_run:
    workflows: ['Pipeline: Test, Lint, Build']
    types: [completed]
jobs:
  pr_comment:
    if: github.event.workflow_run.event == 'pull_request' && github.event.workflow_run.conclusion == 'success'
    runs-on: ubuntu-latest
    steps:
      - uses: actions/github-script@v3
        with:
          # This snippet is public-domain, taken from
          # https://github.com/oprypin/nightly.link/blob/master/.github/workflows/pr-comment.yml
          script: |
            const {owner, repo} = context.repo;
            const run_id = ${{github.event.workflow_run.id}};
            const pull_head_sha = '${{github.event.workflow_run.head_sha}}';
            const pull_user_id = ${{github.event.sender.id}};

            const issue_number = await (async () => {
              const pulls = await github.pulls.list({owner, repo});
              for await (const {data} of github.paginate.iterator(pulls)) {
                for (const pull of data) {
                  if (pull.head.sha === pull_head_sha && pull.user.id === pull_user_id) {
                    return pull.number;
                  }
                }
              }
            })();
            if (issue_number) {
              core.info(`Using pull request ${issue_number}`);
            } else {
              return core.error(`No matching pull request found`);
            }

            const {data: {artifacts}} = await github.actions.listWorkflowRunArtifacts({owner, re
```
### `.github/workflows/pipeline.yml`
```
name: "Pipeline: Test, Lint, Build"
on:
  push:
    branches:
      - master
    tags:
      - "v*"
  pull_request:
    branches:
      - master

concurrency:
  group: ${{ startsWith(github.ref, 'refs/tags/v') && 'tag' || 'branch' }}-${{ github.ref }}
  cancel-in-progress: true

env:
  CROSS_TAGLIB_VERSION: "2.2.0-1"
  CGO_CFLAGS_ALLOW: "--define-prefix"
  IS_RELEASE: ${{ startsWith(github.ref, 'refs/tags/') && 'true' || 'false' }}

jobs:
  git-version:
    name: Get version info
    runs-on: ubuntu-latest
    outputs:
      git_tag: ${{ steps.git-version.outputs.GIT_TAG }}
      git_sha: ${{ steps.git-version.outputs.GIT_SHA }}
    steps:
      - uses: actions/checkout@v6
        with:
          fetch-depth: 0
          fetch-tags: true

      - name: Show git version info
        run: |
          echo "git describe (dirty): $(git describe --dirty --always --tags)"
          echo "git describe --tags: $(git describe --tags `git rev-list --tags --max-count=1`)"
          echo "git tag: $(git tag --sort=-committerdate | head -n 1)"
          echo "github_ref: $GITHUB_REF"
          echo "github_head_sha: ${{ github.event.pull_request.head.sha }}"
          git tag -l
      - name: Determine git current SHA and latest tag
        id: git-version
        run: |
          GIT_TAG=$(git tag --sort=-committerdate | head -n 1)
          if [ -n "$GIT_TAG" ]; then
            if [[ "$GITHUB_REF" != refs/tags/* ]]; then
              GIT_TAG=${GIT_TAG}-SNAPSHOT
            fi
        
```
### `.github/workflows/push-translations.sh`
```
#!/bin/sh

set -e

I18N_DIR=resources/i18n

# Normalize JSON for deterministic comparison:
# remove empty/null attributes, sort keys alphabetically
process_json() {
  jq 'walk(if type == "object" then with_entries(select(.value != null and .value != "" and .value != [] and .value != {})) | to_entries | sort_by(.key) | from_entries else . end)' "$1"
}

# Get list of all languages configured in the POEditor project
get_language_list() {
  curl -s -X POST https://api.poeditor.com/v2/languages/list \
    -d api_token="${POEDITOR_APIKEY}" \
    -d id="${POEDITOR_PROJECTID}"
}

# Extract language name from the language list JSON given a language code
get_language_name() {
  lang_code="$1"
  lang_list="$2"
  echo "$lang_list" | jq -r ".result.languages[] | select(.code == \"$lang_code\") | .name"
}

# Extract language code from a file path (e.g., "resources/i18n/fr.json" -> "fr")
get_lang_code() {
  filepath="$1"
  filename=$(basename "$filepath")
  echo "${filename%.*}"
}

# Export the current translation for a language from POEditor (v2 API)
export_language() {
  lang_code="$1"
  response=$(curl -s -X POST https://api.poeditor.com/v2/projects/export \
    -d api_token="${POEDITOR_APIKEY}" \
    -d id="${POEDITOR_PROJECTID}" \
    -d language="$lang_code" \
    -d type="key_value_json")

  url=$(echo "$response" | jq -r '.result.url')
  if [ -z "$url" ] || [ "$url" = "null" ]; then
    echo "Failed to export $lang_code: $response" >&2
    return 1
  fi
  echo "$url"
}

# Flatten ne
```
### `.github/workflows/push-translations.yml`
```
name: POEditor export

on:
  push:
    branches:
      - master
    paths:
      - 'resources/i18n/*.json'

jobs:
  push-translations:
    runs-on: ubuntu-latest
    if: ${{ github.repository_owner == 'navidrome' }}
    steps:
      - uses: actions/checkout@v6
        with:
          fetch-depth: 2

      - name: Detect changed translation files
        id: changed
        run: |
          CHANGED_FILES=$(git diff --name-only HEAD~1 HEAD -- 'resources/i18n/*.json' | tr '\n' ' ')
          echo "files=$CHANGED_FILES" >> $GITHUB_OUTPUT
          echo "Changed translation files: $CHANGED_FILES"

      - name: Push translations to POEditor
        if: ${{ steps.changed.outputs.files != '' }}
        env:
          POEDITOR_APIKEY: ${{ secrets.POEDITOR_APIKEY }}
          POEDITOR_PROJECTID: ${{ secrets.POEDITOR_PROJECTID }}
        run: |
          .github/workflows/push-translations.sh ${{ steps.changed.outputs.files }}

```
### `.github/workflows/stale.yml`
```
name: 'Close stale issues and PRs'
on:
  workflow_dispatch:
  schedule:
    - cron: '30 1 * * *'
permissions:
  contents: read
jobs:
  stale:
    permissions:
      issues: write
      pull-requests: write
    runs-on: ubuntu-latest
    steps:
      - uses: dessant/lock-threads@v6
        with:
          process-only: 'issues, prs'
          issue-inactive-days: 120
          pr-inactive-days: 120
          log-output: true
          add-issue-labels: 'frozen-due-to-age'
          add-pr-labels: 'frozen-due-to-age'
          issue-comment: >
            This issue has been automatically locked since there
            has not been any recent activity after it was closed.
            Please open a new issue for related bugs.
          pr-comment: >
            This pull request has been automatically locked since there
            has not been any recent activity after it was closed.
            Please open a new issue for related bugs.
      - uses: actions/stale@v9
        with:
          operations-per-run: 999
          days-before-issue-stale: 180
          days-before-pr-stale: 180
          days-before-issue-close: 30
          days-before-pr-close: 30
          stale-issue-message: >
            This issue has been automatically marked as stale because it has not had
            recent activity. The resources of the Navidrome team are limited, and so we are asking for your help.

            If this is a **bug** and you can still reproduce this error on the <code>master
```
### `Makefile`
```
GO_VERSION=$(shell grep "^go " go.mod | cut -f 2 -d ' ')
NODE_VERSION=$(shell cat .nvmrc)
GO_BUILD_TAGS=netgo,sqlite_fts5

# Set global environment variables, required for most targets
export CGO_CFLAGS_ALLOW=--define-prefix
export ND_ENABLEINSIGHTSCOLLECTOR=false

ifneq ("$(wildcard .git/HEAD)","")
GIT_SHA=$(shell git rev-parse --short HEAD)
GIT_TAG=$(shell git describe --tags `git rev-list --tags --max-count=1`)-SNAPSHOT
else
GIT_SHA=source_archive
GIT_TAG=$(patsubst navidrome-%,v%,$(notdir $(PWD)))-SNAPSHOT
endif

SUPPORTED_PLATFORMS ?= linux/amd64,linux/arm64,linux/arm/v5,linux/arm/v6,linux/arm/v7,linux/386,linux/riscv64,darwin/amd64,darwin/arm64,windows/amd64,windows/386
IMAGE_PLATFORMS ?= $(shell echo $(SUPPORTED_PLATFORMS) | tr ',' '\n' | grep "linux" | grep -v "arm/v5" | tr '\n' ',' | sed 's/,$$//')
PLATFORMS ?= $(SUPPORTED_PLATFORMS)
DOCKER_TAG ?= deluan/navidrome:develop

# Taglib version to use in cross-compilation, from https://github.com/navidrome/cross-taglib
CROSS_TAGLIB_VERSION ?= 2.2.0-1
GOLANGCI_LINT_VERSION ?= v2.10.0

UI_SRC_FILES := $(shell find ui -type f -not -path "ui/build/*" -not -path "ui/node_modules/*")

setup: check_env download-deps install-golangci-lint setup-git ##@1_Run_First Install dependencies and prepare development environment
	@echo Downloading Node dependencies...
	@(cd ./ui && npm ci)
.PHONY: setup

dev: check_env   ##@Development Start Navidrome in development mode, with hot-reload for both frontend and backend
	npx foreman -j Procfi
```
### `Dockerfile`
```
FROM --platform=$BUILDPLATFORM ghcr.io/crazy-max/osxcross:14.5-debian AS osxcross

########################################################################################################################
### Build xx (original image: tonistiigi/xx)
FROM --platform=$BUILDPLATFORM public.ecr.aws/docker/library/alpine:3.20 AS xx-build

# v1.9.0
ENV XX_VERSION=a5592eab7a57895e8d385394ff12241bc65ecd50

RUN apk add -U --no-cache git
RUN git clone https://github.com/tonistiigi/xx && \
    cd xx && \
    git checkout ${XX_VERSION} && \
    mkdir -p /out && \
    cp src/xx-* /out/

RUN cd /out && \
    ln -s xx-cc /out/xx-clang && \
    ln -s xx-cc /out/xx-clang++ && \
    ln -s xx-cc /out/xx-c++ && \
    ln -s xx-apt /out/xx-apt-get

# xx mimics the original tonistiigi/xx image
FROM scratch AS xx
COPY --from=xx-build /out/ /usr/bin/

########################################################################################################################
### Get TagLib
FROM --platform=$BUILDPLATFORM public.ecr.aws/docker/library/alpine:3.20 AS taglib-build
ARG TARGETPLATFORM
ARG CROSS_TAGLIB_VERSION=2.2.0-1
ENV CROSS_TAGLIB_RELEASES_URL=https://github.com/navidrome/cross-taglib/releases/download/v${CROSS_TAGLIB_VERSION}/

# wget in busybox can't follow redirects
RUN <<EOT
    apk add --no-cache wget
    PLATFORM=$(echo ${TARGETPLATFORM} | tr '/' '-')
    FILE=taglib-${PLATFORM}.tar.gz

    DOWNLOAD_URL=${CROSS_TAGLIB_RELEASES_URL}${FILE}
    wget ${DOWNLOAD_URL}

    mkdir /taglib
    
```

## Instructions
1. Read the codebase to understand the relevant code
2. Fix the issue described above
3. Run all tests to verify your fix
4. Do NOT modify unrelated files
5. After completing, list every file you changed and explain why
