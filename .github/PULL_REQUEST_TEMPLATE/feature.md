<!-- Feature PR: new capability or behavior change. Pick the template that matches your change. -->

## Linked issue

<!-- closes #xx (resolves the issue) or refs #xx (partial) — one PR does one thing; split cross-cutting work into multiple PRs. -->

## What changed

<!-- Bullet list: what was added, what behavior changed, what it replaces.
     Call out separately: generated code (auto/), migrations, config keys (embedded + sample), docs. -->

## BVT — backend (mandatory, see CONTRIBUTING.md)

- [ ] `go build ./...` clean
- [ ] `go vet ./...` clean
- [ ] `golangci-lint run ./...` clean
- [ ] `go test ./...` all green
- [ ] Touched `mirc/`: ran `make gen-mir`, no hand edits in `auto/`
- [ ] Touched schema: migration pair in `scripts/migration/postgres/` (PostgreSQL only), verified with a `migration`-tagged build
- [ ] Touched config: `internal/conf/config.yaml` and `config.yaml.sample` updated together
- [ ] No build artifacts committed

## BVT — frontend (mandatory if `web/` changed)

- [ ] `npm run lint` — 0 errors
- [ ] `npm run i18n:check` — no missing / unused keys, zh-CN ↔ en in parity
- [ ] `npm run build` — succeeds
- [ ] No overlapping or overflowing content at 1920 / 1600 / 1366 / 1200 / 1000 / 821 / 375 viewports:
  - [ ] `python scripts/verify_sidebar_830.py` — ALL PASS
  - [ ] `python scripts/measure_width.py` — all viewports pass
  - [ ] `python scripts/screenshot.py` — screenshots attached below

## Regression scripts (run the ones relevant to this change; delete the rest)

<!-- Paste the summary/count lines from each script's output. Explain anything not run. -->

- [ ] `scripts/test_audit_flow.py`: PASS=__ FAIL=__
- [ ] `scripts/test_course_flow.py` (two-phase): PASS=__ FAIL=__
- [ ] `scripts/test_whisper_matrix.py`: PASS=__ FAIL=__

## Impact

<!-- API semantics, config keys, dependencies added/removed (justify any new dependency),
     data migrations, and what deployment side must do (config change, run migration, flush Redis). -->

## Evidence

<!-- Mandatory: screenshots or a recording of your hands-on test on a local instance, showing the commit tested
     (e.g. run `git log -1 --oneline` first). UI changes: show the feature in use, before/after where relevant.
     Backend/API changes: show the requests and responses. Script screenshots alone are not enough. -->
