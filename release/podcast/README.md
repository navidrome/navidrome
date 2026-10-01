# Release podcast

A standalone Go CLI turns **published Navidrome release notes** into a grounded
English recap and an MP3. The local CLI and GitHub workflow share the same source
resolution, evidence checks, cost limits and generation stages. Outputs are
`transcript.txt`, `release-podcast.mp3`, `sources.json`, `evidence.json` and
`manifest.json`. Nothing is uploaded to release assets or social media.

## Local use, including before merge

From a checkout of this PR, with the Go version from `go.mod`:

```sh
go run ./release/podcast --from 0.64.0 --to 0.64.2 \
  --dry-run --output /tmp/navidrome-podcast-validation
```

This resolves the inclusive range of published releases, writes exact sources
and a manifest, and makes **zero OpenAI requests**. It needs GitHub network
access; `GH_TOKEN` is optional for public notes and increases the rate limit.
It needs neither an OpenAI key nor repository variables, and does not spoof
GitHub context. Explicit versions are also supported:

```sh
go run ./release/podcast --tags v0.64.0,v0.64.1,v0.64.2 \
  --mode validate --output /tmp/navidrome-podcast-validation
```

Select at most three distinct releases. Versions may omit the `v` prefix.
Ranges require ordered stable-version endpoints that both exist as published
releases. Listing is bounded to 1,000 release records; larger listings require
explicit `--tags`. Published prereleases require `--include-prereleases` and
explicit tags if they are range endpoints. Drafts are discarded.
Prereleases use SemVer precedence, including numeric identifiers: `rc.2`
precedes `rc.10`, and both precede the corresponding stable release. Thus a
stable lower range bound excludes its own RCs; a stable upper bound includes
its RCs only with `--include-prereleases`.

When you separately decide to incur API usage, make `OPENAI_API_KEY` available
in your local process environment through your own secure setup. **Never put
the key in a CLI argument, log, commit, or transcript.** A repository Actions
secret is not a local environment variable; this tool does not retrieve it.
Choose supported values for the nonsecret `TEXT_MODEL`, `TTS_MODEL` and `VOICE`
variables used in this example:

```sh
go run ./release/podcast --from 0.64.0 --to 0.64.2 \
  --mode audio --allow-paid \
  --text-model "$TEXT_MODEL" --tts-model "$TTS_MODEL" --voice "$VOICE" \
  --output /tmp/navidrome-podcast-preview
```

Install `ffmpeg` (including `ffprobe`) yourself before local audio mode. Media
preflight runs before any paid request. Then listen to
`/tmp/navidrome-podcast-preview/release-podcast.mp3` and compare the transcript
and evidence with the notes. Local preview does **not** require merging the PR
or setting `RELEASE_AUDIO_ENABLED`.

`--mode script --allow-paid --text-model MODEL` generates only the transcript
and evidence. `--dry-run` always forces validation, even if `--mode audio` or
`--allow-paid` is also present. CLI model/voice flags override `AUDIO_TEXT_MODEL`,
`AUDIO_TTS_MODEL` and `AUDIO_VOICE` environment values; there are no model/voice
defaults. No API host, repository, key, shell command or pricing override is
accepted as a CLI option.

Use a fresh output directory for each intentional preview. Paid runs acquire
an exclusive local `.release-podcast.lock` and persist a reservation in
`.attempts/` **before** contacting OpenAI. Failed attempts also remain reserved.
`--force` permits another paid attempt and replacement of generated files;
it cannot bypass an active lock. After a killed process, review the output and
ledger before manually removing a stale lock. Validation refuses directories
containing generated script/audio files so it cannot relabel older audio.
Local deduplication is scoped to the output directory; a new directory or deleted
ledger can permit another charge. Script and audio modes have distinct keys.

## Models and voice

These are supported choices, **not finalized user defaults**:

| Setting | Reviewed options |
| --- | --- |
| Text model | `gpt-6-luna` (low reasoning), `gpt-4.1-mini-2025-04-14` |
| Speech model | `gpt-4o-mini-tts-2025-12-15`, `tts-1`, `tts-1-hd` |
| Legacy voices | `alloy`, `echo`, `fable`, `onyx`, `nova`, `shimmer` |
| Mini TTS voices | Legacy voices plus `ash`, `ballad`, `coral`, `sage`, `verse`, `marin`, `cedar` |

