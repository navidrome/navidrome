#!/usr/bin/env python3
"""Bounded, artifact-only release narration; Python standard library only."""

import argparse
import hashlib
import json
import math
import os
import re
import subprocess
import sys
import urllib.error
import urllib.parse
import urllib.request
from datetime import datetime, timezone
from pathlib import Path

REPOSITORY = "navidrome/navidrome"
OUT = Path("release-audio")
PROMPT = Path(__file__).with_name("prompt.txt").read_text(encoding="utf-8")
MAX_SOURCE_BYTES = 65536
MAX_PROMPT_BYTES = 65536
MAX_OUTPUT_TOKENS = 3000
MAX_SCRIPT_CHARS = 2500
MINI_TTS_INPUT_BYTES = 2000
MAX_COST_USD = 0.10
# Supported profiles are options, NOT a selected/default model. Unknown pricing
# must be reviewed here before a model can be used. Rates checked 2026-10-01.
TEXT_MODELS = {"gpt-4.1-mini-2025-04-14": (0.40, 1.60), "gpt-6-luna": (0.10, 0.50)}
TTS_MODELS = {"tts-1": 15.0, "tts-1-hd": 30.0, "gpt-4o-mini-tts-2025-12-15": None}
VOICES = {"alloy", "echo", "fable", "onyx", "nova", "shimmer"}
MODERN_VOICES = VOICES | {"ash", "ballad", "coral", "sage", "verse", "marin", "cedar"}
SPEECH_INSTRUCTIONS = (
    "Speak in clear, calm English with a warm, measured delivery. Read API as letters."
)
# A conservative modeled allowance, NOT an enforceable speech output-token cap.
MODELED_AUDIO_TOKENS = 6000
TAG = re.compile(
    r"v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-[A-Za-z0-9]+(?:[.-][A-Za-z0-9]+)*)?"
)
INTRO = (
    "Welcome to the Navidrome release recap. This narration uses an AI-generated voice."
)
CLOSING = "For the full details and upgrade guidance, read the official release notes linked alongside this transcript."
SCHEMA = {
    "type": "object",
    "additionalProperties": False,
    "required": ["sentences", "cautions"],
    "properties": {
        "sentences": {
            "type": "array",
            "items": {
                "type": "object",
                "additionalProperties": False,
                "required": ["text", "source_id", "excerpt"],
                "properties": {
                    key: {"type": "string"} for key in ("text", "source_id", "excerpt")
                },
            },
        },
        "cautions": {
            "type": "array",
            "items": {
                "type": "object",
                "additionalProperties": False,
                "required": ["caution_id", "sentence_index"],
                "properties": {
                    "caution_id": {"type": "string"},
                    "sentence_index": {"type": "integer"},
                },
            },
        },
    },
}


class AudioError(Exception):
    """Safe diagnostics: never include remote bodies, headers or secrets."""


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, *_args):
        return None


def digest(value):
    return hashlib.sha256(value).hexdigest()


def canonical(value):
    return json.dumps(value, sort_keys=True, ensure_ascii=False).encode("utf-8")


def write_json(name, value):
    OUT.mkdir(exist_ok=True)
    (OUT / name).write_text(
        json.dumps(value, indent=2, ensure_ascii=False) + "\n", encoding="utf-8"
    )


def read_json(name):
    return json.loads((OUT / name).read_text(encoding="utf-8"))


def request(url, token, payload=None, limit=1048576):
    # No SDK retries; even a timeout may already have incurred a charge.
    req = urllib.request.Request(
        url, data=canonical(payload) if payload is not None else None
    )
    req.add_header("Authorization", "Bearer " + token)
    req.add_header("Content-Type", "application/json")
    req.add_header("Accept", "application/json" if payload is None else "*/*")
    try:
        with urllib.request.build_opener(NoRedirect()).open(
            req, timeout=60
        ) as response:
            data = response.read(limit + 1)
            content_type = response.headers.get_content_type()
    except urllib.error.HTTPError as exc:
        exc.close()
        raise AudioError(f"HTTP {exc.code}; requests are not retried") from None
    except (urllib.error.URLError, TimeoutError, OSError):
        raise AudioError("Network request failed; requests are not retried") from None
    if len(data) > limit:
        raise AudioError("Response exceeds the size limit")
    return data, content_type


