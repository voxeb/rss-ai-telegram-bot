#!/usr/bin/env python3
"""Manual release controller. No dependencies, raw logs, or local source uploads.

Cloud Run v2 PATCH accepts a Job etag (there is no updateMask parameter).
Keep the complete writable configuration in memory, change one image, and fail
on conflicts. Never retry a mutation whose outcome could be ambiguous.
"""
import copy
import datetime as dt
import json
import os
from pathlib import Path
import re
import sys
import time
import urllib.error
import urllib.parse
import urllib.request

REPO = "voxeb/rss-ai-telegram-bot"
REPO_ID = 1432283866
PROJECT = "register-news"
REGION = "europe-west1"
PARENT = f"projects/{PROJECT}/locations/{REGION}"
JOB = f"{PARENT}/jobs/register-news"
IMAGE = f"{REGION}-docker.pkg.dev/{PROJECT}/cloud-run-source-deploy/register-news"
SCHEDULER = f"register-scheduler@{PROJECT}.iam.gserviceaccount.com"
WRITABLE = {"name", "labels", "annotations", "client", "clientVersion",
            "launchStage", "binaryAuthorization", "template"}
OUTPUT = {"uid", "generation", "createTime", "updateTime", "deleteTime",
          "expireTime", "creator", "lastModifier", "observedGeneration",
          "terminalCondition", "conditions", "executionCount",
          "latestCreatedExecution", "reconciling", "satisfiesPzs", "etag"}


class ReleaseError(Exception):
    """Only fixed error codes reach reports; never provider messages."""


def require(condition, code):
    if not condition:
        raise ReleaseError(code)


def valid_digest(value):
    require(isinstance(value, str) and re.fullmatch(r"sha256:[a-f0-9]{64}", value),
            "invalid_digest")
    return value


def valid_sha(value):
    require(isinstance(value, str) and re.fullmatch(r"[a-f0-9]{40}", value), "invalid_source_sha")
    return value


def positive_id(value):
    require(isinstance(value, str) and re.fullmatch(r"[1-9][0-9]{0,19}", value), "invalid_run_id")
    return value


def valid_build_id(value):
    require(isinstance(value, str) and re.fullmatch(
        r"[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}", value), "invalid_build_id")
    return value


def timestamp(value):
    return dt.datetime.fromisoformat(value.replace("Z", "+00:00"))


def now():
    return dt.datetime.now(dt.timezone.utc).isoformat()


def request_json(host, path, token, method="GET", body=None):
    require(host in {"run.googleapis.com", "cloudbuild.googleapis.com",
                     "artifactregistry.googleapis.com", "logging.googleapis.com",
                     "api.github.com"}, "untrusted_api_host")
    data = None if body is None else json.dumps(body).encode()
    request = urllib.request.Request(f"https://{host}/{path}", data=data, method=method,
                                     headers={"Authorization": f"Bearer {token}",
                                              "Content-Type": "application/json"})
    try:
        with urllib.request.urlopen(request, timeout=30) as response:
            return json.load(response)
    except urllib.error.HTTPError as error:
        # Discard the response body, which can echo credentials or user data.
        raise ReleaseError(f"api_http_{error.code}") from None
    except (OSError, ValueError):
        raise ReleaseError("api_transport_or_json_error") from None


