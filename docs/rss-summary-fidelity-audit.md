# Source-fidelity audit: 30 pilot summaries

**Needs revision for unattended publication. Review coverage: complete for the 30 original summaries against the saved source texts.** This is an AI-assisted source comparison, not independent verification of the news or human editorial sign-off.

## Finding and decision

This full source comparison supersedes the pilot setup record’s preliminary two-case spot-check; that setup record describes the earlier state.

**4 of 30 original summaries (13.3%) contain at least one material fidelity problem.** The four are S02, S03, S14 and S26. A summary is counted once even if it has multiple problems. Thirteen other summaries have only minor precision/attribution issues; thirteen have no issue identified in this review. These are review classifications, not a production-wide accuracy estimate.

The maintainer reported that summaries are substantially more useful than the publisher's short descriptions for personal reading. That supports the format's value for this reader. The fidelity results support keeping editorial review before unattended distribution while the generation method is improved. They do not approve a new output architecture or establish the error rate of the existing Telegram bot.

All four materially affected summaries met the requested 40–80-word length target. A length check would not have caught these problems. Across all 30, raw lengths were 57–83 words (median 70); 29 met the length target.

## Scope and method

- **Population and unit:** all 30 articles in the fixed local Register pilot sample; one original model output per article. No exclusions or replacement outputs. The visible publisher headline is context for judging omissions; it does not excuse a contradictory claim in the summary.
- **Reference:** the saved publisher RSS `content:encoded` bodies, with their headlines and canonical links. The source XML hash is `bc7a690245788ab9f70cd7feb2895d8da371bdbb7182e2c393ad69e55fff54ca`. No fresh article version or outside knowledge replaced that snapshot.
- **Original outputs:** `quality.json`'s `raw_summary`, before displayed editorial corrections. All 30 were reconciled with the local SQLite model-output records; no summary mismatches or missing rows were found. Plain article text in SQLite matched the saved RSS for all 30. This checks lineage, not truth.
- **Configuration:** Gemini 3.5 Flash-Lite through the RSSBrew pilot, temperature 0.2, 400-output-token ceiling, a custom 40–80-word prompt and the local Gemini message adapter. The production bot uses a different prompt, article-extraction path and output limit.
- **Review:** two independent AI reviewer blocks covered S01–S10 and S11–S20; the lead covered S21–S30 and directly reconciled the evidence for every material finding. All source bodies and original summaries were read. The reviewers were not blinded; there was no human adjudication, inter-rater agreement study or external fact-check.
- **Material error:** a wrong or unsupported central fact, unjustified source/causal certainty, or omission that changes reasonable understanding or action. Optional context, supporting detail and stylistic compression are not errors merely because they are absent.
- **Minor issue:** a narrower attribution, procedural status, scope or precision problem that leaves the central event intact. Borderline cases are retained as minor rather than promoted to material.

The source is the comparison reference, not certified ground truth. A summary can faithfully repeat inaccurate reporting. One consecutive sample from one publisher/date/configuration cannot estimate performance across all news, sources, prompts, models or production runs. No confidence interval or overall factual-accuracy score is inferred.

Three diagnostic source facts were retained per article in local review notes. They were selected after generation, so this audit does not claim to implement a pre-registered three-fact comprehension test or the full multi-user experiment proposed in [market research](market-research.md).

## Material findings and verified corrections

### S02 — persistent infection warning lost

