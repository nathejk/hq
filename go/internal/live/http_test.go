package live

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// readEvent reads one SSE event (up to the blank line separator), skipping
// comment lines such as the heartbeat.
func readEvent(t *testing.T, r *bufio.Reader) (event string, data string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)

	for time.Now().Before(deadline) {
		line, err := r.ReadString('\n')
		if err != nil {
			t.Fatalf("reading stream: %v", err)
		}
		line = strings.TrimRight(line, "\n")

		switch {
		case line == "" || strings.HasPrefix(line, ":"):
			continue // separator or comment
		case strings.HasPrefix(line, "event: "):
			event = strings.TrimPrefix(line, "event: ")
		case strings.HasPrefix(line, "data: "):
			data = strings.TrimPrefix(line, "data: ")
			return event, data
		}
	}
	t.Fatal("timed out reading an event")
	return "", ""
}

// serve starts the handler against a real HTTP server, so the test exercises the
// actual flushing path rather than a recorder that buffers everything.
func serve(t *testing.T, h StreamHandler) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return srv
}

func TestStreamSendsHeadersAndInitialResync(t *testing.T) {
	hub := newTestHub(t)
	srv := serve(t, StreamHandler{Hub: hub, DefaultYear: func() string { return "2026" }})

	resp, err := http.Get(srv.URL + "/api/stream")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()

	// Content type matters more than it looks: this same mux serves the SPA
	// through a fallback that returns index.html for unknown paths, so a route
	// typo would deliver HTML and the client would fail parsing an "event
	// stream" with no obvious cause.
	if got := resp.Header.Get("Content-Type"); got != "text/event-stream" {
		t.Errorf("Content-Type = %q, want text/event-stream", got)
	}
	if got := resp.Header.Get("Cache-Control"); got != "no-cache" {
		t.Errorf("Cache-Control = %q, want no-cache", got)
	}

	event, data := readEvent(t, bufio.NewReader(resp.Body))
	if event != SignalResync {
		t.Errorf("first event = %q, want %q", event, SignalResync)
	}

	var got Signal
	if err := json.Unmarshal([]byte(data), &got); err != nil {
		t.Fatalf("unmarshalling %q: %v", data, err)
	}
	if got.Type != SignalResync {
		t.Errorf("payload type = %q, want %q", got.Type, SignalResync)
	}
}

func TestStreamDeliversSignals(t *testing.T) {
	hub := newTestHub(t)
	srv := serve(t, StreamHandler{Hub: hub, DefaultYear: func() string { return "2026" }})

	resp, err := http.Get(srv.URL + "/api/stream?year=2026")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	reader := bufio.NewReader(resp.Body)
	readEvent(t, reader) // initial resync

	hub.Publish(changedSignal("patrulje", "p-1", "2026", "started"))

	event, data := readEvent(t, reader)
	if event != SignalEntityChanged {
		t.Errorf("event = %q, want %q", event, SignalEntityChanged)
	}

	var got Signal
	if err := json.Unmarshal([]byte(data), &got); err != nil {
		t.Fatalf("unmarshalling %q: %v", data, err)
	}
	want := changedSignal("patrulje", "p-1", "2026", "started")
	if got != want {
		t.Errorf("signal = %+v, want %+v", got, want)
	}
}

func TestStreamAppliesEntityFilterFromQuery(t *testing.T) {
	hub := newTestHub(t)
	srv := serve(t, StreamHandler{Hub: hub, DefaultYear: func() string { return "2026" }})

	resp, err := http.Get(srv.URL + "/api/stream?entities=sos,%20patrulje")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	reader := bufio.NewReader(resp.Body)
	readEvent(t, reader)

	hub.Publish(changedSignal("scan", "s-1", "2026", "scanned"))
	hub.Publish(changedSignal("sos", "c-1", "2026", "commented"))

	// The scan is filtered out; whitespace around the comma must not break the
	// entity names.
	_, data := readEvent(t, reader)
	var got Signal
	if err := json.Unmarshal([]byte(data), &got); err != nil {
		t.Fatal(err)
	}
	if got.Entity != "sos" {
		t.Errorf("received %+v, want the sos signal only", got)
	}
}

func TestStreamDefaultsYearWhenAbsent(t *testing.T) {
	hub := newTestHub(t)
	srv := serve(t, StreamHandler{Hub: hub, DefaultYear: func() string { return "2026" }})

	resp, err := http.Get(srv.URL + "/api/stream")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	reader := bufio.NewReader(resp.Body)
	readEvent(t, reader)

	// Subscribed to the default year, so another year's signal must not arrive.
	hub.Publish(changedSignal("patrulje", "old", "2025", "updated"))
	hub.Publish(changedSignal("patrulje", "new", "2026", "updated"))

	_, data := readEvent(t, reader)
	var got Signal
	if err := json.Unmarshal([]byte(data), &got); err != nil {
		t.Fatal(err)
	}
	if got.ID != "new" {
		t.Errorf("received %+v, want only the default-year signal", got)
	}
}