class Cloud:
    def __init__(self):
        self.token = os.environ["GCP_ACCESS_TOKEN"]

    def call(self, host, path, method="GET", body=None):
        return request_json(host, path, self.token, method, body)

    def job(self):
        return self.call("run.googleapis.com", f"v2/{JOB}")

    def image_exists(self, digest):
        valid_digest(digest)
        resource = urllib.parse.quote("register-news@" + digest, safe="")
        result = self.call("artifactregistry.googleapis.com",
                           f"v1/{PARENT}/repositories/cloud-run-source-deploy/dockerImages/{resource}")
        require(result.get("uri") == IMAGE + "@" + digest, "registry_image_mismatch")

    def patch(self, body):
        return self.call("run.googleapis.com", f"v2/{JOB}", "PATCH", body)

    def executions(self, since):
        result, page = [], ""
        # Lists are newest-first. Bound both pagination and overall waiting.
        for _ in range(10):
            query = urllib.parse.urlencode({"pageSize": 100, "pageToken": page})
            data = self.call("run.googleapis.com", f"v2/{JOB}/executions?{query}")
            result.extend(data.get("executions", []))
            if result and timestamp(result[-1]["createTime"]) < timestamp(since):
                return result
            page = data.get("nextPageToken", "")
            if not page:
                return result
        raise ReleaseError("execution_list_limit")

    def logs(self, execution):
        query = (f'resource.type="cloud_run_job" AND resource.labels.job_name="register-news" '
                 f'AND resource.labels.location="{REGION}" AND '
                 f'labels."run.googleapis.com/execution_name"="{execution}"')
        data = self.call("logging.googleapis.com", "v2/entries:list", "POST",
                         {"resourceNames": [f"projects/{PROJECT}"], "filter": query,
                          "orderBy": "timestamp asc", "pageSize": 1000})
        require(not data.get("nextPageToken"), "execution_log_limit")
        return log_counters(data.get("entries", []))


def writable_job(job):
    require(not set(job) - WRITABLE - OUTPUT, "unsupported_job_fields")
    require(job.get("name") == JOB and job.get("etag"), "invalid_job_identity_or_etag")
    return {key: copy.deepcopy(value) for key, value in job.items() if key in WRITABLE}


def job_digest(job):
    containers = job["template"]["template"]["containers"]
    require(len(containers) == 1, "expected_one_container")
    image = containers[0]["image"]
    require(image.startswith(IMAGE + "@"), "current_image_not_pinned_to_application")
    for variable in containers[0].get("env", []):
        require(variable.get("name") != "TEST_MODE" or
                not variable.get("valueSource") and variable.get("value", "").lower() != "true",
                "test_mode_enabled")
    return valid_digest(image[len(IMAGE) + 1:])


def config_without_image(job):
    config = writable_job(job)
    config["template"]["template"]["containers"][0].pop("image")
    return config


def update_body(before, latest, target):
    require(latest.get("etag") == before["etag"], "job_changed_before_update")
    require(writable_job(latest) == writable_job(before), "job_configuration_changed")
    body = writable_job(before)
    body["template"]["template"]["containers"][0]["image"] = IMAGE + "@" + valid_digest(target)
    body["etag"] = before["etag"]  # Server-side precondition closes the GET/PATCH race.
    return body


def build_request(sha, account, tag):
    valid_sha(sha)
    require(account == f"register-news-builder@{PROJECT}.iam.gserviceaccount.com", "invalid_build_account")
    return {"source": {"gitSource": {"url": f"https://github.com/{REPO}.git", "revision": sha}},
            "steps": [{"name": "gcr.io/cloud-builders/docker",
                       "args": ["build", "--no-cache", "-t", tag, "."]}],
            "images": [tag], "timeout": "900s",
            "serviceAccount": f"projects/{PROJECT}/serviceAccounts/{account}",
            "options": {"logging": "CLOUD_LOGGING_ONLY"}}


def verified_build(build, sha, tag):
    require(build.get("status") == "SUCCESS", "build_not_successful")
    for source in (build.get("source", {}).get("gitSource", {}),
                   build.get("sourceProvenance", {}).get("resolvedGitSource", {})):
        require(source.get("revision") == sha and source.get("url") == f"https://github.com/{REPO}.git",
                "build_source_mismatch")
    images = build.get("results", {}).get("images", [])
    require(len(images) == 1 and images[0].get("name") == tag, "build_image_mismatch")
    return valid_digest(images[0].get("digest"))


