// Package live turns domain events into invalidation signals for connected
// browsers, and fans them out over SSE.
//
// The design is deliberately entity-agnostic: the stream carries *signals, not
// data*, and every signal is derived from the event subject alone. That keeps one
// serialization path (REST), stops pushed and fetched shapes drifting apart, and
// means making a new entity live is a wiring change rather than new code here.
//
// See PRD 004, "Live updates for the SPA" (roadmap/prd/ — folder by status).
package live

import (
	"errors"
	"strings"

	"github.com/jrgensen/cqrs"
)

// Signal names. These reach the browser as the SSE `event:` line, so the client
// can dispatch on them and additional kinds stay additive.
const (
	SignalEntityChanged = "entity.changed"
	SignalResync        = "resync"
)

// ErrNotASignal reports a subject this package cannot express as a signal.
//
// Returned rather than yielding a zero-valued Signal on purpose: a signal naming
// entity "" would invalidate nothing while appearing to work, which is worse than
// no signal at all.
var ErrNotASignal = errors.New("live: subject cannot be expressed as a signal")

// Signal is one invalidation notice.
//
// It carries no entity data, which is what keeps the stream free of its own
// authorisation model: learning that payment/123 changed reveals nothing, and the
// client's refetch still goes through the normal authorised endpoint.
type Signal struct {
	// Type is SignalEntityChanged or SignalResync.
	Type string `json:"type"`

	// Entity is the subject's entity token: "patrulje", "payment", "sos", …
	// Note this is the domain token, not a UI name, and it may be non-ASCII
	// ("gøgler") or plural for collection-level events ("checkgroups").
	Entity string `json:"entity,omitempty"`

	// ID is empty for subjects that carry none — year-level and
	// collection-level events — in which case the signal means "something of
	// this type changed".
	ID string `json:"id,omitempty"`

	Year string `json:"year,omitempty"`

	// Event is everything after the id, so dotted names survive whole
	// ("status.changed", "armNumber.assigned", "lines.changed").
	//
	// Advisory only. Signals are coalesced per (entity, id), so when several
	// events collapse the surviving name is arbitrary — never decide what to
	// refetch from it.
	Event string `json:"event,omitempty"`
}

// Key identifies what a signal is about, for coalescing. Two signals with the
// same key are interchangeable as far as a client is concerned: both mean
// "refetch this".
func (s Signal) Key() string {
	return s.Entity + ":" + s.ID
}

// Resync tells a client to revalidate everything it holds.
//
// Emitted on subscribe, and when a client's buffer would otherwise overflow —
// collapsing a backlog into one of these is what lets the hub avoid dropping
// invalidations silently.
func Resync() Signal {
	return Signal{Type: SignalResync}
}

// The subject convention, as documented in .rules and as actually published by
// this repo's commands:
//
//	NATHEJK.{year}.{entity}.{id}.{event…}
//
// with three real variations to survive:
//
//   - the event may contain dots: "status.changed", "armNumber.assigned"
//   - year-level events carry no entity or id: "NATHEJK.{year}.created"
//   - collection-level events carry no id: "NATHEJK.{year}.checkgroups.sorted"
//
// Both "NATHEJK." and "NATHEJK:" appear in this codebase; subject.FromStr
// normalises the first ":" to "." before we see it, so no special handling is
// needed here — but see the tests, which pin that behaviour precisely because it
// is invisible and load-bearing.
const domain = "NATHEJK"

// SignalFromSubject derives a signal from an event subject.
//
// The whole point of the live-update design rests here: because the subject
// already encodes year, entity, id and event, one function serves every entity
// that exists or ever will, and no per-entity code is needed to make a page live.
func SignalFromSubject(subj cqrs.Subject) (Signal, error) {
	parts := strings.Split(subj.Subject(), ".")

	// parts[0] is the domain; at minimum we need a year after it.
	if len(parts) < 3 || !strings.EqualFold(parts[0], domain) {
		return Signal{}, ErrNotASignal
	}

	year := parts[1]
	if year == "" {
		return Signal{}, ErrNotASignal
	}

	switch len(parts) {
	case 3:
		// NATHEJK.{year}.{event} — the year entity itself changed. Report it as
		// the "year" entity so a client can depend on it by name; the year value
		// doubles as the id, since that is what identifies which year changed.
		return Signal{
			Type:   SignalEntityChanged,
			Entity: "year",
			ID:     year,
			Year:   year,
			Event:  parts[2],
		}, nil

	case 4:
		// NATHEJK.{year}.{entity}.{event} — collection-level, no id.
		if parts[2] == "" {
			return Signal{}, ErrNotASignal
		}
		return Signal{
			Type:   SignalEntityChanged,
			Entity: parts[2],
			Year:   year,
			Event:  parts[3],
		}, nil

	default:
		// NATHEJK.{year}.{entity}.{id}.{event…}
		if parts[2] == "" || parts[3] == "" {
			return Signal{}, ErrNotASignal
		}
		return Signal{
			Type:   SignalEntityChanged,
			Entity: parts[2],
			ID:     parts[3],
			Year:   year,
			Event:  strings.Join(parts[4:], "."),
		}, nil
	}
}
