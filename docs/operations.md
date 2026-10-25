# Operations

This runbook describes existing behavior and how to inspect it. Initial repository baseline: [6174929](https://github.com/voxeb/rss-ai-telegram-bot/commit/61749299e7f49573fa36bcf6f56dd0025a2aaf8b), reviewed on 2026-10-25. See [architecture](architecture.md) for subsequent corrections, limitations, and open decisions.

## Deployment evidence

### Headline normalization deployment on 2026-10-25 (UTC)

The maintainer explicitly authorized deployment of merged [PR #10](https://github.com/voxeb/rss-ai-telegram-bot/pull/10), which fixes [Issue #9](https://github.com/voxeb/rss-ai-telegram-bot/issues/9), from the exact source commit below. Evidence was collected directly through authenticated Cloud Build, Artifact Registry, Cloud Run, Cloud Scheduler, and Cloud Logging reads. The existing Job was updated with **only `--image`**; no manual execution or test publication was requested.

| Evidence | Verified value |
|---|---|
| Project / region / Job | `register-news` / `europe-west1` / `register-news` |
| Source commit | [`047539115b660ca3c9c50914a46282f729614b23`](https://github.com/voxeb/rss-ai-telegram-bot/commit/047539115b660ca3c9c50914a46282f729614b23) |
| Cloud Build | [`19f7b15a-69d8-4779-a08c-6d34765c39df`](https://console.cloud.google.com/cloud-build/builds;region=europe-west1/19f7b15a-69d8-4779-a08c-6d34765c39df?project=register-news), `SUCCESS` |
| Build start / completion | `2026-10-25T13:54:50.462128692Z` / `2026-10-25T13:56:17.057179Z` |
| Source verification | Both `source.gitSource.revision` and `sourceProvenance.resolvedGitSource.revision` equal the full source SHA above; both repository URLs are `https://github.com/voxeb/rss-ai-telegram-bot.git` |
| Image repository / build tag | `europe-west1-docker.pkg.dev/register-news/cloud-run-source-deploy/register-news` / `git-047539115b66` |
| New immutable image digest | `sha256:e36010d5d0e7206cfe4c1a6c9102d1e60a3edbc8d42e3627986287821e5005bd`; identical in Cloud Build results, Artifact Registry, the updated Job, and the checked execution |
| Image push completed | `2026-10-25T13:56:16.115943959Z` |
| Previous image / rollback digest | `sha256:a99d6697569dc969c833a288cd030bac796a2fd9396a9022997da1adbef14ce6` in the same repository; resolved from the existing `git-3f61326847fc` tag and independently checked in pre-update execution `register-news-qm8g2` |
| Job readiness | Generation / observed generation changed from `8/8` to `9/9`; `Ready=True` after the update, before the 14:00 UTC schedule |
| First scheduled execution after the update | `register-news-lp669`, created `2026-10-25T14:00:13.200093Z` |
| Execution start / completion | `2026-10-25T14:00:16.555427Z` / `2026-10-25T14:01:00.324534Z` |
| Execution result | `Completed=True`, `succeededCount: 1`, no failed task reported; `TEST_MODE` not enabled |
| Application counters | At `2026-10-25T14:00:57.982999Z`: `Published 0 new articles (attempted 0, failed 0)` |
| Error inspection | Three execution-scoped log entries retrieved after completion; none matched error severity or text for errors, failed processing, panic, fatal errors, quota, HTTP 429, or HTTP 402. The successful batch summary's `failed 0` is not an error |

The build repeated the existing Dockerfile process used by build `7b6abcc3-71dd-4532-988e-db3ae1e42cee`: the `gcb-internal` step ran `docker build --network cloudbuild --no-cache`, using the legacy worker release and the same default build service account. Source was fetched directly from GitHub at the full SHA; the local checkout and its unrelated work were not uploaded.

```bash
gcloud builds submit https://github.com/voxeb/rss-ai-telegram-bot.git \
  --project=register-news --region=europe-west1 \
  --git-source-revision=047539115b660ca3c9c50914a46282f729614b23 \
  --tag=europe-west1-docker.pkg.dev/register-news/cloud-run-source-deploy/register-news:git-047539115b66 \
  --worker-release=legacy --timeout=3600s --async

gcloud run jobs update register-news \
  --project=register-news --region=europe-west1 \
  --image=europe-west1-docker.pkg.dev/register-news/cloud-run-source-deploy/register-news@sha256:e36010d5d0e7206cfe4c1a6c9102d1e60a3edbc8d42e3627986287821e5005bd
```

**Configuration preservation:** before/after comparison found only the image and the automatically regenerated `spec.template.metadata.labels.client.knative.dev/nonce` changed within the Job spec. Replacing the new nonce with the value inherited by the pre-update execution exactly reproduces the saved pre-update spec fingerprint after excluding the image. The task spec, including environment values and secret references, resources, service account, timeout, retry settings, and other container settings, also matches pre-update execution `register-news-sdq4r` after excluding the image. Environment values were compared in memory and were not printed or saved. Task count remains one, task retries zero, and timeout `180s`.

The Scheduler configuration fingerprint is unchanged: `register-news-every-5min` remains `ENABLED`, `*/5 * * * *`, `Etc/UTC`, targeting the existing Job's `:run` endpoint. Scheduler logs show the 14:00 UTC attempt and HTTP `200`, corresponding to execution creation. No Scheduler settings, APIs, IAM, or secrets were changed; no direct Firestore writes, history resets, manual Job executions, or `TEST_MODE` runs were performed. Naturally scheduled application work retains its normal Firestore behavior.

Execution log scope used:

```text
resource.type="cloud_run_job"
resource.labels.job_name="register-news"
resource.labels.location="europe-west1"
labels."run.googleapis.com/execution_name"="register-news-lp669"
```

**Verification boundary:** the source-to-image mapping, image-only runtime update, scheduling, and successful execution are verified. **Zero attempts do not validate successful publication or headline rendering in Telegram.** This run did not exercise article extraction, Gemini generation, Telegram delivery, or headline normalization on a published message. No channel message was independently inspected or edited. Confirm those paths from a future naturally scheduled run with an actual publication; the current observation does not establish that pending or exhausted articles are absent.

Rollback command saved before the update, **not executed**:

```bash
gcloud run jobs update register-news \
  --project=register-news --region=europe-west1 \
  --image=europe-west1-docker.pkg.dev/register-news/cloud-run-source-deploy/register-news@sha256:a99d6697569dc969c833a288cd030bac796a2fd9396a9022997da1adbef14ce6
```

This documentation is isolated from the existing operational-audit work in [PR #8](https://github.com/voxeb/rss-ai-telegram-bot/pull/8); that branch and working copy were preserved. The rollback command changes only the image and does not undo messages or state writes; any rollback remains a separate maintainer decision.

### Verified deployment on 2026-10-24 (UTC)

The maintainer built the merged [PR #5](https://github.com/voxeb/rss-ai-telegram-bot/pull/5) revision directly from GitHub with `gcloud builds submit --git-source-revision` and updated the existing Job's image. The evidence below was cross-checked from **maintainer-provided CLI output**, not direct access to the cloud account. A merge or image tag alone is not the evidence for this mapping.

| Evidence | Verified value |
|---|---|
| Project / region / Job | `register-news` / `europe-west1` / `register-news` |
| Source commit | [`3f61326847fce5030dc65df3178e537dad8034f7`](https://github.com/voxeb/rss-ai-telegram-bot/commit/3f61326847fce5030dc65df3178e537dad8034f7) |
| Cloud Build | `7b6abcc3-71dd-4532-988e-db3ae1e42cee`, `SUCCESS` |
| Source verification | Both `source.gitSource.revision` and `sourceProvenance.resolvedGitSource.revision` match the commit above; repository URL is `https://github.com/voxeb/rss-ai-telegram-bot.git` |
| Image repository / tag | `europe-west1-docker.pkg.dev/register-news/cloud-run-source-deploy/register-news` / `git-3f61326847fc` |
| Immutable image digest | `sha256:a99d6697569dc969c833a288cd030bac796a2fd9396a9022997da1adbef14ce6`; identical in Cloud Build results, Artifact Registry, and the checked execution |
| Image push completed | `2026-10-24T23:52:29.753100014Z` |
| First checked execution | `register-news-5nhct` |
| Execution start / completion | `2026-10-24T23:55:05.983773Z` / `2026-10-24T23:55:41.280261Z` |
| Execution outcome | `succeededCount: 1`; application log at `2026-10-24T23:55:38.115744Z`: `Published 0 new articles (attempted 0, failed 0)` |
| Previous image recorded for rollback | Same image repository, digest `sha256:15dcecba28353d11f480ea3c24a22059f25c16870f8b87cc796bc466ab928405` |

This confirms that the deployed image includes PR #5's Telegram error redaction and that a normal run completed without attempted article processing. It does **not** exercise live Gemini generation, Telegram delivery, or the redaction failure path. Zero attempts can reflect no new work, leases, delayed retries, or exhausted items; it does not prove the absence of pending work.

The preceding executions `register-news-94b82` and `register-news-8mdz7`, created at 23:45 and 23:50 UTC, both used the recorded previous image and reported one successful task. Their source commit and application outcomes were not checked. The observed five-minute cadence is consistent with the documented schedule, but does not verify the Scheduler configuration.

Outstanding checks from [baseline Issue #2](https://github.com/voxeb/rss-ai-telegram-bot/issues/2): Scheduler identity, location, schedule, target and enabled state; alert coverage; actual Gemini tier/quotas; and cloud spending. Production task settings and IAM were not re-inspected during this image update. The previous image is an observed rollback candidate, not a verified source baseline, and may predate the Telegram redaction correction.

### Earlier configuration snapshot

The maintainer supplied the following `gcloud run jobs describe` output during queue troubleshooting. This historical **operator-provided snapshot** is retained for context; its image is superseded by the deployment above and its other settings have not been reverified.

| Setting | Reported value |
|---|---|
| Project / region / Job | `register-news` / `europe-west1` / `register-news` |
| Task count / task retries / timeout | `1` / `0` / `180s` |
| Runtime service account | `register-bot@register-news.iam.gserviceaccount.com` |
| Image repository | `europe-west1-docker.pkg.dev/register-news/cloud-run-source-deploy/register-news` |
| Image digest | `sha256:a616c921dbb2892b136f07eb1f6f873ba87867166214154ffcc9a8fb28718a0d` |

The source commit for this historical digest is **not verified**. Do not label it as containing PR #1 without build/deployment evidence.

After each approved deployment, update the dated evidence above with its source commit, immutable image digest, approving PR, and first checked execution. Keep any remaining uncertainty explicit.

## Inspect before changing anything

Use the actual project and region. These examples show the reported deployment; fork operators should substitute their own values.

```bash
PROJECT_ID="register-news"
REGION="europe-west1"
JOB="register-news"
```

Read only the relevant Job fields; do not export secret values:

```bash
gcloud run jobs describe "$JOB" \
  --project="$PROJECT_ID" --region="$REGION" \
  --format="yaml(spec.template.spec.taskCount,spec.template.spec.template.spec.serviceAccountName,spec.template.spec.template.spec.maxRetries,spec.template.spec.template.spec.timeoutSeconds,spec.template.spec.template.spec.containers[0].image)"

gcloud run jobs executions list \
  --job="$JOB" --project="$PROJECT_ID" --region="$REGION" --limit=10
```

Discover the Scheduler Job before assuming its name or creating another one. Set `SCHEDULER_REGION` to the location actually used; it need not match the Cloud Run region.

```bash
SCHEDULER_REGION="YOUR_SCHEDULER_REGION"
gcloud scheduler jobs list \
  --project="$PROJECT_ID" --location="$SCHEDULER_REGION" \
  --format="table(name,schedule,timeZone,state,httpTarget.uri)"
```

Inspect the selected trigger's identity, target, authorization, schedule, and state in the console. The documented schedule is `*/5 * * * *`; it has not been verified against the live trigger. Match the image digest to Cloud Build/deployment records. A passing GitHub check is not deployment evidence.

## First installation and deployment

Use an authenticated Google Cloud CLI with permission to manage the intended resources. For a new installation, enable Cloud Run, Cloud Build, Artifact Registry, Cloud Scheduler, Firestore, and Secret Manager APIs; use a Firestore Native `(default)` database. Create `telegram-bot-token` and `gemini-api-key` secrets without committing their values. The runtime `register-bot` service account needs `roles/datastore.user` and `roles/secretmanager.secretAccessor` on those two secrets. Use a separate Scheduler identity with permission to invoke the Job.

For an existing deployment, inspect its configuration and source checkout first. Production configuration changes and deployment require explicit maintainer approval. Capture the current image digest and relevant non-secret settings for rollback.

The following command is the installation template; replace all project-specific values and run it from the approved source revision. Its `--set-*` flags replace configuration and must not be copied blindly over an existing Job.

```bash
PROJECT_ID="YOUR_PROJECT_ID"
REGION="europe-west1"
CHANNEL="@YOUR_CHANNEL"
JOB="register-news"

gcloud run jobs deploy "$JOB" \
  --source . --project="$PROJECT_ID" --region="$REGION" \
  --service-account="register-bot@${PROJECT_ID}.iam.gserviceaccount.com" \
  --tasks=1 --max-retries=0 --task-timeout=180s \
  --memory=512Mi --cpu=1 \
  --set-env-vars="GOOGLE_CLOUD_PROJECT=${PROJECT_ID},TELEGRAM_CHANNEL=${CHANNEL}" \
  --set-secrets="TELEGRAM_TOKEN=telegram-bot-token:latest,GEMINI_API_KEY=gemini-api-key:latest"
```

Only on a new installation, before enabling the Scheduler, run the Job once to seed current feed items without publishing them:

```bash
gcloud run jobs execute "$JOB" \
  --project="$PROJECT_ID" --region="$REGION" --wait
```

On an initialized installation the same command performs real publication. Never delete `settings/feed` or publication records to repeat setup: seeding can mark current articles skipped and damage publication history. After initialization, configure the Scheduler to invoke the Job every five minutes. Keep `TEST_MODE` out of its persistent configuration.

## Testing and monitoring

Use the local credential-free tests for development. A one-off integration test below sends a **real Telegram message** and does not update publication history; use an approved destination and execution. It overrides the environment for this execution only.

```bash
gcloud run jobs execute "$JOB" \
  --project="$PROJECT_ID" --region="$REGION" \
  --update-env-vars=TEST_MODE=true --wait
```

Inspect execution logs in Cloud Logging using this filter:

```text
resource.type="cloud_run_job"
resource.labels.job_name="register-news"
resource.labels.location="europe-west1"
```

Example result: `Published 2 new articles (attempted 3, failed 1)`.

| Result | Meaning and follow-up |
|---|---|
| At least one attempt and no publication | Job fails. Inspect the first error, affected URLs, provider responses, and retry state. |
| Some publications and some failures | Job succeeds; individual failures still require inspection. |
| No eligible articles | Job succeeds. This can mean no new work, active leases, delayed retries, or exhausted attempts. |
| Firestore failure or cancellation | Job fails. A Telegram send may already have succeeded; inspect the channel before retrying. |

Recommended controls, **not verified as configured**: failed-execution alerts, detection of individual processing errors, and review of exhausted records. Choose alert thresholds after establishing normal feed activity; silence alone does not prove a fault. An alert on failed Jobs alone misses partial failures. Check application error messages as well as platform status; do not assume plain-text logs have structured error severity.

The image verified in [deployment evidence](#deployment-evidence) includes the Telegram error sanitization from [Issue #3](https://github.com/voxeb/rss-ai-telegram-bot/issues/3). That run did not exercise a Telegram request or failure. Older versions can log the Telegram request URL and token; redact credentials from historical logs before sharing them or attaching evidence to a public Issue. No actual disclosure has been established.

## Recovering an article

1. Identify the original failure from logs and the `articles/{key}` record; fix or resolve its cause first.
2. If a Telegram request had an uncertain result, or Firestore completion failed, inspect the destination channel. Delivery is not atomic with storage, and the application does not store Telegram message IDs.
3. Confirm the item remains in the current RSS feed. The existing worker will not process it after feed eviction, even if a Firestore record exists.
4. After approval for the retry and after any live claim has ended, reset only that article to `status: "retry"`, `attempts: 0`, `owner: ""`, and a Firestore timestamp in the past for `leaseUntil`. Preserve its identity and other fields. Do not reset an active worker's claim.
5. Observe the next normal execution and the channel. It may process other eligible articles as well; new articles have priority. Leave `published` and `seeded` records unchanged.

There is no implemented replay command for items outside the feed. Global quota/service errors currently consume article attempts; do not repeatedly reset many records without resolving the common cause. Retry and stale-news policy remain open decisions.

## Rollback

After explicit approval, pause the known Scheduler trigger if needed and account for any already-running executions. Select a previously recorded, known-good immutable image; check its compatibility with the existing Firestore states. Do not clear Firestore to roll back code.

```bash
PREVIOUS_IMAGE="YOUR_PREVIOUS_IMAGE_URI@sha256:YOUR_RECORDED_DIGEST"
gcloud run jobs update "$JOB" \
  --project="$PROJECT_ID" --region="$REGION" \
  --image="$PREVIOUS_IMAGE"
```

Updating the image does not undo earlier messages or state writes, restore changed Job settings, or pin secrets referenced as `latest`. Restore any changed non-secret settings separately from the recorded baseline. Check one approved execution and its channel effects before resuming the normal schedule; record the rollback and source/image mapping.

## Quotas and costs

The five-attempt run limit, five-attempt article budget, 24,000-byte article input cap, and 300-output-token request limit bound individual work, not total project spending. A retry after Telegram failure regenerates the summary. Firestore reads, builds, image storage, scheduling, logs, and Job execution can also contribute to costs.

Before choosing limits, inspect actual Gemini tier, usage and quotas in AI Studio, plus actual Google Cloud billing and existing alerts. Record a dated result and the agreed spending policy in [baseline Issue #2](https://github.com/voxeb/rss-ai-telegram-bot/issues/2); credentials do not belong in that evidence. Do not infer free operation from the deployment guide. Billing alerts are notifications, not an automatic spending stop.

Checked on 2026-10-25: Google lists `gemini-3.5-flash-lite` and a free tier; actual project access and limits remain unverified. Use the current official sources when changing the model or budget:

- [Gemini model](https://ai.google.dev/gemini-api/docs/models/gemini-3.5-flash-lite), [pricing](https://ai.google.dev/gemini-api/docs/pricing), and [rate limits](https://ai.google.dev/gemini-api/docs/rate-limits).
- [Cloud Run pricing](https://cloud.google.com/run/pricing) and [budget notifications](https://cloud.google.com/billing/docs/how-to/budgets).
- CLI references: [describe](https://cloud.google.com/sdk/gcloud/reference/run/jobs/describe), [deploy](https://cloud.google.com/sdk/gcloud/reference/run/jobs/deploy), [execute](https://cloud.google.com/sdk/gcloud/reference/run/jobs/execute), [executions list](https://cloud.google.com/sdk/gcloud/reference/run/jobs/executions/list), [update](https://cloud.google.com/sdk/gcloud/reference/run/jobs/update), and [Scheduler list](https://cloud.google.com/sdk/gcloud/reference/scheduler/jobs/list).