def github(path):
    data, _ = request(
        "https://api.github.com/repos/" + REPOSITORY + path, os.environ["GH_TOKEN"]
    )
    return json.loads(data)


def parse_tags(raw):
    tags = [tag.strip() for tag in raw.split(",")]
    if not 1 <= len(tags) <= 3 or len(set(tags)) != len(tags):
        raise AudioError("Supply one to three distinct release tags")
    if any(len(tag) > 80 or not TAG.fullmatch(tag) for tag in tags):
        raise AudioError("Tags must be versions such as v0.64.2")
    return sorted(
        tags, key=lambda tag: tuple(int(n) for n in TAG.fullmatch(tag).groups())
    )


def normalize_release(record, allow_prerelease=False):
    tag = record.get("tag_name", "")
    if not isinstance(tag, str) or len(tag) > 80 or not TAG.fullmatch(tag):
        raise AudioError("Release tag must be a single supported version")
    parse_tags(tag)
    if record.get("draft") is not False or not record.get("published_at"):
        raise AudioError("Only published releases are eligible")
    if record.get("prerelease") is not False and not allow_prerelease:
        raise AudioError("Prereleases require manual opt-in")
    body = record.get("body")
    if not isinstance(body, str) or not body.strip():
        raise AudioError("Release notes are empty; add notes before generating")
    if type(record.get("id")) is not int or record["id"] <= 0:
        raise AudioError("Invalid release ID")
    if len(body.encode("utf-8")) > MAX_SOURCE_BYTES:
        raise AudioError(
            "Release notes exceed the source limit; review them without truncation"
        )
    return {
        "id": record["id"],
        "source_id": str(record["id"]),
        "tag": tag,
        "url": "https://github.com/"
        + REPOSITORY
        + "/releases/tag/"
        + urllib.parse.quote(tag, safe=""),
        "published_at": record["published_at"],
        "updated_at": record.get("updated_at"),
        "body": body.replace("\r\n", "\n"),
        "prerelease": record["prerelease"],
    }


def source_input(sources):
    # Drop only HTML comments. Preserve all sections, including anything added
    # after the promotional footer: a warning must never be silently truncated.
    result = []
    for source in sources:
        body = re.sub(r"<!--.*?-->", "", source["body"], flags=re.S)
        result.append(
            {"source_id": source["source_id"], "tag": source["tag"], "body": body}
        )
    return result


def cautions(sources):
    # Every migration paragraph is mandatory. Security is grouped per release;
    # behavioral qualifiers outside those sections are also mandatory.
    result = []
    for source in source_input(sources):
        in_migration = False
        security_added = False
        for line in source["body"].splitlines():
            if line.startswith("## "):
                in_migration = bool(re.search(r"breaking|migration", line, re.I))
            mandatory = in_migration and line.startswith("- ")
            if re.search(r"security", line, re.I) and not security_added:
                mandatory = True
                security_added = True
            mandatory |= bool(
                re.search(
                    r"back.?up|re-?sync|experimental|opt-in|disabled|host networking",
                    line,
                    re.I,
                )
            )
            if mandatory:
                result.append(
                    {
                        "caution_id": f"c{len(result)}",
                        "source_id": source["source_id"],
                        "excerpt": line,
                    }
                )
    return result


