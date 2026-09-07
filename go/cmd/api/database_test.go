package main

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

// The API used to die if MySQL was not already accepting connections when it booted, which
// is a race it loses routinely on `docker compose up` — and because the dev loop only
// rebuilds on a file change, nothing restarted it afterwards. These tests pin the two
// halves of the fix: it waits, and it still gives up.

func testDatabase(t *testing.T, dsn string, budget, maxWait time.Duration) *database {
	t.Helper()
	db := NewDatabase(DatabaseConfig{
		dsn:          dsn,
		maxOpenConns: 1,
		maxIdleConns: 1,
		maxIdleTime:  "1m",
	})
	db.connectBudget = budget
	db.connectMaxWait = maxWait
	return db
}

// unreachableDSN points at a port nothing is listening on, which is what a not-yet-started
// MySQL looks like from the client's side.
func unreachableDSN(t *testing.T) string {
	t.Helper()
	// Bind and immediately close, so the port is almost certainly free and therefore
	// refuses rather than hanging.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("could not find a free port: %v", err)
	}
	addr := l.Addr().String()
	if err := l.Close(); err != nil {
		t.Fatalf("could not release port: %v", err)
	}
	return "root:secret@tcp(" + addr + ")/hq?parseTime=true"
}

func TestOpenGivesUpAfterBudget(t *testing.T) {
	db := testDatabase(t, unreachableDSN(t), 250*time.Millisecond, 25*time.Millisecond)

	start := time.Now()
	err := db.Open()
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected an error when nothing is listening")
	}
	// The message has to say how long it tried and how often: "connection refused" alone
	// is what sent somebody hunting through container logs in the first place.
	for _, want := range []string{"database unreachable", "attempts"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err, want)
		}
	}
	// It must have retried, not merely failed slowly — the whole point is repeated attempts
	// while the database finishes starting.
	if n := attemptCount(t, err.Error()); n < 3 {
		t.Errorf("only %d attempt(s) within a 250ms budget at a 25ms cap; expected several", n)
	}
	// It must actually have waited rather than failing on the first ping.
	if elapsed < 200*time.Millisecond {
		t.Errorf("gave up after %s, expected to keep trying for about the budget", elapsed)
	}
	// And it must not overshoot the budget by a whole backoff step. Generous upper bound:
	// the point is that the budget is respected, not that scheduling is precise.
	if elapsed > 3*time.Second {
		t.Errorf("took %s, which is well past the 250ms budget", elapsed)
	}
}

// attemptCount reads the "after N attempts" out of the failure message.
func attemptCount(t *testing.T, msg string) int {
	t.Helper()
	const prefix = "after "
	i := strings.Index(msg, prefix)
	if i < 0 {
		t.Fatalf("no attempt count in %q", msg)
	}
	var n int
	if _, err := fmt.Sscanf(msg[i+len(prefix):], "%d attempts", &n); err != nil {
		t.Fatalf("could not read attempt count from %q: %v", msg, err)
	}
	return n
}

// A retry loop that swallows a permanent error is worse than no retry loop, so the returned
// error must still carry the driver's own explanation for whoever reads the log.
func TestOpenErrorWrapsTheUnderlyingCause(t *testing.T) {
	db := testDatabase(t, unreachableDSN(t), 100*time.Millisecond, 50*time.Millisecond)

	err := db.Open()
	if err == nil {
		t.Fatal("expected an error")
	}
	var opErr *net.OpError
	if !errors.As(err, &opErr) {
		t.Errorf("error %q does not wrap the underlying network error", err)
	}
}

// Config errors are not connection problems and must fail immediately, not after the budget.
func TestOpenRejectsBadIdleTimeWithoutWaiting(t *testing.T) {
	db := testDatabase(t, unreachableDSN(t), 10*time.Second, time.Second)
	db.cfg.maxIdleTime = "not-a-duration"

	start := time.Now()
	err := db.Open()

	if err == nil {
		t.Fatal("expected an error for an unparseable idle time")
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("took %s; a config error must not be retried", elapsed)
	}
}
