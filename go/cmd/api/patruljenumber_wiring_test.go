package main

import (
	"context"
	"testing"
	"time"

	"github.com/jrgensen/cqrs"
	"github.com/jrgensen/cqrs/cqrstest"
	"github.com/nathejk/shared-go/messages"
	"github.com/nathejk/shared-go/tables"
	"github.com/nathejk/shared-go/tables/order"
	"github.com/nathejk/shared-go/tables/product"
	"github.com/nathejk/shared-go/types"
	"nathejk.dk/internal/live"
	"nathejk.dk/nathejk/table/patrulje"
	"nathejk.dk/nathejk/table/patruljenumber"
)

// These tests live in the composition root on purpose. The number saga's own
// tests drive the bare saga; what is verified here is the seam main() actually
// builds — the saga wrapped in live.Notify — because the wrapper is what carries
// CaughtUp, and a saga that never learns it is live silently publishes nothing.
// That failure has no runtime symptom, so it needs a test rather than a comment.

// --- fakes ---

type wiringOrders struct {
	orders map[string]*order.Order
	seats  int
}

func (f *wiringOrders) GetByID(_ context.Context, orderID string) (*order.Order, error) {
	o, ok := f.orders[orderID]
	if !ok {
		return nil, tables.ErrRecordNotFound
	}
	return o, nil
}

// ListByOwner reports one paid order carrying `seats` seats for whichever owner is
// asked about, which is all these wiring tests need.
func (f *wiringOrders) ListByOwner(_ context.Context, year types.YearSlug, ownerType types.TeamType, ownerID string) ([]order.Order, error) {
	return []order.Order{{
		OrderID:   "o-" + ownerID,
		Year:      year,
		OwnerType: ownerType,
		OwnerID:   ownerID,
		Status:    order.StatusPaid,
		ChangedAt: "2026-07-01 10:00:00",
		Lines: []order.Line{
			{ProductSKU: "participation.patrulje", Quantity: f.seats},
		},
	}}, nil
}

type wiringProducts struct{}

func (wiringProducts) ListEligibleFor(context.Context, types.YearSlug, types.TeamType) ([]product.Product, error) {
	return []product.Product{{SKU: "participation.patrulje", Kind: product.KindParticipation}}, nil
}

type wiringPatruljer struct{ rows []patrulje.Patrulje }

func (f wiringPatruljer) AssignedNumbers(context.Context, types.YearSlug) (map[types.TeamID]string, error) {
	numbers := map[types.TeamID]string{}
	for _, p := range f.rows {
		if p.TeamNumber != "" {
			numbers[p.TeamID] = p.TeamNumber
		}
	}
	return numbers, nil
}

// --- helpers ---

func paidOrder(orderID, teamID string) *order.Order {
	return &order.Order{
		OrderID:   orderID,
		Year:      "2026",
		OwnerType: types.TeamTypePatrulje,
		OwnerID:   teamID,
		Status:    order.StatusPaid,
	}
}

func orderPaidMsg(t *testing.T, orderID string) cqrs.Message {
	t.Helper()
	m := cqrstest.NewMessage(cqrs.SubjectFromStr("NATHEJK:2026.order." + orderID + ".paid"))
	if err := m.SetBody(&messages.NathejkOrderPaid{OrderID: orderID}); err != nil {
		t.Fatalf("set body: %v", err)
	}
	return m
}

func numberAssignedMsg(t *testing.T, teamID, number string) cqrs.Message {
	t.Helper()
	m := cqrstest.NewMessage(cqrs.SubjectFromStr("NATHEJK:2026.patrulje." + teamID + ".numberassigned"))
	if err := m.SetBody(&messages.NathejkPatrolNumberAssigned{
		TeamID:     types.TeamID(teamID),
		TeamNumber: number,
	}); err != nil {
		t.Fatalf("set body: %v", err)
	}
	return m
}

// wireNumberSaga builds the saga and wraps it exactly as main() does.
func wireNumberSaga(t *testing.T, orders *wiringOrders, rows []patrulje.Patrulje) (cqrs.Consumer, *cqrstest.Publisher) {
	t.Helper()
	pub := &cqrstest.Publisher{}
	hub := live.NewHub(live.WithCoalesceWindow(time.Millisecond))
	t.Cleanup(hub.Close)

	saga := patruljenumber.New(pub, orders, wiringProducts{}, wiringPatruljer{rows: rows}, "2026", time.Millisecond)
	return live.Notify(hub, saga), pub
}

func numbersPublished(t *testing.T, pub *cqrstest.Publisher) []string {
	t.Helper()
	var out []string
	for _, m := range pub.Messages {
		var body messages.NathejkPatrolNumberAssigned
		if err := m.Body(&body); err != nil {
			t.Fatalf("body: %v", err)
		}
		out = append(out, body.TeamNumber)
	}
	return out
}

// goLive is what the jetstream subscribe path does once a consumer's backlog has
// drained: assert the optional interface on the handler it was given, and call it.
// If the decorator drops CaughtUp, this fails here rather than mysteriously
// producing a saga that never assigns anything.
func goLive(t *testing.T, c cqrs.Consumer) {
	t.Helper()
	listener, ok := c.(interface{ CaughtUp() })
	if !ok {
		t.Fatal("the wrapped consumer does not advertise CaughtUp; the saga would never go live")
	}
	listener.CaughtUp()
}