Onyx and cedar are candidates to audition for a male-style delivery; perceived
voice gender is subjective. Mini TTS receives fixed calm English instructions;
legacy TTS does not. Unsupported models fail before paid requests. A new model
requires a code/pricing review. Luna is an API alias whose behavior can change;
the other script model and Mini TTS use pinned snapshots.

## GitHub Actions setup and publication

The workflow is `.github/workflows/release-podcast.yml`, named **Release
podcast**. Automatic generation is disabled by default. The secret and variable
names already communicated during planning are retained, so no configuration
rename is required:

1. The maintainer adds repository Actions secret `OPENAI_API_KEY` personally.
   One project key covers Responses and speech when its model/endpoint access
   allows both. Use a dedicated project key and usage alerts.
2. Choose repository variables `RELEASE_AUDIO_TEXT_MODEL`,
   `RELEASE_AUDIO_TTS_MODEL` and `RELEASE_AUDIO_VOICE`.
3. Set `RELEASE_AUDIO_ENABLED` to literal `true` only when paid workflow runs
   are authorized. Blank variables are fine for offline tests/manual validation.
4. After merge, dispatch **Release podcast** from `master`, starting with
   `validate` and `v0.64.0,v0.64.1,v0.64.2`. Models and voice come from repository
   variables, not manual inputs. No named GitHub Environment is configured.

The automatic trigger is **`release: published`**, stable releases only. It
fetches the event's exact release ID, never `/releases/latest`. Tags, edits,
drafts and prereleases do not automatically generate audio. Promotion of an
already-published prerelease may require manual dispatch; manual prereleases
need explicit opt-in.

The existing `pipeline.yml` uses `GITHUB_TOKEN` for GoReleaser and
`release/goreleaser.yml` sets `draft: true`. Those behaviors are unchanged.
Maintainer publication of the draft through GitHub can trigger this workflow;
publication with `GITHUB_TOKEN` generally suppresses downstream release events.
Use manual dispatch after such publication, or separately review an explicit
dispatch integration with narrow Actions permissions. Do not add a broad PAT
or change draft publishing to solve this integration.

Land the workflow before the next tag. Release events are associated with the
tagged commit; historical tags cannot be assumed to contain a new workflow.
Manual dispatch requires the workflow on the default branch and deliberately
rejects other branches. Checkout executes the reviewed default-branch helper,
with credentials not persisted. **Use the first-class local CLI to test the PR
before merge**, rather than bypassing this Actions guard.

The workflow builds the shared Go binary and installs media tools in
credential-free steps. It exposes `OPENAI_API_KEY` only to generation steps,
grants `contents: read` and `actions: read`, and pins actions to verified SHAs.
No PR event can run the paid workflow. The offline PR test workflow has no key.

| Mode | Maximum OpenAI requests | Outputs |
| --- | --- | --- |
| `validate` / `--dry-run` | None | Sources and manifest; modeled text estimate if configured |
| `script` | One Responses request | Transcript, evidence, sources and manifest |
| `audio` | One Responses plus one speech request | Script outputs and validated MP3 |

## Limits, grounding and costs

Target about two minutes: 250–280 total words and 105–145 seconds. Small releases
may be shorter; correctness takes priority over filler. There are no application
or SDK retries/repair calls. Timeouts may already be billed. HTTP requests have
a one-minute timeout, media inspection 30 seconds, and the CLI/workflow ten
minutes. Sources and the complete prompt each have separate 64 KiB caps;
oversized material fails without truncating warnings. Text output is capped at
3,000 tokens including reasoning/evidence. Narration is at most 280 words and
2,500 characters. Mini TTS additionally uses a conservative 2,000 UTF-8-byte
input ceiling including instructions, to stay below its input-token limit.

The model has no tools and receives only public notes as untrusted evidence,
not executable instructions. No embedded links or draft advisories are fetched.
Strict JSON maps sentences to exact source excerpts and required cautions.
IDs/excerpts and migration/security coverage must match; lexical checks retain
the prototype's backup, client resync, experimental/opt-in, plugin networking,
Docker discovery, and slow-storage/32-bit scan cautions. These checks do not
prove semantic entailment or classify language perfectly. **Human factual and
listening review remains required.** Every transcript discloses the AI voice.
`store: false` does not imply zero provider retention.

Reviewed prices, USD per million units, checked 2026-10-01:

