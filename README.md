
# RSS AI Telegram Bot

A lightweight Go application that automatically summarizes technology news from [The Register](https://www.theregister.com/) using Google's Gemini API and publishes concise English summaries to Telegram.

**Live demo:** [@register_news](https://t.me/register_news)

## Why I built this

Traditional RSS-to-Telegram bots usually offer either headlines with links or full article text. Neither is ideal for quickly following technology news.

This project takes a different approach: it extracts the full article, uses an LLM to identify the key facts and implications, and publishes a short summary.

Each Telegram post contains:

- Original article headline
- AI-generated summary (2–4 sentences, up to 80 words)
- Link to the original article

## Architecture

```text
The Register (RSS 2.0)
         |
         v
Google Cloud Scheduler
    (every 5 minutes)
         |
         v
Google Cloud Run Job
      (Go)
         |
         +--> Fetch RSS feed
         |
         +--> Extract article text
         |
         +--> Gemini 3.5 Flash-Lite
         |       |
         |       v
         |    AI summary
         |
         +--> Telegram Bot API
         |
         +--> Firestore
              (article status,
               processing leases)
```

The application runs as a scheduled batch job. No continuously running server or VPS is required.

## Tech stack

| Component | Technology |
|---|---|
| Application | Go |
| AI summarization | Gemini 3.5 Flash-Lite |
| Execution | Google Cloud Run Jobs |
| Scheduling | Cloud Scheduler |
| Persistence | Firestore |
| Secrets | Google Secret Manager |
| Monitoring | Google Cloud Logging |
| Publishing | Telegram Bot API |

## Features

- Checks for new articles every five minutes
- Parses RSS 2.0 and sorts articles chronologically
- Extracts article content from HTML
- Generates concise English summaries
- Tracks processed articles in Firestore
- Uses transactional processing leases to reduce duplicate processing
- Retries failed article processing
- Processes up to five articles per execution
- Supports a one-off test publication mode
- Uses Secret Manager instead of hardcoded API credentials

## Requirements

- Google Cloud project with billing enabled
- Cloud Run, Cloud Build, Artifact Registry, Cloud Scheduler, Firestore, and Secret Manager APIs enabled
- Firestore Native database named `(default)`
- Gemini API key with access to `gemini-3.5-flash-lite`
- Telegram bot with permission to publish in the target channel
- Google Cloud CLI
- Go for local development

## Build

Clone the repository:

```bash
git clone https://github.com/voxeb/rss-ai-telegram-bot.git
cd rss-ai-telegram-bot
```

Build:

```bash
go mod download
go build -o register-news-bot .
```

## Configuration

The application uses the following environment variables:

| Variable | Description |
|---|---|
| `GOOGLE_CLOUD_PROJECT` | Google Cloud project ID |
| `TELEGRAM_CHANNEL` | Telegram channel username |
| `TELEGRAM_TOKEN` | Telegram Bot API token |
| `GEMINI_API_KEY` | Gemini API key |
| `TEST_MODE` | Optional: `true` enables a one-off test post |

The RSS URL, article HTML selector, Gemini model endpoint, and summary prompt are currently configured in `main.go`.

The implementation is tailored to The Register rather than being a general-purpose RSS reader.

## Deploy to Google Cloud Run

Create two secrets in Secret Manager:

- `telegram-bot-token`
- `gemini-api-key`

Create a service account named `register-bot` and grant it:

- `roles/datastore.user` on the project
- `roles/secretmanager.secretAccessor` on the two secrets

Set your deployment variables:

```bash
PROJECT_ID="YOUR_PROJECT_ID"
REGION="europe-west1"
CHANNEL="@YOUR_CHANNEL"

gcloud config set project "$PROJECT_ID"
```

Deploy:

```bash
gcloud run jobs deploy register-news \
  --source . \
  --region="$REGION" \
  --service-account="register-bot@${PROJECT_ID}.iam.gserviceaccount.com" \
  --tasks=1 \
  --max-retries=0 \
  --task-timeout=180s \
  --memory=512Mi \
  --cpu=1 \
  --set-env-vars="GOOGLE_CLOUD_PROJECT=${PROJECT_ID},TELEGRAM_CHANNEL=${CHANNEL}" \
  --set-secrets="TELEGRAM_TOKEN=telegram-bot-token:latest,GEMINI_API_KEY=gemini-api-key:latest"
```

### Initialize Firestore

The first normal execution records existing RSS articles without publishing historical content:

```bash
gcloud run jobs execute register-news \
  --region="$REGION" \
  --wait
```

### Test publication

To publish one test message:

```bash
gcloud run jobs execute register-news \
  --region="$REGION" \
  --update-env-vars=TEST_MODE=true \
  --wait
```

Test mode sends a real Telegram message marked `[TEST]` without updating article publication history.

### Schedule automatic execution

Create a Cloud Scheduler trigger for the Job.

Cron expression:

```text
*/5 * * * *
```

Use a separate Scheduler service account with permission to invoke the Cloud Run Job.

Do not enable `TEST_MODE` in the scheduled configuration.

## Monitoring

Cloud Run Job logs report successful executions, processed articles, and errors.

Example:

```text
Published 2 new articles (attempted 2)
```

Useful alerts include:

- Gemini HTTP 429 — quota exceeded
- Gemini HTTP 402 — billing or credit issue
- Article extraction failures
- Telegram API errors

Gemini usage and rate limits can be monitored in Google AI Studio.

## Limitations

- The RSS feed exposes only a limited number of recent articles. Extended outages can cause missed publications.
- Firestore leases reduce concurrent duplicate processing, but exactly-once Telegram delivery is not guaranteed.
- Article extraction depends on The Register's HTML structure.
- AI summaries may contain inaccuracies or omit important context.
- Gemini model availability and free-tier limits may change.
- Google Cloud services may incur charges. Billing alerts do not automatically stop spending.

## Security

Never commit API keys, Telegram tokens, service-account credentials, or `.env` files.

Use Google Secret Manager and least-privilege service accounts.

The Telegram bot only needs permission to publish messages in the destination channel.

## Live channel

See the bot in action:

**https://t.me/register_news**

## License

No license has been added yet. Until a license is selected, standard copyright restrictions apply.
## Live channel

See the bot in action:

**https://t.me/register_news**

## License









        README.md
Please move or remove them before you merge.
Aborting
Merge with strategy ort failed.
denis_beskov@cloudshell:~/register-news-go$ cd ~/register-news-go

mv README.md README.local.backup
denis_beskov@cloudshell:~/register-news-go$ git pull origin main --allow-unrelated-histories --no-rebase
From https://github.com/voxeb/rss-ai-telegram-bot
 * branch            main       -> FETCH_HEAD
Merge made by the 'ort' strategy.
 README.md | 1 +
 1 file changed, 1 insertion(+)
 create mode 100644 README.md
denis_beskov@cloudshell:~/register-news-go$ git push -u origin main
Enumerating objects: 10, done.
  GNU nano 7.2                        README.md *                               


















# RSS AI Telegram Bot

A lightweight Go application that automatically summarizes technology news from [The Register](https://www.theregister.com/) using Google's Gemini API and publishes concise English summaries to Telegram.

**Live demo:** [@register_news](https://t.me/register_news)

## Why I built this

Traditional RSS-to-Telegram bots usually offer either headlines with links or full article text. Neither is ideal for quickly following technology news.

This project takes a different approach: it extracts the full article, uses an LLM to identify the key facts and implications, and publishes a short summary.

Each Telegram post contains:

- Original article headline
- AI-generated summary (2–4 sentences, up to 80 words)
- Link to the original article

## Architecture

```text
The Register (RSS 2.0)
         |
         v
Google Cloud Scheduler
    (every 5 minutes)
         |
         v
Google Cloud Run Job
      (Go)
         |
         +--> Fetch RSS feed
         |
         +--> Extract article text
         |
         +--> Gemini 3.5 Flash-Lite
         |       |
         |       v
         |    AI summary
         |
         +--> Telegram Bot API
         |
         +--> Firestore
              (article status,
               processing leases)
```

The application runs as a scheduled batch job. No continuously running server or VPS is required.

## Tech stack

| Component | Technology |
|---|---|
| Application | Go |
| AI summarization | Gemini 3.5 Flash-Lite |
| Execution | Google Cloud Run Jobs |
| Scheduling | Cloud Scheduler |
| Persistence | Firestore |
| Secrets | Google Secret Manager |
| Monitoring | Google Cloud Logging |
| Publishing | Telegram Bot API |

## Features

- Checks for new articles every five minutes
- Parses RSS 2.0 and sorts articles chronologically
- Extracts article content from HTML
- Generates concise English summaries
- Tracks processed articles in Firestore
- Uses transactional processing leases to reduce duplicate processing
- Retries failed article processing
- Processes up to five articles per execution
- Supports a one-off test publication mode
- Uses Secret Manager instead of hardcoded API credentials

## Requirements

- Google Cloud project with billing enabled
- Cloud Run, Cloud Build, Artifact Registry, Cloud Scheduler, Firestore, and Secret Manager APIs enabled
- Firestore Native database named `(default)`
- Gemini API key with access to `gemini-3.5-flash-lite`
- Telegram bot with permission to publish in the target channel
- Google Cloud CLI
- Go for local development

## Build

Clone the repository:

```bash
git clone https://github.com/voxeb/rss-ai-telegram-bot.git
cd rss-ai-telegram-bot
```

Build:

```bash
go mod download
go build -o register-news-bot .
```

## Configuration

The application uses the following environment variables:

| Variable | Description |
|---|---|
| `GOOGLE_CLOUD_PROJECT` | Google Cloud project ID |
| `TELEGRAM_CHANNEL` | Telegram channel username |
| `TELEGRAM_TOKEN` | Telegram Bot API token |
| `GEMINI_API_KEY` | Gemini API key |
| `TEST_MODE` | Optional: `true` enables a one-off test post |

The RSS URL, article HTML selector, Gemini model endpoint, and summary prompt are currently configured in `main.go`.

The implementation is tailored to The Register rather than being a general-purpose RSS reader.

## Deploy to Google Cloud Run

Create two secrets in Secret Manager:

- `telegram-bot-token`
- `gemini-api-key`

Create a service account named `register-bot` and grant it:

- `roles/datastore.user` on the project
- `roles/secretmanager.secretAccessor` on the two secrets

Set your deployment variables:

```bash
PROJECT_ID="YOUR_PROJECT_ID"
REGION="europe-west1"
CHANNEL="@YOUR_CHANNEL"

gcloud config set project "$PROJECT_ID"
```

Deploy:

```bash
gcloud run jobs deploy register-news \
  --source . \
  --region="$REGION" \
  --service-account="register-bot@${PROJECT_ID}.iam.gserviceaccount.com" \
  --tasks=1 \
  --max-retries=0 \
  --task-timeout=180s \
  --memory=512Mi \
  --cpu=1 \
  --set-env-vars="GOOGLE_CLOUD_PROJECT=${PROJECT_ID},TELEGRAM_CHANNEL=${CHANNEL}" \
  --set-secrets="TELEGRAM_TOKEN=telegram-bot-token:latest,GEMINI_API_KEY=gemini-api-key:latest"
```

### Initialize Firestore

The first normal execution records existing RSS articles without publishing historical content:

```bash
gcloud run jobs execute register-news \
  --region="$REGION" \
  --wait
```

### Test publication

To publish one test message:

```bash
gcloud run jobs execute register-news \
  --region="$REGION" \
  --update-env-vars=TEST_MODE=true \
  --wait
```

Test mode sends a real Telegram message marked `[TEST]` without updating article publication history.

### Schedule automatic execution

Create a Cloud Scheduler trigger for the Job.

Cron expression:

```text
*/5 * * * *
```

Use a separate Scheduler service account with permission to invoke the Cloud Run Job.

Do not enable `TEST_MODE` in the scheduled configuration.

## Monitoring

Cloud Run Job logs report successful executions, processed articles, and errors.

Example:

```text
Published 2 new articles (attempted 2)
```

Useful alerts include:

- Gemini HTTP 429 — quota exceeded
- Gemini HTTP 402 — billing or credit issue
- Article extraction failures
- Telegram API errors

Gemini usage and rate limits can be monitored in Google AI Studio.

## Limitations

- The RSS feed exposes only a limited number of recent articles. Extended outages can cause missed publications.
- Firestore leases reduce concurrent duplicate processing, but exactly-once Telegram delivery is not guaranteed.
- Article extraction depends on The Register's HTML structure.
- AI summaries may contain inaccuracies or omit important context.
- Gemini model availability and free-tier limits may change.
- Google Cloud services may incur charges. Billing alerts do not automatically stop spending.

## Security

Never commit API keys, Telegram tokens, service-account credentials, or `.env` files.

Use Google Secret Manager and least-privilege service accounts.

The Telegram bot only needs permission to publish messages in the destination channel.

## Live channel

See the bot in action:

**https://t.me/register_news**

## License

No license has been added yet. Until a license is selected, standard copyright restrictions apply.
