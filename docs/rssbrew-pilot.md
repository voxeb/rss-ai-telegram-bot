# Local RSSBrew reading pilot

Date: 2026-10-30. Status: local, fixed-sample experiment; no production changes.

The maintainer authorized preparing the RSSBrew experiment after the [market research](market-research.md) and selected browser viewing first. This is a narrower first step than the proposed multi-participant study: 30 Register articles, one local reader, no automatic refresh or public hosting. A week of actual reading and demand validation has not happened yet.

## Setup

- Repository baseline: `b938dc9`, which includes research PR #18. No application or existing architecture decision was changed.
- RSSBrew source: [`0ccb5399d6ffb080d8c41db2ec5276861830dbd6`](https://github.com/yinan-c/RSSbrew/tree/0ccb5399d6ffb080d8c41db2ec5276861830dbd6), mounted over the app directory of the ARM64 runtime image `yinanc/rssbrew@sha256:ad5545e27b025af2e0d519735d3b5366a5836c9e908a4a820db0c24842f95158`. The image is an older dependency/runtime base, not proof it contains that source revision.
- Separate Compose project: `rssbrew-register-pilot`; app, Redis and a small local file server. Total configured memory ceiling: 512 MiB; ports bind only to `127.0.0.1`. No worker/cron consumer runs and containers do not auto-restart.
- Runtime dependencies observed: Django 5.2.17, OpenAI Python client 3.6.0. The client calls Google's compatibility endpoint, not OpenAI's service.
- Input: first 30 entries of a public Register feed snapshot; SHA-256 of the reduced XML: `bc7a690245788ab9f70cd7feb2895d8da371bdbb7182e2c393ad69e55fff54ca`. The original response had 50 entries; largest selected body was 8,453 UTF-8 bytes. No article-page scraping was needed.
- Model: `gemini-3.5-flash-lite`, English, requested 40–80 words in 2–4 sentences. Requests used temperature 0.2 and a 400-output-token ceiling.
- The existing Gemini secret was read into memory and passed to one-off processing through stdin. It was not printed, written to pilot files, or placed in the persistent web container's environment. Telegram credentials were not read. Cloud configuration and production Firestore were untouched.

Local runtime files and publisher content are under `/private/tmp/rssbrew-pilot`, outside this repository. Neither raw articles nor credentials are committed. This directory is temporary and can disappear after system cleanup; the experiment is not an installed service.

## What required adaptation

1. **Gemini message compatibility:** the unmodified RSSBrew summary function puts its custom prompt in a final `assistant` message. Google's endpoint rejected this with HTTP 400: `Requests ending with a model turn are not supported.` The pilot wrapper moved those instructions into the system message while leaving article text in the user message. The upstream source files were not edited.
2. **Rate control:** the first batch reached a live free-tier quota response, identifying 15 requests/minute for this model. Subsequent missing articles were processed with a five-second interval after a cooldown. The wrapper disables SDK retries and stops a batch on another HTTP 429. This observation concerns the key used in this experiment; it is not a new audit of production billing or effective limits.
3. **Failed-item recovery:** RSSBrew stores a new article before generation and skips an existing article on a later update. A failed summary therefore does not automatically recover through ordinary re-fetching. The pilot retried only unsummarized local rows; successful rows were preserved. Nothing in the production bot was reset.
4. **Summary-only output:** native RSSBrew prepends summaries to the full source content. Its `include_content` flag applies to digests, not individual entries. A local export creates summary-only RSS and a browser page, preserving source URLs, dates and stable item GUIDs. This is additional pilot code, not a native RSSBrew feature.
5. **Factual quality:** source spot-checks found at least two substantive errors: the AWS summary confused annual revenue with company valuation, and Voyager's approaching one-light-day distance milestone became a power milestone. The exported reading page/feed includes explicit source-check corrections; raw AI output remains in local evaluation records. A complete factual audit of all summaries has not been performed, and corrected text must not be counted as error-free automatic output.

These findings make an unattended production recommendation premature. The local reading trial can still establish whether the format is useful.

## Results and validation

- Completed **30/30 summaries** from the fixed sample; raw lengths were **57–83 words**, median **70**. **29/30** met the requested 40–80-word target. The two marked source-check corrections are editorial changes to exported text, not new model generations.
- **43 outbound generation requests:** 30 successes, one rejected message-role diagnostic, and 12 quota rejections in the initial batch. No additional successful generation was made for an already summarized article. The final wrapper ceiling was 30 successes and 44 total attempts; failed diagnostics remain in its ledger.
- Successful responses reported **39,438 input tokens and 2,941 output tokens**. At the [published standard prices](https://ai.google.dev/gemini-api/docs/pricing#gemini-3.5-flash-lite), their paid-price equivalent is approximately **$0.0192**. This is not an invoice or verified charge; the quota response identified free-tier usage, and failed requests returned no usage totals.
- The credential-free validation reran ingestion with a deliberately unusable key: all 30 stored entries were reused and the generation-request ledger remained unchanged.
- Standard-library XML parsing and feedparser both accepted all 30 summary-only RSS items. Checked unique/stable GUIDs, original Register links, publication dates, attribution, and absence of full-body text in summary-only descriptions.
- HTML structure checks found 30 story cards, two view-selection buttons, and two marked corrections. HTTP retrieval of the source snapshot succeeded. The computer-use browser host was unavailable, so visual browser rendering/click behavior was not verified through automation; that limitation is separate from successful HTTP and DOM-structure checks.

Commands run successfully:

```bash
python3 -m py_compile pilot_process.py pilot_export.py
docker compose -p rssbrew-register-pilot -f compose.pilot.yaml config --quiet
docker compose -p rssbrew-register-pilot -f compose.pilot.yaml exec -T rssbrew python /app/pilot_export.py
docker compose -p rssbrew-register-pilot -f compose.pilot.yaml exec -T rssbrew python /app/pilot_validate.py
```

The temporary scripts run inside the isolated pilot, not the production bot. Its application tests were not run because this repository change is documentation-only. No week-long engagement result or complete factual-quality pass is claimed.

## Viewing and operating the local sample

- Reading page: `http://127.0.0.1:18782/` — switches between AI summaries and native publisher descriptions, with original links and collapsible full feed text.
- Summary-only RSS: `http://127.0.0.1:18782/summary.xml`.
- Native RSSBrew output: `http://127.0.0.1:18780/feeds/Register-pilot/` — contains original article prose as well as raw summaries.

These addresses work only on the host Mac. They are not public URLs or links for the phone. The sample does not update while being read, and opening it does not call the model.

After stopping, restart the already-created containers while the temporary files exist:

```bash
cd /private/tmp/rssbrew-pilot
docker compose -p rssbrew-register-pilot -f compose.pilot.yaml start
```

Stop the isolated experiment without deleting its SQLite data:

```bash
cd /private/tmp/rssbrew-pilot
docker compose -p rssbrew-register-pilot -f compose.pilot.yaml stop
```

Do not start the Huey worker or reuse generation credentials for reading the cached sample. Scheduled fetching, sharing beyond this Mac, additional generations and any change to the production bot are outside this completed setup step.

## Next reading decision

Read a few stories in both views. Record whether the summary conveyed what happened, which facts required the source, any misleading statement, and whether the shorter view was worth keeping. The first decision is whether to continue this format after a week; no claim about engagement or willingness to pay is made now. Before public distribution, resolve the publisher-rights questions documented in the market research.
