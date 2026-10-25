# Operations

This runbook describes existing behavior and how to inspect it. Initial repository baseline: [6174929](https://github.com/voxeb/rss-ai-telegram-bot/commit/61749299e7f49573fa36bcf6f56dd0025a2aaf8b), reviewed on 2026-10-25. See [architecture](architecture.md) for subsequent corrections, limitations, and open decisions.

## Managed releases through GitHub Actions

The workflows in [Issue #12](https://github.com/voxeb/rss-ai-telegram-bot/issues/12) are merged and **one-time WIF/IAM setup was applied with approval on 2026-10-25 (UTC)**. See the [applied access configuration and integration limits](release-access.md). The first dispatch passed checks, OIDC, build provenance, and image installation; automatic execution observation failed on the API assumptions tracked in [Issue #14](https://github.com/voxeb/rss-ai-telegram-bot/issues/14). Do not treat that run as end-to-end success. Existing non-image Job settings and the work in PR #8 and PR #11 were preserved.

Once configured, a maintainer's manual dispatch from `main` explicitly authorizes the corresponding image deployment or rollback. Merge/push/PR events do not deploy. Both workflows use `register-news-production-update` with `cancel-in-progress: false`; GitHub may replace a pending run, so this is not a durable queue. Do not cancel a running update casually: a cancelled observer does not undo an accepted cloud update.

### Deploy

Inspect the Job and the latest report first. The expected digest is mandatory and prevents a queued release from replacing a version installed by another run or operator. Substitute the actual digest from the inspection; a tag, another repository, or a malformed digest is rejected.

```bash
gcloud run jobs describe register-news --project=register-news --region=europe-west1 \
  --format='value(spec.template.spec.template.spec.containers[0].image)'

gh workflow run deploy.yml --repo voxeb/rss-ai-telegram-bot --ref main \
  -f expected_current_digest=sha256:YOUR_CURRENT_64_HEX_DIGEST
```

The immutable dispatch `github.sha` is both the checkout and requested Cloud Build revision, even if main advances while the run waits. The workflow executes `.github/scripts/check-go.sh` (formatting, race-enabled tests, vet, Linux build) and production-free release-controller tests on that checkout before obtaining an OIDC access token. Failure stops subsequent release steps. Cloud Build fetches the public GitHub repository at that SHA using `gitSource`; no checkout archive, local files, or Actions authentication material is uploaded. Requested and resolved revisions, built tag/digest, and registry image metadata must agree before update. A separate builder account runs the existing Dockerfile process with Cloud Logging-only build logs; the old Editor-bearing default account is not used.

The controller records the previous pinned digest, checks the expected digest, and retains a pre-build Job snapshot in memory. Immediately before PATCH it rechecks the etag and complete writable configuration, and submits the original etag for a server-side conflict check. It changes one container image by digest, retaining environment, secret references, runtime identity, retries, timeout, resources, labels, and other settings. It refuses unpinned or foreign images, multiple containers, TEST_MODE, unsupported Job fields, or unexpected drift. Mutations are not retried after an ambiguous response; no automatic rollback is performed.

### Rollback without rebuilding

Use a retained report from a completed manual deploy/rollback workflow on this repository's main. Supply its run ID, attempt, the recorded target digest, and the expected current digest:

```bash
gh workflow run rollback.yml --repo voxeb/rss-ai-telegram-bot --ref main \
  -f expected_current_digest=sha256:YOUR_CURRENT_64_HEX_DIGEST \
  -f release_run_id=PRIOR_RUN_ID -f release_run_attempt=1 \
  -f image_digest=sha256:YOUR_RECORDED_64_HEX_DIGEST
```

The artifact must be named `release-report-RUN_ID-ATTEMPT`. GitHub run metadata must identify this repository, `workflow_dispatch`, main, and one of the two allowed workflow paths. The record must match that run, attempt, controller SHA, and mode. A target is accepted only as its verified previous image or a new image recorded as installed. It must still exist under the fixed application image in Artifact Registry. Expired/missing artifacts, arbitrary image URLs, tags, malformed inputs, and unrelated workflow reports fail closed. A failed release's report can be used if it recorded the applicable installed/previous image; a green workflow conclusion alone is not image provenance.

Rollback runs checks on its pinned **controller** SHA and performs no Cloud Build request. `source_sha` describes the restored image when known from the prior report; it is explicitly null for a prior image whose source was not recorded. It never substitutes the controller SHA for an unknown deployed source. Rollback changes neither application state nor previous messages and does not pin secrets referenced as `latest`.

### Automatic evidence and verification boundaries

Each attempt generates a job summary and a `release-report-RUN_ID-ATTEMPT` artifact containing only `release.json` and `release.md`, retained for 90 days (subject to repository retention/deletion policy). Available evidence is saved incrementally and finalized with `always()` even on prerequisite, build, update, or observation failure. Fields include controller/source SHA, build ID/status, old/new digest, image installation, scheduled execution ID and task result, batch counters/error counts, safe error code, and rollback image/dispatch inputs. No raw provider errors, application log text, Job environment, or auth files are uploaded. A lost runner or forced cancellation can prevent artifact upload; the Actions result and any saved summary/build metadata remain the recovery starting point.

```bash
gh run list --repo voxeb/rss-ai-telegram-bot --workflow deploy.yml
gh run list --repo voxeb/rss-ai-telegram-bot --workflow rollback.yml
gh run download RUN_ID --repo voxeb/rss-ai-telegram-bot \
  --name release-report-RUN_ID-ATTEMPT --dir release-evidence
```

The cloud stage waits at most 1,100 seconds for a build, 180 seconds for image readiness, and 900 seconds for the next scheduled execution and its logs, plus bounded API request time. The workflow has a 50-minute overall timeout; the post-check access token lasts at most one hour. The first execution created **after image verification**, by the existing Scheduler account, is checked against the target image. Its resource path must match the project, region, and Job; its parent field may use the short Job name returned by v2 or the full path. Manual/operator executions do not satisfy this check. Job generation/image are checked throughout observation so another operator's update cannot silently validate this release. Log searches use the execution time interval, follow up to ten pages (including empty continuation pages), and reject more than 1,000 entries or repeated page tokens. Only counters and error counts enter reports.

Read the three outcomes separately:

| Report evidence | Meaning |
|---|---|
| `image_installed=true` | Job readiness and target digest/configuration were verified; this is not proof of execution |
| `execution_completed=true`, `execution_succeeded=true` | A Scheduler-created execution on that image completed; this is not proof of publication |
| `article_published=true`, positive published counter | The application reported publication; Telegram channel rendering was not independently inspected |
| `completed_without_publication`, zero attempts | Normal execution completed without testing delivery, generation, or published headline rendering |
| Timeout / failed execution / partial failures / error entries | Workflow fails, retaining available evidence; the image may already be installed. Inspect before another update; no forced run or automatic rollback occurs |

An inconclusive release is not repaired by TEST_MODE, a manual Job execution, Firestore resets, or widening IAM. Obtain separate authorization for such actions. Release history is the automatic Actions evidence; **do not create a documentation PR after every managed release**. Archive reports externally if history beyond artifact retention is required. The dated manual deployment evidence below remains historical context.

Each manual deploy submits at most one build with a 900-second build timeout. Rollback never rebuilds, and observation is read-only, so these workflows do not add Job executions or Gemini/Telegram requests. Builds, retained images, logs, and Actions artifacts can still incur costs; these bounds are not a project spending cap.

## Deployment evidence

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

For managed releases, use the automatically generated Actions evidence described above rather than a per-release documentation PR. Preserve the dated manual evidence here and keep unverified claims explicit.

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
