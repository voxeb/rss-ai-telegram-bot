# Operations

This runbook describes existing behavior and how to inspect it. Initial repository baseline: [6174929](https://github.com/voxeb/rss-ai-telegram-bot/commit/61749299e7f49573fa36bcf6f56dd0025a2aaf8b), reviewed on 2026-10-25. See [architecture](architecture.md) for subsequent corrections, limitations, and open decisions.

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

At that review, outstanding checks from [baseline Issue #2](https://github.com/voxeb/rss-ai-telegram-bot/issues/2) included Scheduler settings, alert coverage, actual Gemini tier/quotas, and cloud spending. The [read-only audit below](#read-only-operational-audit-on-2026-10-25-utc) updates those findings. Production task settings and IAM were not re-inspected during the image update. The previous image is an observed rollback candidate, not a verified source baseline, and may predate the Telegram redaction correction.

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

## Read-only operational audit on 2026-10-25 (UTC)

Direct Cloud API/CLI inspection on 2026-10-25, through approximately 00:16 UTC, in project `register-news`, region `europe-west1`. This is a dated snapshot, separate from the maintainer-provided deployment evidence above. This audit triggered no Jobs, enabled no APIs, read no application secrets, wrote no production Firestore records, and changed no cloud resources. Publication evidence comes from existing application logs and execution metadata; Telegram channel contents were not independently checked.

### Scheduler and executions: verified facts

| Check | Observed result |
|---|---|
| Scheduler | `register-news-every-5min` in `europe-west1`; `ENABLED`; `*/5 * * * *`; `Etc/UTC` |
| Target | HTTP `POST` to `https://europe-west1-run.googleapis.com/apis/run.googleapis.com/v1/namespaces/register-news/jobs/register-news:run` |
| Invocation identity | OAuth service account `register-scheduler@register-news.iam.gserviceaccount.com`; scope `https://www.googleapis.com/auth/cloud-platform` |
| Cloud Run settings | Generation/observed generation `8/8`; one task; zero task retries; `180s` timeout; runtime identity `register-bot@register-news.iam.gserviceaccount.com` |
| Image in inspected completed execution | `register-news-6wzqn` uses digest `sha256:a99d6697569dc969c833a288cd030bac796a2fd9396a9022997da1adbef14ce6`, matching the deployment evidence above; `TEST_MODE` is not enabled |
| Recent execution sample | At the later query, 30 executions created between `2026-10-24T21:55:02Z` and `2026-10-25T00:15:09Z`: 29 had one successful task, none had a failed task, and newest `register-news-q4pz6` had started without a completion result yet |
| Latest completed execution inspected in detail | `register-news-6wzqn`: start `00:10:05.994586Z`, completion `00:10:41.165231Z`, one successful task; log at `00:10:37.584377Z`: `Published 0 new articles (attempted 0, failed 0)` |
| Latest nonzero publication log found | `register-news-5t5kq`, `2026-10-24T16:40:41.694547Z`: `Published 1 new articles`; one successful task, completed `16:40:45.280312Z`; `TEST_MODE` not enabled |

The last publication used the **older** digest `sha256:a616c921dbb2892b136f07eb1f6f873ba87867166214154ffcc9a8fb28718a0d`; its source commit remains unknown. It is evidence that the application reported a real publication, not validation of delivery by the current image. The inspected current-image summaries (23:55, 00:00, 00:05, 00:10 UTC) all report zero attempts. Scheduler configuration and observed executions support normal scheduling; effective IAM bindings were not exhaustively audited, and duplicate triggers in other regions were not searched.

The latest-publication search covered retained logs from `2026-09-25T00:00:00Z`, newest first, with `textPayload=~"Published [1-9][0-9]* new articles"`. A separate query from `2026-10-24T00:00:00Z` through inspection found no entries matching error severity or text for processing failures, all-attempts-failed batches, quota/429 errors, fatal errors, or panics. This is limited to those filters and retained logs; it does not prove that pending/exhausted articles or all possible failures are absent. Historical error bodies were not printed because they may contain credentials.

### Alerting: verified configuration and gaps

The complete project Monitoring policy list contains one enabled policy, **Gemini quota exceeded** (`14871148910083834259`). Its log condition is:

```text
resource.type="cloud_run_job"
resource.labels.job_name="register-news"
("Gemini HTTP 429" OR "Gemini HTTP 402")
```

It references one enabled email notification channel, limits notifications to once per `3600s`, notifies on incident opening, and auto-closes after `604800s` (seven days). Channel delivery and receipt were **not tested**. No other project policies or user-defined log metrics were listed: generic failed executions, partial publication failures, exhausted articles, and publication silence have no dedicated policies in this project. Policies managed in other monitoring projects were not inspected.

Budget presence and thresholds are **unverified**, not confirmed absent. Listing account budgets with an explicit `x-goog-user-project: register-news` returned HTTP `403`, reason `SERVICE_DISABLED`; Service Usage confirms `billingbudgets.googleapis.com` is disabled in both projects identified below. No API was enabled to bypass this restriction, and the Billing console was not inspected.

### Gemini, billing, and unavailable evidence

| Check | Observed result / boundary |
|---|---|
| Cloud Billing | `register-news` has billing enabled and an open USD billing account; this does not establish spending or Gemini tier |
| Separate Gemini project | The same billing account also links `gen-lang-client-0742109771` (number `474202085611`, display name `register-news`). Gemini API is enabled there and one API-key metadata entry exists. In deployment project `register-news` (`130549813403`), Gemini API is disabled and no API keys were listed |
| Key ownership | The separate project is a candidate for the bot's Gemini usage. Its key was **not matched** to the runtime secret; no key values or secret payloads were read |
| Quota definitions | Service Usage in the separate project returns model-specific free-tier entries for `gemini-3.5-flash-lite`: 15 requests/minute, 500 requests/day, 250,000 input tokens/minute. Paid-tier definitions are also returned; these entries do **not** establish which tier or limits apply to the runtime key |
| Active Gemini limits | Actual runtime-key project, active tier, remaining RPM/TPM/RPD allowance, and usage totals remain **unverified**; confirm in AI Studio for the key's owning project |
| Request metrics | Monitoring `serviceruntime.googleapis.com/api/request_count` filtered to Gemini returned no series in either project for `2026-10-24T00:00:00Z`–`2026-10-25T00:15:00Z`; empty metrics do not prove zero requests or zero cost |
| Actual spending | Month-to-date and daily Cloud/Gemini charges, credits, and token totals are **unverified**. No BigQuery datasets were listed in `register-news`; dataset listing in the separate Gemini project returned `400` because BigQuery is not enabled. An export elsewhere is not ruled out; Billing reports/invoices were not inspected |

Public pricing checked on 2026-10-25 lists a free tier and standard paid prices of **$0.30 per million input tokens and $2.50 per million output tokens** for this model ([official pricing](https://ai.google.dev/gemini-api/docs/pricing#gemini-3.5-flash-lite)). These are published rates, not measured charges or proof of free operation. [Active limits are project-specific and visible in AI Studio](https://ai.google.dev/gemini-api/docs/rate-limits); billing-account linkage alone does not prove the runtime tier ([billing guide](https://ai.google.dev/gemini-api/docs/billing)).

Evidence sources: `gcloud scheduler jobs list`, `gcloud run jobs describe`, execution list/details, Cloud Logging `entries.list`, paginated Monitoring alert-policy/channel lists, Logging metric list, Cloud Billing project/account metadata, Budget list (denied), BigQuery dataset lists, and Service Usage service/quota reads. OAuth credentials stayed in memory; notification addresses and raw historical error text are omitted. No cost-producing generation or BigQuery query jobs were submitted.

### Owner decisions and follow-up (proposals, not approved changes)

- Confirm the runtime key's owning project and active Gemini tier in AI Studio; obtain actual usage and Billing reports, including the separate Gemini project. Decide whether to authorize enabling the Budget API or provide console evidence to verify existing budgets.
- Choose a monthly spending target, notification recipients/thresholds, and the intended response to quota or budget exhaustion. Alerts alone do not enforce a spending cap.
- Decide which additional alerts to implement and their thresholds: failed executions, individual/partial failures, exhausted articles, and publication silence appropriate to the feed's activity. Verify delivery of the existing email channel if approved.
- Observe the next **naturally scheduled** run that attempts publication to validate the current image's normal path; investigate state read-only if silence persists. Any forced execution, test post, retry reset, deployment, or cloud configuration change needs separate explicit approval.

This completes part of the follow-up originally tracked in now-closed [Issue #2](https://github.com/voxeb/rss-ai-telegram-bot/issues/2); the unverified items and owner decisions above remain unresolved. A documentation PR does not approve the proposed operational changes.

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

Inspect the selected trigger's identity, target, authorization, schedule, and state. The five-minute schedule and OAuth identity were directly verified in the [dated audit](#read-only-operational-audit-on-2026-10-25-utc); recheck them when investigating later changes. Match the image digest to Cloud Build/deployment records. A passing GitHub check is not deployment evidence.

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

The [dated audit](#alerting-verified-configuration-and-gaps) found an enabled Gemini 429/402 alert, but no dedicated project policies for failed executions, individual processing failures, exhausted articles, or publication silence. These additional controls remain proposals requiring agreed thresholds. Choose thresholds after establishing normal feed activity; silence alone does not prove a fault. An alert on failed Jobs alone misses partial failures. Check application error messages as well as platform status; do not assume plain-text logs have structured error severity.

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

The [dated audit](#gemini-billing-and-unavailable-evidence) records the accessible quota definitions and the concrete gaps in actual tier, usage, spending, and budget evidence. Before choosing limits, inspect AI Studio for the runtime key's project and actual Google Cloud billing reports. Record a dated result and the agreed spending policy in a follow-up Issue/PR linked to [baseline Issue #2](https://github.com/voxeb/rss-ai-telegram-bot/issues/2); credentials do not belong in that evidence. Do not infer free operation from the deployment guide. Billing alerts are notifications, not an automatic spending stop.

Checked on 2026-10-25: Google lists `gemini-3.5-flash-lite` and a free tier; actual project access and limits remain unverified. Use the current official sources when changing the model or budget:

- [Gemini model](https://ai.google.dev/gemini-api/docs/models/gemini-3.5-flash-lite), [pricing](https://ai.google.dev/gemini-api/docs/pricing), and [rate limits](https://ai.google.dev/gemini-api/docs/rate-limits).
- [Cloud Run pricing](https://cloud.google.com/run/pricing) and [budget notifications](https://cloud.google.com/billing/docs/how-to/budgets).
- CLI references: [describe](https://cloud.google.com/sdk/gcloud/reference/run/jobs/describe), [deploy](https://cloud.google.com/sdk/gcloud/reference/run/jobs/deploy), [execute](https://cloud.google.com/sdk/gcloud/reference/run/jobs/execute), [executions list](https://cloud.google.com/sdk/gcloud/reference/run/jobs/executions/list), [update](https://cloud.google.com/sdk/gcloud/reference/run/jobs/update), and [Scheduler list](https://cloud.google.com/sdk/gcloud/reference/scheduler/jobs/list).