def config(mode):
    text = os.environ.get("AUDIO_TEXT_MODEL", "")
    tts = os.environ.get("AUDIO_TTS_MODEL", "")
    voice = os.environ.get("AUDIO_VOICE", "")
    if mode != "validate" and text not in TEXT_MODELS:
        raise AudioError("Set RELEASE_AUDIO_TEXT_MODEL to a reviewed supported model")
    voices = MODERN_VOICES if tts == "gpt-4o-mini-tts-2025-12-15" else VOICES
    if mode == "audio" and (tts not in TTS_MODELS or voice not in voices):
        raise AudioError(
            "Set RELEASE_AUDIO_TTS_MODEL and RELEASE_AUDIO_VOICE to reviewed options"
        )
    return {
        "text_model": text,
        "tts_model": tts,
        "voice": voice,
        "speed": 1.0,
        "speech_instructions": SPEECH_INSTRUCTIONS
        if tts == "gpt-4o-mini-tts-2025-12-15"
        else "",
    }


def narration_byte_limit(selected):
    if selected["speech_instructions"]:
        return MINI_TTS_INPUT_BYTES - len(selected["speech_instructions"].encode())
    return MAX_SCRIPT_CHARS


def text_payload(sources, selected):
    payload = {
        "model": selected["text_model"],
        "store": False,
        "max_output_tokens": MAX_OUTPUT_TOKENS,
        "instructions": PROMPT,
        "input": json.dumps(
            {
                "sources": source_input(sources),
                "required_cautions": cautions(sources),
                "introduction": INTRO,
                "closing": CLOSING,
                "narration_limits": {
                    "max_words": 280,
                    "max_characters": MAX_SCRIPT_CHARS,
                    "max_utf8_bytes": narration_byte_limit(selected),
                },
            },
            ensure_ascii=False,
        ),
        "text": {
            "format": {
                "type": "json_schema",
                "name": "release_narration",
                "strict": True,
                "schema": SCHEMA,
            }
        },
    }
    if selected["text_model"] == "gpt-6-luna":
        payload["reasoning"] = {"effort": "low"}
    return payload


def cost_bound(payload, selected, mode):
    size = len(canonical(payload))
    if size > MAX_PROMPT_BYTES:
        raise AudioError(
            "Complete prompt exceeds its byte limit; sources must be reviewed, never truncated"
        )
    if selected["text_model"] not in TEXT_MODELS:
        return None
    # Byte-level tokenizers cannot emit more text tokens than UTF-8 bytes;
    # add 1024 for protocol framing. Count the whole JSON envelope, schema too.
    input_rate, output_rate = TEXT_MODELS[selected["text_model"]]
    bound = ((size + 1024) * input_rate + MAX_OUTPUT_TOKENS * output_rate) / 1000000
    if mode == "audio":
        if TTS_MODELS[selected["tts_model"]] is None:
            # English speech input: at most 2500 ASCII characters + instruction
            # bytes, conservatively treated as tokens. Enforce 2000 below too.
            bound += (
                (MAX_SCRIPT_CHARS + len(SPEECH_INSTRUCTIONS.encode())) * 0.60
                + MODELED_AUDIO_TOKENS * 12
            ) / 1000000
        else:
            bound += MAX_SCRIPT_CHARS * TTS_MODELS[selected["tts_model"]] / 1000000
    if bound > MAX_COST_USD:
        raise AudioError(
            "Modeled request cost exceeds $0.10; review models or source size"
        )
    return round(bound, 6)


def duplicate(reservation):
    # Fail closed if the ledger cannot be fully scanned within this bound.
    for page in range(1, 101):
        records = github(f"/actions/artifacts?per_page=100&page={page}")["artifacts"]
        if any(
            (item["name"] == reservation or item["name"].startswith(reservation + "-"))
            and not item["expired"]
            for item in records
        ):
            return True
        if len(records) < 100:
            return False
    raise AudioError("Artifact ledger exceeds lookup limit; manual review required")


def emit_output(key, value):
    path = os.environ.get("GITHUB_OUTPUT")
    if path:
        with open(path, "a", encoding="utf-8") as output:
            output.write(f"{key}={value}\n")


