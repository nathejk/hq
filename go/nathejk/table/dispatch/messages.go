package dispatch

import "github.com/nathejk/shared-go/types"

// SectionDispatchableSet marks an organisation section as being (or no longer being)
// a dispatch unit: a subsection of logistics that holds a vehicle, a driver and
// possibly a co-driver, and that tours may be assigned to.
//
// This is a fact about kørsel rather than about the section, which is why it is a
// dispatch event and a dispatch table instead of a column on shared-go's section —
// the same decision PRD 001 took for the nødtelefon's assignable sections, for the
// same reason. Published on NATHEJK.{year}.dispatch.section.{slug}.dispatchable, with
// the new state in the body rather than as two subjects: it is one fact with two
// values, and a consumer matching two subjects can handle one and silently miss the
// other.
type SectionDispatchableSet struct {
	SectionSlug  types.Slug `json:"sectionSlug"`
	Dispatchable bool       `json:"dispatchable"`
}

// --- Tasks: NATHEJK.{year}.dispatch.{taskId}.{event} ---

// TaskCreated opens a task. Almost everything is optional, which is not laxity: PRD 009
// §8 names desk discipline as the feature's biggest risk, and the mitigation is that the
// written path is the fastest path.
type TaskCreated struct {
	TaskID      TaskID   `json:"taskId"`
	Kind        Kind     `json:"kind"`
	Priority    Priority `json:"priority,omitempty"`
	Description string   `json:"description"`
	SpaceNeeds  string   `json:"spaceNeeds,omitempty"`
	Pickup      Place    `json:"pickup"`
	Dropoff     Place    `json:"dropoff"`

	// CreatedUts is the waiting clock. On the event rather than taken from the message
	// time by the projection, because a task backdated by an operator ("they have been
	// standing there since half past") must wait from when they say, and because replay
	// must not move it.
	CreatedUts   int64  `json:"createdUts"`
	NotBeforeUts *int64 `json:"notBeforeUts,omitempty"`
	DeadlineUts  *int64 `json:"deadlineUts,omitempty"`

	// Where a pickup came from, so the nødtelefon operator can read the expected time
	// off the case without opening the dispatch board.
	SosID     types.SosID      `json:"sosId,omitempty"`
	TeamID    types.TeamID     `json:"teamId,omitempty"`
	MemberIDs []types.MemberID `json:"memberIds,omitempty"`
}

// TaskUpdated carries the task's editable fields in full, after the edit.
//
// A whole-field event rather than a sparse patch: the command has already merged the
// operator's PATCH onto the current row, and putting the merged result on the event means
// the projection has no merge logic and replay cannot depend on the order two overlapping
// edits are applied in. The cost is that the event does not say which field an operator
// touched — which the timeline entry does.
type TaskUpdated struct {
	TaskID       TaskID   `json:"taskId"`
	Kind         Kind     `json:"kind"`
	Priority     Priority `json:"priority,omitempty"`
	Description  string   `json:"description"`
	SpaceNeeds   string   `json:"spaceNeeds,omitempty"`
	Pickup       Place    `json:"pickup"`
	Dropoff      Place    `json:"dropoff"`
	NotBeforeUts *int64   `json:"notBeforeUts,omitempty"`
	DeadlineUts  *int64   `json:"deadlineUts,omitempty"`

	// Changed names the fields the operator actually sent, for the timeline line. Not
	// used by the projection.
	Changed []string `json:"changed,omitempty"`
}

// TaskPlanned says a task is now in a tour.
//
// Published alongside the tour's own stops.changed rather than derived from it, so that a
// task's state is event-driven and its timeline says "lagt i tur" at the moment it
// happened. The redundancy is deliberate: the tour event is the plan, this is the
// consequence for one task, and the driver app will care about exactly one of them.
type TaskPlanned struct {
	TaskID TaskID `json:"taskId"`
	TourID TourID `json:"tourId"`
}

// TaskUnplanned returns a task to the queue.
//
// It carries no clock: the waiting clock is never reset, because the scout has been
// waiting since the call and not since the re-plan (PRD 009 §5).
type TaskUnplanned struct {
	TaskID TaskID `json:"taskId"`
	TourID TourID `json:"tourId"`
}

