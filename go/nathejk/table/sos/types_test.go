package sos

import "testing"

func TestSeverityValid(t *testing.T) {
	for _, s := range []Severity{SeverityGreen, SeverityYellow, SeverityRed} {
		if !s.Valid() {
			t.Errorf("Severity(%q).Valid() = false, want true", s)
		}
	}
	// The empty severity is rejected on input even though a case may have none
	// stored: "unset" is the projection's business, not something a client asks
	// for.
	for _, s := range []Severity{"", "GREEN", "orange", "rød"} {
		if s.Valid() {
			t.Errorf("Severity(%q).Valid() = true, want false", s)
		}
	}
}

func TestStatusValid(t *testing.T) {
	for _, s := range []Status{StatusOpen, StatusClosed} {
		if !s.Valid() {
			t.Errorf("Status(%q).Valid() = false, want true", s)
		}
	}
	for _, s := range []Status{"", "Open", "afsluttet", "deleted"} {
		if s.Valid() {
			t.Errorf("Status(%q).Valid() = true, want false", s)
		}
	}
}

func TestNewIDsArePrefixedAndUnique(t *testing.T) {
	// The prefix is what makes an id readable in a log line or a stream subject,
	// and every other id type in the platform carries one.
	a, b := NewSosID(), NewSosID()
	if a == b {
		t.Error("NewSosID() returned the same id twice")
	}
	if len(a) < len("sos-") || a[:4] != "sos-" {
		t.Errorf("NewSosID() = %q, want a sos- prefix", a)
	}

	c, d := NewCommentID(), NewCommentID()
	if c == d {
		t.Error("NewCommentID() returned the same id twice")
	}
	if len(c) < len("soscomment-") || c[:11] != "soscomment-" {
		t.Errorf("NewCommentID() = %q, want a soscomment- prefix", c)
	}
}
