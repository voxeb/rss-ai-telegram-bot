# RSS summary configuration experiment

**Completed on 2026-10-31 UTC. Recommendation: continue a reviewed browser pilot; do not adopt V2 for unattended publication.** The candidate did not pass the predeclared gate. This document changes neither the production prompt nor publication policy.

## Decision and scope

The maintainer found AI summaries substantially more useful for personal reading than publisher descriptions. The next question is whether a more explicit summarization configuration reduces source-fidelity problems enough to support a reviewed RSS reading pilot. User value and factual reliability are separate questions.

This follows the [market research](market-research.md), [local pilot PR #19](https://github.com/voxeb/rss-ai-telegram-bot/pull/19), and [30-item source audit PR #20](https://github.com/voxeb/rss-ai-telegram-bot/pull/20). Both earlier experiment PRs were open at the start. Relevant architecture references are F3, L7 and Q5; this experiment does not change them.

## Frozen method

- Protocol and [candidate instructions](prompts/rss-summary-v2.txt) were frozen at **2026-10-30 21:45:29 UTC**, before reading the held-out bodies or generating their summaries. Generation started on October 31 UTC after command approval. Candidate SHA-256: `bb63303fbdcf9477c94b777e8488c3cd82db87c4b84b351e18cf5d616db29014` (UTF-8 text with the final newline stripped).
- **Held-out sample:** 30 URLs absent from the original 30-item pilot: the first 20 unseen items in the saved general feed and the first 10 additional unseen items in the saved security feed, preserving XML order. All had non-empty content and fit the 24,000-byte serialized-message cap; none was excluded. This convenience sample overrepresents security and is not representative of all news. Separate articles can cover related incidents; URL separation is not topic independence.
- **Paired comparison:** each new article receives one V1 and one V2 output. V1 reproduces the earlier pilot's generic HTML system instruction plus its custom 40–80-word prompt and body-only input. V2 uses the same system instruction, the more explicit candidate instructions, and a separate headline with the identical body. This tests a **prompt-and-input configuration**, not an isolated prompt effect. Neither is the production Go bot's generation configuration.
- Variant order is deterministically varied by URL hash, rather than always running one configuration first. No successful output is regenerated or selected for quality.
- **Regression:** V2 also summarizes all 30 original articles. The prior original outputs and classifications remain frozen; editorial display corrections are excluded from the comparison. These are development cases, not held-out evidence.
- **Common generation settings:** `gemini-3.5-flash-lite`, temperature 0.2, 400 output tokens, Gemini's [OpenAI-compatible API](https://ai.google.dev/gemini-api/docs/openai), SDK retries disabled, 45-second request timeout, at least six seconds between request starts. No thinking parameter is set. There is one draw per configuration/article, with no seed or reproducibility guarantee.
- **Limits:** 90 successful outputs and at most 100 requests for the entire experiment. The runner stops on any provider error; completed outputs are retained. These local request limits are not a cloud billing cap. Credentials travel through stdin into memory and are not written to the evidence or container configuration.

### Metrics and review rules

The primary metric is the number of articles with at least one **material fidelity defect**: a wrong or unsupported central fact, unjustified source/causal certainty, or an omission that changes reasonable understanding or action. A summary is counted once even with multiple defects. Minor attribution, procedural-status, scope or precision issues are recorded separately. Optional supporting detail and ordinary compression are not errors just because they are absent. The visible source headline provides omission context but cannot excuse a contradictory summary claim.

Secondary measures are minor-only classifications, completion, word counts excluding the attribution label, and provider-reported token usage. Token cost is a standard-price equivalent, not verified spending.

**Predeclared pilot gate:** zero observed material V2 defects on the 30 held-out articles, no new material regression defects, and at least 27/30 held-out V2 outputs within 40–80 words. Passing would justify further reviewed reading, not unattended production publication, a production accuracy claim, or an architectural decision. The small, selected sample cannot establish a reliable population error rate.

Review is AI-assisted against saved publisher bodies and headlines, not independent verification of the news or human editorial sign-off. A/B labels hide configuration identity during the initial item comparison, but the reviewer authored V2 and knows its content; this is not a fully blinded or independent review.

## Results

### Held-out comparison: the same 30 articles

| Measure | Previous V1 | Candidate V2 |
|---|---:|---:|
| Articles with a material fidelity defect | 1 / 30 | 1 / 30 |
| Articles with minor issues only | 9 / 30 | 3 / 30 |
| Articles with no issue identified | 20 / 30 | 26 / 30 |
| Within the 40–80-word target | 27 / 30 | 18 / 30 |
| Word count: minimum / median / maximum | 52 / 69 / 87 | 59 / 75 / 133 |
| Successful completions | 30 / 30 | 30 / 30 |
| Provider-reported input / output tokens | 36,346 / 3,018 | 42,325 / 3,580 |
| Standard-price equivalent, USD | $0.01845 | $0.02165 |

The material-error pair counts are **28 neither, one V1-only and one V2-only**. V2 avoids V1's material omission on H30 but introduces a different material problem on H29. This sample therefore does **not demonstrate a reduction in material defects**. Fewer minor-only classifications are a descriptive observation from this AI-assisted review, not independent evidence of a general improvement.

The two material findings are:

- **H29, V2 — remediation reversed.** In the [GMO virtual-desktop incident](https://www.theregister.com/security/2026/10/27/japanese-virtual-desktop-cloud-hacked-so-hard-it-asks-to-be-banned/5305610), V2 advises changing passwords from untrusted devices. The source instead recommends a device not used for trading. It also warns that changing a password may leave trading sessions active and says to disconnect all sessions or contact the provider. V2 omits that continuing-risk qualification. The affirmative advice inversion is material regardless of how omissions are classified.
- **H30, V1 — dangerous trigger omitted.** In the [npm worm report](https://www.theregister.com/security/2026/10/26/worm-armed-42-github-repos-by-hiding-a-payload-too-big-to-fail/5305570), V1 mentions a destructive wiper but omits that revoking the stolen GitHub token triggers home-directory deletion. This condition changes the normal credential-response action of an affected developer. V2 retains the token-revocation trigger. Counting this omission as material is an editorial judgment under the predeclared action-changing-omission rule.

**Sensitivity:** if material incidence counted only affirmative wrong/unsupported claims and excluded critical omissions, the held-out counts would be **V1 0/30, V2 1/30**. That alternative definition does not support promoting V2 either. Neither definition estimates production error rates.

Length compliance worsened: V2 exceeded 80 words on twelve articles, including a 133-word UNM summary. These are unedited outputs, not excerpts cut to meet the target. More detailed instructions are not a length validator. Both material cases completed normally: H29 has 83 words, whereas H30 has 64. Length alone cannot establish fidelity.

### Regression: the original 30 development articles

V2 has **0 material, 4 minor-only and 26 no-issue-identified classifications** on the old sample. The four minor cases are S02, S20, S22 and S26. Twenty-three outputs meet 40–80 words; lengths range from 55 to 102, with a median of 72.5. All 30 complete successfully.

The earlier audit classified the original outputs as four material, thirteen minor-only and thirteen without an identified issue. That earlier generation happened before this run, uses one different draw per article and is development data; it is **not** a contemporaneous held-out comparison or proof that V2 caused the difference.

| Prior material case | V2 observation | Limit |
|---|---|---|
| S02: extension persistence | Explicitly says a simple update does not clear persistence. | Minor overstatement makes the vendor's scripts mandatory and drops the OS scope of the admin-rights limitation. |
| S03: Voyager quantity and cause | No power substitution or asserted safe-mode explanation remains. | The one-light-day quantity is omitted entirely; successful preservation of that measurement was not demonstrated. |
| S14: AWS forecast | Correctly says annual revenue, with the five-to-ten-year horizon. | One development case, not evidence of general financial reliability. |
| S26: enterprise text-to-SQL | Limits the zero-percent result to MIT's warehouse and identifies EnterpriseDB provenance. | Minor issues remain: no initial-baseline qualifier or later improvement into the 30-percent range; contributed content becomes a more specific sponsored relationship. |

The prior four material defects were not repeated under this review's classifications. That does not mean every original fact was preserved or each revised output is issue-free. New minor issues include an ambiguous server/client compilation description in S20 and merged build/preview timing in S22.

### Gate and cost

| Predeclared condition | Result |
|---|---|
| Zero material V2 defects on the held-out 30 | **Fail:** one, H29. |
| No new material regression defects | Pass within this source review. |
| At least 27/30 held-out V2 outputs within 40–80 words | **Fail:** 18/30. |
| Overall pilot gate | **Not passed.** |

There were **90 successful outputs in 90 requests**, with no provider errors or retries during this run. The regression block used 45,628 input and 3,265 output tokens. Across all three blocks, reported usage was **124,299 input and 9,863 output tokens**. Using Google's standard text prices of $0.30 per million input tokens and $2.50 per million output tokens, the equivalent is approximately **$0.062**: `(124299 × 0.30 + 9863 × 2.50) / 1000000`. These rates were [checked on October 31](https://ai.google.dev/gemini-api/docs/pricing); this is not verified billing and excludes review-agent use, human editing and hosting costs.

Low inference cost does not resolve the quality problem. Human review time and willingness to pay remain unmeasured.

## Interpretation and recommendation

**Verified within this experiment:** the candidate has fewer minor-only classifications, fails the length target more often, and has a new material remediation error on the held-out sample. The original four material problems are absent in the regression classifications, with residual minor issues and some omitted facts. The private comparison page retains all outputs unedited and shows their review notes alongside publisher descriptions.

**Hypotheses:** explicit units, attribution and remediation instructions may help some cases; a longer rule list may also encourage longer outputs. Headline context may help disambiguation. This design cannot isolate those mechanisms, and one stochastic output per configuration/article cannot establish reproducibility.

**Recommendation:** retain the specialized Telegram bot and continue the browser experiment with source review. Keep V2 as an experimental candidate rather than replacing the live prompt. The current evidence supports investigating independent source verification and measured human review effort before unattended RSS distribution; another model agreeing would not establish correctness. Do not generalize the project into a publishing framework on these results.

The immediate useful next activity is a personal reading session with the existing comparison page: record whether the essential facts are understood without opening the article, when the description is sufficient, and the actual time needed to review/correct summaries. No additional generation is needed to start that session. The maintainer's positive feedback is evidence of personal usefulness, not market demand, comprehension-test performance or willingness to pay.

**Unresolved:** acceptable residual risk, human review cost, benefit of a separately evaluated verifier, performance across publishers/days/genres, reader preference for shorter summaries, provider-model stability, and terms for public redistribution. The local preview does not settle those questions or authorize public publication.

## Evidence and limitations

The [public article-level ledger](research/rss-prompt-v2-experiment.json) contains all 60 source links/hashes, 90 output hashes, classifications, concise evidence, the frozen protocol and aggregate arithmetic. The local evidence directory is `/private/tmp/rssbrew-pilot/prompt-v2/`: source snapshots, job order, raw outputs, request ledger and masked-label mapping. Full publisher bodies and complete outputs remain local. Temporary files may be removed by system cleanup; hashes cannot reconstruct missing source material, so this is not a permanently self-contained public replication dataset.

Source feeds: [general](https://www.theregister.com/headlines.atom), [security](https://www.theregister.com/security/headlines.atom). The `.atom` endpoints returned RSS 2.0; the experiment uses their `content:encoded` bodies. No additional article fetch substitutes for the snapshot.

The production Go bot, Telegram publications, Firestore state, scheduling, cloud configuration and existing architectural decisions are outside this experiment. Repository changes are documentation only. Any production prompt change needs its own behavior Issue and reviewed implementation PR.

All 60 saved bodies and all 90 new summaries were read in the source comparison. One AI reviewer authored the candidate and completed the review; there was no independent second reviewer, human adjudication or inter-rater agreement measurement. Initial held-out notes used A/B labels; final adjudication and reporting happened after resolving the mapping. The source is a fidelity reference, not certified truth: for example, H19 includes a 62-percent increase claim alongside rounded 13-to-20-million-ton figures that do not yield that percentage. This source inconsistency was retained as a limitation, not silently corrected using outside information.

Generation uses a requested model alias, not an independently pinned provider backend. Source URLs are disjoint between development and holdout, but related Google earnings and security incidents appear in both. The sample and prompt were selected from one publisher, and the candidate's author reviewed the output; no causal, market-wide, human-comprehension or production-quality claim follows.

Completed checks: reconciled all 60 unique source URLs and HTML hashes; all 90 output hashes, nonempty classifications, word counts, successful finish states and usage entries; checked cohort totals, paired material membership and gate arithmetic. The local reading page returned HTTP 200 with bytes matching the exported artifact, 30 article cards, 90 text variants and 60 review notes. Browser rendering and button interaction were not visually verified; only source/structure and HTTP delivery were checked. Reading or rebuilding the static page makes no generation calls.

Reviewed documentation links, terminology and the staged diff. Go tests were not run because this PR contains only documentation and research evidence. No application code, infrastructure, production configuration or architectural decision was modified; no merge or deployment is implied.