// TaskUnderway says the tour carrying this task has set off.
type TaskUnderway struct {
	TaskID TaskID `json:"taskId"`
	TourID TourID `json:"tourId"`
}

// TaskPickedUp is people aboard: the moment custody changes.
//
// Distinct from completion because they are different moments — the car still has to get
// to HQ — and because this is the event that fills Hønsegården's *På vej* (PRD 007 §8).
// The unit is a section slug rather than a vehicle id: the unit is who took them, and it
// survives a car being swapped mid-night.
type TaskPickedUp struct {
	TaskID      TaskID           `json:"taskId"`
	SectionSlug types.Slug       `json:"sectionSlug,omitempty"`
	MemberIDs   []types.MemberID `json:"memberIds,omitempty"`
	AtUts       int64            `json:"atUts"`
}

type TaskCompleted struct {
	TaskID TaskID `json:"taskId"`
	AtUts  int64  `json:"atUts"`
}

// TaskCancelled always carries a reason. A cancelled task with no explanation is the one
// thing a handover cannot recover from.
type TaskCancelled struct {
	TaskID TaskID `json:"taskId"`
	Reason string `json:"reason"`
	AtUts  int64  `json:"atUts"`
}

// --- Tours: NATHEJK.{year}.tour.{tourId}.{event} ---

// TourCreated opens a tour for one unit.
type TourCreated struct {
	TourID       TourID     `json:"tourId"`
	SectionSlug  types.Slug `json:"sectionSlug"`
	DepartureUts *int64     `json:"departureUts,omitempty"`
	Notes        string     `json:"notes,omitempty"`
	CreatedUts   int64      `json:"createdUts"`
}

// TourUpdated carries departure, unit and notes in full after an edit, for the reason
// TaskUpdated does.
type TourUpdated struct {
	TourID       TourID     `json:"tourId"`
	SectionSlug  types.Slug `json:"sectionSlug"`
	DepartureUts *int64     `json:"departureUts,omitempty"`
	Notes        string     `json:"notes,omitempty"`
	Changed      []string   `json:"changed,omitempty"`
}

// Stop is one place on a tour, with what is done there.
type Stop struct {
	StopID StopID `json:"stopId"`
	Place  Place  `json:"place"`

	// PlannedUts is derived from the tour's departure plus a per-leg allowance unless
	// Override says a dispatcher typed it. Deriving keeps planning fast — nobody types
	// six times at 3am — and the override is what makes deriving acceptable: the moment
	// the desk knows better it can say so, and the stops after re-derive from there.
	PlannedUts *int64 `json:"plannedUts,omitempty"`
	Override   bool   `json:"override,omitempty"`

	// VisitedUts is carried so that a whole-list replacement cannot lose it. Visited
	// stops are fixed, and the projection rebuilds the list from this event alone.
	VisitedUts *int64 `json:"visitedUts,omitempty"`

	Tasks []StopTask `json:"tasks"`
}

// StopTask is one task actioned at a stop.
type StopTask struct {
	TaskID TaskID `json:"taskId"`
	Role   Role   `json:"role"`
}

// StopsChanged replaces a tour's whole ordered stop list.
//
// The whole list in one event rather than add/remove/move events, matching the single
// PUT it comes from: a reorder is one operator intent, and three finer-grained events
// would make a half-applied reorder representable — a state the board would then have to
// render.
//
// The order of the slice *is* the order of the stops. No sort keys on the wire: two
// sources of truth for an ordering is how orderings get corrupted.
type StopsChanged struct {
	TourID TourID `json:"tourId"`
	Stops  []Stop `json:"stops"`
}

type TourUnderway struct {
	TourID TourID `json:"tourId"`
	AtUts  int64  `json:"atUts"`
}

// StopVisited is a stop reached. It advances the tour and progresses the tasks at it —
// but those task transitions are published as their own events, so nothing has to infer
// a task's state from a tour's stop rows.
type StopVisited struct {
	TourID TourID `json:"tourId"`
	StopID StopID `json:"stopId"`
	AtUts  int64  `json:"atUts"`
}

type TourCompleted struct {
	TourID TourID `json:"tourId"`
	AtUts  int64  `json:"atUts"`
}

type TourCancelled struct {
	TourID TourID `json:"tourId"`
	Reason string `json:"reason"`
	AtUts  int64  `json:"atUts"`
}
