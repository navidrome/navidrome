# Release audio

`Release audio` turns **published GitHub release notes** into a grounded English
recap, then an MP3, with a transcript and source/evidence records in Actions
artifacts. The target is about two minutes: 250–280 words and 105–145 seconds.
Small releases may be shorter; correctness takes priority over filler. Nothing
is uploaded to release assets or social media. Audio failure cannot block the
existing software release pipeline.

## Setup (maintainer)

Generation is **disabled by default**. No model or voice is selected by default.
Choose models and audition a stock voice before enabling production generation.

1. Review the workflow/helper and model pricing profiles below.
2. Set repository variables `RELEASE_AUDIO_TEXT_MODEL`,
   `RELEASE_AUDIO_TTS_MODEL`, and `RELEASE_AUDIO_VOICE` to reviewed options.
3. Add `OPENAI_API_KEY` yourself in GitHub Actions Secrets. Use a dedicated
   OpenAI project key with the necessary model/endpoint access and usage alerts.
4. After deciding to permit paid generation, set the repository variable
   `RELEASE_AUDIO_ENABLED` to the literal `true`.
5. Once merged, run **Actions → Release audio → Run workflow** on `master`.
   Start with `validate` and `v0.64.0,v0.64.1,v0.64.2`, then explicitly choose
   `script` or `audio` when ready to incur API usage.
6. Compare the transcript and evidence with the sources, then listen to the
   entire MP3 before public use. Every transcript discloses the AI voice.

Supported options (not finalized user choices):

| Setting | Reviewed options |
| --- | --- |
| Text model | `gpt-6-luna` (low reasoning), `gpt-4.1-mini-2025-04-14` |
| Speech model | `gpt-4o-mini-tts-2025-12-15`, `tts-1`, `tts-1-hd` |
| Legacy voices | `alloy`, `echo`, `fable`, `onyx`, `nova`, `shimmer` |
| Mini TTS voices | Legacy voices plus `ash`, `ballad`, `coral`, `sage`, `verse`, `marin`, `cedar` |

Onyx and cedar are candidates to audition for the requested male-style delivery;
perceived voice gender is subjective. The API selects a stock voice by name.
Mini TTS receives fixed calm English speaking instructions. Legacy TTS models do
not accept those instructions. Unsupported models fail before paid requests;
adding a model requires a code/pricing review. Luna is a verified API alias;
unlike the pinned 4.1/Mini TTS snapshots, its behavior can change over time.

## Publication and manual modes

The automatic trigger is `release: published`, with draft and prerelease guards.
Automatic generation is skipped while the enablement variable is unset/false.
It resolves the **event's release ID**, never `/releases/latest`. Tag pushes,
note edits, and unpublished drafts do not generate audio. Promotion of an
already-published prerelease may require manual dispatch. Manual input accepts
at most three distinct version tags, sorted by numeric version; it produces
one combined recap. Prereleases need the explicit manual opt-in.

The current `pipeline.yml` calls GoReleaser with `GITHUB_TOKEN`, and
`release/goreleaser.yml` has `draft: true`. This implementation leaves that
publishing behavior intact. Publish the prepared draft through GitHub as a
maintainer to trigger audio. **Publication by `GITHUB_TOKEN` generally suppresses
downstream release events.** If that becomes the publication method, dispatch
this workflow manually after successful publication, or separately review an
explicit `workflow_dispatch` integration with narrow Actions permissions.
Do not add a broad PAT or change draft publication to make this work.

Land the workflow before the next release tag. GitHub associates release events
with the tagged commit, so historical tags cannot be assumed to contain a newly
added workflow. Dispatch from the default branch for the 0.64 prototype; do not
move tags. Manual dispatch requires the workflow on the default branch. The
helper always executes from the reviewed default branch, with credentials not
persisted in checkout. Branch protection should protect this implementation.

| Mode | OpenAI requests | Output |
| --- | --- | --- |
| `validate` | None | Exact sources, manifest, modeled estimate when models are configured |
| `script` | At most one Responses request | Transcript, evidence/caution map, sources, manifest |
| `audio` | At most one Responses request and one speech request | Script outputs plus validated MP3 |

## Limits, evidence, and costs

There are no HTTP/SDK/repair retries. Timeouts can already be billed. Requests
have a 60-second timeout; the job has a 10-minute timeout. Sources and the full
prompt have separate 64 KiB byte caps. Oversized material fails without
truncation. Text output is capped at 3,000 tokens (including reasoning and the
structured evidence map); narration at 280 words and 2,500 characters.
Mini TTS adds a conservative 2,000 UTF-8-byte input ceiling, including speaking
instructions, to stay below its 2,000-input-token limit without a tokenizer
dependency. A dense script may need shortening; it is never truncated.

