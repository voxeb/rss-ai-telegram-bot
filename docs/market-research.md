# AI-enriched RSS: market research and product opportunities

Research date: **2026-10-30**. Status: **research and recommendations, not an approved roadmap**.

## Recommendation

**Keep the specialized Telegram bot as the supported product (A), test demand for an optional enriched RSS output (B), and defer a general publishing framework (C).** Run a small, separately approved comparison with real RSS readers before changing the application or production.

The most important feed finding is that **The Register currently supplies substantial article text in `content:encoded`, despite a very short `description`**. AI adds compression and a consistent reading format, rather than necessarily unlocking facts absent from RSS. The existing parser ignores the longer field. Conversely, a field named `content` can still contain only a teaser: Engadget and heise demonstrate this.

There is already direct competition for AI-enriched RSS output, especially RSSBrew and RSS-GPT. Commercial readers also automate article summaries; RSS.app combines AI briefs, XML output, and Telegram. A plausible opportunity is a small, understandable tool that generates one attributed summary and reuses it across destinations. **Demand, willingness to pay, superior accuracy, and lower total operating effort remain hypotheses.** This research establishes supply and technical constraints, not market size or product-market fit.

## Evidence and repository baseline

Labels used throughout:

- **Verified fact:** observed in downloaded XML, inspected repository files, or the linked first-party documentation. Product documentation verifies a documented capability, not its runtime quality.
- **Assessment:** interpretation of sampled content, including whether it conveys essential facts and whether it appears complete.
- **Hypothesis:** a user or market assumption that needs testing.
- **Recommendation:** proposed future action; no implementation or operational approval follows from this document.
- **Unresolved:** evidence or an owner decision is still missing.

