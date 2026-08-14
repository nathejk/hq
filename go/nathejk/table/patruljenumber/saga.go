// Package patruljenumber assigns running team numbers to accepted patruljer.
//
// A patrulje is "accepted" once it has paid for at least MinSeats seats. At that
// point it gets the next number for the year — max assigned so far + 1 — which is
// published as NATHEJK.{year}.patrulje.{teamId}.numberassigned and projected into
// patrulje.teamNumber by the patrulje projector. See PRD 003.
//
// # Why a saga and not a projector
//
// The number is not derivable from any single event: it depends on every number
// already handed out that year. So this consumer keeps that running state in
// memory, rebuilt from the event log on every start, and publishes a new event
// when a patrulje becomes eligible.
//
// That makes it a saga, with the consequence that **exactly one service may mount
// it**. Subscriptions are ephemeral ordered consumers with no queue group, so
// every process receives every message rather than sharing them out; two mounts
// would both see an unnumbered patrulje and both publish. The patrulje
// projector's UPDATE is unconditional, so duplicates would not converge on one
// number — they would overwrite each other. hq owns this saga; tilmelding owns
// the order Pay saga. Neither mounts the other's, and neither can be scaled to
// two replicas without breaking this.
//
// # Replay
//
// Publishing during catch-up would re-issue every number on every restart, so
// nothing is published until CaughtUp fires. Note that CaughtUp only arrives
// because live.Notify forwards it: hq wraps every consumer, and the jetstream
// subscribe path discovers the interface by asserting on the handler it is given
// — the wrapper. See internal/live/notify.go.
package patruljenumber

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jrgensen/cqrs"
	"github.com/nathejk/shared-go/messages"
	"github.com/nathejk/shared-go/tables"
	"github.com/nathejk/shared-go/tables/order"
	"github.com/nathejk/shared-go/tables/product"
	"github.com/nathejk/shared-go/types"
	"nathejk.dk/nathejk/table/patrulje"
)

// MinSeats is how many paid seats make a patrulje accepted. A patrulje is 3–7
// people, so three is "enough of a team to come".
const MinSeats = 3

// DefaultSettle is the total budget spent waiting for the order projection to
// reflect an order this saga was just told is paid. Spread across
// DefaultAttempts reads rather than paid up front, so the common case — the
// projection is already current — costs nothing.
const DefaultSettle = 2 * time.Second

// DefaultAttempts is how many times a trigger is re-read before giving up on an
// order that has not been projected as paid.
const DefaultAttempts = 5

// OrderReader is the slice of the order read API this saga needs: the order the
// trigger names, and the paid quantity per SKU for its owner.
//
// Declared here rather than taking order.Queries wholesale so the saga can be
// tested without a database, and so it states exactly what it reads.
// order.Queries satisfies it.
type OrderReader interface {
	GetByID(ctx context.Context, orderID string) (*order.Order, error)
	PaidQuantityBySKU(ctx context.Context, year types.YearSlug, ownerType types.TeamType, ownerID string) (map[string]int, error)
}

// ProductReader answers which SKUs count as seats.
type ProductReader interface {
	ListEligibleFor(ctx context.Context, year types.YearSlug, ownerType types.TeamType) ([]product.Product, error)
}

// PatruljeReader supplies the numbers that already exist in the read model but
// were never announced as events — assigned by hand, or written directly by
// another projector (klan/consumer.go inserts patrulje rows carrying a
// teamNumber). Replaying numberassigned alone would not see them.
type PatruljeReader interface {
	GetAll(ctx context.Context, f patrulje.Filter) ([]patrulje.Patrulje, error)
}