func TestStreamEmitsHeartbeat(t *testing.T) {
	hub := newTestHub(t)
	srv := serve(t, StreamHandler{
		Hub:         hub,
		Heartbeat:   10 * time.Millisecond,
		DefaultYear: func() string { return "2026" },
	})

	resp, err := http.Get(srv.URL + "/api/stream")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	reader := bufio.NewReader(resp.Body)
	readEvent(t, reader) // initial resync

	// Look for a comment line, which is what keeps an idle connection alive
	// through a proxy's idle timeout.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("reading stream: %v", err)
		}
		if strings.HasPrefix(line, ":") {
			return
		}
	}
	t.Error("no heartbeat comment observed")
}

// Regression test for a failure that is invisible until it matters: the API's
// http.Server sets WriteTimeout to protect ordinary endpoints from slow clients,
// and that deadline applies to the whole response. On a stream it does not refuse
// the connection — it lets the first events through and then kills writes
// mid-flight, which reads like "the proxy is buffering" rather than "our own
// server hung up".
//
// A short WriteTimeout plus a fast heartbeat reproduces it in under a second.
func TestStreamSurvivesServerWriteTimeout(t *testing.T) {
	hub := newTestHub(t)

	srv := httptest.NewUnstartedServer(StreamHandler{
		Hub:         hub,
		Heartbeat:   20 * time.Millisecond,
		DefaultYear: func() string { return "2026" },
	})
	srv.Config.WriteTimeout = 100 * time.Millisecond
	srv.Config.ReadTimeout = 100 * time.Millisecond
	srv.Start()
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/api/stream")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	reader := bufio.NewReader(resp.Body)
	readEvent(t, reader) // initial resync

	// Well past the write timeout: 15 heartbeats at 20ms is ~300ms, three times the
	// 100ms deadline, so this cannot pass by finishing before the deadline bites.
	heartbeats := 0
	deadline := time.Now().Add(3 * time.Second)
	for heartbeats < 15 && time.Now().Before(deadline) {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("stream died after %d heartbeats: %v", heartbeats, err)
		}
		if strings.HasPrefix(line, ":") {
			heartbeats++
		}
	}

	if heartbeats < 15 {
		t.Errorf("only %d heartbeats before the deadline; the write timeout is still killing the stream", heartbeats)
	}
}

func TestStreamUnsubscribesWhenClientDisconnects(t *testing.T) {
	hub := newTestHub(t)
	srv := serve(t, StreamHandler{Hub: hub, DefaultYear: func() string { return "2026" }})

	ctx, cancel := context.WithCancel(context.Background())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/api/stream", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	readEvent(t, bufio.NewReader(resp.Body))

	if got := hub.ClientCount(); got != 1 {
		t.Fatalf("ClientCount() = %d while connected, want 1", got)
	}

	cancel()
	resp.Body.Close()

	// A disconnected browser must not leave a subscription behind.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if hub.ClientCount() == 0 {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Errorf("ClientCount() = %d after disconnect, want 0", hub.ClientCount())
}

// The announcement must arrive before the initial resync. A client that validated
// its dependencies only when the set arrived would otherwise have already reacted to
// signals it could not check.
func TestStreamAnnouncesEntitiesBeforeAnySignal(t *testing.T) {
	hub := newTestHub(t)
	set := EntitySet{Entities: []string{"klan", "qr"}, Exhaustive: false}
	srv := serve(t, StreamHandler{
		Hub:         hub,
		DefaultYear: func() string { return "2026" },
		Entities:    &set,
	})

	resp, err := http.Get(srv.URL + "/api/stream")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	r := bufio.NewReader(resp.Body)

	event, data := readEvent(t, r)
	if event != SignalEntities {
		t.Fatalf("first event = %q, want %q", event, SignalEntities)
	}

	var got EntitySet
	if err := json.Unmarshal([]byte(data), &got); err != nil {
		t.Fatalf("unmarshalling %q: %v", data, err)
	}
	if len(got.Entities) != 2 || got.Entities[0] != "klan" || got.Entities[1] != "qr" {
		t.Errorf("entities = %q, want [klan qr]", got.Entities)
	}
	// Round-tripping this matters: a client that read a non-exhaustive set as
	// exhaustive would reject dependencies that are in fact legitimate.
	if got.Exhaustive {
		t.Error("exhaustive = true, want false to survive the wire")
	}

	if event, _ := readEvent(t, r); event != SignalResync {
		t.Errorf("second event = %q, want %q", event, SignalResync)
	}
}

// A handler with no set configured must stream exactly as before, so the
// announcement is additive rather than required.
func TestStreamWithoutEntitiesAnnouncesNothing(t *testing.T) {
	hub := newTestHub(t)
	srv := serve(t, StreamHandler{Hub: hub, DefaultYear: func() string { return "2026" }})

	resp, err := http.Get(srv.URL + "/api/stream")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()

	if event, _ := readEvent(t, bufio.NewReader(resp.Body)); event != SignalResync {
		t.Errorf("first event = %q, want %q", event, SignalResync)
	}
}
