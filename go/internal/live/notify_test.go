package live

import (
	"errors"
	"testing"
	"time"

	"github.com/jrgensen/cqrs"
	"github.com/jrgensen/stream/caughtup"
)

// --- fakes ---

type fakePublisher struct {
	published []Signal
}

func (f *fakePublisher) Publish(s Signal) {
	f.published = append(f.published, s)
}

type fakeConsumer struct {
	subjects []cqrs.Subject
	err      error
	handled  int
}

func (f *fakeConsumer) Consumes() []cqrs.Subject { return f.subjects }

func (f *fakeConsumer) HandleMessage(cqrs.Message) error {
	f.handled++
	return f.err
}

// fakeMessage is the minimum that satisfies cqrs.Message for these tests: only
// the subject is read by the decorator.
type fakeMessage struct {
	subject cqrs.Subject
}

func (m fakeMessage) Subject() cqrs.Subject { return m.subject }
func (m fakeMessage) Time() time.Time       { return time.Time{} }
func (m fakeMessage) Sequence() uint64      { return 0 }
func (m fakeMessage) Body(any) error        { return nil }
func (m fakeMessage) Meta(any) error        { return nil }
func (m fakeMessage) RawBody() any          { return nil }
func (m fakeMessage) RawMeta() any          { return nil }

func message(subject string) cqrs.Message {
	return fakeMessage{subject: cqrs.SubjectFromStr(subject)}
}

// --- tests ---

func TestNotifyPassesConsumesThrough(t *testing.T) {
	subjects := []cqrs.Subject{
		cqrs.SubjectFromStr("NATHEJK:*.patrulje.*.started"),
		cqrs.SubjectFromStr("NATHEJK:*.patrulje.*.signedup"),
	}
	inner := &fakeConsumer{subjects: subjects}

	got := Notify(&fakePublisher{}, inner).Consumes()

	// Decorating must not change which subjects a projection is subscribed to.
	if len(got) != len(subjects) {
		t.Fatalf("Consumes() returned %d subjects, want %d", len(got), len(subjects))
	}
	for i := range got {
		if got[i].Subject() != subjects[i].Subject() {
			t.Errorf("subject %d = %q, want %q", i, got[i].Subject(), subjects[i].Subject())
		}
	}
}

func TestNotifyPublishesAfterSuccessfulHandle(t *testing.T) {
	pub := &fakePublisher{}
	inner := &fakeConsumer{}

	if err := Notify(pub, inner).HandleMessage(message("NATHEJK.2026.patrulje.p-1.started")); err != nil {
		t.Fatalf("HandleMessage returned %v", err)
	}

	if inner.handled != 1 {
		t.Errorf("wrapped consumer handled %d messages, want 1", inner.handled)
	}
	if len(pub.published) != 1 {
		t.Fatalf("published %d signals, want 1", len(pub.published))
	}
	got := pub.published[0]
	want := Signal{Type: SignalEntityChanged, Entity: "patrulje", ID: "p-1", Year: "2026", Event: "started"}
	if got != want {
		t.Errorf("published %+v, want %+v", got, want)
	}
}

func TestNotifyPublishesNothingWhenProjectionFails(t *testing.T) {
	failure := errors.New("insert failed")
	pub := &fakePublisher{}
	inner := &fakeConsumer{err: failure}

	err := Notify(pub, inner).HandleMessage(message("NATHEJK.2026.patrulje.p-1.started"))

	// The error must reach the stream unchanged — a signalling side-effect may
	// not mask a projection failure.
	if !errors.Is(err, failure) {
		t.Errorf("HandleMessage err = %v, want %v", err, failure)
	}
	// And a failed projection must not announce a change that did not happen.
	if len(pub.published) != 0 {
		t.Errorf("published %d signals after a failure, want 0", len(pub.published))
	}
}

func TestNotifyIgnoresSubjectsThatAreNotSignals(t *testing.T) {
	// The stream's own catch-up sentinel travels as a message with subject
	// "<domain>.caughtup". It is not a domain event, so it must produce no
	// signal — and must not fail, or the stream would retry it forever.
	pub := &fakePublisher{}
	inner := &fakeConsumer{}

	msg := caughtup.NewCaughtupMessage("NATHEJK")
	if err := Notify(pub, inner).HandleMessage(msg); err != nil {
		t.Fatalf("HandleMessage returned %v for a caughtup sentinel", err)
	}

	if inner.handled != 1 {
		t.Errorf("wrapped consumer handled %d messages, want 1", inner.handled)
	}
	if len(pub.published) != 0 {
		t.Errorf("published %d signals for a caughtup sentinel, want 0", len(pub.published))
	}
}

