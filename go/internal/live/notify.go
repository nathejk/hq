package live

import "github.com/jrgensen/cqrs"

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
// # The deadletter caveat, accepted deliberately
//
// cmd/api/main.go builds its Writer as cqrs/deadletter wrapping sqlpersister.
// deadletter diverts a failing statement to a table instead of failing the
// projection loop — so HandleMessage can return nil while the read model was *not*
// updated, and this decorator will then emit a signal for a change that is not
// visible.
//
// We accept that rather than removing deadletter from notified consumers, because
// the failure is benign and self-correcting: the client refetches, sees no change,
// and the next signal or resync brings it up to date. The alternative would trade
// projection resilience — the reason deadletter exists — for signal precision that
// nothing depends on. Recorded here so the next reader finds a decision rather
// than an accident.
func Notify(p Publisher, c cqrs.Consumer) cqrs.Consumer {
	return notifier{publisher: p, consumer: c}
}

// NotifyAll wraps several consumers, for the wiring in cmd/api/main.go.
//
// Wrapping in bulk keeps that call site readable: nineteen Notify(hub, …) calls
// would bury which consumers exist under how they are decorated.
func NotifyAll(p Publisher, consumers ...cqrs.Consumer) []cqrs.Consumer {
	wrapped := make([]cqrs.Consumer, 0, len(consumers))
	for _, c := range consumers {
		wrapped = append(wrapped, Notify(p, c))
	}
	return wrapped
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