type saga struct {
	p         cqrs.Publisher
	orders    OrderReader
	products  ProductReader
	patruljer PatruljeReader
	year      types.YearSlug

	settle   time.Duration
	attempts int

	// mu guards assigned and maxNumber. HandleMessage is called from a single
	// goroutine per consumer, so messages alone would need no lock — but CaughtUp
	// seeds the same state, and the jetstream subscribe path may call it from its
	// own goroutine (immediately, when a consumer starts with no backlog).
	// Contention is nil: one writer plus a one-shot seed.
	mu        sync.Mutex
	assigned  map[types.TeamID]bool
	maxNumber int

	live atomic.Bool

	// sleep is a test seam; nil means time.Sleep.
	sleep func(time.Duration)
}

// New wires the number-assignment saga.
//
// year is the season it assigns for; events from other seasons are ignored, since
// previous years are closed and their numbering is history. settle 0 takes
// DefaultSettle.
func New(p cqrs.Publisher, orders OrderReader, products ProductReader, patruljer PatruljeReader, year types.YearSlug, settle time.Duration) *saga {
	if settle <= 0 {
		settle = DefaultSettle
	}
	return &saga{
		p:         p,
		orders:    orders,
		products:  products,
		patruljer: patruljer,
		year:      year,
		settle:    settle,
		attempts:  DefaultAttempts,
		assigned:  map[types.TeamID]bool{},
	}
}

// The saga fills both roles the wiring needs: a consumer for the mux, and a
// catch-up listener so the live gate can open.
var (
	_ cqrs.Consumer           = (*saga)(nil)
	_ interface{ CaughtUp() } = (*saga)(nil)
)

// CaughtUp marks the saga live: history has been replayed, so subsequent events
// are new and assignments may be published.
//
// Before opening the gate it folds in the numbers that exist in the read model
// but never appeared as events. If that read fails the gate stays shut and this
// saga publishes nothing for the rest of the process. That is the deliberately
// conservative choice: a patrulje left unnumbered is fixed by the next restart,
// whereas re-issuing a number already held by another patrulje is not — the
// projector's UPDATE is unconditional, so the two teams would simply share it.
func (s *saga) CaughtUp() {
	if err := s.seedFromReadModel(); err != nil {
		log.Printf("patruljenumber: seeding existing team numbers failed, staying dormant rather than risk re-issuing a number: %v", err)
		return
	}
	s.live.Store(true)
}

// seedTimeout bounds the one seeding read. Generous: it runs once at startup, and
// timing out costs the saga the whole process.
const seedTimeout = 30 * time.Second

// seedFromReadModel folds every existing teamNumber for the year into the running
// state: the team counts as assigned, and its number raises the high-water mark.
// PRD 003's example — a manual 300 means the next automatic number is 301, not 1.
//
// Timing: this runs at catch-up rather than at construction because hq rebuilds
// its read model from the stream on every start, so at construction the patrulje
// table is empty.
//
// Catch-up of *this* consumer does not prove the patrulje projector has finished
// its own replay — they are independent consumers — so the mark seeded here can
// be too low. Two things keep that from issuing a duplicate: the assigned set is
// also fed by replayed numberassigned events, and every number this saga issues
// is observed on the way back, raising the mark again.
func (s *saga) seedFromReadModel() error {
	ctx, cancel := context.WithTimeout(context.Background(), seedTimeout)
	defer cancel()

	patruljer, err := s.patruljer.GetAll(ctx, patrulje.Filter{YearSlug: s.year})
	if err != nil {
		return err
	}
	for _, p := range patruljer {
		if p.TeamID == "" || p.TeamNumber == "" {
			continue
		}
		s.markAssigned(p.TeamID, p.TeamNumber)
	}
	return nil
}

func (s *saga) Consumes() []cqrs.Subject {
	return []cqrs.Subject{
		// The trigger.
		cqrs.SubjectFromStr("NATHEJK:*.order.*.paid"),
		// Its own output, so a replay rebuilds "who is numbered" and "highest
		// number used" — without which the first live assignment would collide.
		cqrs.SubjectFromStr("NATHEJK:*.patrulje.*.numberassigned"),
	}
}

