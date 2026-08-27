// Package dispatch is the kørsel domain (PRD 009): the tasks the cars are asked to
// do, the tours they are put into, and which organisation sections may run them.
//
// English identifiers throughout — `dispatch`, `tour`, `task`, `stop` — while the
// interface says "Kørsel". Danish names are kept for the domain words that have no
// clean English equivalent (patrulje, klan, lok); these have one.
//
// The package is written to shared-go's guidelines so it *could* be lifted later,
// which is why nothing here imports nathejk.dk/... and the acting user is passed in
// by the caller rather than read from a request context. It stays hq-owned for now
// (PRD 009 §8): tilmelding has no use for it.
package dispatch

import (
	"github.com/nathejk/shared-go/types"
)

// Actor is who performed an action. Passed in by the HTTP handler, as in `sos`.
//
// hq authenticates nobody yet, so UserID is empty in practice (PRD 009 §8,
// "Attribution without authentication"): the *unit* a tour belongs to is an explicit
// choice by the dispatcher, precisely because the person at the keyboard is not the
// person in the car, and that choice does not become redundant when identity arrives.
type Actor struct {
	UserID types.UserID
	Name   string
}
