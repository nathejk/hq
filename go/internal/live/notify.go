package live

import (
	"sync"
	"sync/atomic"

	"github.com/jrgensen/cqrs"
)

// Publisher accepts signals for delivery. The Hub satisfies it.
//
// Declared as an interface so the decorator can be tested without a hub, and so
// nothing here depends on how fan-out works.
type Publisher interface {
	Publish(Signal)
}

// Notify wraps a projection consumer so that applying an event also tells
// connected browsers to refetch.
//
// Wrapping is what keeps this design free of per-entity code: one decorator makes
// every entity live, present and future.
//
//	mux.AddConsumer(live.Notify(hub, patruljetable), live.Notify(hub, paymenttable))
//
// # Why a decorator rather than a second subscriber
//
// If the hub subscribed to the stream independently it could signal *before* the
// projection committed. The client would refetch, read the old row, and display
// stale data that no later event corrects — the single most likely source of "it
// flickered back to the old value" bugs. Wrapping makes the ordering structural:
// the signal cannot precede the write.
//
// That is sound because the write is synchronous: sqlpersister.Writer.Consume
// executes db.Exec inline and MariaDB autocommits, so a nil return from
// HandleMessage means the read model has changed.
//
// # Failed projections, and the deadletter caveat
//
// Today `cmd/api/main.go` builds its Writer as a plain `sqlpersister`
// (`main.go:164`), so a failing statement returns an error from `Consume`, the
// wrapped consumer returns it, and this decorator publishes nothing. That is the
// behaviour we want: a change that did not land is not announced. It is currently
// observable — the `signup` projection fails on every event because its table is
// missing columns the entity now writes, and no signals result.
//
// The caveat applies **if** `cqrs/deadletter` is ever introduced, as the layout
// skill describes: it diverts a failing statement to a table instead of failing
// the projection loop, so `HandleMessage` would return nil while the read model
// was *not* updated, and this decorator would then emit a signal for a change that
// is not visible. That failure would be benign and self-correcting — the client
// refetches, sees no change, and the next signal or resync brings it up to date —
// so it would not be a reason to keep deadletter away from notified consumers. But
// it should be a decision taken knowingly rather than discovered.
func Notify(p Publisher, c cqrs.Consumer) cqrs.Consumer {
	return NotifyAll(p, c)[0]
}

// catchupListener mirrors stream.CatchupListener structurally, so this package
// need not import stream to forward it — the same trick the order saga uses to
// implement it.
type catchupListener interface {
	CaughtUp()
}

// catchupNotifier is NotifyAll's return for a consumer where *something* listens
// for catch-up: the inner consumer, the publisher, or both.
type catchupNotifier struct {
	notifier

	// listener is the inner consumer when it listens for catch-up; nil otherwise.
	// Nothing in hq implements it today except the patrulje number saga.
	listener catchupListener

	// report tells the publisher's gate that this one consumer has drained its
	// backlog; nil when the publisher does not gate.
	report func()
}

// CaughtUp forwards to whoever is waiting for it.
//
// The inner consumer first: its behaviour changes at this moment (the number saga
// starts publishing), and the hub's gate opening is what makes that visible.
func (c catchupNotifier) CaughtUp() {
	if c.listener != nil {
		c.listener.CaughtUp()
	}
	if c.report != nil {
		c.report()
	}
}

// NotifyAll wraps several consumers, for the wiring in cmd/api/main.go.
//
// Wrapping in bulk keeps that call site readable: nineteen Notify(hub, …) calls
// would bury which consumers exist under how they are decorated.
//
// It is also what lets the hub know when the read model has finished replaying:
// catch-up is reported per consumer by the stream, so "caught up" for the hub
// means *every* consumer in one NotifyAll call has reported. That is why the set
// is passed together rather than wrapped one at a time — a consumer wrapped
// separately would open the gate on its own, while the others were still
// replaying.
func NotifyAll(p Publisher, consumers ...cqrs.Consumer) []cqrs.Consumer {
	gate := newCatchupGate(p, len(consumers))

	wrapped := make([]cqrs.Consumer, 0, len(consumers))
	for _, c := range consumers {
		n := notifier{publisher: p, consumer: c}

		// Preserve the inner consumer's optional catch-up interface. The
		// jetstream Subscribe path discovers it by asserting on the handler it is
		// given, and that handler is this decorator — so a consumer that behaves
		// differently while replaying would silently never be told it had caught
		// up, and would go on replaying forever. A decorator must be transparent
		// about the interfaces it wraps.
		listener, listens := c.(catchupListener)
		if !listens {
			listener = nil
		}
		report := gate.reporter()

		// Only advertise CaughtUp when something actually wants it. Adding it
		// unconditionally would ask the stream to track catch-up for consumers
		// nobody is waiting on.
		if listener == nil && report == nil {
			wrapped = append(wrapped, n)
			continue
		}
		wrapped = append(wrapped, catchupNotifier{notifier: n, listener: listener, report: report})
	}
	return wrapped
}

// catchupGate counts consumers down to the publisher's CaughtUp.
//
// Nil when the publisher does not gate, so callers need not branch: a nil gate
// hands out nil reporters.
type catchupGate struct {
	publisher catchupListener
	remaining int64
}

func newCatchupGate(p Publisher, consumers int) *catchupGate {
	listener, ok := p.(catchupListener)
	if !ok || consumers == 0 {
		return nil
	}
	return &catchupGate{publisher: listener, remaining: int64(consumers)}
}

// reporter returns the report function for one consumer. Safe to call repeatedly
// and from several goroutines; only the first call counts, matching the stream's
// own guarantee loosely enough that a double report cannot open the gate early.
func (g *catchupGate) reporter() func() {
	if g == nil {
		return nil
	}
	var once sync.Once
	return func() { once.Do(g.done) }
}

func (g *catchupGate) done() {
	if atomic.AddInt64(&g.remaining, -1) > 0 {
		return
	}
	g.publisher.CaughtUp()
}

type notifier struct {
	publisher Publisher
	consumer  cqrs.Consumer
}

// Consumes passes through unchanged: decorating must not alter which subjects a
// projection sees.
func (n notifier) Consumes() []cqrs.Subject {
	return n.consumer.Consumes()
}

func (n notifier) HandleMessage(msg cqrs.Message) error {
	// Projection first, and its error wins. A signalling side-effect must never
	// swallow or mask a projection failure, and a failed projection must not
	// announce a change that did not happen.
	if err := n.consumer.HandleMessage(msg); err != nil {
		return err
	}

	// Subjects that carry no entity — the stream's own "caughtup" sentinel, or
	// anything off-convention — simply produce no signal. Not an error: the
	// projection succeeded, and refusing the message would make the stream
	// retry it forever.
	signal, err := SignalFromSubject(msg.Subject())
	if err != nil {
		return nil
	}

	n.publisher.Publish(signal)
	return nil
}
