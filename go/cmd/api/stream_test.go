package main

import (
	"bufio"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"nathejk.dk/internal/live"
)

// The stream is mounted on the mux by hand rather than on the httprouter, because
// it must bypass app.Metrics: metricsResponseWriter does not implement
// http.Flusher. This exercises the real chain — ServeMux pattern precedence,
// authenticate, and the flush — since getting any of those wrong produces a
// stream that never delivers anything.
func TestStreamRouteBypassesMetricsAndStreams(t *testing.T) {
	hub := live.NewHub(live.WithCoalesceWindow(5 * time.Millisecond))
	defer hub.Close()

	app := &application{
		config: config{webroot: t.TempDir()},
		live:   hub,
	}

	srv := httptest.NewServer(app.routes())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/stream?year=2026")
	if err != nil {
		t.Fatalf("GET /api/stream: %v", err)
	}
	defer resp.Body.Close()

	// The same mux serves the SPA through a fallback that returns index.html for
	// unknown paths, so a mis-registered pattern would answer with HTML rather
	// than 404 and the client would fail parsing an "event stream".
	if got := resp.Header.Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("Content-Type = %q, want text/event-stream (is the route registered?)", got)
	}

	reader := bufio.NewReader(resp.Body)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("reading stream: %v", err)
		}
		// Reaching the client at all proves the response was flushed through the
		// middleware chain rather than buffered until close.
		if strings.HasPrefix(line, "event: "+live.SignalResync) {
			return
		}
	}
	t.Error("no resync event arrived; the response was not flushed")
}
