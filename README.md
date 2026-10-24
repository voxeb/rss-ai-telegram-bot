# RSS AI Telegram Bot

A lightweight Go application that summarizes technology news from [The Register](https://www.theregister.com/) using Google's Gemini API and publishes concise English summaries to Telegram.

**Live demo:** [@register_news](https://t.me/register_news)

## Why I built this

Traditional RSS-to-Telegram bots usually offer either headlines with links or full article text. Neither is ideal for quickly following technology news. This bot extracts article text and asks an LLM for the key facts and practical implications.

Each post contains the original headline, an AI-generated summary, and a link to the article. The prompt requests 2–4 English sentences and at most 80 words; output accuracy and exact length are not guaranteed.

## Scope and documentation

The current implementation is tailored to The Register and one destination channel per execution. The feed, HTML extractor, Gemini model endpoint, and English prompt are configured in code.

- [Architecture and project baseline](docs/architecture.md): implemented capabilities, recorded decisions, article states, limitations, and proposals awaiting agreement.
- [Operations](docs/operations.md): deployment evidence, installation, inspection, monitoring, recovery, rollback, and costs.
- [Issues](https://github.com/voxeb/rss-ai-telegram-bot/issues): work items and questions requiring action.

Repository behavior and live deployment state are tracked separately. The operations document identifies which production settings are reported and which still need verification.

## Architecture

Cloud Scheduler invokes a Cloud Run Job on the documented five-minute schedule. The Go process fetches RSS, acquires article claims in Firestore, extracts article text, requests a Gemini summary, sends it to Telegram, and records the result. No continuously running application server is required.

| Component | Technology |
|---|---|
| Application | Go 1.24 |
| AI summarization | Gemini 3.5 Flash-Lite |
| Execution / scheduling | Google Cloud Run Jobs / Cloud Scheduler |
| State and processing leases | Firestore |
| Credential injection | Google Secret Manager |
| Logs | Google Cloud Logging |
| Publishing | Telegram Bot API |

## Features

- Parses RSS 2.0 and extracts article content from HTML.
- Generates English summaries and publishes formatted posts with source links.
- Seeds existing feed articles on first normal execution without publishing them.
- Uses transactional Firestore leases to reduce concurrent duplicate processing.
- Prioritizes first attempts over retries; sorts chronologically within each pass.
- Attempts at most five articles per run, including failures.
- Persists a five-attempt article budget with retry delays of 5, 10, 20, and 40 minutes.
- Logs results and fails the Job when all attempted publications fail.
- Provides a one-off test publication mode and credential-free automated tests.

## Build and test

```bash
git clone https://github.com/voxeb/rss-ai-telegram-bot.git
cd rss-ai-telegram-bot
go mod download
go build -o register-news-bot .
go test -race -count=1 ./...
go vet ./...
```

Tests cover queue priority, failures at each publication stage, attempt limits, retry delays and exhaustion, duplicate feed items, leases, cancellation, and transaction callback retries. They use in-memory state and stubbed external operations, require no cloud credentials, and send no Telegram messages. GitHub Actions runs formatting checks, tests, vet, and a Linux build on pushes and pull requests.

## Configuration and deployment

| Environment variable | Description |
|---|---|
| `GOOGLE_CLOUD_PROJECT` | Google Cloud project ID |
| `TELEGRAM_CHANNEL` | Destination channel username |
| `TELEGRAM_TOKEN` | Telegram Bot API token; inject from Secret Manager |
| `GEMINI_API_KEY` | Gemini API key; inject from Secret Manager |
| `TEST_MODE` | Optional: `true` sends one real test post without changing publication history |

Deployment requires a Google Cloud project with billing enabled; Cloud Run, Cloud Build, Artifact Registry, Cloud Scheduler, Firestore, and Secret Manager APIs; a Firestore Native `(default)` database; a Gemini key with access to the configured model; and a Telegram bot allowed to publish in the channel.

Follow the [installation and deployment procedure](docs/operations.md#first-installation-and-deployment). For an existing installation, inspect the current state before changing settings or repeating initialization. Test mode publishes a real message and must not be enabled in the scheduled configuration.

## Limitations and security

- Work is selected from the current RSS feed. Extended outages or feed eviction can cause missed publications, including retries whose records remain in Firestore.
- Exactly-once Telegram delivery is not guaranteed; ambiguous network or storage failures can result in duplicates.
- Extraction depends on The Register's HTML. AI summaries may be inaccurate or omit context.
- Provider outages and quota failures use the same article retry budget. A failed Telegram send can cause another Gemini generation on retry.
- Partial failures do not necessarily fail the Job. Monitoring must include individual errors; see [operations](docs/operations.md#testing-and-monitoring).
- Gemini availability, quotas, and prices can change. Cloud services may incur charges; application limits and billing alerts do not guarantee a spending cap.

Never commit API keys, Telegram tokens, service-account credentials, or `.env` files. Use Secret Manager and least-privilege service accounts. Telegram transport errors can currently contain a token-bearing URL: redact credentials before sharing logs. This is a known code-derived risk; see the [limitations register](docs/architecture.md#known-limitations).

## Contributing

For a bug or behavior change, describe the problem and expected result in an Issue. Use a focused branch and PR, include tests where practical, and update the relevant documentation when behavior changes. Small documentation corrections can go directly to a PR. Keep proposed features separate from accepted decisions.

Merge after the relevant CI checks pass. Production deployment and configuration changes require explicit maintainer approval; a merge is not a deployment. Record the source commit and image digest when an approved deployment occurs.

## License

[MIT License](LICENSE). Copyright © 2026 Denis Beskov.