def summary(message):
    path = os.environ.get("GITHUB_STEP_SUMMARY")
    if path:
        with open(path, "a", encoding="utf-8") as output:
            output.write(message + "\n")


def prepare():
    event = json.loads(
        Path(os.environ["GITHUB_EVENT_PATH"]).read_text(encoding="utf-8")
    )
    if os.environ.get("GITHUB_REPOSITORY") != REPOSITORY:
        raise AudioError("This workflow is restricted to navidrome/navidrome")
    manual = os.environ.get("GITHUB_EVENT_NAME") == "workflow_dispatch"
    inputs = event.get("inputs", {}) if manual else {}
    allow = inputs.get("include_prereleases") in (True, "true")
    force = inputs.get("force_regenerate") in (True, "true")
    mode = inputs.get("mode", "validate") if manual else "audio"
    if mode not in {"validate", "script", "audio"}:
        raise AudioError("Invalid mode")
    if manual:
        expected_ref = "refs/heads/" + event["repository"]["default_branch"]
        if os.environ.get("GITHUB_REF") != expected_ref:
            raise AudioError("Manual generation must use the default branch")
        sources = [
            normalize_release(
                github("/releases/tags/" + urllib.parse.quote(tag, safe="")), allow
            )
            for tag in parse_tags(inputs.get("tags", ""))
        ]
    else:
        if event.get("action") != "published":
            raise AudioError("Expected a release published event")
        normalize_release(event["release"])
        sources = [normalize_release(github(f"/releases/{event['release']['id']}"))]
        if sources[0]["tag"] != event["release"]["tag_name"]:
            raise AudioError("Release identity changed")
    if len({source["id"] for source in sources}) != len(sources):
        raise AudioError("Duplicate release IDs")
    if (
        sum(len(source["body"].encode("utf-8")) for source in sources)
        > MAX_SOURCE_BYTES
    ):
        raise AudioError("Combined sources exceed the limit")
    selected = config(mode)
    payload = text_payload(sources, selected)
    bound = cost_bound(payload, selected, mode)
    # Reserve by release identities AND mode; a script-only run may later get audio.
    # Model/source changes do not silently bypass the paid-attempt ledger.
    reservation = "release-audio-attempt-" + digest(
        canonical(
            {"repo": REPOSITORY, "ids": sorted(s["id"] for s in sources), "mode": mode}
        )
    )
    enabled = os.environ.get("AUDIO_ENABLED") == "true"
    if mode != "validate" and not enabled:
        raise AudioError(
            "Paid generation is disabled; set RELEASE_AUDIO_ENABLED after reviewing setup"
        )
    if mode != "validate" and os.environ.get("AUDIO_KEY_CONFIGURED") != "true":
        raise AudioError(
            "Add OPENAI_API_KEY in Actions Secrets before reserving a paid attempt"
        )
    repeated = mode != "validate" and duplicate(reservation)
    manifest = {
        "version": 1,
        "mode": mode,
        "config": selected,
        "cost_bound_usd": bound,
        "source_sha256": digest(canonical(sources)),
        "prompt_sha256": digest(PROMPT.encode()),
        "implementation_sha": subprocess.run(
            ["git", "rev-parse", "HEAD"], capture_output=True, text=True, check=True
        ).stdout.strip(),
        "event_sha": os.environ.get("GITHUB_SHA", ""),
        "run_id": os.environ.get("GITHUB_RUN_ID", ""),
        "run_attempt": os.environ.get("GITHUB_RUN_ATTEMPT", ""),
        "created_at": datetime.now(timezone.utc).isoformat(),
        "reservation": reservation,
        "cost_is_modeled": True,
        "modeled_audio_tokens": MODELED_AUDIO_TOKENS
        if selected["speech_instructions"]
        else None,
        "requests": {"text": 0, "speech": 0},
        "force_regenerate": force,
        "include_prereleases": allow,
        "status": "duplicate" if repeated and not force else "prepared",
    }
    manifest["generation_sha256"] = digest(
        canonical(
            {
                "sources": sources,
                "config": selected,
                "prompt": PROMPT,
                "implementation": manifest["implementation_sha"],
            }
        )
    )
    write_json("sources.json", sources)
    write_json("manifest.json", manifest)
    emit_output(
        "reservation", reservation + f"-{manifest['run_id']}-{manifest['run_attempt']}"
    )
    emit_output("prepared", "true")
    emit_output(
        "generate",
        str(mode != "validate" and manifest["status"] != "duplicate").lower(),
    )
    summary(
        f"Release audio: {manifest['status']}; mode: {mode}. Modeled API bound: {bound} USD.\n"
        "Review artifacts contain the exact published sources. Listen and compare the transcript before distribution."
    )