| Model | Input | Output |
| --- | ---: | ---: |
| GPT-6 Luna | $0.10/text token | $0.50/text token |
| GPT-4.1 mini | $0.40/text token | $1.60/text token |
| TTS-1 | $15/character | — |
| TTS-1 HD | $30/character | — |
| GPT-4o mini TTS | $0.60/text token | $12/audio token |

Preflight rejects a **modeled allowance above $0.10**, using all serialized
prompt bytes plus 1,024 framing tokens and maximum text output. Legacy TTS uses
the character cap. Mini TTS uses a conservative **6,000 audio-token allowance**
plus input; its API offers no enforceable output-token/dollar cap, so this is
an estimate, **not a billing guarantee**. Prices, anomalous audio duration,
taxes, GitHub usage and deliberate later runs can change costs. Project budget
alerts are soft thresholds. The original approximately $0.03 example applied
to illustrative 4.1 mini + TTS-1 inputs, not every supported model combination.
Neither this PR nor an estimate authorizes a paid prototype run.

## Actions duplicate attempts and recovery

Jobs serialize separately from the build pipeline. Before paid requests, the
GitHub API's exact `name` filter looks up the deterministic reservation name;
unrelated repository artifacts do not enter pagination. A bounded lookup of up
to 10,000 matching reservations fails closed on errors or overflow. A reservation
artifact is uploaded first, keyed by repository, sorted release IDs and mode.
Matching attempts are skipped even after failure or
source/model edits; explicit manual `force_regenerate=true` permits another
paid attempt. Rollups and single releases are different source sets.
Use a fresh manual dispatch for forced generation. Reservation names repeat
across runs but are immutable within a run: a rerun of an already-reserved
forced attempt fails at upload before any paid request, preserving the ledger.

Reservations last 90 days, review outputs/script checkpoints 30 days, limited
by repository policy. Deleted/expired artifacts permit another attempt, so
this is best-effort deduplication, not a permanent billing ledger. GitHub can
replace an older pending concurrency run; dispatch it manually if needed.

Both paid stages re-fetch notes and stop if changed/withdrawn. Scripts are
checkpointed before TTS. Invalid scripts never reach speech, and error JSON
never becomes MP3. `ffprobe` checks codec/duration and `ffmpeg` decodes the full
file, restricted to MP3 and local file/pipe protocols. Duration deviations are
flagged without regeneration. Manifests record source/prompt/script/audio hashes,
the generation fingerprint, commits, config, request counts, text usage and
duration. Child media/git commands do not inherit API credentials.

Do not force regeneration to fix delivery. If the runner/checkpoint is lost,
automatic cross-run checkpoint recovery is not implemented: review the saved
script and decide whether a new deliberate attempt is warranted. It may be
billed. Actions artifacts expire and require signed-in repository read access;
they are not permanent anonymous podcast URLs.

## Offline verification

```sh
go test -race -count=1 -v ./release/podcast
```

HTTP transports are mocked and unexpected requests fail. Public v0.64.0/.1/.2
fixtures test omissions, unsafe versions/output, inclusive ranges, paid guards,
local locking/reservations, changed sources/checkpoints, Actions trust/config,
artifact expiry/lookup limits, budget caps, redirects and no retries. Real MP3
decoding uses a local synthetic tone with mocked OpenAI. CI installs media
tools so that test runs rather than skips. The tool uses Go's standard library;
no Python implementation or additional Go module dependency is required.

## References

- [GitHub release events](https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows#release)
- [GITHUB_TOKEN event suppression](https://docs.github.com/en/actions/how-tos/writing-workflows/choosing-when-your-workflow-runs/triggering-a-workflow)
- [GitHub exact-name artifact filter](https://docs.github.com/en/rest/actions/artifacts#list-artifacts-for-a-repository)
- [SemVer prerelease precedence](https://semver.org/)
- [GPT-6 Luna](https://developers.openai.com/api/docs/models/gpt-6-luna)
- [GPT-4.1 mini](https://developers.openai.com/api/docs/models/gpt-4.1-mini)
- [Mini TTS](https://developers.openai.com/api/docs/models/gpt-4o-mini-tts)
- [TTS-1](https://developers.openai.com/api/docs/models/tts-1)
- [TTS-1 HD](https://developers.openai.com/api/docs/models/tts-1-hd)
- [Speech guide](https://developers.openai.com/api/docs/guides/text-to-speech)
