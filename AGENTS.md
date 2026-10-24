# Codex instructions

1. Before work, read [README.md](README.md), [architecture](docs/architecture.md),
   [operations](docs/operations.md), and the related Issue when applicable.
   Do not rely on access to other chat histories.
2. Distinguish implemented behavior, agreed decisions, proposals, and open
   questions. Verify production state separately from repository state.
3. Communicate with the maintainer in Russian. Write code, comments,
   documentation, and public materials (including Issues and PRs) in English.
4. Keep the architecture simple and dependencies justified. Consider publication
   reliability, duplicates, retry limits, and operating costs in every change.
5. Check Git status, fetch current `origin/main`, and review open PRs before
   editing. Preserve existing local changes. Use separate branches and PRs;
   behavior changes need an Issue, but this instructions document does not.
6. For Go changes, follow [CI](.github/workflows/test.yml):

   ```bash
   test -z "$(gofmt -l .)"
   go test -race -count=1 ./...
   go vet ./...
   CGO_ENABLED=0 GOOS=linux go build -trimpath -o register-news-bot .
   ```

   For documentation-only changes, review text, links, and the diff; Go checks
   are not required locally.
7. Record agreed decisions in `docs/architecture.md` and operational facts and
   deployment evidence in `docs/operations.md`; identify unverified claims.
8. Never expose secrets in files, command output, logs, Issues, or PRs. Gather
   facts and exact errors before diagnosing or proposing fixes; redact secrets
   before displaying or sharing evidence.
9. Deployment, cloud resource changes, production Firestore writes, and real
   publications require explicit maintainer approval. `TEST_MODE=true` also
   sends a real Telegram message.
10. PRs must describe changes, validation results, and limitations. A merged PR
    is not evidence of deployment.
11. When handing work to another session, leave Issue/PR links, agreed decisions,
    completed checks, and remaining questions in the relevant Issue or PR.