// --- tests ---

// The property PRD 003 §9 cares about most: restarting must not re-issue numbers.
// A restart is a replay, so history is delivered before CaughtUp — and a team that
// is already numbered must come out the other side with nothing published.
func TestNumberSagaPublishesNothingForAlreadyNumberedTeamOnReplay(t *testing.T) {
	orders := &wiringOrders{
		orders: map[string]*order.Order{"order-1": paidOrder("order-1", "team-1")},
		seats:  4,
	}
	consumer, pub := wireNumberSaga(t, orders, nil)

	// Replay: the assignment, then the paid order that originally caused it.
	for _, msg := range []cqrs.Message{
		numberAssignedMsg(t, "team-1", "7"),
		orderPaidMsg(t, "order-1"),
	} {
		if err := consumer.HandleMessage(msg); err != nil {
			t.Fatalf("replay: %v", err)
		}
	}
	if len(pub.Messages) != 0 {
		t.Fatalf("published during replay: %v", pub.Subjects())
	}

	goLive(t, consumer)

	// Live now, and the same order is re-delivered (or another order for the same
	// patrulje is paid). Still nothing: the team holds a number.
	if err := consumer.HandleMessage(orderPaidMsg(t, "order-1")); err != nil {
		t.Fatalf("live: %v", err)
	}
	if len(pub.Messages) != 0 {
		t.Errorf("re-numbered an already-numbered team: %v", numbersPublished(t, pub))
	}
}

// The same wiring must actually work once live — otherwise the test above would
// pass for the wrong reason (a saga that can never publish at all).
func TestNumberSagaAssignsThroughTheWrapperOnceLive(t *testing.T) {
	orders := &wiringOrders{
		orders: map[string]*order.Order{"order-1": paidOrder("order-1", "team-1")},
		seats:  3,
	}
	consumer, pub := wireNumberSaga(t, orders, nil)
	goLive(t, consumer)

	if err := consumer.HandleMessage(orderPaidMsg(t, "order-1")); err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if got := numbersPublished(t, pub); len(got) != 1 || got[0] != "1" {
		t.Fatalf("numbers = %v, want [1]", got)
	}
	if got, want := pub.Subjects()[0], "NATHEJK.2026.patrulje.team-1.numberassigned"; got != want {
		t.Errorf("subject = %q, want %q", got, want)
	}
}

// Numbers issued within one process are strictly increasing and distinct, and
// continue past a number that only exists in the read model.
func TestNumberSagaIssuesStrictlyIncreasingDistinctNumbers(t *testing.T) {
	orders := &wiringOrders{
		orders: map[string]*order.Order{
			"order-1": paidOrder("order-1", "team-1"),
			"order-2": paidOrder("order-2", "team-2"),
			"order-3": paidOrder("order-3", "team-3"),
		},
		seats: 5,
	}
	// A manual 300 already in the read model: the sequence must continue past it.
	consumer, pub := wireNumberSaga(t, orders, []patrulje.Patrulje{
		{TeamID: "team-manual", TeamNumber: "300"},
	})
	goLive(t, consumer)

	for _, id := range []string{"order-1", "order-2", "order-3"} {
		if err := consumer.HandleMessage(orderPaidMsg(t, id)); err != nil {
			t.Fatalf("HandleMessage %s: %v", id, err)
		}
	}

	got := numbersPublished(t, pub)
	want := []string{"301", "302", "303"}
	if len(got) != len(want) {
		t.Fatalf("numbers = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("numbers = %v, want %v", got, want)
		}
	}
}

// No reuse after cancellation. The saga does not subscribe to order.cancelled at
// all, so a cancellation cannot clear `assigned` or lower the high-water mark —
// this pins that: the cancelled team keeps its number, and the next qualifier gets
// the number after it rather than the freed one.
func TestNumberSagaDoesNotRecycleNumbersAfterCancellation(t *testing.T) {
	orders := &wiringOrders{
		orders: map[string]*order.Order{
			"order-1": paidOrder("order-1", "team-1"),
			"order-2": paidOrder("order-2", "team-2"),
		},
		seats: 4,
	}
	consumer, pub := wireNumberSaga(t, orders, nil)
	goLive(t, consumer)

	if err := consumer.HandleMessage(orderPaidMsg(t, "order-1")); err != nil {
		t.Fatalf("first assignment: %v", err)
	}

	// team-1 cancels. Even if the event reaches this consumer, it must not be
	// acted on — the number is obsoleted, not returned to the pool.
	cancelled := cqrstest.NewMessage(cqrs.SubjectFromStr("NATHEJK:2026.order.order-1.cancelled"))
	if err := cancelled.SetBody(&messages.NathejkOrderCancelled{OrderID: "order-1"}); err != nil {
		t.Fatalf("set body: %v", err)
	}
	if err := consumer.HandleMessage(cancelled); err != nil {
		t.Fatalf("cancellation: %v", err)
	}

	if err := consumer.HandleMessage(orderPaidMsg(t, "order-2")); err != nil {
		t.Fatalf("second assignment: %v", err)
	}

	got := numbersPublished(t, pub)
	want := []string{"1", "2"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("numbers = %v, want %v — a cancelled number must not be reissued", got, want)
	}
}