func (s *saga) HandleMessage(msg cqrs.Message) error {
	if !s.forOurSeason(msg.Subject()) {
		return nil
	}
	switch {
	case msg.Subject().Match("NATHEJK.*.patrulje.*.numberassigned"):
		return s.observeAssignment(msg)
	case msg.Subject().Match("NATHEJK.*.order.*.paid"):
		return s.considerOrder(msg)
	default:
		return nil
	}
}

// forOurSeason reports whether an event belongs to the season this saga numbers.
//
// Subjects are NATHEJK.<year>.<entity>...; cqrs normalises the domain separator,
// so the year is always Parts()[1]. An empty year means "every season", which is
// only useful in tests.
func (s *saga) forOurSeason(subj cqrs.Subject) bool {
	if s.year == "" {
		return true
	}
	parts := subj.Parts()
	if len(parts) < 2 {
		return false
	}
	return parts[1] == string(s.year)
}

// observeAssignment folds an assignment into the running state. It publishes
// nothing — including for the saga's own events coming back around, which is how
// an optimistic local assignment gets confirmed.
func (s *saga) observeAssignment(msg cqrs.Message) error {
	var body messages.NathejkPatrolNumberAssigned
	if err := msg.Body(&body); err != nil {
		return err
	}
	if body.TeamID == "" {
		return nil
	}
	s.markAssigned(body.TeamID, body.TeamNumber)
	return nil
}

// markAssigned records that a team holds a number and raises the high-water mark.
//
// A non-numeric number still marks the team assigned — it is a number somebody
// meant — but cannot raise a mark it has no value for.
func (s *saga) markAssigned(teamID types.TeamID, number string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.markAssignedLocked(teamID, number)
}

func (s *saga) markAssignedLocked(teamID types.TeamID, number string) {
	s.assigned[teamID] = true
	if n, err := strconv.Atoi(number); err == nil && n > s.maxNumber {
		s.maxNumber = n
	}
}

// isAssigned reports whether the team already holds a number.
func (s *saga) isAssigned(teamID types.TeamID) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.assigned[teamID]
}

// outcome is what one pass at a triggering order found, and so whether reading
// again could change the answer.
type outcome int

const (
	// settled: nothing more to do — already numbered, not a patrulje, published,
	// or genuinely not yet eligible.
	settled outcome = iota

	// unprojected: the order is not in the read model as paid yet, though the
	// event saying so is on the stream. Only another consumer advancing fixes it,
	// so this is worth re-reading.
	unprojected
)

func (s *saga) considerOrder(msg cqrs.Message) error {
	var body messages.NathejkOrderPaid
	if err := msg.Body(&body); err != nil {
		return err
	}
	if body.OrderID == "" {
		return nil
	}

	// Reading up to attempts times rather than once: PaidQuantityBySKU counts
	// lines on orders the order projector has already written as paid, and that
	// projector is an independent consumer. Right after order.paid the seat count
	// can therefore read as zero for an order that is in fact paid — and nothing
	// would ever trigger this saga for that order again.
	attempts := s.attempts
	if attempts < 1 {
		attempts = 1
	}
	wait := s.settle / time.Duration(attempts)
	for i := 0; i < attempts; i++ {
		if i > 0 {
			// Waiting even during replay: the only thing that resolves an
			// unprojected order is another consumer making progress, and
			// back-to-back reads in this goroutine give it no chance to. The cost
			// is self-limiting — the order projector uses the pause to get ahead
			// of this consumer and stays there.
			s.nap(wait)
		}
		res, err := s.attempt(body.OrderID)
		if err != nil {
			return err
		}
		if res == settled {
			return nil
		}
	}
	// The order never showed up as paid. It resolves on a later replay, since the
	// events are on the stream permanently, but until then the patrulje is
	// unnumbered despite having paid — worth saying out loud.
	log.Printf("patruljenumber: order %s still not projected as paid after %d attempts; its patrulje may be unnumbered until the next replay", body.OrderID, attempts)
	return nil
}