func TestNotifyIgnoresOffConventionSubjects(t *testing.T) {
	pub := &fakePublisher{}
	inner := &fakeConsumer{}

	for _, subject := range []string{"", "NATHEJK", "OTHER.2026.patrulje.p-1.started"} {
		if err := Notify(pub, inner).HandleMessage(message(subject)); err != nil {
			t.Errorf("HandleMessage(%q) returned %v, want nil", subject, err)
		}
	}

	if len(pub.published) != 0 {
		t.Errorf("published %d signals for off-convention subjects, want 0", len(pub.published))
	}
}

func TestNotifyAllWrapsEveryConsumer(t *testing.T) {
	pub := &fakePublisher{}
	a, b, c := &fakeConsumer{}, &fakeConsumer{}, &fakeConsumer{}

	wrapped := NotifyAll(pub, a, b, c)
	if len(wrapped) != 3 {
		t.Fatalf("NotifyAll returned %d consumers, want 3", len(wrapped))
	}

	for _, w := range wrapped {
		if err := w.HandleMessage(message("NATHEJK.2026.payment.x-1.received")); err != nil {
			t.Fatalf("HandleMessage returned %v", err)
		}
	}

	for i, inner := range []*fakeConsumer{a, b, c} {
		if inner.handled != 1 {
			t.Errorf("consumer %d handled %d messages, want 1", i, inner.handled)
		}
	}
	if len(pub.published) != 3 {
		t.Errorf("published %d signals, want 3", len(pub.published))
	}
}

// The hub is the production Publisher; keep them structurally compatible.
func TestHubSatisfiesPublisher(t *testing.T) {
	var _ Publisher = NewHub()
}

// catchupConsumer is a consumer that also listens for catch-up, like the order
// Pay saga.
type catchupConsumer struct {
	fakeConsumer
	caughtUp int
}

func (c *catchupConsumer) CaughtUp() { c.caughtUp++ }

// A wrapped consumer is handed to stream.Subscribe in place of the original, and
// the jetstream path discovers catch-up by asserting on it. If the decorator
// dropped the interface, a saga that only waits for a lagging projection once
// live would behave as if it were replaying forever — with no error to say so.
func TestNotifyForwardsCaughtUp(t *testing.T) {
	inner := &catchupConsumer{}

	wrapped := Notify(&fakePublisher{}, inner)

	listener, ok := wrapped.(interface{ CaughtUp() })
	if !ok {
		t.Fatalf("wrapped consumer no longer advertises CaughtUp")
	}
	listener.CaughtUp()
	if inner.caughtUp != 1 {
		t.Errorf("inner CaughtUp called %d times, want 1", inner.caughtUp)
	}

	// Still a working notifier.
	if err := wrapped.HandleMessage(message("NATHEJK.2026.payment.x-1.received")); err != nil {
		t.Fatalf("HandleMessage returned %v", err)
	}
	if inner.handled != 1 {
		t.Errorf("inner handled %d messages, want 1", inner.handled)
	}
}

// The converse: a plain projection must not be advertised as a listener, or the
// stream would track catch-up for consumers that do not care.
func TestNotifyDoesNotInventCaughtUp(t *testing.T) {
	wrapped := Notify(&fakePublisher{}, &fakeConsumer{})

	if _, ok := wrapped.(interface{ CaughtUp() }); ok {
		t.Errorf("plain consumer was advertised as a catch-up listener")
	}
}

// NotifyAll must preserve it too — that is the call site main.go uses.
func TestNotifyAllForwardsCaughtUp(t *testing.T) {
	inner := &catchupConsumer{}

	wrapped := NotifyAll(&fakePublisher{}, &fakeConsumer{}, inner)

	listener, ok := wrapped[1].(interface{ CaughtUp() })
	if !ok {
		t.Fatalf("NotifyAll dropped CaughtUp")
	}
	listener.CaughtUp()
	if inner.caughtUp != 1 {
		t.Errorf("inner CaughtUp called %d times, want 1", inner.caughtUp)
	}
}
