package patruljenumber

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jrgensen/cqrs"
	"github.com/jrgensen/cqrs/cqrstest"
	"github.com/nathejk/shared-go/messages"
	"github.com/nathejk/shared-go/tables"
	"github.com/nathejk/shared-go/tables/order"
	"github.com/nathejk/shared-go/tables/product"
	"github.com/nathejk/shared-go/types"
)

// --- fakes ---

// fakeOrders returns orders[call], last entry repeating, so a test can model a
// projection that lags and then catches up. A nil entry stands for an order the
// projector has not written yet. paid is what PaidQuantityBySKU reports.
type fakeOrders struct {
	orders []*order.Order
	paid   map[string]int
	err    error
	calls  int
}

func (f *fakeOrders) GetByID(context.Context, string) (*order.Order, error) {
	if f.err != nil {
		return nil, f.err
	}
	if len(f.orders) == 0 {
		return nil, tables.ErrRecordNotFound
	}
	i := f.calls
	if i > len(f.orders)-1 {
		i = len(f.orders) - 1
	}
	f.calls++
	if f.orders[i] == nil {
		return nil, tables.ErrRecordNotFound
	}
	return f.orders[i], nil
}

func (f *fakeOrders) PaidQuantityBySKU(context.Context, types.YearSlug, types.TeamType, string) (map[string]int, error) {
	return f.paid, nil
}

// fakeProducts is the 2026-shaped catalogue: one participation SKU plus
// merchandise, so tests prove merchandise does not count as seats.
type fakeProducts struct {
	products []product.Product
	err      error
}

func (f fakeProducts) ListEligibleFor(context.Context, types.YearSlug, types.TeamType) ([]product.Product, error) {
	return f.products, f.err
}

func catalogue() fakeProducts {
	return fakeProducts{products: []product.Product{
		{SKU: "participation.patrulje", Kind: product.KindParticipation},
		{SKU: "tshirt.adult", Kind: product.KindMerchandise},
	}}
}

// --- builders ---

func paidMsg(t *testing.T, orderID string) cqrs.Message {
	t.Helper()
	return paidMsgYear(t, "2026", orderID)
}

func paidMsgYear(t *testing.T, year, orderID string) cqrs.Message {
	t.Helper()
	m := cqrstest.NewMessage(cqrs.SubjectFromStr("NATHEJK:" + year + ".order." + orderID + ".paid"))
	if err := m.SetBody(&messages.NathejkOrderPaid{OrderID: orderID}); err != nil {
		t.Fatalf("set body: %v", err)
	}
	return m
}