def validate_record(record, run, run_id, attempt, target):
    expected_paths = {f".github/workflows/{mode}.yml": mode for mode in ("deploy", "rollback")}
    path = run.get("path", "").split("@")[0]
    require(run.get("id") == int(run_id) and run.get("head_branch") == "main"
            and run.get("event") == "workflow_dispatch" and path in expected_paths
            and run.get("status") == "completed" and run.get("head_repository", {}).get("id") == REPO_ID,
            "untrusted_release_run")
    require(record.get("schema_version") == 1 and record.get("repository") == REPO
            and record.get("run_id") == run_id and record.get("run_attempt") == attempt
            and int(attempt) <= run.get("run_attempt", 0)
            and record.get("controller_sha") == run.get("head_sha")
            and record.get("mode") == expected_paths[path], "release_record_mismatch")
    require(record.get("previous_digest") == target or
            record.get("new_digest") == target and record.get("image_installed") is True,
            "image_not_recorded_as_installed")
    source = record.get("source_sha") if record.get("new_digest") == target else None
    return valid_sha(source) if source else None


def log_counters(entries):
    summaries, errors = [], 0
    for entry in entries:
        text = entry.get("textPayload", "")
        summary = re.fullmatch(r"(?:\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2} )?"
                               r"Published (\d+) new articles \(attempted (\d+), failed (\d+)\)", text)
        if summary:
            published, attempted, failed = map(int, summary.groups())
            require(0 <= published <= attempted <= 5 and 0 <= failed <= attempted
                    and published + failed <= attempted, "invalid_batch_counters")
            summaries.append({"published": published, "attempted": attempted, "failed": failed})
        if entry.get("severity") in {"ERROR", "CRITICAL", "ALERT", "EMERGENCY"} or not summary and (re.search(
                r"(?i)(error|fail|panic|fatal|quota|429|402)", text) or entry.get("jsonPayload")):
            # Unknown structured payloads are not silently assumed to be safe.
            errors += 1
    require(len(summaries) <= 1, "ambiguous_batch_summaries")
    return (summaries[0] if summaries else None), errors


