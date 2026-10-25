"""Safety and failure-path tests. All cloud and GitHub operations are faked."""
import copy
import datetime as dt
import io
import json
import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch
import urllib.error

import release as r

OLD = "sha256:" + "a" * 64
NEW = "sha256:" + "b" * 64
SHA = "c" * 40
BUILD = "12345678-abcd-1234-abcd-123456789abc"
SECRET = "synthetic-secret-must-not-appear"


def job(digest=OLD, generation="1"):
    return {"name": r.JOB, "etag": "version-" + generation, "generation": generation,
            "observedGeneration": generation, "terminalCondition": {"state": "CONDITION_SUCCEEDED"},
            "template": {"taskCount": 1, "template": {"serviceAccount": "runtime@example.com",
                "maxRetries": 0, "timeout": "180s", "containers": [{"image": r.IMAGE + "@" + digest,
                "env": [{"name": "TOKEN", "value": SECRET}], "resources": {"limits": {"memory": "512Mi"}}}]}}}


def execution(image=NEW, creator=r.SCHEDULER, success=True):
    return {"name": r.JOB + "/executions/example-1", "job": r.JOB, "creator": creator,
            "createTime": "2026-10-25T10:05:00Z", "completionTime": "2026-10-25T10:05:40Z",
            "template": {"containers": [{"image": r.IMAGE + "@" + image}]},
            "conditions": [{"type": "Completed", "state": "CONDITION_SUCCEEDED" if success else "CONDITION_FAILED"}],
            "failedCount": 0 if success else 1}


class FakeClock:
    value = 0

    def clock(self):
        return self.value

    def sleep(self, seconds):
        self.value += seconds


class FakeCloud:
    def __init__(self):
        self.current = job()
        self.patches = []
        self.execs = [execution()]
        self.counts = {"published": 0, "attempted": 0, "failed": 0}
        self.errors = 0
        self.exists = []

    def job(self):
        return copy.deepcopy(self.current)

    def image_exists(self, digest):
        self.exists.append(digest)

    def patch(self, body):
        self.patches.append(copy.deepcopy(body))
        self.current = job(NEW, "2")
        return {"name": r.PARENT + "/operations/example"}

    def executions(self, since):
        return copy.deepcopy(self.execs)

    def logs(self, name):
        return self.counts, self.errors


