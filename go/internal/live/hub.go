package live

import (
	"context"
	"slices"
	"sync"
	"time"
)

// Defaults chosen for a handful of operators on an internal tool, not a public
// fan-out. Both are configurable because the right values are empirical.
const (
	// How long signals accumulate before being fanned out. Several projections
	// consume the same event (patrulje, patruljestatus and spejderstatus all
	// consume patrulje.*.started), so without this a single event produces
	// several identical signals. It also smooths mass operations such as
	// collecting a whole patrol.
	//
	// Note replay is not a burst source: the boot gate means no client is
	// connected while a build replays, so the whole history passes with nobody
	// listening. This window is for live bursts only.
	DefaultCoalesceWindow = 75 * time.Millisecond

	// Per-client queue depth before the backlog collapses into one resync.
	DefaultBufferSize = 64
)

// Filter narrows what a client receives.
//
// Both fields are "no opinion" when empty, so a zero Filter receives everything.
type Filter struct {
	// Year restricts signals to one event year. Callers resolve "current year"
	// before constructing the Filter — the hub does not know today's date.
	Year string

	// Entities restricts to these entity tokens. Empty means all. This is what
	// keeps a checkpoint-scan rush off screens that do not display scans.
	Entities []string
}

func (f Filter) allows(s Signal) bool {
	// A resync is a control message: it means "revalidate what you hold", which
	// is true regardless of what the client subscribed to. Filtering it would
	// strand a client that missed something.
	if s.Type == SignalResync {
		return true
	}
	if f.Year != "" && s.Year != "" && s.Year != f.Year {
		return false
	}
	if len(f.Entities) > 0 && !slices.Contains(f.Entities, s.Entity) {
		return false
	}
	return true
}

type client struct {
	ch     chan Signal
	filter Filter
}

// Hub fans signals out to connected clients.
//
// It knows nothing about HTTP: that seam lives in the handler, so the fan-out
// logic — which is where the subtle failure modes are — can be tested without a
// server.
type Hub struct {
	coalesceWindow time.Duration
	bufferSize     int

	mu      sync.Mutex
	clients map[*client]struct{}
	pending map[string]Signal
	timer   *time.Timer
	closed  bool
}

// HubOption configures a Hub.
type HubOption func(*Hub)

// WithCoalesceWindow overrides how long signals accumulate before fan-out.
func WithCoalesceWindow(d time.Duration) HubOption {
	return func(h *Hub) { h.coalesceWindow = d }
}

// WithBufferSize overrides the per-client queue depth.
func WithBufferSize(n int) HubOption {
	return func(h *Hub) { h.bufferSize = n }
}

func NewHub(opts ...HubOption) *Hub {
	h := &Hub{
		coalesceWindow: DefaultCoalesceWindow,
		bufferSize:     DefaultBufferSize,
		clients:        make(map[*client]struct{}),
		pending:        make(map[string]Signal),
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// Subscribe registers a client and returns the channel its signals arrive on.
//
// The channel is closed when ctx is cancelled — which for an HTTP handler means
// the browser went away — so a caller needs no explicit unsubscribe. It receives
// an immediate resync, because a client that has just connected does not know
// what it missed.
func (h *Hub) Subscribe(ctx context.Context, filter Filter) <-chan Signal {
	c := &client{
		ch:     make(chan Signal, h.bufferSize),
		filter: filter,
	}

	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		close(c.ch)
		return c.ch
	}
	h.clients[c] = struct{}{}
	// Fits by construction: the buffer is empty and at least 1 deep.
	c.ch <- Resync()
	h.mu.Unlock()

	go func() {
		<-ctx.Done()
		h.remove(c)
	}()

	return c.ch
}

// remove detaches a client and closes its channel.
//
// Done under the same lock that guards sending, which is what makes closing the
// channel safe: no publisher can be mid-send on a client that is no longer in the
// map.
func (h *Hub) remove(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[c]; !ok {
		return
	}
	delete(h.clients, c)
	close(c.ch)
}

// Publish queues a signal for delivery.
//
// Never blocks and never fails: it is called from projection consumers, whose
// progress must not depend on how promptly browsers read their sockets.
func (h *Hub) Publish(s Signal) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.closed {
		return
	}

	// A resync bypasses coalescing: it is a control message, and delaying "you
	// are out of date" by a window helps nobody.
	if s.Type == SignalResync {
		h.broadcastLocked(s)
		return
	}

	// Last write wins per (entity, id). Two events about one instance are
	// interchangeable to a client — both mean "refetch this" — so the surviving
	// event name is arbitrary, which is why Signal.Event is documented as
	// advisory only.
	h.pending[s.Key()] = s

	if h.timer == nil {
		h.timer = time.AfterFunc(h.coalesceWindow, h.flush)
	}
}

func (h *Hub) flush() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.timer = nil
	if len(h.pending) == 0 {
		return
	}

	for key, s := range h.pending {
		delete(h.pending, key)
		h.broadcastLocked(s)
	}
}

// broadcastLocked sends to every interested client. Caller holds h.mu.
//
// Every send is non-blocking, which is what allows this to run under the lock:
// no client can stall the hub or another client.
func (h *Hub) broadcastLocked(s Signal) {
	for c := range h.clients {
		if !c.filter.allows(s) {
			continue
		}
		send(c, s)
	}
}

// send delivers one signal, collapsing the client's backlog if it is full.
//
// This is the decision that matters most in this file. A slow or sleeping client
// (a backgrounded tab, a closed laptop) needs a policy, and both obvious ones are
// wrong:
//
//   - An unbounded buffer turns a sleeping tab into a memory leak.
//   - Dropping signals leaves the client silently stale forever, with no error
//     anywhere — worse than having no live updates at all, because the UI still
//     looks live.
//
// So on overflow we neither block nor drop: the backlog is discarded and replaced
// by a single resync, meaning "you have missed something, revalidate everything".
// The client already runs that path on reconnect, so overflow degrades into
// well-tested behaviour. A slow client gets coarser updates; never wrong ones.
func send(c *client, s Signal) {
	select {
	case c.ch <- s:
		return
	default:
	}

	// Full. Drain what is queued — it is all superseded by a resync — and queue
	// the resync in its place.
	for {
		select {
		case <-c.ch:
		default:
			select {
			case c.ch <- Resync():
			default:
				// A concurrent reader refilled it; it will get the next resync.
			}
			return
		}
	}
}

// ClientCount reports how many clients are connected. For diagnostics and tests.
func (h *Hub) ClientCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.clients)
}

// Close detaches every client and stops accepting signals.
func (h *Hub) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.closed {
		return
	}
	h.closed = true

	if h.timer != nil {
		h.timer.Stop()
		h.timer = nil
	}
	for c := range h.clients {
		delete(h.clients, c)
		close(c.ch)
	}
}
