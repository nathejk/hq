package live

import (
	"context"
	"sync"
	"testing"
	"time"
)

// Short window so tests stay fast; the production default is 75ms.
const testWindow = 5 * time.Millisecond

func newTestHub(t *testing.T, opts ...HubOption) *Hub {
	t.Helper()
	h := NewHub(append([]HubOption{WithCoalesceWindow(testWindow)}, opts...)...)
	t.Cleanup(h.Close)
	return h
}

// recv waits for one signal, failing rather than hanging the suite.
func recv(t *testing.T, ch <-chan Signal) Signal {
	t.Helper()
	select {
	case s, ok := <-ch:
		if !ok {
			t.Fatal("channel closed while waiting for a signal")
		}
		return s
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for a signal")
		return Signal{}
	}
}

// expectQuiet asserts nothing more arrives, allowing for the coalesce window.
func expectQuiet(t *testing.T, ch <-chan Signal) {
	t.Helper()
	select {
	case s := <-ch:
		t.Fatalf("expected no further signal, got %+v", s)
	case <-time.After(testWindow * 6):
	}
}

func changedSignal(entity, id, year, event string) Signal {
	return Signal{Type: SignalEntityChanged, Entity: entity, ID: id, Year: year, Event: event}
}

func TestSubscribeDeliversImmediateResync(t *testing.T) {
	h := newTestHub(t)
	ch := h.Subscribe(context.Background(), Filter{})

	// A client that just connected does not know what it missed.
	if got := recv(t, ch); got.Type != SignalResync {
		t.Errorf("first signal = %+v, want a resync", got)
	}
}

func TestCoalescesRepeatedSignalsForOneInstance(t *testing.T) {
	h := newTestHub(t)
	ch := h.Subscribe(context.Background(), Filter{})
	recv(t, ch) // initial resync

	// Several projections handling one event, plus a burst of edits.
	h.Publish(changedSignal("patrulje", "p-1", "2026", "started"))
	h.Publish(changedSignal("patrulje", "p-1", "2026", "numberassigned"))
	h.Publish(changedSignal("patrulje", "p-1", "2026", "updated"))

	got := recv(t, ch)
	if got.Entity != "patrulje" || got.ID != "p-1" {
		t.Errorf("got %+v, want a patrulje/p-1 signal", got)
	}
	// One signal, not three: the client only needs to know to refetch.
	expectQuiet(t, ch)
}

func TestDoesNotCoalesceDifferentInstances(t *testing.T) {
	h := newTestHub(t)
	ch := h.Subscribe(context.Background(), Filter{})
	recv(t, ch)

	h.Publish(changedSignal("patrulje", "p-1", "2026", "started"))
	h.Publish(changedSignal("patrulje", "p-2", "2026", "started"))
	h.Publish(changedSignal("payment", "x-1", "2026", "received"))

	seen := map[string]bool{}
	for range 3 {
		seen[recv(t, ch).Key()] = true
	}
	for _, want := range []string{"patrulje:p-1", "patrulje:p-2", "payment:x-1"} {
		if !seen[want] {
			t.Errorf("missing signal for %s (saw %v)", want, seen)
		}
	}
}

func TestFiltersByEntity(t *testing.T) {
	h := newTestHub(t)
	ch := h.Subscribe(context.Background(), Filter{Entities: []string{"sos", "patrulje"}})
	recv(t, ch)

	h.Publish(changedSignal("scan", "s-1", "2026", "scanned"))
	h.Publish(changedSignal("sos", "c-1", "2026", "commented"))

	// The scan must not arrive at all — that is what keeps a checkpoint rush off
	// screens that do not display scans.
	if got := recv(t, ch); got.Entity != "sos" {
		t.Errorf("got %+v, want the sos signal only", got)
	}
	expectQuiet(t, ch)
}

func TestFiltersByYear(t *testing.T) {
	h := newTestHub(t)
	ch := h.Subscribe(context.Background(), Filter{Year: "2026"})
	recv(t, ch)

	h.Publish(changedSignal("patrulje", "old", "2025", "updated"))
	h.Publish(changedSignal("patrulje", "new", "2026", "updated"))

	if got := recv(t, ch); got.ID != "new" {
		t.Errorf("got %+v, want only the 2026 signal", got)
	}
	expectQuiet(t, ch)
}