class Release:
    def __init__(self, cloud, record, save, clock=time.monotonic, sleep=time.sleep):
        self.cloud, self.record, self.save = cloud, record, save
        self.clock, self.sleep = clock, sleep

    def note(self, **fields):
        self.record.update(fields)
        self.save(self.record)

    def snapshot(self, expected):
        before = self.cloud.job()
        current = job_digest(before)
        self.note(previous_digest=current, rollback_image=IMAGE + "@" + current)
        require(current == valid_digest(expected), "unexpected_current_digest")
        require(not before.get("reconciling") and
                before.get("terminalCondition", {}).get("state") == "CONDITION_SUCCEEDED",
                "job_not_ready")
        writable_job(before)
        self.cloud.image_exists(current)
        return before

    def build(self, sha, account):
        tag = IMAGE + f":gha-{self.record['run_id']}-{self.record['run_attempt']}-{sha[:12]}"
        operation = self.cloud.call("cloudbuild.googleapis.com", f"v1/{PARENT}/builds", "POST",
                                    build_request(sha, account, tag))
        build_id = operation.get("metadata", {}).get("build", {}).get("id", "")
        valid_build_id(build_id)
        self.note(build_id=build_id)
        deadline = self.clock() + 1100
        while self.clock() < deadline:
            build = self.cloud.call("cloudbuild.googleapis.com", f"v1/{PARENT}/builds/{build_id}")
            status = build.get("status", "STATUS_UNKNOWN")
            require(status in {"STATUS_UNKNOWN", "PENDING", "QUEUED", "WORKING", "SUCCESS", "FAILURE",
                               "INTERNAL_ERROR", "TIMEOUT", "CANCELLED", "EXPIRED"}, "unknown_build_status")
            self.note(build_status=status)
            if build.get("status") == "SUCCESS":
                digest = verified_build(build, sha, tag)
                self.cloud.image_exists(digest)
                self.note(new_digest=digest, source_verified=True)
                return digest
            require(build.get("status") in {"STATUS_UNKNOWN", "PENDING", "QUEUED", "WORKING"}, "build_failed")
            self.sleep(15)
        raise ReleaseError("build_wait_timeout")

    def install(self, before, target):
        require(target != self.record["previous_digest"], "image_already_installed")
        self.cloud.image_exists(target)
        body = update_body(before, self.cloud.job(), target)
        self.note(update_requested_at=now(), new_digest=target, update_outcome="pending")
        self.note(rollback_inputs={"expected_current_digest": target, "image_digest": self.record["previous_digest"],
                                  "release_run_id": self.record["run_id"],
                                  "release_run_attempt": self.record["run_attempt"]})
        # On transport failure the update may have succeeded. Report uncertainty;
        # do not send another PATCH or perform an automatic rollback.
        operation = self.cloud.patch(body)
        self.note(update_outcome="accepted", update_operation=operation.get("name"))
        deadline = self.clock() + 180
        while self.clock() < deadline:
            job = self.cloud.job()
            require(job_digest(job) == target, "image_changed_after_update")
            require(config_without_image(before) == config_without_image(job), "configuration_changed_after_update")
            if not job.get("reconciling"):
                require(job.get("terminalCondition", {}).get("state") == "CONDITION_SUCCEEDED",
                        "job_update_failed")
                require(job.get("generation") == job.get("observedGeneration"), "job_generation_unobserved")
                self.note(image_installed=True, installed_at=now(), installed_generation=job["generation"],
                          update_outcome="verified", verification="image_installed")
                return job
            self.sleep(5)
        raise ReleaseError("image_install_timeout")

    def observe(self, installed, target):
        deadline = self.clock() + 900
        execution_id = None
        while self.clock() < deadline:
            current = self.cloud.job()
            require(job_digest(current) == target and current.get("generation") == installed["generation"],
                    "job_changed_during_observation")
            candidates = [e for e in self.cloud.executions(self.record["installed_at"])
                          if e.get("creator") == SCHEDULER
                          and timestamp(e["createTime"]) >= timestamp(self.record["installed_at"])]
            candidates.sort(key=lambda e: timestamp(e["createTime"]))
            if candidates:
                execution = next((e for e in candidates if e["name"] == execution_id), candidates[0])
                execution_id = execution["name"]
                require(execution.get("job") == JOB and
                        execution["template"]["containers"][0]["image"] == IMAGE + "@" + target,
                        "scheduled_execution_image_mismatch")
                self.note(execution_id=execution_id.rsplit("/", 1)[-1], execution_creator=SCHEDULER)
                if execution.get("completionTime"):
                    completed = any(c.get("type") == "Completed" and c.get("state") == "CONDITION_SUCCEEDED"
                                    for c in execution.get("conditions", []))
                    self.note(execution_completed=True, execution_succeeded=completed,
                              execution_completion_time=execution["completionTime"],
                              succeeded_tasks=execution.get("succeededCount", 0),
                              failed_tasks=execution.get("failedCount", 0),
                              verification="execution_completed")
                    counts, errors = self.cloud.logs(self.record["execution_id"])
                    self.note(counters=counts, error_entries=errors,
                              article_published=None if counts is None else counts["published"] > 0)
                    require(completed and not execution.get("failedCount") and not execution.get("cancelledCount"),
                            "scheduled_execution_failed")
                    require(errors == 0 and not (counts and counts["failed"]), "execution_errors_detected")
                    if counts:
                        self.note(verification="publication_reported" if counts["published"] else "completed_without_publication")
                        return
            self.sleep(15)
        raise ReleaseError("scheduled_execution_or_logs_timeout")