def recheck_sources(manifest, sources):
    latest = [
        normalize_release(
            github(f"/releases/{source['id']}"), manifest["include_prereleases"]
        )
        for source in sources
    ]
    if digest(canonical(latest)) != manifest["source_sha256"]:
        raise AudioError(
            "Published notes changed; stop and review a new manual attempt"
        )


def validate_script(result, sources):
    if not isinstance(result, dict) or set(result) != {"sentences", "cautions"}:
        raise AudioError("Invalid narration schema")
    sentences = result["sentences"]
    if not isinstance(sentences, list) or not 1 <= len(sentences) <= 35:
        raise AudioError("Invalid narration sentence count")
    by_id = {source["source_id"]: source for source in sources}
    for sentence in sentences:
        if not isinstance(sentence, dict) or set(sentence) != {
            "text",
            "source_id",
            "excerpt",
        }:
            raise AudioError("Invalid sentence schema")
        if not all(
            isinstance(value, str) and value.strip() for value in sentence.values()
        ):
            raise AudioError("Empty or invalid sentence evidence")
        source = by_id.get(sentence["source_id"])
        if (
            source is None
            or len(sentence["excerpt"]) < 12
            or sentence["excerpt"] not in source["body"]
        ):
            raise AudioError("Sentence evidence is absent from its source")
    required = {item["caution_id"]: item for item in cautions(sources)}
    covered = set()
    if not isinstance(result["cautions"], list):
        raise AudioError("Invalid caution coverage")
    for item in result["cautions"]:
        if not isinstance(item, dict) or set(item) != {"caution_id", "sentence_index"}:
            raise AudioError("Invalid caution schema")
        index = item["sentence_index"]
        caution = required.get(item["caution_id"])
        if caution is None or type(index) is not int or not 0 <= index < len(sentences):
            raise AudioError("Invalid caution reference")
        if sentences[index]["source_id"] != caution["source_id"]:
            raise AudioError("Caution mapped to the wrong release")
        covered.add(item["caution_id"])
    if covered != set(required):
        raise AudioError("Missing required migration/security/qualifier coverage")
    text = (
        INTRO
        + "\n\n"
        + " ".join(s["text"].strip() for s in sentences)
        + "\n\n"
        + CLOSING
    )
    if not 100 <= len(text.split()) <= 280 or len(text) > MAX_SCRIPT_CHARS:
        raise AudioError("Narration must be 100-280 words and at most 2500 characters")
    if re.search(r"https?://|www\.|[@`<>{}\[\]#]|\$\(|\x00|[\x01-\x08\x0b-\x1f]", text):
        raise AudioError("Narration contains markup, URL, handle or executable content")
    if sum(ord(c) < 128 for c in text) / len(text) < 0.95:
        raise AudioError("Narration must be English plain text")
    if any(source["tag"].removeprefix("v") not in text for source in sources):
        raise AudioError("Narration must identify each release version")
    validate_qualifiers(sentences, sources)
    return text + "\n"


