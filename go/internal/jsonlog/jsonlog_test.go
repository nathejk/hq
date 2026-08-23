package jsonlog

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// A logger that crashes while reporting a crash is the worst place for a nil dereference: it
// replaces the diagnosis with a nil-pointer trace from inside the logging call, and the original
// problem never gets printed. That is what happened on every graceful shutdown of the API until
// task 107, so the nil case has a test.

func decode(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("log line is not JSON: %v (%q)", err, buf.String())
	}
	return entry
}

func TestPrintErrorWithNilDoesNotPanic(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(buf, LevelInfo)

	// The assertion is that this line returns at all.
	logger.PrintError(nil, nil)

	entry := decode(t, buf)
	if entry["level"] != "ERROR" {
		t.Errorf("level = %v, want ERROR", entry["level"])
	}
	// Not an empty message: a caller that reaches here with nil has a bug of its own, and an empty
	// line would give nobody a thread to pull.
	msg, _ := entry["message"].(string)
	if msg == "" {
		t.Error("a nil error logged an empty message, which reports nothing")
	}
	if !strings.Contains(msg, "nil") {
		t.Errorf("message = %q, want it to say the error was nil", msg)
	}
}

func TestPrintErrorWithAnErrorLogsIt(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(buf, LevelInfo)

	logger.PrintError(errors.New("kunne ikke nå databasen"), map[string]string{"host": "mysql"})

	entry := decode(t, buf)
	if entry["message"] != "kunne ikke nå databasen" {
		t.Errorf("message = %v", entry["message"])
	}
	props, ok := entry["properties"].(map[string]any)
	if !ok || props["host"] != "mysql" {
		t.Errorf("properties = %v", entry["properties"])
	}
}

// PrintFatal exits the process, so it cannot be called from a test — errorMessage is where its nil
// handling lives, and testing that is testing the fix.
func TestErrorMessageHandlesNil(t *testing.T) {
	if got := errorMessage(nil); got == "" {
		t.Error("nil rendered as an empty message")
	}
	if got := errorMessage(errors.New("boom")); got != "boom" {
		t.Errorf("errorMessage = %q, want %q", got, "boom")
	}
}

// Below the minimum level nothing is written at all, which is worth pinning next to the above: a
// guard that only works at ERROR would be no guard at all if the level were raised.
func TestBelowMinLevelWritesNothing(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(buf, LevelError)

	logger.PrintInfo("ignorerbar", nil)

	if buf.Len() != 0 {
		t.Errorf("expected nothing written, got %q", buf.String())
	}
}
