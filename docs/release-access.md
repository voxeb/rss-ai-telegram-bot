# One-time GitHub Actions access setup

This is a concrete **plan, not an applied configuration** for [Issue #12](https://github.com/voxeb/rss-ai-telegram-bot/issues/12). Cloud IAM, APIs, GitHub variables, and production were not changed while preparing the workflows. Review and explicitly authorize the setup separately. Run commands from the merged, reviewed checkout. Do not create or paste JSON service-account keys.

## Read-only inventory on 2026-10-25 (UTC)

| Resource | Observed state / action |
|---|---|
| Project | `register-news`, number `130549813403` |
| Cloud Run | Existing Job `register-news` in `europe-west1`; runtime `register-bot@register-news.iam.gserviceaccount.com`; pinned digest `sha256:e36010d5d0e7206cfe4c1a6c9102d1e60a3edbc8d42e3627986287821e5005bd`; v2 GET returns an etag |
| Scheduler | Existing service account `register-scheduler@register-news.iam.gserviceaccount.com` has Job-scoped `roles/run.invoker`; sampled execution creator is this account |
| Artifact Registry | Existing Docker repository `cloud-run-source-deploy` in `europe-west1`; no repository-level bindings returned (project inheritance still applies) |
| Current Cloud Build account | Last successful build `19f7b15a-69d8-4779-a08c-6d34765c39df` used the default compute account, which has project Editor. Do not grant Actions impersonation of that account |
| Existing identities | Runtime, Scheduler, and default compute service accounts. Proposed release and builder accounts do not exist |
| WIF / custom roles | No workload identity pools in `global`; no project custom roles returned. Proposed pool, provider, roles, and bindings are new |
| APIs already enabled | IAM, IAM Credentials, Cloud Build, Cloud Run, Artifact Registry, Cloud Logging, Cloud Scheduler, Service Usage, and existing application APIs. Reuse them |
| API to enable | `sts.googleapis.com` was not in the enabled-service list |
| GitHub | Repository ID `1432283866`, owner ID `4266793`, default branch `main`; no repository variables or environments returned |
| Preserved work | Operational audit [PR #8](https://github.com/voxeb/rss-ai-telegram-bot/pull/8) and manual headline deployment evidence [PR #11](https://github.com/voxeb/rss-ai-telegram-bot/pull/11) remain independent |

Inventory used `gcloud auth list`, pool/service-account/custom-role/service lists, project/Job/registry/service-account IAM policies, a prior build, and read-only v2 Job/execution and registry image metadata. No secret payloads or credential values were printed or saved. Project policies establish existing bindings, not a complete audit of inherited organization policies or IAM deny rules.

Before applying, repeat those read-only checks and describe the proposed names. If any now exists, inspect its configuration and bindings and reuse it only if they match; do not rerun resource creation blindly. Do not remove existing Editor, runtime, Scheduler, or service-agent bindings as part of this setup.

## Resources and least-privilege bindings

Create one dedicated WIF pool/provider, two service accounts, and the three custom roles committed in `.github/iam/`. Reuse the existing Job, registry, Scheduler, runtime secrets, Cloud Build service agent, and Logging infrastructure.

| Identity | Scope | Rights |
|---|---|---|
| GitHub external identity | Release service account only | `roles/iam.workloadIdentityUser`, constrained by the provider condition and repository ID principalSet |
| `register-news-release` | Existing Job only | Custom `registerNewsJobRelease`: get/update Job, list/get executions; no create/delete/run/cancel |
| `register-news-release` | Project | Custom `registerNewsReleaseController`: create/get builds, list log entries, use enabled services |
| `register-news-release` | Existing registry only | Custom `registerNewsReleaseImageReader`: get Docker image metadata and download artifacts for deployment access |
| `register-news-release` | Builder and runtime accounts only | `roles/iam.serviceAccountUser` (`actAs`), required to submit as the builder and retain the runtime identity on Job update |
| `register-news-builder` | Existing registry only | `roles/artifactregistry.writer` for Docker build/push |
| `register-news-builder` | Project | `roles/logging.logWriter` for `CLOUD_LOGGING_ONLY` build logs |

The new accounts receive no Secret Manager, Firestore, Scheduler administration, or IAM administration roles. The builder cannot deploy. Existing Cloud Build and Cloud Run service agents continue performing their normal service functions. No additional Token Creator grant is planned for the same-project user-managed builder or ordinary WIF impersonation.

IAM cannot limit `run.jobs.update` to the image field: that guarantee is enforced by the reviewed controller, complete configuration comparison, and etag precondition. A release identity that can deploy application code inherently affects code with runtime access. Protect write access to main, workflow files, controller scripts, and WIF administration. The project-level build and log permissions are broader than a single build/log stream; raw responses stay in memory and are not report artifacts.

## Commands to apply only after approval

The following names are absent in the inventory. Stop on any command failure; do not suppress an unexpected error or assume an existing resource is equivalent.

```bash
gcloud services enable sts.googleapis.com --project=register-news

gcloud iam service-accounts create register-news-release \
  --project=register-news --display-name='Register News GitHub Releases'
gcloud iam service-accounts create register-news-builder \
  --project=register-news --display-name='Register News Container Builder'

gcloud iam roles create registerNewsReleaseController --project=register-news \
  --file=.github/iam/release-controller.yaml
gcloud iam roles create registerNewsJobRelease --project=register-news \
  --file=.github/iam/job-release.yaml
gcloud iam roles create registerNewsReleaseImageReader --project=register-news \
  --file=.github/iam/image-reader.yaml

gcloud iam workload-identity-pools create register-news-github \
  --project=register-news --location=global --display-name='Register News GitHub'
gcloud iam workload-identity-pools providers create-oidc github-release \
  --project=register-news --location=global \
  --workload-identity-pool=register-news-github \
  --issuer-uri=https://token.actions.githubusercontent.com \
  --attribute-mapping='google.subject=assertion.sub,attribute.repository_id=assertion.repository_id,attribute.repository_owner_id=assertion.repository_owner_id,attribute.ref=assertion.ref,attribute.event_name=assertion.event_name,attribute.workflow_ref=assertion.workflow_ref' \
  --attribute-condition="$(cat .github/iam/github-provider-condition.cel)"

gcloud iam service-accounts add-iam-policy-binding \
  register-news-release@register-news.iam.gserviceaccount.com --project=register-news \
  --role=roles/iam.workloadIdentityUser \
  --member='principalSet://iam.googleapis.com/projects/130549813403/locations/global/workloadIdentityPools/register-news-github/attribute.repository_id/1432283866'

gcloud projects add-iam-policy-binding register-news \
  --member=serviceAccount:register-news-release@register-news.iam.gserviceaccount.com \
  --role=projects/register-news/roles/registerNewsReleaseController
gcloud run jobs add-iam-policy-binding register-news --project=register-news --region=europe-west1 \
  --member=serviceAccount:register-news-release@register-news.iam.gserviceaccount.com \
  --role=projects/register-news/roles/registerNewsJobRelease
gcloud artifacts repositories add-iam-policy-binding cloud-run-source-deploy \
  --project=register-news --location=europe-west1 \
  --member=serviceAccount:register-news-release@register-news.iam.gserviceaccount.com \
  --role=projects/register-news/roles/registerNewsReleaseImageReader

gcloud iam service-accounts add-iam-policy-binding \
  register-news-builder@register-news.iam.gserviceaccount.com --project=register-news \
  --member=serviceAccount:register-news-release@register-news.iam.gserviceaccount.com \
  --role=roles/iam.serviceAccountUser
gcloud iam service-accounts add-iam-policy-binding \
  register-bot@register-news.iam.gserviceaccount.com --project=register-news \
  --member=serviceAccount:register-news-release@register-news.iam.gserviceaccount.com \
  --role=roles/iam.serviceAccountUser

gcloud artifacts repositories add-iam-policy-binding cloud-run-source-deploy \
  --project=register-news --location=europe-west1 \
  --member=serviceAccount:register-news-builder@register-news.iam.gserviceaccount.com \
  --role=roles/artifactregistry.writer
gcloud projects add-iam-policy-binding register-news \
  --member=serviceAccount:register-news-builder@register-news.iam.gserviceaccount.com \
  --role=roles/logging.logWriter
```

The issuer is GitHub's OIDC endpoint. Default provider audience validation is retained. The CEL condition uses immutable owner/repository IDs, `refs/heads/main`, `workflow_dispatch`, and exactly the deploy/rollback workflow paths on main. A feature-branch copy, PR event, fork, other workflow, or other repository cannot satisfy it. No reusable workflow is involved. Main changes after dispatch cannot change the pinned source SHA of that run.

## GitHub variables (not secrets)

After verifying the bindings above, set these three repository variables. No GitHub secret or persistent cloud key is required. No environment is created automatically.

| Variable | Value |
|---|---|
| `GCP_WIF_PROVIDER` | `projects/130549813403/locations/global/workloadIdentityPools/register-news-github/providers/github-release` |
| `GCP_RELEASE_SERVICE_ACCOUNT` | `register-news-release@register-news.iam.gserviceaccount.com` |
| `GCP_BUILD_SERVICE_ACCOUNT` | `register-news-builder@register-news.iam.gserviceaccount.com` |

```bash
gh variable set GCP_WIF_PROVIDER --repo voxeb/rss-ai-telegram-bot \
  --body projects/130549813403/locations/global/workloadIdentityPools/register-news-github/providers/github-release
gh variable set GCP_RELEASE_SERVICE_ACCOUNT --repo voxeb/rss-ai-telegram-bot \
  --body register-news-release@register-news.iam.gserviceaccount.com
gh variable set GCP_BUILD_SERVICE_ACCOUNT --repo voxeb/rss-ai-telegram-bot \
  --body register-news-builder@register-news.iam.gserviceaccount.com
```

Re-read provider mappings/condition, the new identities' bindings, enabled APIs, and these variable values after setup. Verify no additional grant was required before adding one; capture exact sanitized denial codes. The controller and IAM plan have production-free tests and read-only API shape checks, but WIF exchange, custom-role sufficiency, actual PATCH etag enforcement, and dedicated-account builds have **not** been exercised live. The first approved manual dispatch is the integration validation; a failed check must stop without widening permissions automatically.

## Official references checked on 2026-10-25

- [Google WIF for deployment pipelines](https://docs.cloud.google.com/iam/docs/workload-identity-federation-with-deployment-pipelines) and [GitHub OIDC claims](https://docs.github.com/en/actions/reference/security/oidc).
- [Google authentication Action](https://github.com/google-github-actions/auth): WIF, short-lived access tokens, and disabling credential-file creation.
- [User-managed Cloud Build accounts and Logging](https://docs.cloud.google.com/build/docs/securing-builds/configure-user-specified-service-accounts).
- [Cloud Run Job resource and etag](https://docs.cloud.google.com/run/docs/reference/rest/v2/projects.locations.jobs) and [PATCH request](https://docs.cloud.google.com/run/docs/reference/rest/v2/projects.locations.jobs/patch).
- [Artifact Registry Docker image lookup](https://docs.cloud.google.com/artifact-registry/docs/reference/rest/v1/projects.locations.repositories.dockerImages/get).
