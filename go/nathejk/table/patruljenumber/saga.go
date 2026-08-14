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
	"sort"
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
// trigger names, and every order belonging to a patrulje.
//
// Declared here rather than taking order.Queries wholesale so the saga can be
// tested without a database, and so it states exactly what it reads.
// order.Queries satisfies it.
//
// ListByOwner rather than the purpose-built PaidQuantityBySKU aggregate: the
// backfill needs to know *when* a patrulje qualified, not just that it did, and
// having one way to count seats is worth more than saving a query. Two
// definitions of "paid seats" would be free to drift, and the drift would be
// invisible — the live path and the backfill would simply disagree about who is
// accepted. The cost is a handful of order rows per patrulje.
type OrderReader interface {
	GetByID(ctx context.Context, orderID string) (*order.Order, error)
	ListByOwner(ctx context.Context, year types.YearSlug, ownerType types.TeamType, ownerID string) ([]order.Order, error)
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
// are new and assignments may be published. It then numbers whatever already
// qualifies.
//
// Before opening the gate it folds in the numbers that exist in the read model
// but never appeared as events. If that read fails the gate stays shut and this
// saga publishes nothing for the rest of the process. That is the deliberately
// conservative choice: a patrulje left unnumbered is fixed by the next restart,
// whereas re-issuing a number already held by another patrulje is not — the
// projector's UPDATE is unconditional, so the two teams would simply share it.
func (s *saga) CaughtUp() {
	ctx, cancel := context.WithTimeout(context.Background(), catchupTimeout)
	defer cancel()

	// Read once and use the rows twice: the seed needs the numbers that exist, the
	// backfill needs the patruljer that lack one.
	patruljer, err := s.patruljer.GetAll(ctx, patrulje.Filter{YearSlug: s.year})
	if err != nil {
		log.Printf("patruljenumber: reading existing team numbers failed, staying dormant rather than risk re-issuing a number: %v", err)
		return
	}
	s.seed(patruljer)
	s.live.Store(true)

	// Only now that the gate is open. A patrulje that qualified before this saga
	// existed has no future event to trigger on — every one of its order.paid
	// events is history, and replay deliberately publishes nothing — so without
	// this sweep it would stay unnumbered forever.
	//
	// Unlike the seed, a failure here cannot cause a duplicate number: it only
	// means the backfill did not happen, and the next start tries again. So it logs
	// and carries on rather than shutting the gate on new payments.
	if err := s.backfill(ctx, patruljer); err != nil {
		log.Printf("patruljenumber: backfill incomplete, will retry on the next start: %v", err)
	}
}

// catchupTimeout bounds the seed and the sweep together. Generous: it runs once at
// startup, and the sweep may issue a number per qualifying patrulje.
const catchupTimeout = 2 * time.Minute

// seed folds every existing teamNumber for the year into the running state: the
// team counts as assigned, and its number raises the high-water mark. PRD 003's
// example — a manual 300 means the next automatic number is 301, not 1.
//
// Timing: this runs at catch-up rather than at construction because hq rebuilds
// its read model from the stream on every start, so at construction the patrulje
// table is empty.
//
// Catch-up of *this* consumer does not prove the patrulje projector has finished
// its own replay — they are independent consumers — so the mark seeded here can
// be too low. Two things keep that from issuing a duplicate: the assigned set is
// also fed by replayed numberassigned events, and every number this saga issues is
// observed on the way back, raising the mark again.
func (s *saga) seed(patruljer []patrulje.Patrulje) {
	for _, p := range patruljer {
		if p.TeamID == "" || p.TeamNumber == "" {
			continue
		}
		s.markAssigned(p.TeamID, p.TeamNumber)
	}
}

// candidate is a patrulje the sweep found eligible, and when it became so.
type candidate struct {
	teamID types.TeamID
	// paidAt is when the earliest of its paid orders was last changed, i.e. when it
	// was paid. Used only for ordering.
	paidAt string
}

// backfill numbers every patrulje that already qualifies but holds no number.
//
// Ordering decides who gets which number, so it must be deterministic and
// defensible: earliest payment first, which is the honest reading of "numbering
// reflects payment order" (PRD 003 §5). Sorting is essential rather than cosmetic
// — without it the numbers would follow whatever order the read model returned,
// and any later switch to iterating a map would hand out different numbers on
// every start. teamID breaks ties so the result is total.
//
// paidAt is compared as a string. The column holds Go's time.Time text form, all
// UTC and zero-padded, so lexicographic order is chronological order. If that
// format ever changes this needs a real parse.
//
// Idempotent by construction: after a sweep the patruljer it numbered hold
// numbers, so the next start's seed marks them assigned and the sweep skips them.
func (s *saga) backfill(ctx context.Context, patruljer []patrulje.Patrulje) error {
	skus, err := s.participationSKUs(ctx, s.year)
	if err != nil {
		return err
	}

	var candidates []candidate
	for _, p := range patruljer {
		if p.TeamID == "" || s.isAssigned(p.TeamID) {
			continue
		}
		seats, paidAt, err := s.paidSeats(ctx, s.year, p.TeamID, skus)
		if err != nil {
			return err
		}
		if seats < MinSeats {
			continue
		}
		candidates = append(candidates, candidate{teamID: p.TeamID, paidAt: paidAt})
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].paidAt != candidates[j].paidAt {
			return candidates[i].paidAt < candidates[j].paidAt
		}
		return candidates[i].teamID < candidates[j].teamID
	})

	assigned := 0
	for _, c := range candidates {
		done, err := s.assignNext(c.teamID)
		if err != nil {
			return err
		}
		if done {
			assigned++
		}
	}
	// Logged unconditionally, including the zero: on the first start after this
	// shipped it is a burst of one event per already-paid patrulje, and that should
	// be recognisable in the stream as a backfill rather than a runaway loop — while
	// on every later start the zero is the evidence that the sweep ran and correctly
	// found nothing to do.
	log.Printf("patruljenumber: backfill assigned %d number(s) to patruljer that already qualified", assigned)
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

	seats, _, err := s.paidSeatsFor(ctx, o.Year, teamID)
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

	if _, err := s.assignNext(teamID); err != nil {
		return settled, err
	}
	return settled, nil
}

