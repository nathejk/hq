# 136 — Return after an error response: six handlers write two bodies

**Status:** done
**Priority:** medium
**Created:** 2026-09-03
**Picked up by:** Zed agent
**Started:** 2026-09-03
**Completed:** 2026-09-03

## Description

`showPatruljeListHandler` called `app.ServerErrorResponse(...)` and then *fell
through* to write the success envelope, so a failing request answered with an error
object **followed by** `{"teams": null}` — two JSON documents in one body, which no
client can parse. That is how a plain database timeout reached the operator as an
unexplained failure instead of an error message. Fixed in commit `bc4942d`.

The pattern is not unique to that handler. A sweep of `go/cmd/api` for error
responses that fall through to further code — excluding `switch` arms, which cannot
fall through in Go, and blocks that already end in `return` — leaves exactly six:

| Handler | File |
|---|---|
| `showBadutListHandler` | `go/cmd/api/badut.go:15` |
| `listPersonnelHandler` | `go/cmd/api/personnel.go:15` |
| `mailRecipientsHandler` | `go/cmd/api/mail.go:22` |
| `excelPatruljeHandler` | `go/cmd/api/export.go:22` |
| `excelKlanHandler` | `go/cmd/api/export.go:125` |
| `excelPersonnelHandler` | `go/cmd/api/export.go:222` |

The three list handlers reproduce the patrulje bug exactly: error envelope, then a
second `{"personnel": null}` or a recipient list built from nil.

The three Excel handlers are worse than a malformed body. They keep going and build a
whole spreadsheet from the nil slice, then write it over a response that already
carries an error status and `Content-Type: application/json` — so the operator
downloads an empty-but-plausible `patruljer-20260903….xlsx`. A silently empty export
is a wrong answer presented as a right one, which is the failure mode worth caring
about here: nobody checks a spreadsheet's status code.

Note these six share a cause with the 500 itself. `excelPatruljeHandler` and
`excelKlanHandler` call the same `GetAll` queries that task 135 is about, so they were
failing in production too — just invisibly, as blank exports.

Deliberately **not** in scope: occurrences inside `/* */` blocks
(`badut.go:92`, `klan.go:427`, `mail.go:103`, `patrulje.go:193`). Leave the commented
code alone or delete it as its own task; do not half-revive it.

## Acceptance Criteria

- [x] All six handlers `return` after the error response
- [x] The Excel handlers write no spreadsheet bytes on a failed query — an errored
      export is a JSON error, not a zero-row download
- [x] A test covers at least one list handler and one Excel handler failing, asserting
      a single parseable body and that no XLSX payload follows
- [x] The sweep is repeatable: either a lint/vet check or a documented command in this
      log that finds error responses falling through, so the seventh does not appear
      unnoticed
- [x] `go build ./...`, `go vet ./cmd/api/` and `go test ./cmd/api/...` green

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-09-03 07:40 — Task created. Split out of the `GET /api/patrulje` 500: the missing `return` in `showPatruljeListHandler` was fixed inline (`bc4942d`), and the sweep found six more, three of which silently emit empty Excel exports.
- 2026-09-03 13:20 — Picked up. Plan: add the returns, stop the Excel handlers emitting a workbook on failure, cover both shapes with tests, and replace the throwaway sweep with a check that runs in CI.
- 2026-09-03 13:30 — All six `return` now. ✅ Criterion 1.
- 2026-09-03 13:40 — ✅ Criterion 2, and went further than the criterion asked. The three Excel handlers shared a tail that set `Content-Disposition` and then called `xlsx.Write(w)`, ignoring its error; that tail is now one helper, `writeXlsx`, which renders the workbook to a buffer *before* setting a single header. Rendering first is the point: the headers are the commitment, so once a 200 and an attachment filename are on the wire, any later failure arrives as a corrupt file the browser still saves under a plausible name. As a side effect the response now carries a real `Content-Length`, and the write error is logged instead of dropped.
- 2026-09-03 13:55 — ✅ Criterion 3. `cmd/api/export_errors_test.go`: a failing list handler must answer exactly one JSON document (asserted by decoding and then asking whether anything remains, since the number of documents is what broke the client), a failing Excel handler must send no `Content-Disposition`, no `application/octet-stream` and no `PK\x03\x04` zip magic, and — so the second test cannot be satisfied by never producing a workbook at all — the success path must still return a real xlsx with a Content-Length. Verified non-vacuous by copying the tests into a worktree at 9e549a0 and running them there: both failure tests fail, and the output dumps the empty spreadsheet that used to be appended to the error body.
- 2026-09-03 14:10 — ✅ Criterion 4, as a test rather than a documented command, so CI enforces it: `cmd/api/errorresponse_lint_test.go` walks the package AST and fails when an error responder is not the last thing its block does. Two false starts worth recording. First version flagged 40-odd `app.ServerErrorResponse(...)` followed by `return`, which is the fix, not the bug. Second version then passed — but a deliberately planted canary showed it was passing for the wrong reason: it only looked at error responses appearing *directly* in a block, and the real shape is an error response as the last statement of an `if` guard, with the *enclosing* block carrying on. That is precisely the case the naive check skipped, so the lint would have caught none of the seven. Now it asks about the guard and its enclosing block, exempting a guard immediately followed by `return` and `case` arms (Go does not fall through).
- 2026-09-03 14:15 — Lint verified against the pre-fix tree (worktree at 9e549a0): reports exactly seven — badut.go:15, export.go:22/125/222, mail.go:22, patrulje.go:88, personnel.go:15 — which is the full set found by hand, with no false positives and nothing missed. Canary removed, worktree removed.
- 2026-09-03 14:20 — ✅ Criterion 5. gofmt clean, `go build ./...`, `go vet ./cmd/api/`, `go tool staticcheck ./cmd/api/` and the full `go test ./...` all green. Note `go run honnef.co/go/tools/cmd/staticcheck@2024.1.1` cannot build this module (needs go1.25); the pinned `go tool staticcheck` from go.mod is the one to use.
- 2026-09-03 14:25 — Completed. Six handlers fixed, the Excel tail rewritten to be all-or-nothing, three behaviour tests and one AST lint added. Left alone as scoped: the four occurrences inside `/* */` blocks (badut.go, klan.go, mail.go, patrulje.go) — dead code, worth deleting on its own terms rather than reviving. Also unaddressed: the lint only checks `cmd/api`, and only guards with no `else`; an if/else where both branches respond and the function continues would slip through. Not seen in this codebase, so not built for.