func TestResyncIgnoresFilters(t *testing.T) {
	h := newTestHub(t)
	ch := h.Subscribe(context.Background(), Filter{Year: "2026", Entities: []string{"sos"}})
	recv(t, ch)

	// "You are out of date" is true regardless of what the client subscribed to;
	// filtering it would strand a client that missed something.
	h.Publish(Resync())

	if got := recv(t, ch); got.Type != SignalResync {
		t.Errorf("got %+v, want the resync through the filter", got)
	}
}

func TestOverflowCollapsesToResyncRatherThanDropping(t *testing.T) {
	// A tiny buffer stands in for a client that stopped reading.
	h := newTestHub(t, WithBufferSize(4))
	ch := h.Subscribe(context.Background(), Filter{})

	// Deliberately do not read: fill well past the buffer with distinct keys so
	// none of them coalesce.
	for i := range 50 {
		h.Publish(changedSignal("patrulje", string(rune('a'+i%26))+string(rune('a'+i/26)), "2026", "updated"))
	}
	time.Sleep(testWindow * 4)

	// Drain whatever is queued. The contract is not "you get everything" — it is
	// "you are never left silently stale", so a resync must be in there.
	var sawResync bool
	for {
		select {
		case s := <-ch:
			if s.Type == SignalResync {
				sawResync = true
			}
			continue
		default:
		}
		break
	}
	if !sawResync {
		t.Error("overflow dropped signals without telling the client to resync")
	}
}

func TestSlowClientDoesNotBlockOthersOrThePublisher(t *testing.T) {
	h := newTestHub(t, WithBufferSize(2))

	// Never reads.
	slowCtx, cancelSlow := context.WithCancel(context.Background())
	defer cancelSlow()
	h.Subscribe(slowCtx, Filter{})

	fast := h.Subscribe(context.Background(), Filter{})
	recv(t, fast)

	done := make(chan struct{})
	go func() {
		for i := range 100 {
			h.Publish(changedSignal("payment", string(rune('A'+i%26)), "2026", "received"))
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("a slow client blocked the publisher")
	}

	// And the healthy client still receives.
	recv(t, fast)
}

func TestContextCancelUnsubscribesAndClosesChannel(t *testing.T) {
	h := newTestHub(t)
	ctx, cancel := context.WithCancel(context.Background())
	ch := h.Subscribe(ctx, Filter{})
	recv(t, ch)

	if got := h.ClientCount(); got != 1 {
		t.Fatalf("ClientCount() = %d, want 1", got)
	}

	cancel()

	// The channel must close, so a handler's range loop ends rather than hanging.
	deadline := time.After(2 * time.Second)
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				if got := h.ClientCount(); got != 0 {
					t.Errorf("ClientCount() = %d after cancel, want 0", got)
				}
				return
			}
		case <-deadline:
			t.Fatal("channel was not closed after context cancel")
		}
	}
}

func TestPublishAfterCloseIsSafe(t *testing.T) {
	h := NewHub(WithCoalesceWindow(testWindow))
	ch := h.Subscribe(context.Background(), Filter{})
	h.Close()

	// Channel closed by Close; publishing afterwards must not panic.
	for range ch { //nolint:revive // draining
	}
	h.Publish(changedSignal("patrulje", "p-1", "2026", "started"))
	h.Close() // idempotent
}

func TestSubscribeAfterCloseReturnsClosedChannel(t *testing.T) {
	h := NewHub()
	h.Close()

	ch := h.Subscribe(context.Background(), Filter{})
	if _, ok := <-ch; ok {
		t.Error("expected a closed channel when subscribing to a closed hub")
	}
}

// Run with -race: publishers are projection consumers, which are concurrent.
func TestConcurrentPublishAndSubscribe(t *testing.T) {
	h := newTestHub(t, WithBufferSize(8))

	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range 50 {
				h.Publish(changedSignal("patrulje", string(rune('a'+i%26)), "2026", "updated"))
			}
		}()
	}
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			ch := h.Subscribe(ctx, Filter{})
			for range 3 {
				select {
				case <-ch:
				case <-time.After(time.Second):
					return
				}
			}
		}()
	}
	wg.Wait()
}