// assignNext hands the team the next number and records it, reporting whether it
// did. false means the team already held one.
//
// The whole decision is under the lock, including the publish: the number comes
// from the high-water mark, so releasing between reading it and raising it would
// let the same number go out twice. Locking across I/O is normally worth avoiding,
// but the only other contender is the catch-up sweep, so there is nothing to
// contend with.
//
// Publishing before the local mark is deliberate: the saga's own subscription will
// observe the event on the way back, but two patruljer qualifying in quick
// succession must get distinct numbers without waiting for that round trip.
func (s *saga) assignNext(teamID types.TeamID) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.assigned[teamID] {
		return false, nil
	}
	number := s.maxNumber + 1
	if err := s.publish(teamID, number); err != nil {
		return false, err
	}
	s.markAssignedLocked(teamID, strconv.Itoa(number))
	return true, nil
}

// participationSKUs is the set of SKUs that count as seats for a patrulje.
//
// Read from the catalogue rather than hardcoded, so renaming or splitting the
// participation product does not silently stop acceptance from working.
// Merchandise is excluded — a patrulje that buys four t-shirts has not thereby
// paid for four seats.
func (s *saga) participationSKUs(ctx context.Context, year types.YearSlug) (map[string]bool, error) {
	products, err := s.products.ListEligibleFor(ctx, year, types.TeamTypePatrulje)
	if err != nil {
		return nil, err
	}
	skus := map[string]bool{}
	for _, p := range products {
		if p.Kind == product.KindParticipation {
			skus[p.SKU] = true
		}
	}
	return skus, nil
}

// paidSeatsFor is paidSeats for a single patrulje, fetching the SKU set itself.
// Used by the live path, where there is one patrulje to consider; the sweep hoists
// the SKU lookup out of its loop instead.
func (s *saga) paidSeatsFor(ctx context.Context, year types.YearSlug, teamID types.TeamID) (int, string, error) {
	skus, err := s.participationSKUs(ctx, year)
	if err != nil {
		return 0, "", err
	}
	return s.paidSeats(ctx, year, teamID, skus)
}

// paidSeats counts the seats a patrulje has paid for, and reports when the
// earliest of those payments landed.
//
// Only paid orders count: an open order is an intention, not a payment. This is
// the single definition of "paid seats" — the live path and the backfill both come
// through here, so they cannot disagree about who is accepted.
func (s *saga) paidSeats(ctx context.Context, year types.YearSlug, teamID types.TeamID, skus map[string]bool) (int, string, error) {
	orders, err := s.orders.ListByOwner(ctx, year, types.TeamTypePatrulje, string(teamID))
	if err != nil {
		return 0, "", err
	}
	seats := 0
	firstPaidAt := ""
	for _, o := range orders {
		if o.Status != order.StatusPaid {
			continue
		}
		for _, l := range o.Lines {
			if skus[l.ProductSKU] {
				seats += l.Quantity
			}
		}
		if firstPaidAt == "" || o.ChangedAt < firstPaidAt {
			firstPaidAt = o.ChangedAt
		}
	}
	return seats, firstPaidAt, nil
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
