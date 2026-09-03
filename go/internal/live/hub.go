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
	// Note replay is not a burst source: the catch-up gate below holds signals
	// back until the read model has finished replaying. This window is for live
	// bursts only.
	DefaultCoalesceWindow = 75 * time.Millisecond

	// Per-client queue depth before the backlog collapses into one resync.
	DefaultBufferSize = 64

	// How long the catch-up gate waits before opening itself regardless.
	//
	// The gate is released by the projections reporting that they have drained
	// their backlog (see NotifyAll). If one of them never reports — a consumer
	// wedged on a message, a subject nothing ever publishes to under an
	// unexpected stream state — the gate would stay shut and the SPA would stop
	// being live with no error anywhere, which is the failure mode this package
	// works hardest to avoid. So the gate also opens on a timer: a late release
	// still broadcasts the accumulated collection signals, so the worst case is
	// that clients learn a little late rather than never.
	//
	// Generous on purpose: a cold start replays the whole event log, and opening
	// early is exactly the burst the gate exists to suppress.
	DefaultCatchupTimeout = 5 * time.Minute
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

	catchupTimeout time.Duration

	mu      sync.Mutex
	clients map[*client]struct{}
	pending map[string]Signal
	timer   *time.Timer
	closed  bool

	// catchingUp is true from construction until CaughtUp. While it holds,
	// signals are folded into catchup instead of being broadcast.
	catchingUp   bool
	catchup      map[string]Signal
	catchupTimer *time.Timer
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

// WithCatchupTimeout overrides how long the catch-up gate waits before opening
// itself. See DefaultCatchupTimeout for why the fallback exists at all.
func WithCatchupTimeout(d time.Duration) HubOption {
	return func(h *Hub) { h.catchupTimeout = d }
}

// NewHub returns a hub that starts out *catching up*: it accumulates signals
// without broadcasting until CaughtUp is called.
//
// Gated by default rather than on request, because a hub is built before the
// stream is subscribed and every process therefore begins by replaying history.
// A hub that broadcast during that replay would fan out one signal per historical
// event — tens of thousands of them — to any client that connected while the
// build was running, which is both pointless (the client has nothing cached that
// predates its own connection) and actively harmful: each client's buffer
// overflows, collapses into a resync, and the SPA refetches everything it holds,
// repeatedly, for the duration of the replay.
func NewHub(opts ...HubOption) *Hub {
	h := &Hub{
		coalesceWindow: DefaultCoalesceWindow,
		bufferSize:     DefaultBufferSize,
		catchupTimeout: DefaultCatchupTimeout,
		clients:        make(map[*client]struct{}),
		pending:        make(map[string]Signal),
		catchingUp:     true,
		catchup:        make(map[string]Signal),
	}
	for _, opt := range opts {
		opt(h)
	}
	if h.catchupTimeout > 0 {
		h.catchupTimer = time.AfterFunc(h.catchupTimeout, h.CaughtUp)
	}
	return h
}

// CaughtUp opens the catch-up gate: the read model has finished replaying, so
// what happens from here on is news.
//
// Whatever accumulated during the replay is broadcast now, as one signal per
// (entity type, year) with the instance id stripped — so every list, count and
// other type-level dependency in the SPA revalidates exactly once, and a client
// that connected mid-build ends up consistent with the finished read model.
//
// Ids are dropped rather than kept because a replay says nothing useful about
// individual instances: it touches every row that ever existed, so keeping them
// would reproduce the burst the gate suppressed. A detail view holding one
// instance is covered by the resync it received on connect.
//
// Idempotent, and safe to call from several goroutines: it is reported by each
// projection independently (see NotifyAll) and by the fallback timer.
func (h *Hub) CaughtUp() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.closed || !h.catchingUp {
		return
	}
	h.catchingUp = false
	if h.catchupTimer != nil {
		h.catchupTimer.Stop()
		h.catchupTimer = nil
	}

	for key, s := range h.catchup {
		delete(h.catchup, key)
		h.broadcastLocked(s)
	}
}

// CatchingUp reports whether the gate is still shut. For diagnostics and tests.
func (h *Hub) CatchingUp() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.catchingUp
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

	// A resync bypasses both coalescing and the catch-up gate: it is a control
	// message, and delaying "you are out of date" helps nobody.
	if s.Type == SignalResync {
		h.broadcastLocked(s)
		return
	}

	// Still replaying: remember *that* this type changed, not which instance,
	// and say nothing until the gate opens. Event is dropped with the id — a
	// name picked arbitrarily from thousands of historical events would be
	// misleading in a log, and Signal.Event is advisory anyway.
	if h.catchingUp {
		h.catchup[s.Entity+"@"+s.Year] = Signal{
			Type:   s.Type,
			Entity: s.Entity,
			Year:   s.Year,
		}
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
	if h.catchupTimer != nil {
		h.catchupTimer.Stop()
		h.catchupTimer = nil
	}
	for c := range h.clients {
		delete(h.clients, c)
		close(c.ch)
	}
}