class ControllerTests(unittest.TestCase):
    def setUp(self):
        self.cloud = FakeCloud()
        self.record = {"run_id": "123", "run_attempt": "1"}
        self.saved = []
        self.clock = FakeClock()
        self.controller = r.Release(self.cloud, self.record, lambda data: self.saved.append(copy.deepcopy(data)),
                                    self.clock.clock, self.clock.sleep)

    def observe(self):
        self.cloud.current = job(NEW, "2")
        self.record["installed_at"] = "2026-10-25T10:00:00Z"
        return self.controller.observe(job(NEW, "2"), NEW)

    def test_wrong_expected_digest_never_updates_and_saves_old_image(self):
        with self.assertRaisesRegex(r.ReleaseError, "unexpected_current_digest"):
            self.controller.snapshot(NEW)
        self.assertEqual(self.record["previous_digest"], OLD)
        self.assertEqual(self.cloud.patches, [])
        self.assertNotIn(SECRET, json.dumps(self.saved))

    def test_only_image_changes_with_etag_precondition(self):
        before = self.controller.snapshot(OLD)
        installed = self.controller.install(before, NEW)
        self.assertTrue(self.record["image_installed"])
        self.assertEqual(installed["generation"], "2")
        body = self.cloud.patches[0]
        self.assertEqual(body["etag"], before["etag"])
        body["template"]["template"]["containers"][0]["image"] = r.IMAGE + "@" + OLD
        del body["etag"]
        self.assertEqual(body, r.writable_job(before))
        self.assertNotIn(SECRET, json.dumps(self.saved))

    def test_operator_change_during_build_aborts_without_patch(self):
        before = self.controller.snapshot(OLD)
        self.cloud.current["etag"] = "operator-change"
        with self.assertRaisesRegex(r.ReleaseError, "job_changed_before_update"):
            self.controller.install(before, NEW)
        self.assertEqual(self.cloud.patches, [])

    def test_ambiguous_patch_is_not_retried_or_rolled_back(self):
        before = self.controller.snapshot(OLD)
        with patch.object(self.cloud, "patch", side_effect=r.ReleaseError("api_transport_or_json_error")) as method:
            with self.assertRaises(r.ReleaseError):
                self.controller.install(before, NEW)
        self.assertEqual(method.call_count, 1)
        self.assertEqual(self.record["update_outcome"], "pending")
        self.assertEqual(self.record["new_digest"], NEW)

    def test_configuration_drift_after_patch_fails(self):
        before = self.controller.snapshot(OLD)
        def mutate(body):
            self.cloud.current = job(NEW, "2")
            self.cloud.current["template"]["template"]["maxRetries"] = 3
            return {}
        self.cloud.patch = mutate
        with self.assertRaisesRegex(r.ReleaseError, "configuration_changed_after_update"):
            self.controller.install(before, NEW)

    def test_zero_attempts_is_completed_without_publication(self):
        self.observe()
        self.assertTrue(self.record["execution_completed"])
        self.assertIs(self.record["article_published"], False)
        self.assertEqual(self.record["verification"], "completed_without_publication")

    def test_nonzero_publication_is_reported(self):
        self.cloud.counts = {"published": 1, "attempted": 1, "failed": 0}
        self.observe()
        self.assertTrue(self.record["article_published"])
        self.assertEqual(self.record["verification"], "publication_reported")

    def test_failed_execution_retains_counters(self):
        self.cloud.execs = [execution(success=False)]
        self.cloud.counts = {"published": 0, "attempted": 1, "failed": 1}
        with self.assertRaisesRegex(r.ReleaseError, "scheduled_execution_failed"):
            self.observe()
        self.assertTrue(self.record["execution_completed"])
        self.assertFalse(self.record["execution_succeeded"])
        self.assertEqual(self.record["counters"]["failed"], 1)

    def test_partial_failure_or_error_logs_fail_verification(self):
        for counts, errors in [({"published": 1, "attempted": 2, "failed": 1}, 0),
                               ({"published": 1, "attempted": 1, "failed": 0}, 1)]:
            with self.subTest(counts=counts, errors=errors):
                self.cloud.counts, self.cloud.errors = counts, errors
                with self.assertRaisesRegex(r.ReleaseError, "execution_errors_detected"):
                    self.observe()

    def test_manual_execution_does_not_satisfy_schedule_check(self):
        self.cloud.execs = [execution(creator="operator@example.com")]
        with self.assertRaisesRegex(r.ReleaseError, "scheduled_execution_or_logs_timeout"):
            self.observe()
        self.assertEqual(self.clock.value, 900)
        self.assertNotIn("execution_id", self.record)

    def test_wrong_image_or_intervening_update_is_rejected(self):
        self.cloud.execs = [execution(image=OLD)]
        with self.assertRaisesRegex(r.ReleaseError, "scheduled_execution_image_mismatch"):
            self.observe()
        self.cloud.current = job(NEW, "3")
        with self.assertRaisesRegex(r.ReleaseError, "job_changed_during_observation"):
            self.controller.observe(job(NEW, "2"), NEW)

    def test_missing_logs_times_out_without_claiming_delivery(self):
        self.cloud.counts = None
        with self.assertRaisesRegex(r.ReleaseError, "scheduled_execution_or_logs_timeout"):
            self.observe()
        self.assertTrue(self.record["execution_completed"])
        self.assertIsNone(self.record["article_published"])

    def test_successful_build_verifies_registry_and_retains_provenance(self):
        def call(host, path, method="GET", body=None):
            if method == "POST":
                self.submitted = body
                return {"metadata": {"build": {"id": BUILD}}}
            return {"status": "SUCCESS", "source": self.submitted["source"],
                    "sourceProvenance": {"resolvedGitSource": self.submitted["source"]["gitSource"]},
                    "results": {"images": [{"name": self.submitted["images"][0], "digest": NEW}]}}
        self.cloud.call = call
        target = self.controller.build(SHA, "register-news-builder@register-news.iam.gserviceaccount.com")
        self.assertEqual(target, NEW)
        self.assertTrue(self.record["source_verified"])
        self.assertIn(NEW, self.cloud.exists)

    def test_build_wait_has_a_deadline(self):
        self.cloud.call = lambda host,path,method="GET",body=None: (
            {"metadata":{"build":{"id":BUILD}}} if method == "POST" else {"status":"WORKING"})
        with self.assertRaisesRegex(r.ReleaseError, "build_wait_timeout"):
            self.controller.build(SHA, "register-news-builder@register-news.iam.gserviceaccount.com")
        self.assertLessEqual(self.clock.value, 1110)

    def test_failed_build_retains_id_without_any_patch(self):
        def call(host, path, method="GET", body=None):
            return {"metadata": {"build": {"id": BUILD}}} if method == "POST" else {"status": "FAILURE"}
        self.cloud.call = call
        with self.assertRaisesRegex(r.ReleaseError, "build_failed"):
            self.controller.build(SHA, "register-news-builder@register-news.iam.gserviceaccount.com")
        self.assertEqual(self.record["build_id"], BUILD)
        self.assertEqual(self.cloud.patches, [])

    def test_full_rollback_uses_recorded_source_without_building(self):
        original_sha = "d" * 40
        image_sha = "e" * 40
        run = {"id":42,"head_branch":"main","event":"workflow_dispatch","path":".github/workflows/deploy.yml",
               "status":"completed","head_repository":{"id":r.REPO_ID},"head_sha":original_sha,"run_attempt":1}
        original = {"schema_version":1,"repository":r.REPO,"run_id":"42","run_attempt":"1","mode":"deploy",
                    "controller_sha":original_sha,"source_sha":image_sha,"previous_digest":OLD,
                    "new_digest":NEW,"image_installed":True,"build_id":BUILD}
        with tempfile.TemporaryDirectory() as directory:
            reports = Path(directory)/"reports"
            reports.mkdir()
            source = Path(directory)/"original"
            source.mkdir()
            (source/"release.json").write_text(json.dumps(original))
            current = {"controller_sha":SHA,"run_id":"123","run_attempt":"1","mode":"rollback","checks_passed":True}
            (reports/"release.json").write_text(json.dumps(current))
            with patch.dict(os.environ, {"REPORT_DIR":str(reports),"ROLLBACK_RECORD_DIR":str(source),
                    "GITHUB_REPOSITORY":r.REPO,"GITHUB_REF":"refs/heads/main","GITHUB_EVENT_NAME":"workflow_dispatch",
                    "EXPECTED_CURRENT_DIGEST":OLD,"RELEASE_RUN_ID":"42","RELEASE_RUN_ATTEMPT":"1","TARGET_DIGEST":NEW}), \
                 patch.object(r,"Cloud",return_value=self.cloud), patch.object(r,"github_run",return_value=run), \
                 patch.object(r,"now",return_value="2026-10-25T10:00:00Z"), \
                 patch.object(r.sys,"argv",["release.py","rollback"]):
                r.main()
            result=json.loads((reports/"release.json").read_text())
            self.assertEqual(result["result"],"success")
            self.assertEqual(result["source_sha"],image_sha)
            self.assertEqual(result["origin_build_id"],BUILD)
            self.assertNotIn("build_id",result)
            self.assertEqual(len(self.cloud.patches),1)
            self.assertNotIn(SECRET,json.dumps(result))