func (s *saga) attempt(orderID string) (outcome, error) {
	ctx := context.Background()

	o, err := s.orders.GetByID(ctx, orderID)
	switch {
	case errors.Is(err, tables.ErrRecordNotFound):
		// We are reacting to this order's own paid event, so the order exists on
		// the stream; the projector is simply behind.
		return unprojected, nil
	case err != nil:
		// A failed read is not evidence that a patrulje should stay unnumbered.
		// Returning it dead-letters the message rather than dropping it.
		return settled, err
	}
	if o.OwnerType != types.TeamTypePatrulje {
		return settled, nil
	}
	teamID := types.TeamID(o.OwnerID)
	if teamID == "" {
		return settled, nil
	}
	// Cheap and decisive: a team that already holds a number is never given a
	// second one, live or not.
	if s.isAssigned(teamID) {
		return settled, nil
	}
	if o.Status != order.StatusPaid {
		// The paid event is on the stream but not yet in the read model, so the
		// seat count below would under-count. Retryable.
		return unprojected, nil
	}

	seats, err := s.seatCount(ctx, o.Year, teamID)
	if err != nil {
		return settled, err
	}
	if seats < MinSeats {
		// Genuinely short of a team. Terminal rather than retryable: a two-seat
		// patrulje would otherwise burn the whole budget on every event it
		// triggers. It may qualify later, on its next paid order.
		return settled, nil
	}

	if !s.live.Load() {
		// Replaying. Everything above still ran — cheaply, and it keeps the
		// eligibility path exercised — but publishing here would re-issue every
		// number on every restart.
		return settled, nil
	}

	// Publish first, then record locally. The saga's own subscription will
	// observe the event on the way back, but two patruljer qualifying in quick
	// succession must get distinct numbers without waiting for that round trip.
	//
	// Held under the lock across the publish so the number cannot be handed out
	// twice. Locking across I/O is normally worth avoiding; here the only other
	// contender is the one-shot seed, so there is nothing to contend with.
	s.mu.Lock()
	defer s.mu.Unlock()
	number := s.maxNumber + 1
	if err := s.publish(teamID, number); err != nil {
		return settled, err
	}
	s.markAssignedLocked(teamID, strconv.Itoa(number))
	return settled, nil
}

// seatCount is the patrulje's paid seats: the quantity it has paid for across the
// year's participation products.
//
// The SKUs come from the catalogue rather than a constant, so renaming or
// splitting the participation product does not silently stop acceptance from
// working. Merchandise is excluded — a patrulje that buys four t-shirts has not
// thereby paid for four seats.
func (s *saga) seatCount(ctx context.Context, year types.YearSlug, teamID types.TeamID) (int, error) {
	products, err := s.products.ListEligibleFor(ctx, year, types.TeamTypePatrulje)
	if err != nil {
		return 0, err
	}
	paid, err := s.orders.PaidQuantityBySKU(ctx, year, types.TeamTypePatrulje, string(teamID))
	if err != nil {
		return 0, err
	}
	seats := 0
	for _, p := range products {
		if p.Kind != product.KindParticipation {
			continue
		}
		seats += paid[p.SKU]
	}
	return seats, nil
}

func (s *saga) publish(teamID types.TeamID, number int) error {
	body := messages.NathejkPatrolNumberAssigned{
		TeamID:     teamID,
		TeamNumber: strconv.Itoa(number),
	}
	subj := cqrs.SubjectFromStr(fmt.Sprintf("NATHEJK:%s.patrulje.%s.numberassigned", s.year, teamID))
	msg := s.p.MessageFunc()(subj)
	msg.SetBody(&body)
	if err := s.p.Publish(msg); err != nil {
		log.Printf("patruljenumber: publish number %d for %s: %v", number, teamID, err)
		return err
	}
	return nil
}

func (s *saga) nap(d time.Duration) {
	if s.sleep != nil {
		s.sleep(d)
		return
	}
	time.Sleep(d)
}
