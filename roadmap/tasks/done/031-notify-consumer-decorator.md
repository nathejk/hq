# 031 — `notify` consumer decorator

**Status:** done
**Priority:** high
**Created:** 2026-08-08
**Picked up by:** agent session (zed)
**Started:** 2026-08-08
**Completed:** 2026-08-08

## Description

The bridge between projections and the hub: a decorator that wraps any
`cqrs.Consumer` and publishes a signal **after** the wrapped consumer has applied the
event.

```go
mux.AddConsumer(notify(hub, patruljetable), notify(hub, paymenttable), …)
```

One decorator makes every entity live, present and future, with no per-entity code —
which is the property that justifies the whole signals-not-data design.

### The ordering trap this exists to avoid

If the hub subscribed to the stream independently, it could signal **before** the
projection committed: the client would refetch, read the *old* row, and display stale
data that no later event corrects. This is the most likely source of "it flickered
back to the old value" bugs.

So the notifier wraps, and emits only after `HandleMessage` returns `nil`. That is
sound because the write is synchronous — `sqlpersister.Writer.Consume` executes
`db.Exec` inline and MariaDB autocommits.

**One caveat, precisely.** `cqrs/deadletter` is a `Writer` decorator that diverts
failing statements to a table instead of failing the projection loop. If a notified
consumer is wired through it, `HandleMessage` can return `nil` while the read model was
*not* updated — producing a signal for a change that is not visible. `main.go` builds
its writer as `deadletter` wrapping `sqlpersister`, so **this applies here**. Decide
and document: either do not compose `deadletter` beneath a notified consumer, or accept
that a diverted statement yields a signal whose refetch shows no change (which is
recoverable — the next signal or resync corrects it — but must be a decision, not an
accident).

### Notes

- Must not change the wrapped consumer's behaviour: same subjects, same errors, and a
  failure must still propagate. A signalling side-effect may never swallow a
  projection error.
- Unparseable subject (029) → no signal, and do not fail the message.
- Keep it dependency-light: the decorator takes an interface the hub satisfies, so it
  can be tested with a fake.

## Acceptance Criteria

- [x] `notify(publisher, consumer)` returns something satisfying `cqrs.Consumer`
- [x] `Consumes()` is passed through unchanged
- [x] A signal is published only after `HandleMessage` returns nil
- [x] `HandleMessage` errors propagate unchanged and produce **no** signal
- [x] An unparseable subject produces no signal and no error
- [x] The `deadletter` interaction is documented in code, with the chosen behaviour
- [x] Tests with a fake publisher and a fake consumer, including the error path
- [x] `go test ./...`, `go vet ./...`, `go tool staticcheck ./...` pass

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-08 18:44 — Task created. Depends on 029, 030.
- 2026-08-08 19:48 — Picked up. Plan: `internal/live/notify.go` with a `Publisher`
  interface (the hub satisfies it), `Notify(p, consumer)` and a `NotifyAll` helper for
  wiring the nineteen consumers in main.go. Tests with fakes for both sides.
- 2026-08-08 19:54 — Added `internal/live/notify.go`: `Notify(p, c)` plus `NotifyAll`.
  Decisions:
  • **Projection first, and its error wins.** A signalling side-effect must never mask a
    projection failure, and a failed projection must not announce a change that did not
    happen. Both directions are tested.
  • **A non-signal subject is not an error.** The stream's own `caughtup` sentinel
    arrives as a message with subject `<domain>.caughtup`, and off-convention subjects
    exist too. Returning an error there would make the stream retry the message forever;
    the projection succeeded, so the message is done. Just no signal.
  • **`Publisher` is an interface**, so the decorator is testable without a hub and
    depends on nothing about how fan-out works. Added a compile-time check that `*Hub`
    still satisfies it.
- 2026-08-08 19:58 — **deadletter decision recorded** (the ticket asked for it):
  `main.go` builds its Writer as `deadletter` wrapping `sqlpersister`, so a diverted
  statement means `HandleMessage` can return nil while the read model was *not* updated
  — and this decorator will then signal a change that is not visible.
  Accepted rather than removing `deadletter` from notified consumers, because the
  failure is benign and self-correcting: the client refetches, sees no change, and the
  next signal or resync brings it up to date. Removing it would trade projection
  resilience — the entire reason `deadletter` exists — for signal precision nothing
  depends on. Written into the doc comment so the next reader finds a decision rather
  than an accident.
- 2026-08-08 20:02 — ✅ 7 tests (including the real `caughtup.NewCaughtupMessage`
  sentinel rather than a hand-rolled stand-in). `go test -race` clean, gofmt/vet/
  staticcheck clean, full suite green.
- 2026-08-08 20:04 — Completed. The pieces now exist to make any entity live; 032 adds
  the HTTP surface and 033 does the wiring.