class ValidationTests(unittest.TestCase):
    def test_digest_rejects_tags_other_apps_injection_and_malformed_values(self):
        for value in ["latest", r.IMAGE + "@" + OLD, OLD.upper(), OLD + "\n", "$(command)", None]:
            with self.subTest(value=value), self.assertRaises(r.ReleaseError):
                r.valid_digest(value)

    def test_job_rejects_tag_unknown_fields_multiple_containers_and_test_mode(self):
        for change in [lambda j:j.update(unknownField=True),
                       lambda j:j["template"]["template"]["containers"][0].update(image=r.IMAGE + ":latest"),
                       lambda j:j["template"]["template"]["containers"].append({"image": "other"}),
                       lambda j:j["template"]["template"]["containers"][0]["env"].append({"name":"TEST_MODE", "value":"true"}),
                       lambda j:j["template"]["template"]["containers"][0]["env"].append({"name":"TEST_MODE", "valueSource":{"secretKeyRef":{}}})]:
            data = job()
            change(data)
            with self.assertRaises(r.ReleaseError):
                r.writable_job(data)
                r.job_digest(data)

    def test_build_fetches_remote_sha_without_local_files_or_credentials(self):
        with patch.dict(os.environ, {"GCP_ACCESS_TOKEN": SECRET, "GOOGLE_APPLICATION_CREDENTIALS": "/tmp/auth.json"}):
            data = r.build_request(SHA, "register-news-builder@register-news.iam.gserviceaccount.com", r.IMAGE+":test")
        self.assertEqual(data["source"], {"gitSource": {"url":"https://github.com/"+r.REPO+".git", "revision":SHA}})
        self.assertNotIn(SECRET, json.dumps(data))
        self.assertNotIn("/tmp/auth.json", json.dumps(data))
        self.assertEqual(data["options"]["logging"], "CLOUD_LOGGING_ONLY")

    def test_build_provenance_and_image_must_match(self):
        source = {"revision": SHA, "url":"https://github.com/"+r.REPO+".git"}
        build = {"status":"SUCCESS", "source":{"gitSource":source},
                 "sourceProvenance":{"resolvedGitSource":copy.deepcopy(source)},
                 "results":{"images":[{"name":"tag", "digest":NEW}]}}
        self.assertEqual(r.verified_build(build, SHA, "tag"), NEW)
        for mutate in [lambda b:b["sourceProvenance"]["resolvedGitSource"].update(revision="d"*40),
                       lambda b:b["results"]["images"][0].update(name="other"),
                       lambda b:b.update(status="FAILURE")]:
            changed = copy.deepcopy(build)
            mutate(changed)
            with self.assertRaises(r.ReleaseError):
                r.verified_build(changed, SHA, "tag")

    def test_rollback_requires_trusted_run_and_recorded_image(self):
        run = {"id":123, "head_branch":"main", "event":"workflow_dispatch", "path":".github/workflows/deploy.yml",
               "status":"completed", "head_repository":{"id":r.REPO_ID}, "head_sha":SHA,"run_attempt":1}
        record = {"schema_version":1,"repository":r.REPO,"run_id":"123","run_attempt":"1", "mode":"deploy",
                  "controller_sha":SHA,"source_sha":SHA,"previous_digest":OLD,"new_digest":NEW,"image_installed":True}
        self.assertEqual(r.validate_record(record, run, "123", "1", NEW), SHA)
        self.assertIsNone(r.validate_record(record, run, "123", "1", OLD))
        for field, value in [("event","push"),("head_branch","feature"),("path",".github/workflows/test.yml"),
                             ("head_repository",{"id":1}),("status","in_progress")]:
            with self.subTest(field=field), self.assertRaises(r.ReleaseError):
                r.validate_record(record, {**run,field:value}, "123", "1", NEW)
        for field,value in [("run_attempt","2"),("controller_sha","e"*40),("image_installed",False)]:
            with self.subTest(field=field), self.assertRaises(r.ReleaseError):
                r.validate_record({**record,field:value}, run, "123", "1", NEW)
        with self.assertRaises(r.ReleaseError):
            r.validate_record(record, run, "123", "1", "sha256:"+"f"*64)

    def test_logs_return_only_counters_and_error_count(self):
        entries = [{"textPayload":"2026/10/25 10:00:00 Published 0 new articles (attempted 0, failed 0)"},
                   {"severity":"ERROR", "textPayload":SECRET}, {"textPayload":"Gemini HTTP 429: " + SECRET}]
        counts, errors = r.log_counters(entries)
        self.assertEqual(counts, {"published":0,"attempted":0,"failed":0})
        self.assertEqual(errors, 2)
        self.assertNotIn(SECRET, json.dumps((counts, errors)))
        with self.assertRaises(r.ReleaseError):
            r.log_counters([entries[0], entries[0]])
        self.assertEqual(r.log_counters([{**entries[0], "severity":"ERROR"}])[1], 1)

    def test_api_errors_never_echo_body_or_token(self):
        error = urllib.error.HTTPError("https://example", 409, SECRET, {}, io.BytesIO(SECRET.encode()))
        with patch("urllib.request.urlopen", side_effect=error) as call:
            with self.assertRaisesRegex(r.ReleaseError, "^api_http_409$"):
                r.request_json("run.googleapis.com", "v2/"+r.JOB, SECRET, "PATCH", {})
        self.assertEqual(call.call_count, 1)

    def test_execution_pagination_stops_at_observation_boundary(self):
        cloud = object.__new__(r.Cloud)
        recent = execution()
        old = {**recent,"createTime":"2026-10-24T00:00:00Z"}
        cloud.call = lambda *args: {"executions":[recent,old],"nextPageToken":"older-history"}
        self.assertEqual(len(cloud.executions("2026-10-25T10:00:00Z")), 2)

    def test_main_stops_before_cloud_access_if_checks_failed_or_wrong_ref(self):
        for ref, passed in [("refs/heads/main",False),("refs/heads/other",True)]:
            with tempfile.TemporaryDirectory() as directory, patch.dict(os.environ, {
                    "REPORT_DIR":directory,"GITHUB_REPOSITORY":r.REPO,"GITHUB_REF":ref,"GITHUB_EVENT_NAME":"workflow_dispatch"}):
                data = {"controller_sha":SHA,"run_id":"123","run_attempt":"1","mode":"deploy","checks_passed":passed}
                file=Path(directory)/"release.json"
                file.write_text(json.dumps(data))
                with patch.object(r, "Cloud") as cloud, patch.object(r.sys,"argv",["release.py","deploy"]), patch("sys.stderr",io.StringIO()):
                    with self.assertRaises(SystemExit):
                        r.main()
                    cloud.assert_not_called()
                self.assertEqual(json.loads(file.read_text())["result"], "failure")


