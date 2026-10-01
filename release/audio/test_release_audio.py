"""Offline acceptance tests. Unmocked network access is always an error."""

import copy
import json
import os
import shutil
import subprocess
import tempfile
import unittest
import urllib.error
from pathlib import Path
from types import SimpleNamespace
from unittest.mock import patch

import release_audio as audio

RECORDS = json.loads(
    Path(__file__).with_name("testdata").joinpath("releases.json").read_text()
)
SOURCES = [audio.normalize_release(record) for record in RECORDS]


def narration():
    lines = [
        (
            0,
            "Version 0.64.0 introduced experimental Jellyfin music support, which must be explicitly enabled.",
        ),
        (
            0,
            "Back up your database before upgrading because internal IDs change; clients may need to resync cached IDs.",
        ),
        (
            0,
            "Plugin authors must migrate to the host HTTP service and review restrictions on private or loopback network addresses.",
        ),
        (
            0,
            "Shares now belong to their creator, and admins cannot create them for another user.",
        ),
        (
            0,
            "Negative configuration durations are rejected at startup, and unknown options produce warnings.",
        ),
        (
            0,
            "Security fixes protect library access and plugin networking, alongside improvements to artwork, sorting and playlist imports.",
        ),
        (
            0,
            "Database restore also avoids wiping existing data when the backup file is missing.",
        ),
        (
            1,
            "Version 0.64.1 is a security release fixing five vulnerabilities; upgrade as soon as practical.",
        ),
        (
            1,
            "Jellyfin client compatibility improves, and Quick Connect makes signing in easier.",
        ),
        (
            1,
            "Local discovery is opt-in, and Docker users need host networking for discovery broadcasts.",
        ),
        (
            1,
            "Smart playlists can reference another playlist by path, while the interface follows your selected language for dates.",
        ),
        (
            2,
            "Version 0.64.2 fixes scan failures and database lock contention on slow storage.",
        ),
        (2, "It also fixes scans on 32-bit builds with invalid track metadata."),
        (
            2,
            "Security fixes sanitize download names and prevent an admin password from reaching logs.",
        ),
    ]
    result = {
        "sentences": [
            {
                "text": text,
                "source_id": SOURCES[index]["source_id"],
                "excerpt": SOURCES[index]["body"][:40],
            }
            for index, text in lines
        ],
        "cautions": [],
    }
    # Short exact excerpts are only mechanical evidence in this fake response.
    # Human review is still needed for entailment; qualifier tests below ensure
    # consequential conditions cannot disappear just by filling the evidence map.
    for caution in audio.cautions(SOURCES):
        index = next(
            i
            for i, s in enumerate(result["sentences"])
            if s["source_id"] == caution["source_id"]
        )
        result["cautions"].append(
            {"caution_id": caution["caution_id"], "sentence_index": index}
        )
    return result


class ReleaseAudioTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.out = Path(self.tmp.name)
        self.patch(audio, "OUT", new=self.out)
        self.patch(
            audio.urllib.request,
            "build_opener",
            side_effect=AssertionError("Unmocked network access"),
        )
        self.env = {
            "AUDIO_ENABLED": "true",
            "AUDIO_KEY_CONFIGURED": "true",
            "AUDIO_TEXT_MODEL": "gpt-6-luna",
            "AUDIO_TTS_MODEL": "gpt-4o-mini-tts-2025-12-15",
            "AUDIO_VOICE": "onyx",
            "OPENAI_API_KEY": "test-never-a-real-key",
            "GH_TOKEN": "offline",
            "GITHUB_REPOSITORY": audio.REPOSITORY,
            "GITHUB_EVENT_NAME": "workflow_dispatch",
            "GITHUB_REF": "refs/heads/master",
            "GITHUB_RUN_ID": "123",
            "GITHUB_RUN_ATTEMPT": "1",
            "GITHUB_OUTPUT": str(self.out / "outputs"),
            "GITHUB_STEP_SUMMARY": str(self.out / "summary"),
            "GITHUB_EVENT_PATH": str(self.out / "event.json"),
        }
        env_patch = patch.dict(os.environ, self.env)
        env_patch.start()
        self.addCleanup(env_patch.stop)
        self.event = {
            "repository": {"default_branch": "master"},
            "inputs": {"tags": "v0.64.2,v0.64.0,v0.64.1", "mode": "validate"},
        }
        self.save_event()

    def patch(self, target, name, **kwargs):
        p = patch.object(target, name, **kwargs)
        value = p.start()
        self.addCleanup(p.stop)
        return value

    def save_event(self):
        Path(self.env["GITHUB_EVENT_PATH"]).write_text(json.dumps(self.event))

    def fake_github(self, path):
        if path.startswith("/actions/artifacts"):
            return {"artifacts": []}
        for record in RECORDS:
            if path in (
                "/releases/tags/" + record["tag_name"],
                f"/releases/{record['id']}",
            ):
                return copy.deepcopy(record)
        raise AssertionError(path)

    def prepare(self, mode="audio"):
        self.event["inputs"]["mode"] = mode
        self.save_event()
        self.patch(audio, "github", side_effect=self.fake_github)
        audio.prepare()

    def test_tags_are_bounded_and_sorted(self):
        self.assertEqual(
            audio.parse_tags("v0.64.2, v0.64.0,v0.64.1"),
            ["v0.64.0", "v0.64.1", "v0.64.2"],
        )
        for raw in (
            "",
            "v0.64.0,",
            "v0.64.0,v0.64.0",
            "v1.0.0,v2.0.0,v3.0.0,v4.0.0",
            "$(touch /tmp/pwn)",
            "../../foo",
            "v1.0.0\nmalicious",
            "v01.2.3",
        ):
            with self.subTest(raw=raw), self.assertRaises(audio.AudioError):
                audio.parse_tags(raw)

    def test_ineligible_releases(self):
        for values in (
            {"draft": True},
            {"prerelease": True},
            {"body": " "},
            {"published_at": None},
            {"id": -1},
            {"id": True},
            {"body": "x" * 65537},
            {"tag_name": "v0.64.0,v0.64.1"},
            {"tag_name": "v1.2.3١"},
        ):
            record = dict(RECORDS[0], **values)
            with self.subTest(values=list(values)), self.assertRaises(audio.AudioError):
                audio.normalize_release(record)

    def test_manual_prerelease_opt_in(self):
        record = dict(RECORDS[0], prerelease=True, tag_name="v0.64.0-rc.1")
        self.assertTrue(audio.normalize_release(record, True)["prerelease"])

    def test_actual_prototype_qualifiers_are_detected(self):
        text = " ".join(c["excerpt"] for c in audio.cautions(SOURCES))
        for term in (
            "back up your database",
            "re-sync",
            "Plugin authors",
            "experimental",
            "security release",
            "opt-in",
            "host networking",
        ):
            self.assertIn(term, text)

    def test_source_warnings_after_footer_are_preserved(self):
        source = dict(
            SOURCES[0],
            body="Notes\n## Helping out\nThanks\n## Migration\n- Back up before upgrade.",
        )
        self.assertIn("Back up", audio.source_input([source])[0]["body"])
        self.assertTrue(audio.cautions([source]))

    def test_validate_mode_never_contacts_openai(self):
        self.prepare("validate")
        self.assertIn("generate=false", (self.out / "outputs").read_text())
        self.assertEqual(
            audio.read_json("manifest.json")["requests"], {"text": 0, "speech": 0}
        )

    def test_default_branch_and_repository_required(self):
        for values in (
            {"GITHUB_REF": "refs/heads/untrusted"},
            {"GITHUB_REPOSITORY": "attacker/navidrome"},
        ):
            with (
                self.subTest(values=values),
                patch.dict(os.environ, values),
                self.assertRaises(audio.AudioError),
            ):
                audio.prepare()

    def test_published_event_uses_exact_release_id(self):
        self.event = {
            "action": "published",
            "release": RECORDS[2],
            "repository": {"default_branch": "master"},
        }
        self.save_event()
        calls = self.patch(audio, "github", side_effect=self.fake_github)
        with patch.dict(os.environ, {"GITHUB_EVENT_NAME": "release"}):
            audio.prepare()
        self.assertEqual(
            calls.call_args_list[0].args[0], f"/releases/{RECORDS[2]['id']}"
        )
        self.assertEqual(audio.read_json("sources.json")[0]["tag"], "v0.64.2")

    def test_paid_generation_requires_explicit_configuration(self):
        for values in (
            {"AUDIO_TEXT_MODEL": "unknown"},
            {"AUDIO_TTS_MODEL": "unknown"},
            {"AUDIO_VOICE": "custom-voice"},
        ):
            with (
                self.subTest(values=values),
                patch.dict(os.environ, values),
                self.assertRaises(audio.AudioError),
            ):
                audio.config("audio")
        with (
            patch.dict(os.environ, {"AUDIO_ENABLED": "false"}),
            self.assertRaises(audio.AudioError),
        ):
            self.prepare()
        with (
            patch.dict(os.environ, {"AUDIO_KEY_CONFIGURED": "false"}),
            self.assertRaises(audio.AudioError),
        ):
            self.prepare()

    def test_price_and_input_bounds(self):
        selected = audio.config("audio")
        payload = audio.text_payload(SOURCES, selected)
        self.assertLess(audio.cost_bound(payload, selected, "audio"), 0.10)
        self.assertEqual(payload["reasoning"], {"effort": "low"})
        self.assertFalse(payload["store"])
        self.assertNotIn("tools", payload)
        with self.assertRaises(audio.AudioError):
            audio.cost_bound(dict(payload, input="x" * 65537), selected, "audio")
        with (
            patch.object(audio, "MAX_COST_USD", 0.001),
            self.assertRaises(audio.AudioError),
        ):
            audio.cost_bound(payload, selected, "audio")

    def test_duplicate_attempt_and_expiry(self):
        for expired in (False, True):
            with patch.object(
                audio,
                "github",
                return_value={"artifacts": [{"name": "key-123-1", "expired": expired}]},
            ):
                self.assertEqual(audio.duplicate("key"), not expired)

    def test_duplicate_lookup_paginates_and_fails_closed(self):
        page = {"artifacts": [{"name": "other", "expired": False}] * 100}
        with patch.object(
            audio, "github", side_effect=[page, {"artifacts": []}]
        ) as lookup:
            self.assertFalse(audio.duplicate("key"))
            self.assertEqual(lookup.call_count, 2)
        with (
            patch.object(audio, "github", return_value=page),
            self.assertRaises(audio.AudioError),
        ):
            audio.duplicate("key")

    def test_duplicate_is_skipped_unless_manually_forced(self):
        self.event["inputs"]["mode"] = "audio"
        self.save_event()
        self.patch(audio, "github", side_effect=self.fake_github)
        with patch.object(audio, "duplicate", return_value=True):
            audio.prepare()
            self.assertEqual(audio.read_json("manifest.json")["status"], "duplicate")
            self.event["inputs"]["force_regenerate"] = "true"
            self.save_event()
            audio.prepare()
            self.assertEqual(audio.read_json("manifest.json")["status"], "prepared")

    def test_accepts_grounded_prototype(self):
        text = audio.validate_script(narration(), SOURCES)
        self.assertIn("AI-generated voice", text)
        self.assertIn("0.64.2", text)

    def test_missing_evidence_or_cautions_are_rejected(self):
        for mutation in (
            lambda r: r["sentences"][0].update(source_id="unknown"),
            lambda r: r["sentences"][0].update(excerpt="invented unsupported excerpt"),
            lambda r: r["cautions"].pop(),
            lambda r: r["cautions"][0].update(sentence_index=999),
        ):
            result = narration()
            mutation(result)
            with self.subTest(mutation=mutation), self.assertRaises(audio.AudioError):
                audio.validate_script(result, SOURCES)

    def test_security_migration_and_opt_in_omissions_are_rejected(self):
        for term in (
            "Back up",
            "resync",
            "experimental",
            "host HTTP",
            "opt-in",
            "host networking",
            "32-bit",
            "slow storage",
        ):
            result = narration()
            for sentence in result["sentences"]:
                sentence["text"] = sentence["text"].replace(term, "some detail")
            with self.subTest(term=term), self.assertRaises(audio.AudioError):
                audio.validate_script(result, SOURCES)

    def test_output_injection_and_length_are_rejected(self):
        for text in (
            " https://evil.example",
            " `code`",
            " $(cat secret)",
            " <script>",
            " @handle",
            "x" * 2501,
        ):
            result = narration()
            result["sentences"][0]["text"] += text
            with self.subTest(text=text[:30]), self.assertRaises(audio.AudioError):
                audio.validate_script(result, SOURCES)

    def test_source_edits_stop_before_paid_call(self):
        self.prepare()
        manifest = audio.read_json("manifest.json")
        changed = dict(RECORDS[0], body=RECORDS[0]["body"] + "\nNew warning")
        with (
            patch.object(audio, "github", return_value=changed),
            self.assertRaises(audio.AudioError),
        ):
            audio.recheck_sources(manifest, SOURCES)

    def test_script_invalid_output_never_produces_transcript(self):
        self.prepare()
        self.patch(
            audio, "openai_json", return_value=({"sentences": [], "cautions": []}, {})
        )
        with self.assertRaises(audio.AudioError):
            audio.script()
        self.assertFalse((self.out / "transcript.txt").exists())
        self.assertEqual(audio.read_json("manifest.json")["requests"]["text"], 1)
        with self.assertRaises(audio.AudioError):
            audio.script()

    def test_script_checkpoint_and_speech_response_validation(self):
        self.prepare()
        self.patch(
            audio, "openai_json", return_value=(narration(), {"input_tokens": 1000})
        )
        audio.script()
        self.patch(
            audio.subprocess, "run", return_value=type("Probe", (), {"returncode": 0})()
        )
        call = self.patch(
            audio, "request", return_value=(b'{"error":"bad"}', "application/json")
        )
        with self.assertRaises(audio.AudioError):
            audio.speech()
        self.assertFalse((self.out / "release-audio.mp3").exists())
        self.assertEqual(call.call_count, 1)
        self.assertEqual(
            call.call_args.args[2]["instructions"], audio.SPEECH_INSTRUCTIONS
        )

    def test_tampered_checkpoint_rejected_before_speech(self):
        self.prepare()
        self.patch(audio, "openai_json", return_value=(narration(), {}))
        audio.script()
        (self.out / "transcript.txt").write_text("tampered")
        with self.assertRaises(audio.AudioError):
            audio.speech()

    def test_mini_tts_input_limit(self):
        self.prepare()
        self.patch(audio, "openai_json", return_value=(narration(), {}))
        audio.script()
        self.patch(
            audio.subprocess, "run", return_value=type("Probe", (), {"returncode": 0})()
        )
        with patch.object(audio, "SPEECH_INSTRUCTIONS", "x" * 3000):
            manifest = audio.read_json("manifest.json")
            manifest["config"]["speech_instructions"] = audio.SPEECH_INSTRUCTIONS
            audio.write_json("manifest.json", manifest)
            with self.assertRaises(audio.AudioError):
                audio.speech()

    def valid_script(self):
        self.prepare()
        self.patch(audio, "openai_json", return_value=(narration(), {}))
        audio.script()

    def test_valid_mp3_is_saved_with_checksums_and_duration(self):
        self.valid_script()
        probe = {"format": {"duration": "120.5"}, "streams": [{"codec_name": "mp3"}]}
        self.patch(
            audio.subprocess,
            "run",
            return_value=SimpleNamespace(returncode=0, stdout=json.dumps(probe)),
        )
        self.patch(audio, "request", return_value=(b"mock-mp3-content", "audio/mpeg"))
        audio.speech()
        manifest = audio.read_json("manifest.json")
        self.assertEqual(manifest["status"], "audio_validated")
        self.assertEqual(manifest["duration_seconds"], 120.5)
        self.assertFalse(manifest["duration_needs_review"])
        self.assertEqual(
            manifest["audio_sha256"],
            audio.digest((self.out / "release-audio.mp3").read_bytes()),
        )
        self.assertFalse((self.out / "audio.tmp").exists())
        with self.assertRaises(audio.AudioError):
            audio.speech()

    def test_invalid_mp3_metadata_never_becomes_an_artifact(self):
        self.valid_script()
        for duration, codec in (("nan", "mp3"), ("0", "mp3"), ("120", "aac")):
            manifest = audio.read_json("manifest.json")
            manifest.update(
                status="script_validated", requests={"text": 1, "speech": 0}
            )
            audio.write_json("manifest.json", manifest)
            probe = {
                "format": {"duration": duration},
                "streams": [{"codec_name": codec}],
            }
            with (
                patch.object(
                    audio.subprocess,
                    "run",
                    return_value=SimpleNamespace(
                        returncode=0, stdout=json.dumps(probe)
                    ),
                ),
                patch.object(audio, "request", return_value=(b"invalid", "audio/mpeg")),
            ):
                with (
                    self.subTest(duration=duration, codec=codec),
                    self.assertRaises(audio.AudioError),
                ):
                    audio.speech()
            self.assertFalse((self.out / "release-audio.mp3").exists())
            self.assertFalse((self.out / "audio.tmp").exists())

    @unittest.skipUnless(
        shutil.which("ffmpeg") and shutil.which("ffprobe"), "ffmpeg/ffprobe unavailable"
    )
    def test_real_mp3_decode_with_mocked_openai(self):
        # Synthetic local tone, not a paid narration or an auditioned voice.
        tone = self.out / "tone.mp3"
        subprocess.run(
            [
                "ffmpeg",
                "-v",
                "error",
                "-f",
                "lavfi",
                "-i",
                "sine=frequency=440",
                "-t",
                "0.2",
                "-c:a",
                "libmp3lame",
                str(tone),
            ],
            check=True,
            capture_output=True,
            timeout=30,
        )
        self.valid_script()
        self.patch(audio, "request", return_value=(tone.read_bytes(), "audio/mpeg"))
        audio.speech()
        manifest = audio.read_json("manifest.json")
        self.assertGreater(manifest["duration_seconds"], 0)
        self.assertTrue(manifest["duration_needs_review"])
        self.assertEqual(manifest["requests"], {"text": 1, "speech": 1})

    def test_mp3_decode_failure_cleans_temporary_file(self):
        self.valid_script()

        def run(args, **kwargs):
            if args[0] == "ffmpeg" and "-xerror" in args:
                raise subprocess.CalledProcessError(1, args, stderr=b"untrusted-data")
            probe = {"format": {"duration": "120"}, "streams": [{"codec_name": "mp3"}]}
            return SimpleNamespace(returncode=0, stdout=json.dumps(probe))

        self.patch(audio.subprocess, "run", side_effect=run)
        self.patch(audio, "request", return_value=(b"broken-mp3", "audio/mpeg"))
        with self.assertRaises(subprocess.CalledProcessError):
            audio.speech()
        self.assertFalse((self.out / "release-audio.mp3").exists())
        self.assertFalse((self.out / "audio.tmp").exists())

    def test_config_change_after_prepare_is_rejected(self):
        self.prepare()
        with (
            patch.dict(os.environ, {"AUDIO_VOICE": "cedar"}),
            self.assertRaises(audio.AudioError),
        ):
            audio.script()

    def test_legacy_tts_does_not_receive_instructions(self):
        with patch.dict(os.environ, {"AUDIO_TTS_MODEL": "tts-1"}):
            self.valid_script()
            self.patch(
                audio.subprocess, "run", return_value=SimpleNamespace(returncode=0)
            )
            call = self.patch(
                audio, "request", return_value=(b"error", "application/json")
            )
            with self.assertRaises(audio.AudioError):
                audio.speech()
            self.assertNotIn("instructions", call.call_args.args[2])

    def test_responses_refusal_and_incomplete_are_rejected(self):
        for response in (
            {"status": "incomplete"},
            {
                "status": "completed",
                "output": [{"type": "message", "content": [{"type": "refusal"}]}],
            },
        ):
            with (
                patch.object(
                    audio,
                    "request",
                    return_value=(json.dumps(response).encode(), "application/json"),
                ),
                self.assertRaises(audio.AudioError),
            ):
                audio.openai_json({})

    def test_complete_structured_response_is_parsed(self):
        expected = narration()
        response = {
            "status": "completed",
            "usage": {"input_tokens": 500},
            "output": [
                {"type": "reasoning"},
                {
                    "type": "message",
                    "content": [{"type": "output_text", "text": json.dumps(expected)}],
                },
            ],
        }
        with patch.object(
            audio,
            "request",
            return_value=(json.dumps(response).encode(), "application/json"),
        ):
            result, usage = audio.openai_json({})
        self.assertEqual(result, expected)
        self.assertEqual(usage, {"input_tokens": 500})

    def test_response_size_limit_and_credentials_not_in_payload(self):
        response = SimpleNamespace(
            headers=SimpleNamespace(get_content_type=lambda: "audio/mpeg")
        )
        response.read = lambda limit: b"x" * limit
        context = unittest.mock.MagicMock()
        context.__enter__.return_value = response
        opener = unittest.mock.MagicMock()
        opener.open.return_value = context
        with (
            patch.object(audio.urllib.request, "build_opener", return_value=opener),
            self.assertRaises(audio.AudioError),
        ):
            audio.request(
                "https://api.openai.com/v1/audio/speech",
                "test-secret",
                {"input": "public text"},
                limit=100,
            )
        req = opener.open.call_args.args[0]
        self.assertNotIn(b"test-secret", req.data)
        self.assertEqual(req.get_header("Authorization"), "Bearer test-secret")

    def test_http_errors_and_timeouts_are_not_retried_or_leaked(self):
        for error in (
            urllib.error.HTTPError("url", 429, "secret-body", {}, None),
            TimeoutError("secret-body"),
        ):
            opener = type("Opener", (), {})()
            with (
                patch.object(audio.urllib.request, "build_opener", return_value=opener),
                patch.object(opener, "open", create=True, side_effect=error) as call,
            ):
                with self.assertRaises(audio.AudioError) as caught:
                    audio.request("https://api.openai.com/v1/responses", "secret")
                self.assertNotIn("secret", str(caught.exception))
                self.assertEqual(call.call_count, 1)

    def test_redirects_are_disabled(self):
        self.assertIsNone(audio.NoRedirect().redirect_request(None))


if __name__ == "__main__":
    unittest.main()