def save_report(record):
    path = Path(os.environ["REPORT_DIR"]) / "release.json"
    temporary = path.with_suffix(".tmp")
    temporary.write_text(json.dumps(record, indent=2) + "\n")
    temporary.replace(path)


def github_run(run_id):
    positive_id(run_id)
    return request_json("api.github.com", f"repos/{REPO}/actions/runs/{run_id}", os.environ["GH_TOKEN"])


def main():
    path = Path(os.environ["REPORT_DIR"]) / "release.json"
    record = json.loads(path.read_text())
    try:
        require(os.environ.get("GITHUB_REPOSITORY") == REPO and os.environ.get("GITHUB_REF") == "refs/heads/main"
                and os.environ.get("GITHUB_EVENT_NAME") == "workflow_dispatch", "manual_main_only")
        valid_sha(record["controller_sha"])
        positive_id(record["run_id"])
        positive_id(record["run_attempt"])
        command = sys.argv[1]
        if command == "preflight":
            record["expected_current_digest"] = valid_digest(os.environ["EXPECTED_CURRENT_DIGEST"])
            require(os.environ.get("RELEASE_SERVICE_ACCOUNT") ==
                    f"register-news-release@{PROJECT}.iam.gserviceaccount.com", "invalid_release_account")
            if record["mode"] == "rollback":
                positive_id(os.environ["RELEASE_RUN_ID"])
                positive_id(os.environ["RELEASE_RUN_ATTEMPT"])
                valid_digest(os.environ["TARGET_DIGEST"])
            save_report(record)
            return
        if command == "checks-passed":
            record["checks_passed"] = True
            save_report(record)
            return
        if command == "validate-record-run":
            run = github_run(os.environ["RELEASE_RUN_ID"])
            # Validate run identity before downloading its artifact.
            stub = {"schema_version": 1, "repository": REPO, "mode": "deploy" if run.get("path", "").split("@")[0].endswith("/deploy.yml") else "rollback",
                    "run_id": os.environ["RELEASE_RUN_ID"], "run_attempt": positive_id(os.environ["RELEASE_RUN_ATTEMPT"]),
                    "controller_sha": run.get("head_sha"), "previous_digest": valid_digest(os.environ["TARGET_DIGEST"])}
            validate_record(stub, run, stub["run_id"], stub["run_attempt"], stub["previous_digest"])
            return
        require(command in {"deploy", "rollback"} and record["mode"] == command, "invalid_mode")
        require(record.get("checks_passed") is True, "checks_not_passed")
        cloud = Cloud()
        controller = Release(cloud, record, save_report)
        before = controller.snapshot(os.environ["EXPECTED_CURRENT_DIGEST"])
        if command == "deploy":
            target = controller.build(valid_sha(record["controller_sha"]), os.environ["BUILD_SERVICE_ACCOUNT"])
            controller.note(source_sha=record["controller_sha"])
        else:
            run_id = positive_id(os.environ["RELEASE_RUN_ID"])
            attempt = positive_id(os.environ["RELEASE_RUN_ATTEMPT"])
            target = valid_digest(os.environ["TARGET_DIGEST"])
            original = json.loads((Path(os.environ["ROLLBACK_RECORD_DIR"]) / "release.json").read_text())
            source = validate_record(original, github_run(run_id), run_id, attempt, target)
            origin = original.get("build_id") if original.get("new_digest") == target else None
            if origin is not None:
                valid_build_id(origin)
            controller.note(source_sha=source, rollback_record_run=run_id, rollback_record_attempt=attempt,
                            origin_build_id=origin,
                            new_digest=target)
        installed = controller.install(before, target)
        controller.observe(installed, target)
        controller.note(result="success")
    except Exception as error:
        record.update(result="failure", error_code=str(error) if isinstance(error, ReleaseError) else "internal_error")
        save_report(record)
        print("Release stopped; consult the sanitized release report.", file=sys.stderr)
        raise SystemExit(1) from None


if __name__ == "__main__":
    main()