class WorkflowEvidenceTests(unittest.TestCase):
    def test_workflow_bootstrap_and_failure_finalizer_produce_valid_reports(self):
        root = Path(__file__).resolve().parents[1] / "workflows"
        for mode in ("deploy", "rollback"):
            text = (root / (mode+".yml")).read_text()
            blocks = []
            lines = text.splitlines()
            for i,line in enumerate(lines):
                if line == "          python3 - <<'PY'":
                    end = lines.index("          PY", i+1)
                    blocks.append("\n".join(x[10:] for x in lines[i+1:end]))
            self.assertEqual(len(blocks), 2)
            with tempfile.TemporaryDirectory() as directory, patch.dict(os.environ, {
                "REPORT_DIR":directory,"RELEASE_MODE":mode,"GITHUB_REPOSITORY":r.REPO,"GITHUB_SHA":SHA,
                "GITHUB_RUN_ID":"123","GITHUB_RUN_ATTEMPT":"1","WORKFLOW_RESULT":"failure",
                "GITHUB_STEP_SUMMARY":directory+"/summary.md","GITHUB_ENV":directory+"/github-env"}):
                for code in blocks:
                    exec(compile(code, mode+".yml", "exec"), {})
                data=json.loads((Path(directory)/"release.json").read_text())
                self.assertEqual(data["result"], "failure")
                self.assertEqual(data["controller_sha"], SHA)
                self.assertIn("Zero attempts", (Path(directory)/"summary.md").read_text())


if __name__ == "__main__":
    unittest.main()
