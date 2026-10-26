# Architecture and project baseline

Initial baseline reviewed on 2026-10-25 against [6174929](https://github.com/voxeb/rss-ai-telegram-bot/tree/61749299e7f49573fa36bcf6f56dd0025a2aaf8b), including [PR #1](https://github.com/voxeb/rss-ai-telegram-bot/pull/1). Telegram error handling and tests below include the correction tracked in [Issue #3](https://github.com/voxeb/rss-ai-telegram-bot/issues/3). Update this document when behavior changes. Deployment evidence belongs in [operations.md](operations.md).

**Evidence:** "Implemented" means confirmed in repository code; "documented deployment" means a configuration described in the installation guide. Neither establishes live production state. Proposals below are not accepted requirements.

## Implemented capabilities

| ID | Capability | Current behavior |
|---|---|---|
| F1 | Feed ingestion | Reads The Register's RSS 2.0 items, including ordered article categories, normalizes headline HTML to plain text, and sorts by parsed publication time within each processing pass. Normal and test runs share the parser. The URL ends in `.atom`, but the parser expects RSS. |
| F2 | Article extraction | Extracts paragraphs and headings from `div.bodytext`, rejects text shorter than 250 bytes, and truncates input at 24,000 bytes. |
| F3 | Summarization | Calls `gemini-3.5-flash-lite`; the prompt requests English, 2–4 sentences, at most 80 words, key facts and implications. These are prompt instructions, not validated output guarantees. |
| F4 | Telegram publication | Sends the plain-text headline escaped inside a bold HTML element, an escaped summary, an optional source-category hashtag, and the original-article link. Normal and test modes share this HTML formatting. |
| F5 | Initialization | The first normal run marks current RSS items `seeded` and creates the initialization marker without publishing historical articles. |
| F6 | Duplicate suppression | Uses persistent identity, transactional five-minute processing leases, and ownership checks when completing claims. |
| F7 | Bounded retries | Prioritizes first attempts; allows at most five claims per run and five attempts per article, including expired processing claims. Failed attempts wait at least 5, 10, 20, then 40 minutes. |
| F8 | Diagnostics and test mode | Logs batch outcomes; all-attempts-failed batches fail the Job. Telegram errors are sanitized before returning to callers, retaining cancellation and timeout identification. `TEST_MODE=true` sends a real `[TEST]` message without updating publication history. |
| F9 | Automated validation | RSS headline/category fixtures, the full hashtag dictionary and normalization/selection tests, queue tests, Firestore-client tests with an RPC stub, and publication/error/log tests with an HTTP stub; CI checks formatting, race-enabled tests, vet, and a Linux build. |
| F10 | Controlled releases | Manual-main-only deploy/rollback workflows pin the controller commit, run shared CI checks before cloud authentication, use WIF, and generate sanitized release reports. Initial cloud access setup and live workflow integration remain unverified; see [access plan](release-access.md). |

Sources: [main.go](../main.go), [feed.go](../feed.go), [feed_test.go](../feed_test.go), [hashtags.go](../hashtags.go), [hashtags_test.go](../hashtags_test.go), [queue.go](../queue.go), [queue_test.go](../queue_test.go), [firestore_test.go](../firestore_test.go), [telegram_test.go](../telegram_test.go), and [CI workflow](../.github/workflows/test.yml). The [Go CI run](https://github.com/voxeb/rss-ai-telegram-bot/actions/runs/40139823191) and [CodeQL run](https://github.com/voxeb/rss-ai-telegram-bot/actions/runs/40139823209) passed on the initial baseline commit; this is a dated observation, not a promise about future commits or production.

Headline normalization runs once after XML decoding, before claims, summarization, or publication. It extracts HTML text, decodes HTML entities once, collapses whitespace, and omits comments and script/style/template content. Inline text stays adjacent; line breaks and common block elements separate words. Encoded literal angle brackets remain text. GUIDs and URLs are unchanged, preserving article identity. This does not edit previously published messages or migrate existing Firestore titles.

## Recorded decisions

These choices are embodied in the code or deployment guide. Historical alternatives and motives are not inferred. Keep future accepted decisions in this table with their approving PR.

| ID | Decision | Status and consequence |
|---|---|---|
| A1 | One Go batch process in a container | Implemented with Go 1.24 and a non-root runtime image. Cloud Run Jobs invoked every five minutes is the documented deployment. |
| A2 | Firestore stores state and coordinates claims | Implemented using `articles` and `settings`. No separate message broker; pending work is still selected from the current feed. |
| A3 | Article key is SHA-256 of GUID, a pipe separator, and URL | Implemented. A changed GUID or URL can create a different identity; there is no content-based or per-channel identity. |
| A4 | First attempts precede retries | Implemented in PR #1. Each pass is oldest-first; retries can wait when all five slots go to new articles. Global publication order is not guaranteed. |
| A5 | Application-managed retry budget | Implemented in PR #1. The deployment guide sets Cloud Run task retries to zero. Retry history survives process restarts. |
| A6 | Secrets arrive through environment variables | Implemented. Secret Manager injection and separate application/Scheduler identities are documented deployment choices; live IAM is unverified. |
| A7 | Specialize for the current use case | Implemented for one source and one target channel per execution. Feed URL, extractor, model endpoint, and English prompt are code constants. |
| A8 | Manual, conflict-checked release automation | Agreed in [Issue #12](https://github.com/voxeb/rss-ai-telegram-bot/issues/12). Deploy and rollback share a non-cancelling concurrency group, require an expected current digest, use the Cloud Run v2 etag precondition, and preserve all writable Job configuration except the image. A manual dispatch is release approval; a merge does not deploy. |
| A9 | Separate release authority and build authority | Workflow OIDC is constrained to repository/owner IDs, main, manual dispatch, and the two workflow paths. A dedicated builder receives registry/logging rights, while the release identity updates the existing Job. IAM setup is a plan requiring separate approval, not a deployed capability. |
| A10 | Automatic release evidence | Actions summaries and immutable per-attempt report artifacts replace per-release documentation PRs. Reports distinguish installed image, execution completion, and application-reported publication. Observation never triggers executions or automatic rollback. |
| A11 | Deterministic source-category hashtags | Agreed in [Issue #16](https://github.com/voxeb/rss-ai-telegram-bot/issues/16). Use the explicit dictionary below and the first recognized RSS category, with no AI classification or additional API calls. Missing or unknown metadata does not block publication. |

## RSS category hashtags

Each item's `category` elements are decoded into an ordered string slice. At the shared Telegram send boundary, dictionary lookup ignores case and collapses whitespace using Go's Unicode-aware whitespace splitting. The first recognized category supplies at most one hashtag, in its own paragraph between the summary and `Read full article →`. If no category is recognized, the previous message format is preserved exactly, without an extra empty paragraph. The headline, summary, hashtag, and link remain HTML-escaped.

| RSS category | Hashtag |
|---|---|
| ai and ml | #AI |
| security | #Security |
| cyber-crime | #Security |
| systems | #Systems |
| personal tech | #PersonalTech |
| virtualization | #Virtualization |
| networks | #Networking |
| public sector | #PublicSector |
| os platforms | #OperatingSystems |
| devops | #DevOps |
| software | #Software |
| applications | #Applications |
| science | #Science |
| columnists | #Opinion |
| offbeat | #Offbeat |
| bofh | #BOFH |

`security` and `cyber-crime` are independent categories with the same result. Categories are optional in-memory metadata, not stored in Firestore or passed to Gemini. This adds no dependencies or external requests and leaves article keys, deduplication, queue order, retry limits, and the Firestore schema unchanged. Test mode retains its `[TEST]` summary prefix and uses the same hashtag selection and formatting. Existing Telegram messages are not edited.

The source's sections sometimes indicate genre rather than topic: `columnists`, `offbeat`, and `bofh` are source labels, not semantic analysis of the article. Unknown future sections receive no tag until explicitly mapped. Fixture tests and stubbed HTTP publication tests cover parsing, the complete dictionary, normalization, first-recognized selection, and messages with and without tags in both modes; live Telegram rendering is not validated by those tests.

## Processing and stored state

Each normal run fetches the feed, checks initialization, then performs two passes over its items: unattempted articles, followed by eligible retries. Only acquired claims count toward the shared run limit; each article is attempted at most once per execution. A successful claim precedes HTML extraction, Gemini generation, Telegram delivery, and the Firestore result update. The program deadline is 110 seconds; the HTTP client timeout is 25 seconds. Five attempts is an upper bound, not guaranteed throughput.

| Stored data | Purpose |
|---|---|
| `settings/feed.initialized` | Initialization timestamp; the document's existence determines whether seeding is required. |
| `articles/{key}.status` | `seeded`, `processing`, `retry`, `published`, or `failed`. |
| `owner`, `leaseUntil` | Random claim owner and processing lease; `leaseUntil` also represents the earliest retry time. |
| `attempts` | Transactionally incremented claim count, including a claim whose worker subsequently crashes. |
| `url`, `title`, `updatedAt` | Article reference and most recent state update. Seeded records instead initially contain `status`, `url`, and `created`. |

`seeded`, `published`, and `failed` records are not automatically claimed. A failed attempt becomes `retry`, or `failed` when its budget is exhausted. An expired final processing claim is retired when the article is encountered again. Completion requires the current owner. Legacy retry/processing records without `attempts` receive a fresh budget and still yield to first attempts.

Article bodies, generated summaries, Telegram message IDs, and a durable list of all discovered pending items are not stored. [Operations](operations.md#recovering-an-article) describes the existing manual recovery procedure.

## Known limitations

| ID | Confirmed behavior or code-derived risk | Consequence |
|---|---|---|
| L1 | Only current RSS items are selected | Disappeared items cannot be retried; outages can also miss items that were never observed. |
| L2 | Telegram delivery and Firestore completion are separate operations | A lost response or failed completion write can lead to duplicate delivery. Exactly-once delivery is not guaranteed. |
| L3 | Publication errors share the same retry policy | Quota exhaustion, a provider outage, and a broken article can each exhaust an article's budget. |
| L4 | Retries repeat extraction and generation | A Telegram failure can cause another Gemini call; summaries are not reused. |
| L5 | Per-run limits are not a project spending cap | No application-wide daily request budget, token-usage accounting, or shared provider cooldown exists. |
| L6 | Partial success and no eligible work return success | Job status alone does not reveal individual failures or exhausted work. Alert configuration is unverified. |
| L7 | Extraction and summaries depend on external content | HTML changes can break extraction; the prompt's length, language, and factual requirements are not fully validated. |
| L8 | Test coverage focuses on coordination and Telegram failures | RSS/HTML fixtures provide only basic coverage; live provider integration is untested. The Firestore stub is not an emulator. |
| L10 | Hashtags mirror source categories | A section can identify genre rather than topic; only the first recognized category is shown, and unknown categories have no tag. |

Resolved in code: **L9**, credential-bearing Telegram errors ([Issue #3](https://github.com/voxeb/rss-ai-telegram-bot/issues/3)). The send boundary removes the request URL, redacts the configured token from remaining text, and retains safe cancellation/timeout causes without wrapping the original error. Regression tests cover returned errors and logs in both the queue and test-publication paths. Deployment of this correction was verified from maintainer-provided build and execution output on 2026-10-24 UTC; see [operations evidence](operations.md#deployment-evidence). The checked run attempted no publications, so it did not exercise Telegram delivery or redaction. Historical logs can still contain credentials. No actual disclosure has been established.

## Open questions and proposals

These remain open. Acceptance of this documentation does not select new recovery, budget, or product behavior. Record discussion in [baseline Issue #2](https://github.com/voxeb/rss-ai-telegram-bot/issues/2) and update decisions only when agreed.

| ID | Question | Proposal for discussion |
|---|---|---|
| Q1 | Which operational controls are live? | The 2026-10-24 deployment's source/image mapping and one successful run are recorded in [operations](operations.md#deployment-evidence). Verify Scheduler settings, alerts, quotas, spending, and a run that attempts publication. |
| Q2 | What recovery coverage and publication delay are required? | Retain discovered pending articles beyond feed eviction; decide separately whether unobserved articles need recovery and when news becomes too old. |
| Q3 | How should exhausted attempts and ambiguous delivery be handled? | Agree a manual review/retry procedure and acceptable duplicate-versus-omission tradeoffs before changing automation. |
| Q4 | What budget and quota policy is acceptable? | Agree an amount and response to provider limits before implementing a shared pause or spending controls. |
| Q5 | Should scope expand? | Keep The Register, English summaries, and one destination for the next phase unless a concrete request changes the scope. |

Possible later work: a durable discovered-article backlog, explicit provider-outage handling, and reuse of generated summaries. These are proposals, not implemented features or an approved roadmap. The existing queue-priority fix is complete in PR #1.
