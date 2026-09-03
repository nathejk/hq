# 136 — Return after an error response: six handlers write two bodies

**Status:** open
**Priority:** medium
**Created:** 2026-09-03
**Picked up by:**
**Started:**
**Completed:**

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

- [ ] All six handlers `return` after the error response
- [ ] The Excel handlers write no spreadsheet bytes on a failed query — an errored
      export is a JSON error, not a zero-row download
- [ ] A test covers at least one list handler and one Excel handler failing, asserting
      a single parseable body and that no XLSX payload follows
- [ ] The sweep is repeatable: either a lint/vet check or a documented command in this
      log that finds error responses falling through, so the seventh does not appear
      unnoticed
- [ ] `go build ./...`, `go vet ./cmd/api/` and `go test ./cmd/api/...` green

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-09-03 07:40 — Task created. Split out of the `GET /api/patrulje` 500: the missing `return` in `showPatruljeListHandler` was fixed inline (`bc4942d`), and the sweep found six more, three of which silently emit empty Excel exports.