The versioned prompt treats notes as untrusted evidence and requests strict JSON
with sentence-to-source excerpts and required caution coverage. Source IDs and
literal excerpts must match. Migration paragraphs, security messages, and
experimental/opt-in warnings must be mapped; lexical guards preserve the
prototype's backup, client-resync, plugin networking, Docker discovery, and
32-bit/slow-storage scan cautions. Neither literal matches nor keyword checks
prove a paraphrase is true. **Human factual and listening review remains
required.** English/markup checks are heuristics, not a language classifier.
No source links or draft advisories are fetched. Only public release-note text
and the validated script go to fixed OpenAI endpoints. `store: false` does not
imply zero provider retention.

Rates checked 2026-10-01, USD per million units:

| Model | Input | Output |
| --- | ---: | ---: |
| GPT-6 Luna | $0.10/text token | $0.50/text token |
| GPT-4.1 mini | $0.40/text token | $1.60/text token |
| TTS-1 | $15/character | — |
| TTS-1 HD | $30/character | — |
| GPT-4o mini TTS | $0.60/text token | $12/audio token |

Preflight rejects a modeled request allowance above **$0.10**, counting the
whole serialized prompt as UTF-8 bytes plus 1,024 framing tokens and maximum
text output. Legacy speech uses the character ceiling. Mini TTS uses a
conservative **6,000 audio-token allowance** plus input. Its API does not expose
an enforceable output-token or dollar cap, so this is a modeled allowance,
not a billing guarantee. The transcript/request caps bound the work; changed
prices, anomalous speech length, taxes, GitHub usage, and subsequent deliberate
runs remain outside the estimate. Do not reuse legacy character pricing for
Mini TTS. Project budget alerts are soft thresholds, not hard spend caps.

The original approximately $0.03 single-pass estimate described illustrative
4.1 mini + TTS-1 inputs. It is not a promise for every supported combination,
and neither that estimate nor this PR authorizes a paid prototype run.

## Duplicate attempts, checkpoints, and retention

All release-audio jobs serialize separately from the build pipeline. Before
any paid request, the helper checks up to 10,000 repository artifacts, failing
closed if the lookup fails or exceeds that bound. A reservation artifact is
uploaded **before** generation, keyed by repository, sorted release IDs, and
mode. Attempts with the same key are skipped even after failure or source/model
changes. `force_regenerate=true` is a deliberate manual opt-in to another paid
attempt. Script and audio modes have separate keys; a rollup and a single
release are different source sets. Force attempts are also recorded.

Reservation artifacts last 90 days, limited by repository retention policy;
review outputs and script checkpoints last 30 days. This is best-effort
deduplication within retained artifacts, not a permanent exactly-once ledger.
Deleting/expiring reservations permits another attempt. GitHub may replace an
older pending concurrency run; dispatch that run manually if needed.

The validated script is checkpointed before speech. Sources are fetched again
before each paid stage; unpublished/deleted/edited notes stop the attempt.
Invalid scripts never reach TTS; error JSON never becomes an MP3. `ffprobe`
checks codec and duration, and `ffmpeg` decodes the entire MP3 before acceptance.
Duration deviations are flagged without regenerating. Manifests include
implementation/event commits, source/prompt/script/audio hashes, models, voice,
and a complete generation fingerprint,
request counts, text usage, and duration. Failed attempts retain safe diagnostics
and any completed checkpoints through the final artifact step.

A rerun skips a reserved attempt; do not force regeneration to fix an upload.
If the runner/files are lost, this implementation deliberately has no automatic
cross-run script recovery: download the checkpoint for review, and decide
whether a new manual forced attempt is warranted. That attempt may be billed.
Artifacts need a signed-in GitHub account with repository read access and expire;
they are not permanent anonymous podcast URLs.

## Offline verification

```sh
python3 -m unittest discover -s release/audio -p 'test_*.py' -v
```

The suite blocks unmocked network access. Fixtures preserve the public bodies
of v0.64.0/.1/.2 and test consequential omissions, unsafe tags/output, disabled
generation, exact event identity, changing notes, duplicates/expiry/force,
budget caps, error responses, and no retries. The separate PR test workflow
never receives `OPENAI_API_KEY`. Generation uses only Python's standard library;
the Ubuntu runner must have `ffprobe` and `ffmpeg` before speech is requested.

## References

- [GitHub release events](https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows#release)
- [GITHUB_TOKEN event suppression](https://docs.github.com/en/actions/how-tos/writing-workflows/choosing-when-your-workflow-runs/triggering-a-workflow)
- [GPT-6 Luna](https://developers.openai.com/api/docs/models/gpt-6-luna)
- [GPT-4.1 mini](https://developers.openai.com/api/docs/models/gpt-4.1-mini)
- [Mini TTS](https://developers.openai.com/api/docs/models/gpt-4o-mini-tts)
- [TTS-1](https://developers.openai.com/api/docs/models/tts-1)
- [TTS-1 HD](https://developers.openai.com/api/docs/models/tts-1-hd)
- [Speech API guide and AI-voice disclosure](https://developers.openai.com/api/docs/guides/text-to-speech)