def validate_qualifiers(sentences, sources):
    # Lexical guards for consequential source conditions. These complement the
    # evidence map; neither can prove semantic entailment. Human review remains.
    rules = [
        (
            r"back up your database before upgrading",
            [r"back.?up", r"database", r"before.{0,40}upgrad"],
        ),
        (r"may need to re-sync", [r"re.?sync"]),
        (
            r"experimental Jellyfin",
            [r"experimental", r"enabl|opt.in|default.off|disabled by default"],
        ),
        (
            r"Plugin authors",
            [r"plugin", r"host.{0,20}HTTP|host.{0,20}network", r"private|loopback|LAN"],
        ),
        (r"security release.*?Upgrade", [r"security", r"upgrad"]),
        (r"opt-in LAN auto-discovery", [r"opt.in", r"Docker", r"host networking"]),
        (r"slow storage", [r"slow.{0,20}storage", r"scan|lock"]),
        (r"32-bit builds", [r"32.bit", r"scan"]),
    ]
    for source in sources:
        narration = [
            s["text"] for s in sentences if s["source_id"] == source["source_id"]
        ]
        body = re.sub(r"[*`]", "", source["body"])
        for trigger, requirements in rules:
            if re.search(trigger, body, re.I | re.S) and not any(
                all(re.search(term, sentence, re.I) for term in requirements)
                for sentence in narration
            ):
                raise AudioError(
                    "Narration omits a consequential source qualifier or action"
                )


def openai_json(payload):
    data, content_type = request(
        "https://api.openai.com/v1/responses", os.environ["OPENAI_API_KEY"], payload
    )
    if content_type != "application/json":
        raise AudioError("Text API returned an unexpected content type")
    result = json.loads(data)
    if result.get("status") != "completed":
        raise AudioError("Text response incomplete or refused")
    parts = [
        part
        for output in result.get("output", [])
        if output.get("type") == "message"
        for part in output.get("content", [])
    ]
    if any(part.get("type") == "refusal" for part in parts):
        raise AudioError("Text response refused")
    texts = [part["text"] for part in parts if part.get("type") == "output_text"]
    if len(texts) != 1:
        raise AudioError("Expected one structured narration")
    return json.loads(texts[0]), result.get("usage")


def load_stage(stage):
    manifest, sources = read_json("manifest.json"), read_json("sources.json")
    if os.environ.get("AUDIO_ENABLED") != "true" or not os.environ.get(
        "OPENAI_API_KEY"
    ):
        raise AudioError("Paid generation needs explicit enablement and OPENAI_API_KEY")
    if config(manifest["mode"]) != manifest["config"]:
        raise AudioError("Model configuration changed after preparation")
    if manifest["mode"] == "validate" or manifest["requests"][stage] != 0:
        raise AudioError("Stage not authorized or already attempted")
    recheck_sources(manifest, sources)
    if stage == "text" and manifest["status"] != "prepared":
        raise AudioError("Script stage requires a fresh prepared attempt")
    return manifest, sources


def script():
    manifest, sources = load_stage("text")
    payload = text_payload(sources, manifest["config"])
    cost_bound(payload, manifest["config"], manifest["mode"])
    manifest["requests"]["text"] = 1
    manifest["status"] = "script_requested"
    write_json("manifest.json", manifest)
    result, usage = openai_json(payload)
    text = validate_script(result, sources)
    if len(text.rstrip("\n").encode()) > narration_byte_limit(manifest["config"]):
        raise AudioError(
            "Narration exceeds the configured speech input limit; review a shorter script"
        )
    (OUT / "transcript.txt").write_text(text, encoding="utf-8")
    write_json("evidence.json", result)
    manifest.update(
        status="script_validated",
        text_usage=usage,
        script_sha256=digest(text.encode()),
        word_count=len(text.split()),
        character_count=len(text.rstrip("\n")),
    )
    write_json("manifest.json", manifest)


