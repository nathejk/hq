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
	"github.com/google/uuid"
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

// TaskID identifies one thing that needs moving.
type TaskID types.ID

// TourID identifies one car's run.
type TourID types.ID

// StopID identifies a place on a tour.
//
// Minted per stop rather than derived from (tour, position), because a stop's identity
// must survive a reorder: the driver-facing
// POST /api/dispatch/tour/:id/stop/:stopId/visited would otherwise mark whatever has
// since moved into third place.
type StopID types.ID

// The server mints every id, so a client cannot choose or collide with one.
func NewTaskID() TaskID { return TaskID("dispatchtask-" + uuid.New().String()) }
func NewTourID() TourID { return TourID("dispatchtour-" + uuid.New().String()) }
func NewStopID() StopID { return StopID("dispatchstop-" + uuid.New().String()) }

// Kind is what sort of job a task is.
//
// Four values because they read differently on a board and default their places
// differently — a delivery leaves HQ, a collection returns to it — and not because
// their lifecycles differ. They do not: one state machine serves all four.
type Kind string

const (
	// KindPickup is people. The only kind that changes custody, and the reason the
	// pickedup transition exists.
	KindPickup Kind = "pickup"
	// KindTransport is a thing from A to B, neither end being HQ.
	KindTransport Kind = "transport"
	// KindCollection is fetching something to HQ.
	KindCollection Kind = "collection"
	// KindDelivery is taking something out from HQ.
	KindDelivery Kind = "delivery"
)

func (k Kind) Valid() bool {
	switch k {
	case KindPickup, KindTransport, KindCollection, KindDelivery:
		return true
	}
	return false
}

// Priority is how urgent a task is.
//
// The SOS severity vocabulary — same values, same Danish labels, same theme colours —
// so that two race-night desks do not have two words for urgent and a pickup created
// from a red case can arrive red.
//
// Defined here rather than imported from the `sos` package on purpose (PRD 009 §8): a
// delivery of dinner must not depend on the emergency-phone entity to know what "rød"
// means. Three duplicated string constants is the cheaper wrong thing. When `sos` is
// lifted to shared-go (task 055), a shared types.Severity is the obvious home for
// both, and that is the moment to converge them.
type Priority string

const (
	PriorityGreen  Priority = "green"
	PriorityYellow Priority = "yellow"
	PriorityRed    Priority = "red"
)

// Valid reports whether p is one of the three. The empty priority is not valid, but a
// task may legitimately have none — most tasks are ordinary — so callers check Valid
// only on input.
func (p Priority) Valid() bool {
	switch p {
	case PriorityGreen, PriorityYellow, PriorityRed:
		return true
	}
	return false
}

// TaskState is where a task has got to.
//
// queued → planned → underway → done, plus cancelled from anywhere. A pickup
// additionally records `pickedUp` on the way to done, which is a timestamp rather than
// a state: custody changes when people get in the car, and that is not when the task
// finishes.
type TaskState string

const (
	TaskStateQueued    TaskState = "queued"
	TaskStatePlanned   TaskState = "planned"
	TaskStateUnderway  TaskState = "underway"
	TaskStateDone      TaskState = "done"
	TaskStateCancelled TaskState = "cancelled"
)

// TourState is where a tour has got to.
//
// The constants are prefixed `TourState`/`TaskState` rather than the shorter `TourPlanned`,
// because the event payloads in messages.go own the short names — `TaskPlanned` is the
// event that says a task went into a tour, and it is worth more there than as a state.
type TourState string

const (
	TourStatePlanned   TourState = "planned"
	TourStateUnderway  TourState = "underway"
	TourStateCompleted TourState = "completed"
	TourStateCancelled TourState = "cancelled"
)

// PlaceKind says what a place refers to.
//
// PlaceText is not a fallback for missing data: "på Slangerupvej ved skovbrynet" is the
// normal way to describe where a scout is standing, and a picker that only offered
// known locations would be worked around by typing the road name into the description.
type PlaceKind string

const (
	PlaceCheckpoint PlaceKind = "checkpoint"
	PlaceLok        PlaceKind = "lok"
	PlaceHQ         PlaceKind = "hq"
	PlaceText       PlaceKind = "text"
)

// Place is where something is picked up or dropped off.
//
// Type + reference + label rather than a foreign key, for two independent reasons: a
// place may be free text, and a checkpoint's name should stay as it was on the task
// even if the checkpoint is later renamed. The label is therefore a copy on purpose,
// not a denormalisation to be cleaned up.
type Place struct {
	Kind  PlaceKind `json:"kind"`
	RefID string    `json:"refId,omitempty"`
	Label string    `json:"label"`
}

// Role is what happens to a task at a stop.
//
// A task that moves something occupies two stops, and the board must not let the
// unload be ordered before the load — which is only checkable because the pair is
// labelled.
type Role string

const (
	RoleLoad   Role = "load"
	RoleUnload Role = "unload"
	// RoleAction is a task that is simply done at one place.
	RoleAction Role = "action"
)

// ActivityType tags an entry on the timeline.
//
// A string rather than an enum with exhaustive switches, for the reason sos learned:
// the set grows, and both the projection and the SPA must tolerate a type they do not
// recognise rather than failing on it.
type ActivityType string

const (
	ActivityTaskCreated   ActivityType = "task.created"
	ActivityTaskUpdated   ActivityType = "task.updated"
	ActivityTaskPlanned   ActivityType = "task.planned"
	ActivityTaskUnplanned ActivityType = "task.unplanned"
	ActivityTaskUnderway  ActivityType = "task.underway"
	ActivityTaskPickedUp  ActivityType = "task.pickedup"
	ActivityTaskCompleted ActivityType = "task.completed"
	ActivityTaskCancelled ActivityType = "task.cancelled"

	ActivityTourCreated      ActivityType = "tour.created"
	ActivityTourUpdated      ActivityType = "tour.updated"
	ActivityTourStopsChanged ActivityType = "tour.stops.changed"
	ActivityTourUnderway     ActivityType = "tour.underway"
	ActivityTourStopVisited  ActivityType = "tour.stop.visited"
	ActivityTourCompleted    ActivityType = "tour.completed"
	ActivityTourCancelled    ActivityType = "tour.cancelled"
)