Reviewed current `origin/main` at [f7c8a1bfb521285825b39fc42fa27ab40d48de50](https://github.com/voxeb/rss-ai-telegram-bot/tree/f7c8a1bfb521285825b39fc42fa27ab40d48de50): [AGENTS.md](../AGENTS.md), [README](../README.md), [architecture](architecture.md), [operations](operations.md), `main.go`, `feed.go`, and `queue.go`. No open PRs were returned before editing. This is a documentation-only investigation; production was not inspected or changed.

Relevant existing references are **A7/Q5** (specialization and scope expansion), **F1–F4** (ingestion, extraction, generation, publication), **A2–A5** (storage, identity, ordering, retries), and **L1–L7** (delivery, availability, cost, and accuracy limitations) in [architecture](architecture.md). They remain unchanged.

Verified repository behavior:

- One source, The Register, and one Telegram destination per execution; the feed URL, source-specific HTML selector, English prompt, and Gemini endpoint are in code. RSS categories supply a deterministic optional hashtag.
- RSS parsing reads `description` into `entry.Summary`, but generation uses separately fetched article HTML, not that field. `content:encoded` and Atom entries are not parsed.
- Firestore stores processing/publication state and article references. It does **not** retain generated summaries, article bodies, Telegram message IDs, or a complete durable discovered-item backlog.
- Claims, retries, and publication status are shared around a Telegram operation. A second output cannot safely be modeled by merely sending twice and reusing one `published` flag.
- Five claims per run, five attempts per article, and an input truncation of 24,000 bytes bound individual work. Failed delivery can repeat generation. Selection from the current feed can lose work after feed eviction.

The operations document contains dated deployment and cost evidence. This report does not refresh that evidence or infer current spending from published prices.

## Publisher feed inspection

### Method and limits

Downloaded public XML with `curl -L --fail --max-time 40` without authentication. The twelve successful feeds were retrieved on **2026-10-30, 20:50:07–20:50:47 UTC**. Inspected the **first five entries in document order per feed: 60 entries total**, without excluding deals, opinion, sponsored-looking material, or newsletters. This describes what a subscriber actually receives; it is not a representative random sample of technology journalism.

Parsed XML namespaces, decoded entities, stripped HTML and script/style/template text, inserted spaces at common block boundaries, collapsed whitespace, and counted whitespace-separated tokens. Counts are approximate words, not model tokens. They include visible captions, calls to action, and read-more text where present. The median and range describe only these five entries. Duplicate fields are counted separately, never added together.

**Essential facts** means the title plus the best available feed text identifies the event, actors, and an important consequence or constraint. “Yes” does not mean all context, quotations, or supporting evidence is present. These are researcher assessments, not a user comprehension study or independent verification of the news claims.

“Full text” below means apparently complete article prose in the sample, not a publisher guarantee. Checked two Register article pages and one each from 404 Media, TechRadar, and MIT Technology Review against their feed bodies: narrative endings matched, with site chrome, biographies, or promotional differences. A second 404 page was explicitly a paid-member preview. Ars article-page requests returned **403**; excerpt classification relies on feed truncation/read-more markers. VentureBeat's feed returned **429** and is excluded from the twelve successful inspections, not treated as empty. No access restriction was bypassed. No authenticated subscriber feed was inspected.

Namespace abbreviations: `atom` = `http://www.w3.org/2005/Atom`; `content` = `http://purl.org/rss/1.0/modules/content/`; `dc` = `http://purl.org/dc/elements/1.1/`. `media:content` belongs to Media RSS and is image/media metadata, not automatically article text. TechRadar's `dc:description` includes author metadata; it is not its article standfirst. The [RSS specification](https://www.rssboard.org/rss-specification) and [Atom specification](https://www.rfc-editor.org/rfc/rfc4287) define the formats, not the editorial completeness of any field.

### Fields, lengths, and usefulness

All rows were inspected **2026-10-30**. Lengths are **median [minimum–maximum] words**, n = 5 for every listed text field.

| Publisher / inspected feed | Actual format and text fields | Approximate lengths | Content assessment | Essential facts without opening? |
|---|---|---|---|---|
| [The Register](https://www.theregister.com/headlines.atom) | RSS 2.0: `description`; `content:encoded` | 10 [6–14]; 552 [489–654] | Witty teaser plus apparently full article text | Description: no. Encoded body: yes in checked examples. |
| [404 Media](https://www.404media.co/rss/) | RSS 2.0: `description`; `content:encoded` | 20 [9–32]; 935 [155–1,334] | Standfirst plus mixed full prose and member-only preview | Open article body: yes. Member preview: incomplete. |
| [Ars Technica](https://feeds.arstechnica.com/arstechnica/index) | RSS 2.0: `description`; `content:encoded` | 13 [10–14]; 170 [127–227] | Standfirst plus several opening paragraphs and read-more link | Partly: event and some specifics, remaining evidence/context absent. |
| [The Verge](https://www.theverge.com/rss/index.xml) | Atom: `atom:summary`; `atom:content` | 56 [56–56]; 112 [109–124] | Truncated lead; longer content also truncates mid-narrative | Partly: often the main event, incomplete qualifications. |
| [TechCrunch](https://techcrunch.com/feed/) | RSS 2.0: `description`; no encoded body in sample | 23 [17–36] | Short editorial standfirst, sometimes a teaser | Partly: title plus standfirst gives the topic/event, few details. |
| [WIRED](https://www.wired.com/feed/rss) | RSS 2.0: `description`; no encoded body in sample | 22 [19–25] | Short editorial standfirst/teaser | Partly: one useful fact, little context; shopping items also present. |
| [BleepingComputer](https://www.bleepingcomputer.com/feed/) | RSS 2.0: `description`; no encoded body in sample | 29 [19–47] | Factual editorial mini-summary, ending in `[...]` | Often the immediate incident; not enough for remediation or full scope. |
| [Engadget](https://www.engadget.com/rss.xml) | RSS 2.0: `description`; identical `content:encoded` | Both 15 [9–23] | Brief standfirst duplicated into a content field | Usually partial; the second field adds no facts. |
| [ZDNET](https://www.zdnet.com/news/rss.xml) | Atom: `atom:summary`; no Atom content in sample | 19 [13–25] | Short editorial standfirst/teaser | Partly: headline and one constraint, little supporting detail. |
| [TechRadar](https://www.techradar.com/rss) | RSS 2.0: `description`; identical prose in `dc:content` and `content:encoded` | 24 [16–28]; bodies both 565 [461–1,033] | Standfirst plus apparently full prose, including commerce/deals | Body: yes for checked report. Description alone: partial. |
| [heise online](https://www.heise.de/rss/heise-atom.xml) | Atom: identical `atom:summary` and `atom:content` | Both 22 [17–23], German | Short editorial standfirst duplicated into content | Partial: useful event/topic, not the underlying explanation or review. |
| [MIT Technology Review](https://www.technologyreview.com/feed/) | RSS 2.0: `description`; `content:encoded` | 55 [55–55]; 1,039 [752–1,190] | Truncated lead plus apparently full prose; includes newsletter roundups | Body: yes for checked article. Newsletter essentials span multiple stories. |

Redirects observed: The Register resolves to `https://api.theregister.com/api/v1/article?orderBy=published&site_id=2&remapper=rss&content=node` (still RSS, despite `.atom` in the public URL); ZDNET to `https://www.zdnet.com/rss/news/`; TechRadar to `https://www.techradar.com/feeds.xml`. Determine format from XML, not filename.

### Actual examples

Descriptions below **paraphrase actual sampled entries**, rather than reproducing publisher text. Numbers describe their source feed fields, not the length of these paraphrases. Linked articles identify the examples; the field contents were read from the XML above.

#### The Register

Native short-field opening (verbatim fragment): “$220 million? For all of you? Yeah, we can probably dig that” …

- [GM driver-data settlement](https://www.theregister.com/security/2026/10/30/gm-cracks-the-petty-cash-box-for-multistate-driver-tracking-settlement/5306721) — `description` 14; `content:encoded` 615 words. The teaser mentions the settlement amount sarcastically. The body explains the $220 million agreement, the participating jurisdictions, data sharing, and consent allegations.
- [Claude Code extension incident](https://www.theregister.com/cyber-crime/2026/10/30/anthropic-that-claude-code-extension-compromise-couldnt-be-us/5306682) — `description` 10; `content:encoded` 489 words. The short standfirst points at a supply-chain attack; the body distinguishes Cursor marketplace access from an alleged breach of Anthropic and describes the response.

#### 404 Media

Native short-field opening (verbatim fragment): “Writer and readers share what they're feeling and doing in the aftermath” …

- [AO3 outage reactions](https://www.404media.co/ao3-down-outage-reactions/) — `description` 20; `content:encoded` 1334 words. The description signals community reactions. The long body supplies the security-incident context, expected downtime, and accounts from writers/readers.
- [Behind the Blog: Tech Support](https://www.404media.co/behind-the-blog-tech-support/) — `description` 9; `content:encoded` 155 words. The description only identifies the weekly topics. The body is a 155-word opening section; the article page explicitly gates the rest for paid members.

#### Ars Technica

Native short-field opening (verbatim fragment): “A meteorite found in the Sahara is a rare piece of Mars'” …

- [Ancient Martian meteorite](https://arstechnica.com/science/2026/10/rare-meteorite-from-mars-gives-us-a-view-into-its-ancient-crust/) — `description` 14; `content:encoded` 227 words. The description gives the discovery topic. The excerpt identifies Teghaza 001, its estimated age, and a sampling explanation, then sends readers to the full article.
- [Fujifilm tape expansion](https://arstechnica.com/gadgets/2026/10/fujifilm-to-boost-tape-production-5x-as-ai-fuels-archiving-demand/) — `description` 12; `content:encoded` 127 words. The description explains tape’s continuing utility. The excerpt adds the roughly ¥10 billion investment, fivefold capacity target, and expansion period.

#### The Verge

Native short-field opening (verbatim fragment): “Asked about supply chain shortages, like the ongoing RAM shortage, OpenAI CEO” …

- [RAM allocation remarks](https://www.theverge.com/ai-artificial-intelligence/1023820/sam-altman-ram-shortage-cloud-servers) — `atom:summary` 56; `atom:content` 121 words. The 56-word summary begins an interview account; the 121-word content cuts through a quotation. It conveys the stance but is not a completed editorial synthesis.
- [Shared AI scraping opt-out](https://www.theverge.com/tech/1023763/openai-google-microsoft-ai-scraping-standard-opt-out) — `atom:summary` 56; `atom:content` 112 words. The summary names participating AI firms. Longer content explains the proposed policy signal and its robots.txt location before truncating.

#### TechCrunch

Native short-field opening (verbatim fragment): “Like a real-life episode of "Suits," the demise of famed white-shoe Wall” …

- [Crosby and Weil Gotshal](https://techcrunch.com/2026/10/30/ai-law-firm-crosby-bids-on-disintegrating-famed-firm-weil-gotshal/) — `description` 36 words. The standfirst frames turmoil at a law firm and an AI bidder; most transaction detail remains outside the feed.
- [Satlyt satellite computing test](https://techcrunch.com/2026/10/30/satlyt-takes-another-step-toward-turning-satellites-into-space-data-centers/) — `description` 20 words. The standfirst reports a successful test toward spacecraft-based data centers, without the technical test results.

#### WIRED

Native short-field opening (verbatim fragment): “Thanks to a lucky break, flawed code in a tainted Claude extension” …

- [Tainted Claude extension](https://www.wired.com/story/trojan-malware-targets-thousands-of-anthropic-claude-developer-systems/) — `description` 25 words. The standfirst gives a roughly 5,000-machine exposure and an implementation flaw limiting the attack. It does not explain the full chain.
- [AO3 outage coping guides](https://www.wired.com/story/people-are-creating-survival-guides-to-cope-with-the-ao3-apocalypse/) — `description` 19 words. The standfirst identifies a security incident and potentially weeks of downtime; reactions and underlying evidence remain in the article.

#### BleepingComputer

Native short-field opening (verbatim fragment): “Hardware wallet device maker Trezor has received credible reports about a suspected” …

- [Trezor implant warning](https://www.bleepingcomputer.com/news/security/trezor-warns-of-suspected-crypto-theft-implant-in-hardware-wallet/) — `description` 29 words. The description names the suspected implant, Safe 5 model, and reseller/region: a useful compact incident summary.
- [Malicious extension in Cursor](https://www.bleepingcomputer.com/news/security/malicious-claude-code-extension-on-cursor-planted-ssh-backdoors/) — `description` 37 words. The description adds the roughly 11-hour distribution window and up to 5,000 exposed developers, but omits full response instructions.

#### Engadget

Native short-field opening (verbatim fragment): “Snap is now the only defendant in the case.” …

- [Meta settlement](https://www.engadget.com/2302956/meta-settles-another-social-media-addiction-case-just-before-trial/) — `description` 9; `content:encoded` 9 words. The headline identifies the settlement; both body fields add only that Snap remains the sole defendant, in nine words.
- [Bluesky moderation inbox](https://www.engadget.com/2302932/bluesky-now-has-a-dedicated-moderation-inbox/) — `description` 15; `content:encoded` 15 words. Both fields describe a beta feature for following reports and account strikes. Neither gives an extended explanation.

#### ZDNET

Native short-field opening (verbatim fragment): “Only a quarter of Android phones have even made it to Android” …

- [Android adoption](https://www.zdnet.com/tech/android-17-os-adoption-numbers-october-2026/) — `atom:summary` 13 words. The headline supplies the Android 17 adoption figure; the 13-word summary compares Android 16 adoption, without explaining update availability.
- [Galaxy sleep tracking](https://www.zdnet.com/health/advanced-sleep-apnea-tracking-galaxy-watches/) — `atom:summary` 19 words. The 19-word summary names initial supported devices and FDA approval; detailed rollout and functionality are absent.

#### TechRadar

Native short-field opening (verbatim fragment): “But its entry-level GPU is two generations old, and its AI pitch” …

- [DT Research rugged laptop](https://www.techradar.com/pro/us-police-forces-could-soon-get-a-major-tech-upgrade-thanks-to-dt-researchs-new-rugged-laptops-which-pack-in-nvidia-graphics-and-ai-processing) — `dc:content` 1033; `description` 16; `content:encoded` 1033 words. The short description critiques the GPU/AI pitch. The body includes configurations, expected availability, and deployment constraints.
- [Dell laptop offer](https://www.techradar.com/pro/dells-15-touchscreen-laptop-with-1tb-of-storage-is-the-office-machine-id-recommend-to-remote-workers-especially-now-its-usd300-off-at-best-buy) — `dc:content` 461; `description` 28; `content:encoded` 461 words. The description lists main specifications; full prose adds the price/deal and buying rationale. This is commerce content, not solely a news event.

#### heise online

Native short-field opening (verbatim fragment): “GPS-Tracker helfen beim Aufspüren entlaufener Haustiere und kosten zwischen 22 und 90” …

- [Pet GPS tracker comparison](https://www.heise.de/bestenlisten/testsieger/top-10-der-beste-gps-tracker-fuer-hund-und-katze-im-test-viel-besser-als-airtag/mqeplqh?wt_mc=rss.red.ho.ho.atom.beitrag.beitrag) — `atom:summary` 23; `atom:content` 23 words. Both German fields give the price range and comparison topic, but no ranking or test findings.
- [Digital euro service suppliers](https://www.heise.de/news/Zentralbank-stellt-die-Weichen-fuer-den-digitalen-Euro-11506763.html?wt_mc=rss.red.ho.ho.atom.beitrag.beitrag) — `atom:summary` 19; `atom:content` 19 words. Both fields identify telecom providers involved in connectivity consortia; the longer project explanation is absent.

#### MIT Technology Review

Native short-field opening (verbatim fragment): “GLP-1 weight-loss drugs are one of the biggest biotech breakthroughs of the” …

- [The Download roundup](https://www.technologyreview.com/2026/10/30/1147843/the-download-glp-1s-eating-disorders-forever-chemicals/) — `description` 55; `content:encoded` 1039 words. The 55-word lead starts the first topic. Encoded text includes a multi-topic newsletter with human-written story summaries and links.
- [GLP-1s and eating disorders](https://www.technologyreview.com/2026/10/30/1147838/eating-disorders-rise-glp-1s-weight-loss-drugs/) — `description` 55; `content:encoded` 1190 words. The lead stops mid-narrative. The long body includes research context, clinical concerns, and qualifications; it is article text, not a concise summary.

### What this establishes

**Verified:** the available input is heterogeneous. Some feeds already carry much more text than the default description; others duplicate a teaser under a more promising tag. **Assessment:** BleepingComputer’s compact editorial descriptions can already serve rapid incident awareness, while long Register/TechRadar/MIT bodies need compression for a short briefing. The MIT roundup also shows that human editorial summaries are an existing alternative.

**Hypothesis:** a consistent, faithful 40–80-word summary will help readers who find a standfirst too thin and a full body too long. Rewriting a 15-word teaser alone cannot recover missing facts; any such output must be labeled as based on limited source material. Full article input, where permitted and available, is needed to test the stronger promise.

The current Register feed could reduce dependence on HTML extraction in a future design, but completeness across article types, long-term feed stability, and permission must be checked first. This observation does not approve a parser or extractor change.

## Competitor analysis

Checked **2026-10-30** against first-party documentation and GitHub. Six products cover three commercial services and three open-source alternatives. No paid account, end-to-end trial, or performance benchmark was used. **ND** means not documented in the inspected sources, not proven impossible. OPML subscription export, JSON APIs, ordinary RSS forwarding, and AI-enriched RSS are different capabilities.

### Capabilities

| Product | Automatic per-article summary | Telegram publishing | Output RSS containing AI text | Self-hosting |
|---|---|---|---|---|
| Inoreader | Yes: rule action creates summaries for matching new articles | External automation via output feed/API; native Telegram publishing ND. Telegram feed ingestion is a separate feature | Ordinary output feeds verified; AI-summary inclusion in those public feeds **unverified**. API explicitly exposes Intelligence summaries | Hosted service; self-hosted server ND |
| Readwise Reader | Yes for manually saved documents; automatic Feed summaries require the user's OpenAI API key | ND; a custom integration would need separate verification | ND; documented exports include OPML, CSV, article files, and Markdown summaries, not a continuous enriched RSS endpoint | Hosted service; self-hosted server ND |
| RSS.app | Scheduled **multi-article AI Brief** verified; one automatic summary per incoming article unverified | Yes, native bot; AI Brief documentation includes forwarding to Telegram | Yes for generated AI Briefs in XML; do not equate a brief with per-article enrichment | Hosted service; self-hosted server ND |
| RSSBrew | Yes: prepends brief/longer article summaries; also daily/weekly digests | Native output ND; its Telegram discussion group is not publishing support | Yes: processed feeds with summaries and optional digests | Yes, Docker deployment and database |
| RSS-GPT | Yes: scheduled script attaches summaries to feed articles | Native output ND | Yes: generated XML hosted through GitHub Pages/raw files | Yes, operator-controlled fork/workflow; custom hosting possible, not a packaged server |
| Ember | Yes: summarize every article, or selected starred/saved/board articles | ND | ND; documented JSON/Fever API and OPML export do not establish enriched RSS output | Yes, Go application, SQLite, optional local Ollama |

### Costs, licensing, and customization

USD list prices observed on the research date; annual billing prices are monthly equivalents, not a month-to-month commitment. Taxes, payment availability, usage add-ons, and operator time are excluded. No service was purchased.

| Product | Published cost / license | Customization and meaningful constraint | Primary evidence |
|---|---|---|---|
| Inoreader | Pro $7.50/month billed annually or $9.99 monthly; proprietary hosted service. January documentation states 1 million included AI tokens/month; confirm present allowance at checkout | Rules, prompts, language/persona, output feeds. April documentation allows provider choice; own keys/model selection require an Intelligence add-on or Team Intelligence, superseding the older unrestricted-key description | [Pricing](https://www.inoreader.com/pricing), [automatic summary rules](https://www.inoreader.com/blog/2025/10/introducing-new-rule-triggers-and-actions-translations-summaries-and-more.html), [AI allowance](https://www.inoreader.com/blog/2026/01/get-insights-with-inoreader-intelligence.html), [April change](https://www.inoreader.com/blog/2026/04/choose-your-ai-platform-for-intelligence-features.html), [output feeds](https://www.inoreader.com/features/outgoing-feeds), [API summaries](https://www.inoreader.com/developers/stream-contents) |
| Readwise Reader | Reader-only $7.99/month billed yearly; Full Readwise $9.99/month billed yearly; proprietary service. Feed auto-summaries also consume the user's API credits | Editable Ghostreader prompts; reading/highlight workflows. A reader subscription is not a ready-made publishing service | [Pricing](https://readwise.io/pricing), [Ghostreader / Feed key requirement](https://docs.readwise.io/reader/docs/faqs/ghostreader), [export formats](https://docs.readwise.io/reader/docs/faqs/exporting) |
| RSS.app | Basic $8.32, Developer $16.64, Pro $83.32/month equivalents with annual billing; proprietary service; paid plans include 5 AI credits/month | Model, articles per brief, scheduling, filters, and output formatting. Credits per run vary; additional-credit unit prices and realistic monthly brief volume were not established | [Pricing](https://rss.app/en/pricing), [AI Brief](https://help.rss.app/en/articles/14692230-how-to-generate-ai-summaries-for-rss-automatically), [credit accounting](https://help.rss.app/en/articles/14691648-guide-to-ai-credits), [Telegram bot](https://help.rss.app/en/articles/8880712-how-to-add-rss-feeds-to-telegram) |
| RSSBrew | AGPL-3.0; no software subscription fee documented; operator pays hosting, database, and chosen model usage | UI, nested filters, aggregate/tag feeds, custom prompts, OpenAI-compatible model endpoints. Public README says development/documentation are ongoing; runtime quality untested | [README at inspected revision](https://github.com/yinan-c/RSSbrew/blob/0ccb5399d6ffb080d8c41db2ec5276861830dbd6/README.md), [license](https://github.com/yinan-c/RSSbrew/blob/0ccb5399d6ffb080d8c41db2ec5276861830dbd6/LICENSE), [product documentation](https://rssbrew.com/) |
| RSS-GPT | MIT; no software subscription fee; operator pays API and any applicable Actions/hosting charges | Configuration for sources, merging, filters, language and summary length; prompt/script changes possible. README says upstream feed updates are suspended and points to RSSBrew; this is not proof the fork workflow is unusable | [README at inspected revision](https://github.com/yinan-c/RSS-GPT/blob/4e38db3ad7af51185a0bd25c0c3df0aaca4a5b5f/README.md), [license](https://github.com/yinan-c/RSS-GPT/blob/4e38db3ad7af51185a0bd25c0c3df0aaca4a5b5f/LICENSE), [configuration guide](https://yinan.me/rss-gpt-manual-en.html) |
| Ember | MIT; no software subscription fee; local inference uses hardware/electricity, or hosted inference incurs provider charges | Model/backend controls, article selection, filtering, and source changes. Small local-model quality and actual operating resource needs were not benchmarked | [README at inspected revision](https://github.com/brandonhon/ember/blob/5e6113d26501e03745cdfe9d650bf5a20915b1f9/README.md), [license](https://github.com/brandonhon/ember/blob/5e6113d26501e03745cdfe9d650bf5a20915b1f9/LICENSE) |

**Assessment:** RSSBrew is the closest documented substitute for option C and much of B. RSS-GPT demonstrates an established batch-to-static-feed approach. Inoreader and Reader reduce the need for a separate tool for personal reading, even where publishing needs remain unmet. RSS.app already covers multichannel delivery for briefs. The opportunity is therefore a workflow preference and execution-quality hypothesis, not an unoccupied feature category.

**Unresolved competitor checks:** does Inoreader's public output include generated summaries with usable attribution? Can RSS.app automatically produce one item per article at a viable credit cost? Do either satisfy the maintainer's real workflow without custom code? Documentation alone does not answer these; a small authorized trial should precede a build-versus-adopt decision.

## Product options

All future behavior in this section is proposed. Complexity estimates are relative engineering judgments, not delivery commitments or approved architecture.

| Dimension | A. Specialized Telegram bot | B. Optional enriched RSS beside Telegram | C. Reusable RSS-to-AI publishing tool |
|---|---|---|---|
| User value | Existing compact English stream for a known channel | Same summaries in existing readers, searchable/read-later workflows and downstream tools; supports people who prefer RSS | Configurable sources, prompts, providers and destinations for multiple operators |
| Complexity | Lowest incremental scope; maintain extractor and delivery reliability | Moderate: persist summaries, serialize safe XML, serve a stable URL, separate destination outcomes and support retention/corrections | Highest: RSS/Atom normalization, multiple extraction strategies, source configuration, validation, packaging, upgrades and support |
| AI and infrastructure costs | Current generation and retries plus cloud baseline | Near-zero extra inference **only if the same stored summary is reused**; adds storage, hosting/egress and feed builds/reads | More sources and variants increase inference and retrieval; quotas/fairness become operator concerns |
| Reliability | Known feed-eviction and ambiguous Telegram-delivery risks persist | Stable GUIDs, atomic feed replacement, last-good snapshot, independent retries; RSS must remain useful during a Telegram outage | Per-source health, durable discovery, deduplication across feeds, independent sink status, bounded failure handling and SSRF controls for configurable URLs |
| Storage | Current state/reference records; no summary archive | Durable summary and provenance, original dates/IDs, output status; bounded feed window and correction/deletion history | Wider backlog, provider/prompt versions, per-destination state, retention policies, migrations/backups |
| Attribution and legal exposure | Linked AI summaries already require appropriate source handling | A reusable public endpoint increases syndication/republication reach; clearly identify publisher and generated text | Different publisher terms and jurisdictions multiply review work; source permissions cannot be generalized from one feed |
| Product risk | Limited audience/channel dependency | Demand may be too small, or native readers may already suffice | Competes directly with maintained tools and risks becoming infrastructure/support work without users |

### What B actually requires

The endpoint is the easy part; the reusable publication record is the material change. A possible minimal record would include original article ID/URL, publisher, title, source publication time, generated summary, generation time/model/prompt version, and separate Telegram/feed outcomes. These are conceptual needs, **not a selected schema**.

Generate once and reuse the same approved text. RSS failure should retry feed construction without calling the model again; Telegram retries should not create fresh feed IDs or new summaries. Keep the last valid feed when generation or publication fails. Decide independently whether an article can appear in RSS before Telegram succeeds. The current single `published` state cannot express that policy.

A static feed snapshot built by a batch worker is a candidate worth comparing with a live endpoint; it is not an architectural decision. Consider stable item GUIDs, original publication dates, explicit correction timestamps, XML/HTML escaping, a canonical source link, conditional HTTP caching, item limits, and atomic replacement. Polling clients can create duplicates if IDs change after an edit. Feed caching and client polling introduce delay even after successful generation. No subscriber should trigger a new AI call.

### Cost and storage scenarios

**Verified price input:** [Gemini 3.5 Flash-Lite standard pricing](https://ai.google.dev/gemini-api/docs/pricing#gemini-3.5-flash-lite), checked 2026-10-30, lists **$0.30/million input tokens and $2.50/million output tokens**, including thinking tokens. Free-tier availability does not establish the runtime key's tier, allowance, or actual bill.

**Illustrative assumptions:** 3,000 total input tokens and 150 billable output tokens per successful article; 30 days/month; one generation, with no cache/grounding/tools. This is a planning scenario, not observed token consumption. Byte limits cannot be converted directly into these token assumptions.

`monthly inference = articles/day × 30 × attempts/article × (3,000 × $0.30 + 150 × $2.50) / 1,000,000`

| Scenario | Articles/month | Inference at one generation | At five equally sized generations/article |
|---|---:|---:|---:|
| 50 articles/day | 1,500 | $1.91 | $9.56 |
| 500 articles/day | 15,000 | $19.13 | $95.63 |

Five generations is a retry sensitivity scenario, not a claim that every attempt reaches Gemini. Actual output/reasoning tokens, source sizes, failures and pricing may differ. Cloud Run, Scheduler, Firestore operations/indexes, logs, build artifacts, feed hosting, egress, taxes and human support are additional. B adds no generation cost only when reuse works; per-subscriber personalization or regeneration changes the economics.

**Storage assumptions:** 2 KiB per compact summary/provenance record, 20 KiB total per record if retaining a source body as well, 30-day retention, 50 articles/day. That is roughly **2.9 MiB** or **29.3 MiB** respectively; at 500/day it is **29.3 MiB** or **293 MiB**, before indexes, backups, versions and database overhead. A 100-item feed at 2 KiB/item is about **200 KiB** before XML overhead. With 100 readers polling hourly, a full uncached response would transfer roughly **13.7 GiB/month**; shared caches and HTTP 304 responses can reduce that. These figures describe capacity, not hosting charges or measured demand. A small storage footprint does not remove the need to choose retention and deletion behavior.

## Attribution, rights, and source handling

**Verified:** [The Register's terms, section 2](https://www.theregister.com/profile/terms_and_conditions_of_use) describe personal, noncommercial website use and restrict copying/distribution without prior written authorization. Public feed availability does not itself establish permission for an AI syndication service. Applicability to a specific feed workflow, existing permission, and enforceability were not resolved here. This concerns A as well as B/C; wider distribution increases the practical exposure.

**Verified legal context:** the [U.S. Copyright Office](https://www.copyright.gov/fair-use/) explains that fair use depends on the facts and legal application of a particular use. It is not a worldwide permission or a fixed word-count allowance. No jurisdiction-specific legal conclusion is made here. Publisher terms beyond the cited Register terms have not been comprehensively reviewed.

**Recommendations for any future experiment or product:**

- Display publisher, original headline/link and original publication time, with an explicit AI-generated-summary label. Preserve author and source categories when available; do not imply publisher endorsement. Attribution is necessary context, not a substitute for permission.
- Publish concise original summaries rather than copied full bodies or substantial excerpts. Do not redistribute subscriber-only material or bypass paywalls. A body being exposed in a feed does not prove redistribution rights.
- Establish a source allowlist and rights basis before sharing a pilot. Check source terms and crawling policies, use modest retrieval rates, and retain only the source material needed for the agreed evaluation/retention period.
- Preserve uncertainty and attribution for allegations, security incidents and opinion. Do not invent implications from a teaser. Treat source text as untrusted input to the model and exclude unrelated instructions/ads from generation.
- Provide a correction/removal contact and a way to withdraw an item. Feed caches and downstream copies can outlive deletion. Review model-provider content handling and the applicable account tier before sending article text.

**Unresolved:** publisher permission/licensing, operating jurisdiction, commercial intent, treatment of quoted material/images, source retention, corrections, and deletion propagation. These require decisions before public or commercial expansion. This research does not contact publishers or authorize publication.

## Minimal experiment and decision gates

**Recommendation:** start with a **two-week concierge pilot for five consenting people who already use RSS readers**, limited to The Register and English. First compare an existing enriched-feed tool (prefer RSSBrew for evaluation) with a minimal candidate workflow. Do not build a general framework. The pilot below requires a new behavior Issue and explicit authorization for paid generation, external hosting/sharing, or messages; none was run for this research.

1. **Preflight:** record who wants RSS, which two reader clients they actually use, an agreed source-rights basis, retention window, and a hard experiment budget. Proposed ceilings: **$10 incremental service/API spend and four hours of setup work**, with stop-on-limit rather than an alert alone. If permission or five participants cannot be obtained, stop and keep A; do not assume demand.
2. **Build-versus-adopt check:** spend at most the first setup hour testing the documented RSSBrew path with permission-cleared inputs. Verify per-article enrichment, canonical links, stable IDs, and output readability. If it satisfies the pilot, use it to test demand before proposing changes to this repository. Do not add the same source to production or change Telegram publishing.
3. **Quality comparison:** select **30 consecutive eligible articles** during the pilot; log every excluded/restricted/unavailable item and its reason. Produce candidate 40–80-word summaries in an isolated evaluation environment. Compare native title/description, native best-content field, and AI summary. Each participant reads 12 different articles, four per condition; rotate article assignments and condition order across participants to reduce recall and order effects. This yields 60 reading tasks, scored against three facts each (180 answers). Establish three key facts per article from the permitted full source, reviewed by a human before scoring. Use the same record across both reader clients; do not regenerate per participant.
4. **RSS workflow:** after rights/hosting approval, share a bounded pilot feed of at most 100 summary-only entries, retaining evaluation records for 30 days and excluding full source bodies from the public endpoint. Test duplicate rebuilds, one injected generation failure, and one interrupted feed update in an isolated harness. Keep production Telegram untouched. Collect opt-in feedback rather than interpreting raw HTTP requests as people.

Proposed pass criteria must be agreed before the pilot starts:

| Question | Measure and pass threshold | Interpretation |
|---|---|---|
| Are summaries faithful? | Human review of all 30: **zero material factual errors** (wrong actor, event, number, certainty, or critical omission that changes meaning); at least **27/30** contain the three predefined essential facts | A material error triggers correction and review; a new evaluation is needed after changing the prompt |
| Does AI add useful compression? | On balanced reading tasks, median time to identify the essential facts is **at least 25% lower than best native feed content**, with fact-answer accuracy no more than **5 percentage points lower** | Measure against full feed content, not only the weakest teaser |
| Is it better than the teaser? | Fact-answer accuracy is **at least 10 percentage points higher** than title/description alone | Prevents optimizing for brevity without improved understanding |
| Do people choose RSS? | At least **3/5** use the feed on three distinct days in week two and explicitly ask to keep it; at least **2/5** prefer it to their existing reader-summary workflow | Self-reported dated use is acceptable for this small pilot; polling counts are not engagement |
| Does delivery behave predictably? | Both selected clients parse all valid items; **zero duplicate IDs on rebuild/restart**, last-good feed survives injected failure; at least **95%** of eligible pilot items appear within **15 minutes of discovery**, with measured client delay reported separately | Failures are counted, including exclusions from output; not a production SLA |
| Is the extra output inexpensive? | Actual incremental spend **≤ $10**; at most **one successful generation per article** reused across readers; setup **≤ four hours** and ongoing operation **≤ 30 minutes/week** | Account for provider and hosting costs, failures and human review time separately |

Small-sample results are directional, not statistically representative. Record per-condition task times, accuracy denominators, exclusions, costs and operational failures in the pilot report. Do not claim a pass if the article count or participant count is not reached.

**Decision:** retain A if the pilot fails to recruit, quality is inadequate, permissions remain unresolved, or users prefer existing tools. Consider a separately scoped B Issue only if quality, value, reliability and cost gates all pass **and** a specific gap justifies adding this repository's output rather than using RSSBrew. Investigate C only after at least **three independent operators** request multiple sources/destinations and show why existing tools are insufficient. Any later Issue must revisit Q2–Q4 (recovery, ambiguous delivery, budget), not only Q5.

## Questions left open

1. Is the maintainer optimizing personal reading, a public news channel, reusable open-source software, or paid publishing? Who specifically wants output RSS?
2. What use and redistribution rights cover each source and the proposed audience? Which jurisdiction and provider data-handling terms apply?
3. Is a 40–80-word English summary materially better than the full Register feed or native reader summaries, and for which article types?
4. Can an existing tool satisfy the real workflow? Inoreader summary export and RSS.app per-article mode/cost still need direct verification.
5. Should RSS publication be independent of Telegram success? What are the retention, correction, deletion, freshness and duplicate policies?
6. What are the runtime Gemini tier, actual token usage, total cloud bill, and acceptable recurring budget? The existing operations evidence does not settle current values.
7. Which feeds and languages would a second operator need? A single-day twelve-publisher inspection cannot establish future extraction reliability or demand.

## Inspection audit notes

The following compact manifest records the downloaded XML hashes and the first-five word counts in feed order. It helps distinguish this observation from a later changing feed. Raw publisher XML/full articles were used temporarily for inspection and are not committed; hashes alone cannot reconstruct them. The linked feed URLs and dated examples above remain the public references.

| Publisher | SHA-256 of inspected XML | First-five counts: short field / best body, when separate |
|---|---|---|
| The Register | `c8f26d6e96d2b62a407050cc93b8d60e46629b026a1a5e2f052f8c26cab78396` | 14, 10, 6, 9, 13 / 615, 489, 505, 654, 552 |
| 404 Media | `cd9654a0c2a77199dd878a5c329a99ccf325b9e338163f9201f41de39e63aac1` | 20, 28, 32, 9, 15 / 1334, 1063, 935, 155, 784 |
| Ars Technica | `3d5ffe350aa0f7fa91e82b0fb3f3b16ca60c4e33189fa4cd328f9d6d5dbb4662` | 14, 12, 13, 10, 14 / 227, 127, 170, 154, 173 |
| The Verge | `cbbe8b55f312ca0a610400d4f2e233967ad4ccf5d2f867fdb412fc76e84ed299` | 56, 56, 56, 56, 56 / 121, 112, 111, 109, 124 |
| TechCrunch | `c429b490b2b6da8600813a0292ae0854b91dbdb27f4303de39edaae54643e7a8` | 36, 20, 23, 17, 27 |
| WIRED | `2583cc27adc39e6248616075d8c636f93d4f8730c4dd2101eafb3959c79b8360` | 25, 19, 20, 22, 25 |
| BleepingComputer | `28bd3a492938abadc607c6b0cffbd4e8779c5ac0d4e85bc32e0579cceb1d1357` | 29, 37, 47, 23, 19 |
| Engadget | `d0160bf081f63077b698f60762d35f7cfe3c2874658136d88810c22e4848b302` | 9, 15, 13, 23, 17 / 9, 15, 13, 23, 17 |
| ZDNET | `d27c2d6267999a3c52de0217c6863dbff0e5f3f478c79c98d1ff322750f9c704` | 13, 19, 25, 22, 14 |
| TechRadar | `1a90626620ba02d5287f6735c4033f12834f10c298ad30161dae23a2c8e0a12c` | 16, 28, 27, 24, 17 / 1033, 461, 565, 461, 567 |
| heise online | `dfe021ac39101c36dbdb48c746aa5bb2cc354b46d129215acc92390e692e8279` | 23, 19, 23, 17, 22 / 23, 19, 23, 17, 22 |
| MIT Technology Review | `6a18b836f31e6e55f1eec5122912035c5c2d4eba4857000c33813ab2cfe332f4` | 55, 55, 55, 55, 55 / 1039, 1190, 1025, 1188, 752 |

Validation for this document: checked source links and their stated limitations, XML field names, sample counts/ranges, arithmetic, repository terminology, and documentation-only diff. Application tests are not required by AGENTS.md for documentation-only work and were not run locally. Recommendations have not been implemented, trialed, merged, or deployed.
