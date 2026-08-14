package patruljenumber

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jrgensen/cqrs"
	"github.com/jrgensen/cqrs/cqrstest"
	"github.com/nathejk/shared-go/messages"
	"github.com/nathejk/shared-go/tables"
	"github.com/nathejk/shared-go/tables/order"
	"github.com/nathejk/shared-go/tables/product"
	"github.com/nathejk/shared-go/types"
	"nathejk.dk/nathejk/table/patrulje"
)

// --- fakes ---

// fakeOrders returns orders[call], last entry repeating, so a test can model a
// projection that lags and then catches up. A nil entry stands for an order the
// projector has not written yet. byOwner is what ListByOwner reports, keyed by
// owner id; owned() builds the common single-order case.
type fakeOrders struct {
	orders  []*order.Order
	byOwner map[string][]order.Order
	err     error
	calls   int
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

func (f *fakeOrders) ListByOwner(_ context.Context, _ types.YearSlug, _ types.TeamType, ownerID string) ([]order.Order, error) {
	if f.err != nil {
		return nil, f.err
	}
	if rows, ok := f.byOwner[ownerID]; ok {
		return rows, nil
	}
	// anyOwner lets a test say "every patrulje has n paid seats" without naming
	// them; tests that care who paid what supply an explicit map.
	return f.byOwner[anyOwner], nil
}

const anyOwner = "*"

// seatOrder is a paid order carrying n seats, paid at the given time.
func seatOrder(teamID string, n int, paidAt string) order.Order {
	return order.Order{
		OrderID:   "o-" + teamID,
		Year:      "2026",
		OwnerType: types.TeamTypePatrulje,
		OwnerID:   teamID,
		Status:    order.StatusPaid,
		ChangedAt: paidAt,
		Lines: []order.Line{
			{ProductSKU: "participation.patrulje", Quantity: n},
		},
	}
}

// owned is the single-paid-order case: this team has n paid seats.
func owned(teamID string, n int) map[string][]order.Order {
	return map[string][]order.Order{teamID: {seatOrder(teamID, n, "2026-07-01 10:00:00")}}
}

// mixedOrder pairs a couple of seats with a pile of t-shirts, to prove that only
// the seats count.
func mixedOrder(teamID string) order.Order {
	o := seatOrder(teamID, 2, "2026-07-01 10:00:00")
	o.Lines = append(o.Lines, order.Line{ProductSKU: "tshirt.adult", Quantity: 4})
	return o
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

// fakePatruljer is the patrulje read model: the numbers that exist without ever
// having been an event.
type fakePatruljer struct {
	rows []patrulje.Patrulje
	err  error
	year types.YearSlug
}

func (f *fakePatruljer) GetAll(_ context.Context, filter patrulje.Filter) ([]patrulje.Patrulje, error) {
	f.year = filter.YearSlug
	return f.rows, f.err
}

func numbered(teamID, number string) patrulje.Patrulje {
	return patrulje.Patrulje{TeamID: types.TeamID(teamID), TeamNumber: number}
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

// newTestSaga wires a saga with fakes, a recording sleep seam, and an empty
// patrulje read model. It is live unless a test says otherwise.
func newTestSaga(orders *fakeOrders) (*saga, *cqrstest.Publisher, *[]time.Duration) {
	return newTestSagaWith(orders, &fakePatruljer{})
}

func newTestSagaWith(orders *fakeOrders, patruljer PatruljeReader) (*saga, *cqrstest.Publisher, *[]time.Duration) {
	pub := &cqrstest.Publisher{}
	var slept []time.Duration
	s := New(pub, orders, catalogue(), patruljer, "2026", 2*time.Second)
	s.sleep = func(d time.Duration) { slept = append(slept, d) }
	s.CaughtUp()
	return s, pub, &slept
}

// seats says every patrulje has n paid seats, which is what most tests need.
func seats(n int) map[string][]order.Order {
	return map[string][]order.Order{anyOwner: {seatOrder(anyOwner, n, "2026-07-01 10:00:00")}}
}

// --- tests ---

// The live gate turns on this interface being discoverable: the jetstream layer
// finds it by runtime type assertion, through live.Notify's wrapper.
func TestImplementsCatchupListener(t *testing.T) {
	var c cqrs.Consumer = New(&cqrstest.Publisher{}, &fakeOrders{}, catalogue(), &fakePatruljer{}, "2026", 0)
	if _, ok := c.(interface{ CaughtUp() }); !ok {
		t.Fatal("saga does not implement CaughtUp; the live gate would never open")
	}
}

// order.Queries must keep satisfying the narrow reader this saga declares, or the
// wiring in main.go stops compiling for a reason that is not obvious there.
func TestReadModelsSatisfyTheReaders(t *testing.T) {
	var _ OrderReader = order.Queries(nil)
	var _ ProductReader = product.Queries(nil)
	var _ PatruljeReader = patrulje.Queries(nil)
}

func TestConsumesTriggerAndOwnOutput(t *testing.T) {
	s := New(&cqrstest.Publisher{}, &fakeOrders{}, catalogue(), &fakePatruljer{}, "2026", 0)

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
		orders:  []*order.Order{paidPatruljeOrder("order-1", "team-1")},
		byOwner: seats(4),
	}, catalogue(), &fakePatruljer{}, "2026", time.Millisecond)
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
		orders:  []*order.Order{paidPatruljeOrder("order-1", "team-1")},
		byOwner: seats(3),
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
		orders:  []*order.Order{paidPatruljeOrder("order-1", "team-2")},
		byOwner: seats(5),
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
		orders:  []*order.Order{paidPatruljeOrder("order-1", "team-1")},
		byOwner: seats(7),
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
		orders:  []*order.Order{paidPatruljeOrder("order-1", "team-1")},
		byOwner: seats(4),
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
		orders:  []*order.Order{paidPatruljeOrder("order-1", "team-1")},
		byOwner: seats(MinSeats - 1),
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
		orders:  []*order.Order{paidPatruljeOrder("order-1", "team-1")},
		byOwner: map[string][]order.Order{"team-1": {mixedOrder("team-1")}},
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
		orders:  []*order.Order{open, paidPatruljeOrder("order-1", "team-1")},
		byOwner: seats(3),
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
		orders:  []*order.Order{nil, paidPatruljeOrder("order-1", "team-1")},
		byOwner: seats(3),
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
	s, pub, slept := newTestSaga(&fakeOrders{orders: []*order.Order{nil}, byOwner: seats(3)})

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

	s, pub, slept := newTestSaga(&fakeOrders{orders: []*order.Order{klanOrder}, byOwner: seats(9)})

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
		orders:  []*order.Order{paidPatruljeOrder("order-1", "team-1")},
		byOwner: seats(5),
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
		byOwner: map[string][]order.Order{
			"team-1": {seatOrder("team-1", 3, "2026-07-01 10:00:00")},
			"team-2": {seatOrder("team-2", 3, "2026-07-02 10:00:00")},
		},
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

// --- seeding from the read model (task 058) ---

// The headline case from PRD 003: a manual 300 and no history means the next
// automatic number is 301, not 1.
func TestSeedsHighWaterMarkFromExistingTeamNumbers(t *testing.T) {
	s, pub, _ := newTestSagaWith(
		&fakeOrders{
			orders:  []*order.Order{paidPatruljeOrder("order-1", "team-2")},
			byOwner: seats(3),
		},
		&fakePatruljer{rows: []patrulje.Patrulje{numbered("team-1", "300")}},
	)

	if err := s.HandleMessage(paidMsg(t, "order-1")); err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if len(pub.Messages) != 1 {
		t.Fatalf("published %d events, want 1: %v", len(pub.Messages), pub.Subjects())
	}
	var body messages.NathejkPatrolNumberAssigned
	if err := pub.Messages[0].Body(&body); err != nil {
		t.Fatalf("body: %v", err)
	}
	if body.TeamNumber != "301" {
		t.Errorf("TeamNumber = %q, want 301 (existing 300 + 1)", body.TeamNumber)
	}
}

// A patrulje numbered by hand has no numberassigned event, so only the read model
// can say it is spoken for.
func TestSeedMarksExistingNumberedTeamAsAssigned(t *testing.T) {
	s, pub, _ := newTestSagaWith(
		&fakeOrders{
			orders:  []*order.Order{paidPatruljeOrder("order-1", "team-1")},
			byOwner: seats(5),
		},
		&fakePatruljer{rows: []patrulje.Patrulje{numbered("team-1", "42")}},
	)

	if err := s.HandleMessage(paidMsg(t, "order-1")); err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if len(pub.Messages) != 0 {
		t.Errorf("a manually numbered patrulje was numbered again: %v", pub.Subjects())
	}
}

func TestSeedSkipsEmptyTeamNumbers(t *testing.T) {
	// team-9 is unnumbered but has no paid orders, so the catch-up sweep leaves it
	// alone; team-1 is not in the read model at all and qualifies live. That keeps
	// this test about the empty number, not about the backfill.
	s, pub, _ := newTestSagaWith(
		&fakeOrders{
			orders:  []*order.Order{paidPatruljeOrder("order-1", "team-1")},
			byOwner: map[string][]order.Order{"team-1": {seatOrder("team-1", 3, "2026-07-01 10:00:00")}},
		},
		&fakePatruljer{rows: []patrulje.Patrulje{numbered("team-9", "")}},
	)

	if err := s.HandleMessage(paidMsg(t, "order-1")); err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if len(pub.Messages) != 1 {
		t.Fatalf("an unnumbered patrulje must still be assignable: %v", pub.Subjects())
	}
	var body messages.NathejkPatrolNumberAssigned
	if err := pub.Messages[0].Body(&body); err != nil {
		t.Fatalf("body: %v", err)
	}
	if body.TeamNumber != "1" {
		t.Errorf("TeamNumber = %q, want 1 — empty numbers must not count as 0 or raise the mark", body.TeamNumber)
	}
}

func TestSeedSkipsNonNumericTeamNumbersButMarksThemAssigned(t *testing.T) {
	s, _, _ := newTestSagaWith(
		&fakeOrders{},
		&fakePatruljer{rows: []patrulje.Patrulje{numbered("team-1", "A-7")}},
	)

	if s.maxNumber != 0 {
		t.Errorf("maxNumber = %d, want 0 (a non-numeric number has no value)", s.maxNumber)
	}
	if !s.isAssigned("team-1") {
		t.Error("a team with a non-numeric number must still count as numbered")
	}
}

// The mark is the highest of both sources, whichever side happens to hold it.
func TestSeedTakesTheHighestOfEventsAndReadModel(t *testing.T) {
	for _, tc := range []struct {
		name        string
		fromEvent   string
		inReadModel string
		want        string
	}{
		{"read model higher", "5", "300", "301"},
		{"event higher", "300", "5", "301"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, pub, _ := newTestSagaWith(
				&fakeOrders{
					orders:  []*order.Order{paidPatruljeOrder("order-1", "team-new")},
					byOwner: seats(3),
				},
				&fakePatruljer{rows: []patrulje.Patrulje{numbered("team-2", tc.inReadModel)}},
			)

			if err := s.HandleMessage(assignedMsg(t, "team-1", tc.fromEvent)); err != nil {
				t.Fatalf("HandleMessage: %v", err)
			}
			if err := s.HandleMessage(paidMsg(t, "order-1")); err != nil {
				t.Fatalf("HandleMessage: %v", err)
			}
			var body messages.NathejkPatrolNumberAssigned
			if err := pub.Messages[0].Body(&body); err != nil {
				t.Fatalf("body: %v", err)
			}
			if body.TeamNumber != tc.want {
				t.Errorf("TeamNumber = %q, want %q", body.TeamNumber, tc.want)
			}
		})
	}
}

// A failed seed must not open the gate: publishing with a too-low mark would hand
// a patrulje a number another one already has, and the projector's UPDATE is
// unconditional, so both would keep it.
func TestFailedSeedLeavesTheSagaDormant(t *testing.T) {
	s, pub, _ := newTestSagaWith(
		&fakeOrders{
			orders:  []*order.Order{paidPatruljeOrder("order-1", "team-1")},
			byOwner: seats(4),
		},
		&fakePatruljer{err: errors.New("database on fire")},
	)

	if s.live.Load() {
		t.Error("saga went live despite a failed seed")
	}
	if err := s.HandleMessage(paidMsg(t, "order-1")); err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if len(pub.Messages) != 0 {
		t.Errorf("published while dormant: %v", pub.Subjects())
	}
}

// Numbers reset per year, so the seed must not drag another season's numbers in.
func TestSeedReadsOnlyOurSeason(t *testing.T) {
	patruljer := &fakePatruljer{}
	newTestSagaWith(&fakeOrders{}, patruljer)

	if patruljer.year != "2026" {
		t.Errorf("seeded with YearSlug %q, want 2026", patruljer.year)
	}
}

// CaughtUp seeds state, and the jetstream subscribe path may call it from its own
// goroutine while messages are being delivered. Run with -race.
func TestSeedingIsSafeAlongsideMessageHandling(t *testing.T) {
	pub := &cqrstest.Publisher{}
	s := New(pub,
		&fakeOrders{
			orders:  []*order.Order{paidPatruljeOrder("order-1", "team-1")},
			byOwner: seats(4),
		},
		catalogue(),
		&fakePatruljer{rows: []patrulje.Patrulje{numbered("team-7", "77")}},
		"2026", time.Millisecond)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		s.CaughtUp()
	}()
	go func() {
		defer wg.Done()
		if err := s.HandleMessage(paidMsg(t, "order-1")); err != nil {
			t.Errorf("HandleMessage: %v", err)
		}
	}()
	wg.Wait()
}

