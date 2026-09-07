# 160 — Bump shared-go and adopt the typed payment method

**Status:** done
**Priority:** high
**Created:** 2026-09-07
**Picked up by:** agent session
**Started:** 2026-09-07
**Completed:** 2026-09-07

## Description

From **PRD 012** §10, phase H0. The gate every other hq task in that PRD waits behind.

Pull the shared-go version carrying tasks 156–159 (the seat-transfer capability) and make
hq build against it. Verify the seam rather than assuming it: hq depends on subject
strings, JSON field names and exported signatures decided in another repo, and a rename
that compiles produces a projection consuming nothing and an empty table, with no build
error anywhere.

Expected from the shared-go upgrade notes (`docs/moving-a-paid-member.md` §11):

1. **One source-compatibility break** — `payment.Payment.Method` is now
   `types.PaymentMethod` rather than `string`.
2. **Two new `payment` columns plus an index**, applied by `cqrs.EnsureColumn` /
   `cqrs.EnsureIndex` in `payment.New`, so an existing dev database gets them on the next
   start with no manual migration.
3. **`spejder.teamId` is now mutable**, but only via the new `reassigned` subject.
4. Everything else additive.

## Notes

- Bumped to `v0.0.0-20260907115955-8b5198064fce` (shared-go `8b51980`).
- The single break landed exactly where the notes predicted: `cmd/api/order.go`, building
  a `paymentRef.Provider` from `p.Method`. Fixed with `string(p.Method)` rather than by
  widening `paymentRef` — that struct is the API's own wire shape and does not want a
  shared-go type in it.
- `klan_test.go`'s `fakeOrderQueries` needed `order.MemberLineReader` embedded once
  `data.Models.Order` became `data.OrderInterface`. Nil interface is deliberate: these
  tests do not call it, and a call should panic rather than quietly return nothing.
- **No new mux consumer.** The transfer command only publishes; `ordertable`,
  `paymenttable` and `spejdertable` were already mounted and already in the `projections`
  slice, so the live signals (`order`, `payment`, `spejder`) flow with no wiring. That is
  what makes PRD 012's phase H3 empty — recorded here rather than as a task that turns out
  to be nothing.
- Not verified here: that the two new payment columns actually appear in a **pre-existing**
  dev database. It needs a real start against a database created before the bump — see
  task 163.

## Acceptance Criteria

- [x] `go/go.mod` pinned to a shared-go version containing tasks 156–159
- [x] `go build ./...` clean
- [x] `go vet ./...` clean
- [x] `go test ./...` green
- [x] `gofmt` clean
- [x] The `Method` type change handled without leaking the shared-go type into hq's wire
      structs
- [x] Confirmed no new consumer is needed for the new subjects

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-09-07 — Bumped to `8b51980`. One compile break, as documented. Fixed, plus the test
  fake. Build, vet, fmt and the full Go test suite green. Confirmed the new subjects need
  no mux change because their projections are already mounted and already wrapped by
  `live.NotifyAll`.
