package live

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// DefaultHeartbeat is how often an idle stream emits a comment line.
//
// Without traffic, an intermediary will eventually consider the connection idle
// and close it — Traefik's default responding-timeout is minutes, so anything
// well under that is safe. The interval is configurable because the value that
// actually works is empirical (see task 035).
const DefaultHeartbeat = 20 * time.Second

// StreamHandler serves signals to one browser over Server-Sent Events.
//
// SSE rather than a websocket: the client never sends anything (writes go over
// REST), EventSource reconnects on its own, and cookies are sent automatically
// same-origin — so it works under the basic auth in front of stage/prod today and
// under the planned JWT cookie without change. A token in an Authorization header
// would have been the one scheme that broke this.
type StreamHandler struct {
	Hub *Hub

	// Heartbeat interval; DefaultHeartbeat when zero.
	Heartbeat time.Duration

	// DefaultYear supplies the year for a request that specifies none. Injected
	// rather than computed here so this package needs no opinion about what
	// "current" means, and so tests are not time-dependent.
	DefaultYear func() string
}

// ServeHTTP streams until the client disconnects.
//
//	@Summary		Live update stream
//	@Description	Server-Sent Events stream of invalidation signals. Each event names
//	@Description	an entity that changed; the client refetches it over the normal REST
//	@Description	endpoints. Carries no entity data, so it needs no authorisation model
//	@Description	of its own. Event names: `entity.changed`, `resync`.
//	@Tags			live
//	@Produce		text/event-stream
//	@Param			year		query	string	false	"Event year; defaults to the current year"
//	@Param			entities	query	string	false	"Comma-separated entity tokens to receive; default all"
//	@Success		200	{string}	string	"event stream"
//	@Router			/stream [get]
func (h StreamHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	filter := h.filterFrom(r)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	// Harmless here, and it saves an afternoon if a buffering proxy is ever put
	// in front: nginx and several others honour it to disable response buffering.
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	// The ResponseWriter may be wrapped by middleware that does not implement
	// http.Flusher — `metricsResponseWriter` in cmd/api/app is exactly that, and
	// a plain w.(http.Flusher) assertion against it fails, leaving the client
	// hanging with no output. ResponseController follows Unwrap() to find the
	// real flusher, so this keeps working whatever is layered on later.
	rc := http.NewResponseController(w)
	flush := func() { _ = rc.Flush() }
	flush()

	signals := h.Hub.Subscribe(r.Context(), filter)

	heartbeat := h.Heartbeat
	if heartbeat <= 0 {
		heartbeat = DefaultHeartbeat
	}
	ticker := time.NewTicker(heartbeat)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			// The browser went away. Subscribe() is bound to the same context,
			// so the hub has already dropped this client.
			return

		case signal, ok := <-signals:
			if !ok {
				return // hub closed, e.g. shutdown
			}
			if err := writeEvent(w, signal); err != nil {
				return // connection is gone; nothing useful to log
			}
			flush()

		case <-ticker.C:
			// A comment line: ignored by EventSource, enough to keep the
			// connection from being judged idle.
			if _, err := fmt.Fprint(w, ": keep-alive\n\n"); err != nil {
				return
			}
			flush()
		}
	}
}

// filterFrom builds the subscription filter from the query string.
//
// The year is a query parameter rather than the X-YearSlug header the rest of the
// API uses, because EventSource cannot set headers. An absent year falls back to
// the current year, matching the server's behaviour elsewhere.
func (h StreamHandler) filterFrom(r *http.Request) Filter {
	q := r.URL.Query()

	year := strings.TrimSpace(q.Get("year"))
	if year == "" && h.DefaultYear != nil {
		year = h.DefaultYear()
	}

	var entities []string
	if raw := strings.TrimSpace(q.Get("entities")); raw != "" {
		for _, e := range strings.Split(raw, ",") {
			if e = strings.TrimSpace(e); e != "" {
				entities = append(entities, e)
			}
		}
	}

	return Filter{Year: year, Entities: entities}
}

// writeEvent emits one SSE event.
//
// The event name is the signal type, so the client dispatches on `event:` and a
// future kind of signal — a deploy notification, say — is additive rather than a
// format change.
func writeEvent(w http.ResponseWriter, s Signal) error {
	payload, err := json.Marshal(s)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", s.Type, payload)
	return err
}