// A non-numeric number arriving as an event still means "this team has one", even
// though it cannot raise the high-water mark.
func TestNonNumericAssignmentStillMarksAssigned(t *testing.T) {
	s, pub, _ := newTestSaga(&fakeOrders{
		orders:  []*order.Order{paidPatruljeOrder("order-1", "team-1")},
		byOwner: seats(4),
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

// --- catch-up backfill (task 061) ---

// The gap 060 found: patruljer that qualified before this saga existed have no
// future event to trigger on, so the sweep at catch-up is the only thing that ever
// numbers them.
func TestBackfillNumbersAlreadyQualifyingPatruljer(t *testing.T) {
	orders := &fakeOrders{byOwner: map[string][]order.Order{
		"team-late":  {seatOrder("team-late", 4, "2026-07-30 15:00:00")},
		"team-early": {seatOrder("team-early", 6, "2026-06-01 09:00:00")},
		"team-mid":   {seatOrder("team-mid", 3, "2026-07-01 12:00:00")},
	}}
	// Deliberately not in payment order, to prove the sweep sorts rather than
	// following whatever the read model returned.
	rows := []patrulje.Patrulje{
		numbered("team-mid", ""),
		numbered("team-late", ""),
		numbered("team-early", ""),
	}

	_, pub, _ := newTestSagaWith(orders, &fakePatruljer{rows: rows})

	// Earliest payment first, and consecutive from 1.
	want := []string{
		"NATHEJK.2026.patrulje.team-early.numberassigned",
		"NATHEJK.2026.patrulje.team-mid.numberassigned",
		"NATHEJK.2026.patrulje.team-late.numberassigned",
	}
	got := pub.Subjects()
	if len(got) != len(want) {
		t.Fatalf("published %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("published %v, want %v", got, want)
		}
	}
	if numbers := numbersOf(t, pub); numbers != "1,2,3" {
		t.Errorf("numbers = %s, want 1,2,3", numbers)
	}
}

// Ordering must not depend on the order the read model happened to return, or a
// restart would hand out different numbers.
func TestBackfillOrderIsStableRegardlessOfInputOrder(t *testing.T) {
	orders := &fakeOrders{byOwner: map[string][]order.Order{
		"team-a": {seatOrder("team-a", 3, "2026-06-01 09:00:00")},
		"team-b": {seatOrder("team-b", 3, "2026-06-02 09:00:00")},
		"team-c": {seatOrder("team-c", 3, "2026-06-03 09:00:00")},
	}}

	first := ""
	for _, rows := range [][]patrulje.Patrulje{
		{numbered("team-a", ""), numbered("team-b", ""), numbered("team-c", "")},
		{numbered("team-c", ""), numbered("team-a", ""), numbered("team-b", "")},
		{numbered("team-b", ""), numbered("team-c", ""), numbered("team-a", "")},
	} {
		_, pub, _ := newTestSagaWith(orders, &fakePatruljer{rows: rows})
		got := strings.Join(pub.Subjects(), " ")
		if first == "" {
			first = got
			continue
		}
		if got != first {
			t.Fatalf("assignment order depends on input order:\n %s\n %s", first, got)
		}
	}
}

// Ties on payment time still have to be total, or the same two patruljer could
// swap numbers between restarts.
func TestBackfillBreaksTiesOnTeamID(t *testing.T) {
	same := "2026-06-01 09:00:00"
	orders := &fakeOrders{byOwner: map[string][]order.Order{
		"team-b": {seatOrder("team-b", 3, same)},
		"team-a": {seatOrder("team-a", 3, same)},
	}}

	_, pub, _ := newTestSagaWith(orders, &fakePatruljer{rows: []patrulje.Patrulje{
		numbered("team-b", ""),
		numbered("team-a", ""),
	}})

	if got, want := pub.Subjects()[0], "NATHEJK.2026.patrulje.team-a.numberassigned"; got != want {
		t.Errorf("first assignment = %q, want %q", got, want)
	}
}

func TestBackfillSkipsAlreadyNumberedAndIneligible(t *testing.T) {
	orders := &fakeOrders{byOwner: map[string][]order.Order{
		"team-numbered": {seatOrder("team-numbered", 5, "2026-06-01 09:00:00")},
		"team-short":    {seatOrder("team-short", MinSeats-1, "2026-06-02 09:00:00")},
		"team-ok":       {seatOrder("team-ok", 3, "2026-06-03 09:00:00")},
		"team-unpaid":   {openOrder("team-unpaid", 7)},
	}}

	_, pub, _ := newTestSagaWith(orders, &fakePatruljer{rows: []patrulje.Patrulje{
		numbered("team-numbered", "12"),
		numbered("team-short", ""),
		numbered("team-unpaid", ""),
		numbered("team-ok", ""),
	}})

	if got, want := len(pub.Messages), 1; got != want {
		t.Fatalf("published %d events, want %d: %v", got, want, pub.Subjects())
	}
	if got, want := pub.Subjects()[0], "NATHEJK.2026.patrulje.team-ok.numberassigned"; got != want {
		t.Errorf("subject = %q, want %q", got, want)
	}
	// 12 is taken, so the sweep continues from it rather than restarting at 1.
	if numbers := numbersOf(t, pub); numbers != "13" {
		t.Errorf("number = %s, want 13 (continuing past the manual 12)", numbers)
	}
}

// A restart replays everything and sweeps again; the second sweep must be a no-op,
// which is what stops the backfill renumbering on every deploy.
func TestBackfillIsIdempotentAcrossRestarts(t *testing.T) {
	orders := &fakeOrders{byOwner: map[string][]order.Order{
		"team-1": {seatOrder("team-1", 4, "2026-06-01 09:00:00")},
	}}

	_, pub, _ := newTestSagaWith(orders, &fakePatruljer{
		rows: []patrulje.Patrulje{numbered("team-1", "")},
	})
	if len(pub.Messages) != 1 {
		t.Fatalf("first start: published %v, want one assignment", pub.Subjects())
	}
	assignedNumber := numbersOf(t, pub)

	// Second start: the projector has since written the number, so the read model
	// now reports it — exactly what the next process would see.
	_, pub2, _ := newTestSagaWith(orders, &fakePatruljer{
		rows: []patrulje.Patrulje{numbered("team-1", assignedNumber)},
	})
	if len(pub2.Messages) != 0 {
		t.Errorf("second start re-numbered: %v", pub2.Subjects())
	}
}

// A failed sweep must not cost the saga its live gate: unlike a failed seed it
// cannot cause a duplicate number, so new payments should still be numbered.
func TestFailedBackfillLeavesTheSagaLive(t *testing.T) {
	// GetAll succeeds (so the seed is fine) but the order reads fail.
	orders := &fakeOrders{err: errors.New("orders unavailable")}
	pub := &cqrstest.Publisher{}
	s := New(pub, orders, catalogue(),
		&fakePatruljer{rows: []patrulje.Patrulje{numbered("team-1", "")}},
		"2026", time.Millisecond)
	s.CaughtUp()

	if !s.live.Load() {
		t.Fatal("a failed backfill must not shut the live gate")
	}
	if len(pub.Messages) != 0 {
		t.Errorf("published despite failing reads: %v", pub.Subjects())
	}
}

// numbersOf returns the published numbers as a comma-joined string, for compact
// assertions.
func numbersOf(t *testing.T, pub *cqrstest.Publisher) string {
	t.Helper()
	var out []string
	for _, m := range pub.Messages {
		var body messages.NathejkPatrolNumberAssigned
		if err := m.Body(&body); err != nil {
			t.Fatalf("body: %v", err)
		}
		out = append(out, body.TeamNumber)
	}
	return strings.Join(out, ",")
}

// openOrder is a paid-for-nothing order: seats requested but not paid.
func openOrder(teamID string, n int) order.Order {
	o := seatOrder(teamID, n, "2026-06-01 09:00:00")
	o.Status = order.StatusOpen
	return o
}