func assignedMsg(t *testing.T, teamID, number string) cqrs.Message {
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

func paidPatruljeOrder(orderID, teamID string) *order.Order {
	return &order.Order{
		OrderID:   orderID,
		Year:      "2026",
		OwnerType: types.TeamTypePatrulje,
		OwnerID:   teamID,
		Status:    order.StatusPaid,
	}
}

// newTestSaga wires a saga with fakes, a recording sleep seam, and (by default)
// enough paid seats to qualify. It is live unless a test says otherwise.
func newTestSaga(orders *fakeOrders) (*saga, *cqrstest.Publisher, *[]time.Duration) {
	pub := &cqrstest.Publisher{}
	var slept []time.Duration
	s := New(pub, orders, catalogue(), "2026", 2*time.Second)
	s.sleep = func(d time.Duration) { slept = append(slept, d) }
	s.CaughtUp()
	return s, pub, &slept
}

func seats(n int) map[string]int {
	return map[string]int{"participation.patrulje": n}
}

// --- tests ---

// The live gate turns on this interface being discoverable: the jetstream layer
// finds it by runtime type assertion, through live.Notify's wrapper.
func TestImplementsCatchupListener(t *testing.T) {
	var c cqrs.Consumer = New(&cqrstest.Publisher{}, &fakeOrders{}, catalogue(), "2026", 0)
	if _, ok := c.(interface{ CaughtUp() }); !ok {
		t.Fatal("saga does not implement CaughtUp; the live gate would never open")
	}
}

// order.Queries must keep satisfying the narrow reader this saga declares, or the
// wiring in main.go stops compiling for a reason that is not obvious there.
func TestOrderQueriesSatisfiesOrderReader(t *testing.T) {
	var _ OrderReader = order.Queries(nil)
	var _ ProductReader = product.Queries(nil)
}

func TestConsumesTriggerAndOwnOutput(t *testing.T) {
	s := New(&cqrstest.Publisher{}, &fakeOrders{}, catalogue(), "2026", 0)

	var got []string
	for _, subj := range s.Consumes() {
		got = append(got, subj.Subject())
	}
	want := []string{"NATHEJK.*.order.*.paid", "NATHEJK.*.patrulje.*.numberassigned"}
	if len(got) != len(want) {
		t.Fatalf("Consumes() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Consumes()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

// The crux of the design: replaying a qualifying order must publish nothing, or
// every restart re-issues every number.
func TestPublishesNothingDuringReplay(t *testing.T) {
	pub := &cqrstest.Publisher{}
	s := New(pub, &fakeOrders{
		orders: []*order.Order{paidPatruljeOrder("order-1", "team-1")},
		paid:   seats(4),
	}, catalogue(), "2026", time.Millisecond)
	// Deliberately no CaughtUp.

	if err := s.HandleMessage(paidMsg(t, "order-1")); err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if len(pub.Messages) != 0 {
		t.Fatalf("published %v during replay, want nothing", pub.Subjects())
	}
}

func TestAssignsNextNumberWhenLive(t *testing.T) {
	s, pub, _ := newTestSaga(&fakeOrders{
		orders: []*order.Order{paidPatruljeOrder("order-1", "team-1")},
		paid:   seats(3),
	})

	if err := s.HandleMessage(paidMsg(t, "order-1")); err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if len(pub.Messages) != 1 {
		t.Fatalf("published %d events, want 1 (%v)", len(pub.Messages), pub.Subjects())
	}
	if got, want := pub.Subjects()[0], "NATHEJK.2026.patrulje.team-1.numberassigned"; got != want {
		t.Errorf("subject = %q, want %q", got, want)
	}
	var body messages.NathejkPatrolNumberAssigned
	if err := pub.Messages[0].Body(&body); err != nil {
		t.Fatalf("body: %v", err)
	}
	if body.TeamID != "team-1" {
		t.Errorf("TeamID = %q, want team-1", body.TeamID)
	}
	if body.TeamNumber != "1" {
		t.Errorf("TeamNumber = %q, want 1 (first of a fresh year)", body.TeamNumber)
	}
}

// A number seen on the stream is both an assignment and a high-water mark, which
// is what makes the sequence continue past manual numbers instead of colliding.
func TestReplayedAssignmentSetsTheFloor(t *testing.T) {
	s, pub, _ := newTestSaga(&fakeOrders{
		orders: []*order.Order{paidPatruljeOrder("order-1", "team-2")},
		paid:   seats(5),
	})

	if err := s.HandleMessage(assignedMsg(t, "team-1", "300")); err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if len(pub.Messages) != 0 {
		t.Fatalf("observing an assignment must publish nothing, got %v", pub.Subjects())
	}
	if err := s.HandleMessage(paidMsg(t, "order-1")); err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}

	var body messages.NathejkPatrolNumberAssigned
	if err := pub.Messages[0].Body(&body); err != nil {
		t.Fatalf("body: %v", err)
	}
	if body.TeamNumber != "301" {
		t.Errorf("TeamNumber = %q, want 301 (max seen + 1)", body.TeamNumber)
	}
}

func TestAlreadyAssignedFromHistoryIsNotReassigned(t *testing.T) {
	s, pub, _ := newTestSaga(&fakeOrders{
		orders: []*order.Order{paidPatruljeOrder("order-1", "team-1")},
		paid:   seats(7),
	})

	if err := s.HandleMessage(assignedMsg(t, "team-1", "12")); err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if err := s.HandleMessage(paidMsg(t, "order-1")); err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if len(pub.Messages) != 0 {
		t.Errorf("already-numbered team was assigned again: %v", pub.Subjects())
	}
}

// Idempotent within one process too: a second paid order for the same patrulje
// (a t-shirt bought later, say) must not hand out a second number.
func TestSecondPaidOrderDoesNotReassign(t *testing.T) {
	s, pub, _ := newTestSaga(&fakeOrders{
		orders: []*order.Order{paidPatruljeOrder("order-1", "team-1")},
		paid:   seats(4),
	})

	for i := 0; i < 2; i++ {
		if err := s.HandleMessage(paidMsg(t, "order-1")); err != nil {
			t.Fatalf("HandleMessage: %v", err)
		}
	}
	if len(pub.Messages) != 1 {
		t.Errorf("published %d events, want 1: %v", len(pub.Messages), pub.Subjects())
	}
}

func TestBelowMinSeatsIsNotAssignedAndDoesNotRetry(t *testing.T) {
	s, pub, slept := newTestSaga(&fakeOrders{
		orders: []*order.Order{paidPatruljeOrder("order-1", "team-1")},
		paid:   seats(MinSeats - 1),
	})

	if err := s.HandleMessage(paidMsg(t, "order-1")); err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if len(pub.Messages) != 0 {
		t.Errorf("under-seated patrulje was assigned: %v", pub.Subjects())
	}
	if len(*slept) != 0 {
		t.Errorf("an under-seated patrulje must be terminal, not retried: waited %v", *slept)
	}
}

// Merchandise is not a seat: four t-shirts and two seats is still two seats.
func TestMerchandiseDoesNotCountAsSeats(t *testing.T) {
	s, pub, _ := newTestSaga(&fakeOrders{
		orders: []*order.Order{paidPatruljeOrder("order-1", "team-1")},
		paid:   map[string]int{"participation.patrulje": 2, "tshirt.adult": 4},
	})

	if err := s.HandleMessage(paidMsg(t, "order-1")); err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if len(pub.Messages) != 0 {
		t.Errorf("t-shirts counted toward seats: %v", pub.Subjects())
	}
}

// The replay race: the order projector has not written status=paid yet. Reading
// again — after a wait, so the other consumer can advance — must find it.
func TestRetriesUntilOrderIsProjectedAsPaid(t *testing.T) {
	open := paidPatruljeOrder("order-1", "team-1")
	open.Status = order.StatusOpen

	s, pub, slept := newTestSaga(&fakeOrders{
		orders: []*order.Order{open, paidPatruljeOrder("order-1", "team-1")},
		paid:   seats(3),
	})

	if err := s.HandleMessage(paidMsg(t, "order-1")); err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if len(*slept) != 1 {
		t.Fatalf("want one wait so the order projector can catch up, got %v", *slept)
	}
	if want := 2 * time.Second / time.Duration(DefaultAttempts); (*slept)[0] != want {
		t.Errorf("wait = %v, want settle/attempts = %v", (*slept)[0], want)
	}
	if len(pub.Messages) != 1 {
		t.Errorf("want an assignment once the projection arrives, got %v", pub.Subjects())
	}
}

// Same race, one step earlier: the order row itself is not there yet.
func TestRetriesWhenOrderNotProjectedAtAll(t *testing.T) {
	s, pub, slept := newTestSaga(&fakeOrders{
		orders: []*order.Order{nil, paidPatruljeOrder("order-1", "team-1")},
		paid:   seats(3),
	})

	if err := s.HandleMessage(paidMsg(t, "order-1")); err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if len(*slept) != 1 {
		t.Fatalf("want one wait, got %v", *slept)
	}
	if len(pub.Messages) != 1 {
		t.Errorf("want an assignment once the order appears, got %v", pub.Subjects())
	}
}

func TestGivesUpAfterMaxAttempts(t *testing.T) {
	s, pub, slept := newTestSaga(&fakeOrders{orders: []*order.Order{nil}, paid: seats(3)})

	if err := s.HandleMessage(paidMsg(t, "order-1")); err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if len(*slept) != DefaultAttempts-1 {
		t.Errorf("want %d waits, got %d", DefaultAttempts-1, len(*slept))
	}
	if len(pub.Messages) != 0 {
		t.Errorf("no assignment expected, got %v", pub.Subjects())
	}
}

func TestNonPatruljeOwnerIsIgnored(t *testing.T) {
	klanOrder := paidPatruljeOrder("order-1", "team-1")
	klanOrder.OwnerType = types.TeamTypeKlan

	s, pub, slept := newTestSaga(&fakeOrders{orders: []*order.Order{klanOrder}, paid: seats(9)})

	if err := s.HandleMessage(paidMsg(t, "order-1")); err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if len(pub.Messages) != 0 {
		t.Errorf("a klan was assigned a patrulje number: %v", pub.Subjects())
	}
	if len(*slept) != 0 {
		t.Errorf("a non-patrulje owner must be terminal, waited %v", *slept)
	}
}

func TestOtherSeasonsAreIgnored(t *testing.T) {
	orders := &fakeOrders{
		orders: []*order.Order{paidPatruljeOrder("order-1", "team-1")},
		paid:   seats(5),
	}
	s, pub, _ := newTestSaga(orders)

	if err := s.HandleMessage(paidMsgYear(t, "2025", "order-1")); err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if len(pub.Messages) != 0 {
		t.Errorf("assigned for another season: %v", pub.Subjects())
	}
	if orders.calls != 0 {
		t.Errorf("read the order model for another season (%d calls)", orders.calls)
	}
}

// Two patruljer qualifying back to back must get distinct consecutive numbers,
// without waiting for the saga's own events to come back around.
func TestConsecutiveQualifiersGetDistinctNumbers(t *testing.T) {
	s, pub, _ := newTestSaga(&fakeOrders{
		orders: []*order.Order{
			paidPatruljeOrder("order-1", "team-1"),
			paidPatruljeOrder("order-2", "team-2"),
		},
		paid: seats(3),
	})

	if err := s.HandleMessage(paidMsg(t, "order-1")); err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if err := s.HandleMessage(paidMsg(t, "order-2")); err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if len(pub.Messages) != 2 {
		t.Fatalf("published %d events, want 2: %v", len(pub.Messages), pub.Subjects())
	}

	var got []string
	for _, m := range pub.Messages {
		var body messages.NathejkPatrolNumberAssigned
		if err := m.Body(&body); err != nil {
			t.Fatalf("body: %v", err)
		}
		got = append(got, body.TeamNumber)
	}
	if got[0] != "1" || got[1] != "2" {
		t.Errorf("numbers = %v, want [1 2]", got)
	}
}

// A read error is not evidence that a patrulje should stay unnumbered: return it
// so the message is dead-lettered rather than silently dropped.
func TestHardReadErrorIsReturned(t *testing.T) {
	boom := errors.New("boom")
	s, pub, _ := newTestSaga(&fakeOrders{err: boom})

	if err := s.HandleMessage(paidMsg(t, "order-1")); !errors.Is(err, boom) {
		t.Fatalf("HandleMessage error = %v, want %v", err, boom)
	}
	if len(pub.Messages) != 0 {
		t.Errorf("published despite a failed read: %v", pub.Subjects())
	}
}

// A non-numeric number still means "this team has one", even though it cannot
// raise the high-water mark.
func TestNonNumericAssignmentStillMarksAssigned(t *testing.T) {
	s, pub, _ := newTestSaga(&fakeOrders{
		orders: []*order.Order{paidPatruljeOrder("order-1", "team-1")},
		paid:   seats(4),
	})

	if err := s.HandleMessage(assignedMsg(t, "team-1", "A-7")); err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if err := s.HandleMessage(paidMsg(t, "order-1")); err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if len(pub.Messages) != 0 {
		t.Errorf("team with a non-numeric number was reassigned: %v", pub.Subjects())
	}
	if s.maxNumber != 0 {
		t.Errorf("maxNumber = %d, want 0 (a non-numeric number has no value)", s.maxNumber)
	}
}