[Cursor extension incident](https://www.theregister.com/cyber-crime/2026/10/30/anthropic-that-claude-code-extension-compromise-couldnt-be-us/5306682). The summary emphasizes failed tunnels and marketplace removal. The saved source explicitly says local persistence can remain and updating the extension alone does not remove it. Omitting this changes the reasonable response of an affected developer. The source separately attributes marketplace removal to Cursor.

**Fixed in the local reading export:** the rewritten summary retains exfiltration and explicitly states that an update is insufficient and cleanup/investigation guidance is needed. Original output and its audit classification remain unchanged.

This omission classification involves editorial judgment. If the metric counted only affirmative wrong/unsupported claims and excluded critical omissions, the incidence would be **3/30 (10.0%)**. The primary 4/30 definition includes action-changing omissions.

### S03 — distance and causal attribution changed

[Voyager communications](https://www.theregister.com/science/2026/10/30/voyager-1-gives-controllers-the-silent-treatment/5306637). The approaching milestone is one light-day of distance, not power. The original output also attributes a likely safe-mode explanation to the mission manager, while the source presents an unconfirmed possibility and quotes the manager wanting to learn what happened.

**Fixed in the local reading export:** distance and signal latency are explicit; the cause remains unresolved and the manager's comment is kept separate. This replaces the earlier partial correction, which addressed only the milestone.

### S14 — financial quantity substituted

[AWS growth forecast](https://www.theregister.com/off-prem/2026/10/30/aws-installs-an-ai-accelerator-every-14-seconds-and-eyes-1-trillion-annual-revenue/5306547). The central forecast concerns annual revenue. The original instead forecasts company valuation, a different quantity. Its assertion that service growth offsets capital expenditure also exceeds the source's narrower discussion of margins, payback and funding ability.

**Fixed in the local reading export:** annual revenue, the five-to-ten-year horizon and the attributed investment plan are retained; the unsupported offset assertion is removed.

### S26 — narrow benchmark generalized

[Postgres enterprise-AI commentary](https://www.theregister.com/databases/2026/10/29/partner-content-built-to-bend-postgress-inventor-and-a-steward-on-what-ai-really-needs/5305957). The source describes Stonebraker's MIT warehouse experiments and Beaver benchmark. The summary turns their reported results into an unqualified ceiling for complex enterprise text-to-SQL generally. It also combines the speakers' positions and omits the EnterpriseDB contribution context.

**Fixed in the local reading export:** the performance statement is restricted and attributed to the named experiments, differing outlooks are preserved, and contributed-content context is explicit.

The local export now has four marked source-check corrections. These are editorial revisions, not evidence that the generator independently produced correct replacements. Raw database summaries were not overwritten, no new provider generation calls were made, and the source snapshot was preserved. Minor issues remain documented rather than silently cleaned up.

## Article-level audit ledger

The [machine-readable ledger](research/rss-summary-fidelity-audit.json) records all 30 source links, source/output hashes, classifications and concise evidence/remedies. Full publisher articles and complete model outputs remain in the local evidence files; they are not republished in this repository. Source paragraph locators refer to the saved snapshot; reviewer notes retain the corresponding text anchors where markup introduces a numbering difference.

| ID / source topic | Original output | Finding or reviewed interpretation |
|---|---|---|
| S01 — [GM consent settlement](https://www.theregister.com/security/2026/10/30/gm-cracks-the-petty-cash-box-for-multistate-driver-tracking-settlement/5306721) | Minor issue only | DC is counted as a state; settlement amount and principal allegations are preserved. |
| S02 — [Cursor extension infection](https://www.theregister.com/cyber-crime/2026/10/30/anthropic-that-claude-code-extension-compromise-couldnt-be-us/5306682) | Material problem | Continuing local persistence/cleanup is omitted; removal attribution also needs precision. |
| S03 — [Voyager communications](https://www.theregister.com/science/2026/10/30/voyager-1-gives-controllers-the-silent-treatment/5306637) | Material problem | Distance becomes power; an unconfirmed explanation is attributed to the manager. |
| S04 — [Early AlpineBSD project](https://www.theregister.com/os-platforms/2026/10/30/alpinebsd-announced-an-ai-free-foss-os/5306589) | Minor issue only | Qualified avoidance of generated material becomes an absolute ban. |
| S05 — [Hubble servicing options](https://www.theregister.com/science/2026/10/30/nasa-ponders-a-hubble-servicing-mission-or-a-controlled-deorbit/5306622) | Minor issue only | Author analysis and one former astronaut’s concern become a plural expert warning. |
| S06 — [AI productivity report](https://www.theregister.com/ai-and-ml/2026/10/30/ai-projects-not-paying-off-keep-at-it-says-uni-productivity-institute/5306581) | No issue identified | Report recommendations and skeptical counter-evidence remain attributed. |
| S07 — [GP website disruption](https://www.theregister.com/public-sector/2026/10/30/nhs-quango-warns-of-risk-to-gp-patient-data-after-website-vendor-collapse/5306647) | No issue identified | Disruption, continuity needs, institutional responsibility and the replacement template are retained. |
| S08 — [Missing GCHQ storage](https://www.theregister.com/security/2026/10/30/british-intelligence-agency-misplaced-thousands-of-storage-devices-years-later-still-cant-say-how-many/5306605) | Minor issue only | A planned tracking system reads as an implemented fix; the agency response is still attributed. |
| S09 — [Accenture recruiting agents](https://www.theregister.com/ai-and-ml/2026/10/30/accenture-ai-agents-are-sifting-through-applicants/5306460) | No issue identified | Recruiting benefits remain executive claims and human final decisions are preserved. |
| S10 — [Household robot insurance](https://www.theregister.com/ai-and-ml/2026/10/30/domestic-droid-gone-amok-dont-worry-youre-covered/5306396) | No issue identified | Coverage scope and expected household-robot demand remain attributed. |
| S11 — [Shelly reseller incident](https://www.theregister.com/cyber-crime/2026/10/30/uk-shelly-reseller-warns-payment-data-stolen-after-malware-found-on-website/5306250) | Minor issue only | The regulator was already notified; only the investigation remains ongoing. |
| S12 — [Google search monetization](https://www.theregister.com/ai-and-ml/2026/10/30/googles-search-revenue-ferrari-hits-an-ai-powered-speed-bump/5306520) | Minor issue only | The broader AI-products interpretation is attributed more widely than the quoted executive statement. |
| S13 — [Farm support anecdote](https://www.theregister.com/systems/2026/10/30/farm-pc-maintained-by-a-14-year-old-dumped-support-chap-in-a-steaming-pile-of-tech-mess/5306217) | Minor issue only | End of mainstream support becomes a claim of no support at all. |
| S14 — [AWS growth forecast](https://www.theregister.com/off-prem/2026/10/30/aws-installs-an-ai-accelerator-every-14-seconds-and-eyes-1-trillion-annual-revenue/5306547) | Material problem | Annual revenue becomes valuation; capital-expenditure payback is also overstated. |
| S15 — [Binance security incident](https://www.theregister.com/security/2026/10/30/in-the-year-ai-re-wrote-the-infosec-rulebook-its-comforting-to-know-crypto-exchanges-still-have-bad-security/5306539) | Minor issue only | No evidence of a private-key leak becomes no leak; initial-investigation wording mitigates severity. |
| S16 — [Intel earnings and supply](https://www.theregister.com/systems/2026/10/30/pc-chip-plunge-wipes-out-a-quarter-of-intels-value/5306531) | Minor issue only | Memory’s PC-demand effect is merged with wafer constraints on server supply. |
| S17 — [AI developer-tool flaws](https://www.theregister.com/security/2026/10/29/researchers-find-bugs-in-5-ai-developer-tools-with-openai-and-microsoft-flaws-still-unpatched/5306508) | No issue identified | Product-specific exploit effects are summarized collectively without saying every tool has every flaw. |
| S18 — [Howler privacy model](https://www.theregister.com/offbeat/2026/10/29/howler-will-have-ai-answer-your-questions-then-burn-the-chat/5306484) | No issue identified | Ephemeral local history is distinguished from possible provider retention. |
| S19 — [Vera DevOps tooling](https://www.theregister.com/ai-and-ml/2026/10/29/startup-wants-to-make-ai-agents-safe-for-devops-duty/5306477) | Minor issue only | A forecast for one agents product is extended to all three planned tools. |
| S20 — [Topcoat web framework](https://www.theregister.com/devops/2026/10/29/topcoat-enables-rust-to-render-server-side-web-apps/5306470) | No issue identified | Framework behavior and limitations for complex applications are retained. |
| S21 — [PHP2Shell exploit chain](https://www.theregister.com/security/2026/10/29/suspected-chinese-attackers-built-a-custom-php2shell-exploit-and-rooted-25-servers/5306456) | No issue identified | Attribution, suspected origin, continuing access risk and independent tooling confirmation are preserved. |
| S22 — [Ubuntu Core Desktop work](https://www.theregister.com/os-platforms/2026/10/29/ubuntu-2704-may-bring-a-preview-of-the-long-awaited-core-desktop/5306426) | Minor issue only | First-person author analysis becomes an unspecified plural outside assessment. |
| S23 — [BattMan manufacturing program](https://www.theregister.com/science/2026/10/29/battman-aims-to-turbocharge-us-batteries-without-robbin-critical-minerals/5306273) | No issue identified | Domestic manufacturing, commercialization barriers and the proposed AI approach are retained. |
| S24 — [GSA internal file sharing](https://www.theregister.com/public-sector/2026/10/29/gsa-left-sensitive-google-drive-files-open-to-staff-contractors/5306430) | Minor issue only | Historical Google-environment findings become a more specific Drive history. |
| S25 — [Oracle AI usage pricing](https://www.theregister.com/applications/2026/10/29/oracle-wants-to-bill-you-for-every-ai-task-not-just-every-seat/5306403) | No issue identified | Usage-based AI pricing, migration claims and financing plans remain attributed. |
| S26 — [Postgres enterprise-AI commentary](https://www.theregister.com/databases/2026/10/29/partner-content-built-to-bend-postgress-inventor-and-a-steward-on-what-ai-really-needs/5305957) | Material problem | One benchmark/experiment result becomes a general enterprise ceiling; provenance is reduced. |
| S27 — [Belgian tax-agency outage](https://www.theregister.com/security/2026/10/29/belgiums-tax-agency-confirms-cyberattack-pulls-services-offline/5306409) | Minor issue only | The correct deadline dates lack the source’s affected-income-category qualification. |
| S28 — [Rust support in systemd](https://www.theregister.com/os-platforms/2026/10/29/systemd-adds-support-for-rust/5306264) | No issue identified | New Rust leaf components, build restrictions and core limitations are preserved. |
| S29 — [Vulcan return target](https://www.theregister.com/offbeat/2026/10/29/critical-launch-for-vulcan-as-ula-targets-friday-the-13th/5306243) | No issue identified | Return target, payload, nozzle history and attributed mitigation remain coherent. |
| S30 — [Ofcom age assurance](https://www.theregister.com/security/2026/10/29/shortlist-of-age-checks-for-the-uks-under-16-social-media-ban-is-out-some-adults-will-need-verifying/5306282) | No issue identified | Proposed methods, ID-access problem and privacy criticism remain attributed; headline covers adult impact. |

## Review scorecards

Counts below concern scoped review units, not independent probabilities or an overall accuracy score. Categories overlap and must not be summed. The fidelity row includes the thirteen minor-only cases as well as the four material cases. The immutable original outputs remain the evaluation target even after correcting the separate reading export.

### Artifact usefulness and quality

| Category | Observed defects | Assessment |
|---|---|---|
| Usefulness and completeness | 0 / 1 | All 30 original outputs were compared with saved source bodies for the stated publication-readiness question. |
| Analytical clarity | 0 / 3 | Definitions, article-level incidence and configuration/sampling limits are explicit; minor and material findings are separated. |
| Visual and interaction consistency | N/A | Not applicable: Markdown/JSON review; browser controls were not reverified. |

### Analytical correctness and robustness

| Category | Observed defects | Assessment |
|---|---|---|
| Source authority and scope | 0 / 30 | The publisher snapshot is the controlling fidelity reference for all 30, not independently certified truth. |
| Fact/value accuracy | 17 / 30 | Four original summaries have material problems; thirteen others have minor issues. This row counts any identified issue, not a model-accuracy percentage. |
| Within-chart agreement | N/A | Not applicable: No quantitative chart is part of this audit. |
| Complete source details | 0 / 30 | Each output has a canonical source URL and source/output hashes; findings retain scoped evidence and remedies. |
| Cross-artifact consistency | 0 / 30 | All 30 original summaries and source texts matched the local database; corrected display copies remain separate. |
| Data-quality controls | 0 / 30 | All 30 source/output pairs are present and unique with no exclusions; all originals, not edited replacements, form the denominator. |
| Conclusion support | 0 / 3 | Incidence arithmetic, the proposed quality-bar assessment and limits on production extrapolation are supported and qualified. |

## Recommended next step

**Continue source review for every pilot item before unattended publication.** The sample shows useful compression but does not satisfy the proposed zero-material-error quality bar. Formatting and low inference cost are insufficient quality controls.

For a separately scoped next experiment:

1. Include the normalized headline with the body. RSSBrew's custom HTML summary path currently sends the body without a separate title; the production bot already supplies a title. Missing headline context is a candidate contributor to some ambiguity, not a proven cause of these errors.
2. Require exact preservation of financial measures, units, populations, status and attribution. Retain absence-of-evidence language, unresolved causes, and continuing-risk/remediation warnings. Preserve contributed-content labels. These rules directly address the observed failure patterns.
3. Use this frozen sample for regression cases, then evaluate a new held-out sample with the revised prompt. Source-reference extraction and a second verifier may help, but their benefit must be measured; a second model agreeing is not proof of correctness.
4. Measure review effort with an actual human reader/editor. Agent elapsed time cannot establish minutes of human QA per article. This audit does not show that selective spot checks or a particular review budget are enough.

No prompt, model, scheduler or production publication policy was changed by this review. Live output changes and any additional paid experiment remain separate work.

## Provenance and completed checks

Repository baseline: `b938dc946996435450d689ebc284d79900cc8842`. [Pilot setup and observations are in PR #19](https://github.com/voxeb/rss-ai-telegram-bot/pull/19), which was still open when this review started. Relevant existing architecture references are F3, L7 and Q5; this document does not change them.

Before editing, reviewed current main, AGENTS.md, README, architecture, operations, market research and the pilot record. Working evidence is under `/private/tmp/rssbrew-audit-*.json` and `/private/tmp/rssbrew-pilot/pilot-input/`. The pre-audit `quality.json` hash is recorded in the public ledger. Temporary source files can be removed by system cleanup; hashes cannot reconstruct missing articles or model outputs.

Checked all 30 ID/URL/hash mappings, classification totals, numerator membership, ratio arithmetic and raw-versus-corrected separation. Compared SQLite source text and original summaries with the saved audit inputs: 30 matched pairs, no missing rows. After the four editorial fixes, credential-free export validation confirmed 30 parsed items, stable unique GUIDs, original links/dates, four marked corrections and zero new AI calls on rebuild. The maintainer gave positive feedback about summary usefulness; browser controls were not reverified and no new screenshot or browser automation check is claimed.

Documentation links, terminology and staged diff were reviewed. Application tests were not run because the repository changes contain only documentation and a review ledger. No merge or production deployment is implied.