def speech():
    manifest, sources = load_stage("speech")
    if manifest["mode"] != "audio" or manifest["status"] != "script_validated":
        raise AudioError("Speech requires a validated audio-mode script")
    text = (OUT / "transcript.txt").read_text(encoding="utf-8")
    if (
        digest(text.encode()) != manifest["script_sha256"]
        or validate_script(read_json("evidence.json"), sources) != text
    ):
        raise AudioError("Script checkpoint changed")
    if (
        not subprocess.run(
            ["ffprobe", "-version"], capture_output=True, check=False
        ).returncode
        == 0
    ):
        raise AudioError("ffprobe is required before speech generation")
    if (
        not subprocess.run(
            ["ffmpeg", "-version"], capture_output=True, check=False
        ).returncode
        == 0
    ):
        raise AudioError("ffmpeg is required before speech generation")
    selected = manifest["config"]
    payload = {
        "model": selected["tts_model"],
        "voice": selected["voice"],
        "input": text.rstrip("\n"),
        "response_format": "mp3",
        "speed": selected["speed"],
    }
    if selected["speech_instructions"]:
        payload["instructions"] = selected["speech_instructions"]
        # Mini TTS accepts at most 2000 input tokens. No dependency/tokenizer:
        # use UTF-8 bytes as a conservative upper bound and fail without truncation.
        if (
            len((payload["input"] + payload["instructions"]).encode())
            > MINI_TTS_INPUT_BYTES
        ):
            raise AudioError(
                "Mini TTS conservative input-token limit exceeded; review a shorter script"
            )
    manifest["requests"]["speech"] = 1
    manifest["status"] = "speech_requested"
    write_json("manifest.json", manifest)
    data, content_type = request(
        "https://api.openai.com/v1/audio/speech",
        os.environ["OPENAI_API_KEY"],
        payload,
        limit=10485760,
    )
    if (
        content_type not in {"audio/mpeg", "audio/mp3", "application/octet-stream"}
        or not data
    ):
        raise AudioError("Speech API did not return audio")
    temporary = OUT / "audio.tmp"
    temporary.write_bytes(data)
    try:
        probe = subprocess.run(
            [
                "ffprobe",
                "-v",
                "error",
                "-show_entries",
                "format=duration:stream=codec_name",
                "-of",
                "json",
                str(temporary),
            ],
            check=True,
            capture_output=True,
            text=True,
            timeout=30,
        )
        info = json.loads(probe.stdout)
        duration = float(info["format"]["duration"])
        if (
            not math.isfinite(duration)
            or duration <= 0
            or not info["streams"]
            or any(s["codec_name"] != "mp3" for s in info["streams"])
        ):
            raise AudioError("Speech response is not a valid nonempty MP3")
        subprocess.run(
            [
                "ffmpeg",
                "-v",
                "error",
                "-xerror",
                "-i",
                str(temporary),
                "-f",
                "null",
                "-",
            ],
            check=True,
            capture_output=True,
            timeout=30,
        )
        temporary.replace(OUT / "release-audio.mp3")
    finally:
        temporary.unlink(missing_ok=True)
    manifest.update(
        status="audio_validated",
        duration_seconds=duration,
        audio_sha256=digest(data),
        duration_needs_review=not 105 <= duration <= 145,
    )
    write_json("manifest.json", manifest)
    summary(
        f"MP3 validated: {duration:.1f} seconds. AI-generated voice. Listen before distribution; duration target is 105-145 seconds."
    )


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("stage", choices=["prepare", "script", "speech"])
    args = parser.parse_args()
    try:
        {"prepare": prepare, "script": script, "speech": speech}[args.stage]()
    except (
        AudioError,
        KeyError,
        ValueError,
        TypeError,
        AttributeError,
        OSError,
        subprocess.SubprocessError,
    ) as exc:
        # Unexpected exceptions are intentionally not printed: remote content
        # and a credential must never appear in logs or workflow commands.
        print(
            "Release audio failed: "
            + (
                str(exc)
                if isinstance(exc, AudioError)
                else "invalid data or unavailable local tool"
            ),
            file=sys.stderr,
        )
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
