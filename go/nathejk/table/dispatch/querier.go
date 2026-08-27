package dispatch

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/nathejk/shared-go/tables"
	"github.com/nathejk/shared-go/types"
)

// Task is one thing that needs moving.
//
// Nullable times are pointers rather than zero values: 0 is 1970, and a zero deadline
// would render on the board as a task that was late fifty-six years ago. "No deadline" and
// "a deadline that has passed" must not be the same value on a screen whose whole purpose
// is showing what is about to be late.
type Task struct {
	ID       TaskID         `json:"id"`
	YearSlug types.YearSlug `json:"year"`

	Kind        Kind      `json:"kind"`
	Priority    Priority  `json:"priority,omitempty"`
	Description string    `json:"description"`
	SpaceNeeds  string    `json:"spaceNeeds,omitempty"`
	Pickup      Place     `json:"pickup"`
	Dropoff     Place     `json:"dropoff"`
	State       TaskState `json:"state"`

	// CreatedUts is the waiting clock, and the one number on this screen that needs no
	// model and is never wrong.
	CreatedUts   int64  `json:"createdUts"`
	NotBeforeUts *int64 `json:"notBeforeUts"`
	DeadlineUts  *int64 `json:"deadlineUts"`
	PickedUpUts  *int64 `json:"pickedUpUts"`
	DoneUts      *int64 `json:"doneUts"`
	CancelledUts *int64 `json:"cancelledUts"`
	CancelReason string `json:"cancelReason,omitempty"`

	SosID     types.SosID      `json:"sosId,omitempty"`
	TeamID    types.TeamID     `json:"teamId,omitempty"`
	MemberIDs []types.MemberID `json:"memberIds"`
	CreatedBy types.UserID     `json:"createdBy,omitempty"`

	// Stops are the stops this task sits on, in tour order — the load and the unload, or
	// the single action. This is where the answer to "when?" comes from: the planned time
	// of the stop, made by a human who knows the roads.
	//
	// Filled in by GetTask and by StopsByTask; the board leaves it empty and matches tasks
	// to stops from the tours it already has.
	Stops []TaskStop `json:"stops,omitempty"`

	// Timeline is filled in by GetTask only. The board neither needs it nor wants the join.
	Timeline []Activity `json:"timeline,omitempty"`
}

// TaskStop is one appearance of a task on a tour.
type TaskStop struct {
	TourID     TourID `json:"tourId"`
	StopID     StopID `json:"stopId"`
	Role       Role   `json:"role"`
	SortOrder  int    `json:"sortOrder"`
	Place      Place  `json:"place"`
	PlannedUts *int64 `json:"plannedUts"`
	Override   bool   `json:"override"`
	VisitedUts *int64 `json:"visitedUts"`
}

// Tour is one car's run.
type Tour struct {
	ID          TourID         `json:"id"`
	YearSlug    types.YearSlug `json:"year"`
	SectionSlug types.Slug     `json:"sectionSlug"`

	DepartureUts *int64    `json:"departureUts"`
	Notes        string    `json:"notes,omitempty"`
	State        TourState `json:"state"`
	CreatedUts   int64     `json:"createdUts"`
	UnderwayUts  *int64    `json:"underwayUts"`
	CompletedUts *int64    `json:"completedUts"`
	CancelledUts *int64    `json:"cancelledUts"`
	CancelReason string    `json:"cancelReason,omitempty"`

	// Stops in order, always non-nil. `[]` and never null: a null collection has broken
	// this repo's rendering three times, most memorably taking a dialog's own close button
	// with it and trapping the operator in a modal.
	Stops []TourStop `json:"stops"`
}

// TourStop is a place on a tour, with what is done there.
type TourStop struct {
	StopID     StopID     `json:"stopId"`
	SortOrder  int        `json:"sortOrder"`
	Place      Place      `json:"place"`
	PlannedUts *int64     `json:"plannedUts"`
	Override   bool       `json:"override"`
	VisitedUts *int64     `json:"visitedUts"`
	Tasks      []StopTask `json:"tasks"`
}

// Visited reports whether the stop has been reached. A method rather than a field the
// client compares against null, so nobody reimplements it — including, eventually, one
// caller getting it wrong.
func (s TourStop) Visited() bool { return s.VisitedUts != nil }

// Activity is one entry on the timeline.
type Activity struct {
	Seq    uint64       `json:"seq"`
	TaskID TaskID       `json:"taskId,omitempty"`
	TourID TourID       `json:"tourId,omitempty"`
	Type   ActivityType `json:"type"`
	Actor  types.UserID `json:"actorUserId,omitempty"`
	Value  string       `json:"value,omitempty"`
	AtUts  int64        `json:"atUts"`
}

// Queries is what the application may read. An interface so cmd/api depends on the
// queries rather than on the table implementation.
type Queries interface {
	DispatchableSections(context.Context, types.YearSlug) ([]types.Slug, error)

	Tasks(context.Context, Filter) ([]*Task, error)
	GetTask(context.Context, types.YearSlug, TaskID) (*Task, error)
	Tours(context.Context, TourFilter) ([]*Tour, error)
	GetTour(context.Context, types.YearSlug, TourID) (*Tour, error)

	// StopsByTask answers "where does each of these tasks appear, and when is it planned
	// for" in one query, for a caller holding tasks but not the tours — the SOS case being
	// the one that matters (PRD 009 §6). Batched rather than per task: a case with four
	// waiting members must not be four round trips.
	StopsByTask(context.Context, types.YearSlug, []TaskID) (map[TaskID][]TaskStop, error)

	// Duty windows, whole roster for the year. Ordered by start, so the caller can answer both
	// "who is on now" and "when does the next unit come on" from one list rather than two
	// queries — there are fewer than ten units and a night's worth of shifts.
	DutyWindows(context.Context, types.YearSlug) ([]Duty, error)
}

// Duty is one window in which a unit is available.
type Duty struct {
	ID          DutyID         `json:"id"`
	YearSlug    types.YearSlug `json:"year"`
	SectionSlug types.Slug     `json:"sectionSlug"`
	StartUts    int64          `json:"startUts"`
	EndUts      int64          `json:"endUts"`
}

// Covers reports whether the window includes an instant.
//
// Half-open on the end: a shift ending at 22.00 does not include 22.00, so two consecutive
// windows do not both claim the same minute — which would make "who is on now" answer twice for
// one unit and read as a configuration error on the strip.
func (d Duty) Covers(uts int64) bool { return d.StartUts <= uts && uts < d.EndUts }

type querier struct {
	db *sql.DB
	r  *goqu.Database
}

const taskColumns = `id, year, kind, priority, description, spaceNeeds,
	pickupKind, pickupRefId, pickupLabel, dropoffKind, dropoffRefId, dropoffLabel,
	state, createdUts, notBeforeUts, deadlineUts, pickedUpUts, doneUts, cancelledUts,
	cancelReason, sosId, teamId, memberIds, createdBy`

// Tasks lists tasks, oldest first.
//
// Oldest first rather than newest: this is a queue, and the question the board asks of it
// is "who has waited longest", not "what came in last".
func (q *querier) Tasks(ctx context.Context, f Filter) ([]*Task, error) {
	query := `SELECT ` + taskColumns + ` FROM dispatch_task WHERE (year = ? OR ? = '')`
	args := []any{f.YearSlug, f.YearSlug}
	if len(f.States) > 0 {
		query += ` AND state IN (` + placeholders(len(f.States)) + `)`
		for _, s := range f.States {
			args = append(args, string(s))
		}
	}
	if f.SosID != "" {
		query += ` AND sosId = ?`
		args = append(args, string(f.SosID))
	}
	query += ` ORDER BY createdUts ASC, id ASC`

	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := []*Task{}
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

// GetTask returns one task with its stops and its full timeline.
func (q *querier) GetTask(ctx context.Context, year types.YearSlug, id TaskID) (*Task, error) {
	if id == "" {
		return nil, tables.ErrRecordNotFound
	}
	query := `SELECT ` + taskColumns + ` FROM dispatch_task WHERE id = ? AND (year = ? OR ? = '')`
	t, err := scanTask(q.db.QueryRowContext(ctx, query, string(id), year, year))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, tables.ErrRecordNotFound
		}
		return nil, err
	}
	stops, err := q.StopsByTask(ctx, year, []TaskID{id})
	if err != nil {
		return nil, err
	}
	t.Stops = stops[id]
	if t.Timeline, err = q.taskTimeline(ctx, id); err != nil {
		return nil, err
	}
	return t, nil
}

const tourColumns = `id, year, sectionSlug, departureUts, notes, state, createdUts,
	underwayUts, completedUts, cancelledUts, cancelReason`

// Tours lists tours with their ordered stops.
//
// Two queries and a fold rather than one join: a join multiplies each tour by its stops
// and each stop by its tasks, and the assembly in Go is both cheaper to read and cheaper
// to be wrong in. This is fewer than ten cars for one night, not a reporting workload.
func (q *querier) Tours(ctx context.Context, f TourFilter) ([]*Tour, error) {
	query := `SELECT ` + tourColumns + ` FROM dispatch_tour WHERE (year = ? OR ? = '')`
	args := []any{f.YearSlug, f.YearSlug}
	if len(f.States) > 0 {
		query += ` AND state IN (` + placeholders(len(f.States)) + `)`
		for _, s := range f.States {
			args = append(args, string(s))
		}
	}
	if f.SectionSlug != "" {
		query += ` AND sectionSlug = ?`
		args = append(args, string(f.SectionSlug))
	}
	// Departure first, with the un-departed last: a tour with no departure yet is a plan
	// being built, and it belongs below the ones that are going somewhere.
	query += ` ORDER BY departureUts IS NULL ASC, departureUts ASC, createdUts ASC`

	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tours := []*Tour{}
	byID := map[TourID]*Tour{}
	for rows.Next() {
		t, err := scanTour(rows)
		if err != nil {
			return nil, err
		}
		tours = append(tours, t)
		byID[t.ID] = t
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(tours) == 0 {
		return tours, nil
	}
	return tours, q.attachStops(ctx, f.YearSlug, byID)
}

// GetTour returns one tour with its ordered stops.
func (q *querier) GetTour(ctx context.Context, year types.YearSlug, id TourID) (*Tour, error) {
	if id == "" {
		return nil, tables.ErrRecordNotFound
	}
	query := `SELECT ` + tourColumns + ` FROM dispatch_tour WHERE id = ? AND (year = ? OR ? = '')`
	t, err := scanTour(q.db.QueryRowContext(ctx, query, string(id), year, year))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, tables.ErrRecordNotFound
		}
		return nil, err
	}
	return t, q.attachStops(ctx, year, map[TourID]*Tour{t.ID: t})
}

// attachStops fills in the stops of the given tours, in order, each with its tasks.
func (q *querier) attachStops(ctx context.Context, year types.YearSlug, tours map[TourID]*Tour) error {
	ids := make([]any, 0, len(tours))
	for id := range tours {
		ids = append(ids, string(id))
	}
	if len(ids) == 0 {
		return nil
	}
	query := `SELECT id, tourId, sortOrder, placeKind, placeRefId, placeLabel,
		plannedUts, plannedOverride, visitedUts
		FROM dispatch_stop WHERE tourId IN (` + placeholders(len(ids)) + `)
		ORDER BY tourId ASC, sortOrder ASC`
	rows, err := q.db.QueryContext(ctx, query, ids...)
	if err != nil {
		return err
	}
	defer rows.Close()

	stops := map[StopID]stopRef{}
	for rows.Next() {
		var (
			s       TourStop
			tourID  string
			planned sql.NullInt64
			visited sql.NullInt64
		)
		if err := rows.Scan(&s.StopID, &tourID, &s.SortOrder, &s.Place.Kind, &s.Place.RefID,
			&s.Place.Label, &planned, &s.Override, &visited); err != nil {
			return err
		}
		s.PlannedUts = nullInt(planned)
		s.VisitedUts = nullInt(visited)
		s.Tasks = []StopTask{}
		tour, ok := tours[TourID(tourID)]
		if !ok {
			continue
		}
		tour.Stops = append(tour.Stops, s)
		// Tour and index, not a pointer into the slice: appending to tour.Stops may
		// reallocate, and a pointer taken before that would be writing the tasks into a
		// copy nobody reads. An index stays correct.
		stops[s.StopID] = stopRef{tour: tour, idx: len(tour.Stops) - 1}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(stops) == 0 {
		return nil
	}

	stopIDs := make([]any, 0, len(stops))
	for id := range stops {
		stopIDs = append(stopIDs, string(id))
	}
	taskRows, err := q.db.QueryContext(ctx,
		`SELECT stopId, taskId, role FROM dispatch_stop_task
		 WHERE stopId IN (`+placeholders(len(stopIDs))+`) ORDER BY taskId ASC`, stopIDs...)
	if err != nil {
		return err
	}
	defer taskRows.Close()
	for taskRows.Next() {
		var stopID string
		var st StopTask
		if err := taskRows.Scan(&stopID, &st.TaskID, &st.Role); err != nil {
			return err
		}
		if ref, ok := stops[StopID(stopID)]; ok {
			ref.tour.Stops[ref.idx].Tasks = append(ref.tour.Stops[ref.idx].Tasks, st)
		}
	}
	return taskRows.Err()
}

// stopRef locates a stop inside the tours being assembled.
type stopRef struct {
	tour *Tour
	idx  int
}

// StopsByTask returns each task's appearances on tours, in tour order.
func (q *querier) StopsByTask(ctx context.Context, year types.YearSlug, ids []TaskID) (map[TaskID][]TaskStop, error) {
	out := map[TaskID][]TaskStop{}
	if len(ids) == 0 {
		return out, nil
	}
	args := make([]any, 0, len(ids)+2)
	for _, id := range ids {
		args = append(args, string(id))
	}
	args = append(args, year, year)
	query := `SELECT st.taskId, st.tourId, s.id, st.role, s.sortOrder,
		s.placeKind, s.placeRefId, s.placeLabel, s.plannedUts, s.plannedOverride, s.visitedUts
		FROM dispatch_stop_task st JOIN dispatch_stop s ON s.id = st.stopId
		WHERE st.taskId IN (` + placeholders(len(ids)) + `) AND (st.year = ? OR ? = '')
		ORDER BY s.sortOrder ASC`
	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			taskID  string
			ts      TaskStop
			planned sql.NullInt64
			visited sql.NullInt64
		)
		if err := rows.Scan(&taskID, &ts.TourID, &ts.StopID, &ts.Role, &ts.SortOrder,
			&ts.Place.Kind, &ts.Place.RefID, &ts.Place.Label, &planned, &ts.Override, &visited); err != nil {
			return nil, err
		}
		ts.PlannedUts = nullInt(planned)
		ts.VisitedUts = nullInt(visited)
		out[TaskID(taskID)] = append(out[TaskID(taskID)], ts)
	}
	return out, rows.Err()
}

// DispatchableSections lists the organisation sections that are dispatch units for the
// given year.
//
// Empty by default and opted into per section, so a fresh year offers no unit until
// somebody says which subsections hold a car. An empty list is not an error: it means
// nobody has decided yet, and the desk can still write tasks down — which is exactly PRD
// 009's "no unit on duty" edge case.
func (q *querier) DispatchableSections(ctx context.Context, year types.YearSlug) ([]types.Slug, error) {
	query := `SELECT sectionSlug FROM dispatchable_section
		WHERE (year = ? OR ? = '') ORDER BY sectionSlug ASC`
	rows, err := q.db.QueryContext(ctx, query, year, year)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	slugs := []types.Slug{}
	for rows.Next() {
		var s types.Slug
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		slugs = append(slugs, s)
	}
	return slugs, rows.Err()
}

func (q *querier) DutyWindows(ctx context.Context, year types.YearSlug) ([]Duty, error) {
	rows, err := q.db.QueryContext(ctx,
		`SELECT id, year, sectionSlug, startUts, endUts FROM dispatch_duty
		 WHERE (year = ? OR ? = '') ORDER BY startUts ASC, sectionSlug ASC`, year, year)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	windows := []Duty{}
	for rows.Next() {
		var d Duty
		if err := rows.Scan(&d.ID, &d.YearSlug, &d.SectionSlug, &d.StartUts, &d.EndUts); err != nil {
			return nil, err
		}
		windows = append(windows, d)
	}
	return windows, rows.Err()
}

func (q *querier) taskTimeline(ctx context.Context, id TaskID) ([]Activity, error) {
	rows, err := q.db.QueryContext(ctx,
		`SELECT seq, taskId, tourId, type, actorUserId, value, createdUts
		 FROM dispatch_activity WHERE taskId = ? ORDER BY seq ASC`, string(id))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	timeline := []Activity{}
	for rows.Next() {
		var a Activity
		var value sql.NullString
		if err := rows.Scan(&a.Seq, &a.TaskID, &a.TourID, &a.Type, &a.Actor, &value, &a.AtUts); err != nil {
			return nil, err
		}
		a.Value = value.String
		timeline = append(timeline, a)
	}
	return timeline, rows.Err()
}

// --- scanning ---

type scanner interface {
	Scan(dest ...any) error
}

func scanTask(row scanner) (*Task, error) {
	var (
		t         Task
		notBefore sql.NullInt64
		deadline  sql.NullInt64
		pickedUp  sql.NullInt64
		done      sql.NullInt64
		cancelled sql.NullInt64
		desc      sql.NullString
		memberIDs sql.NullString
	)
	if err := row.Scan(&t.ID, &t.YearSlug, &t.Kind, &t.Priority, &desc, &t.SpaceNeeds,
		&t.Pickup.Kind, &t.Pickup.RefID, &t.Pickup.Label,
		&t.Dropoff.Kind, &t.Dropoff.RefID, &t.Dropoff.Label,
		&t.State, &t.CreatedUts, &notBefore, &deadline, &pickedUp, &done, &cancelled,
		&t.CancelReason, &t.SosID, &t.TeamID, &memberIDs, &t.CreatedBy); err != nil {
		return nil, err
	}
	t.Description = desc.String
	t.NotBeforeUts = nullInt(notBefore)
	t.DeadlineUts = nullInt(deadline)
	t.PickedUpUts = nullInt(pickedUp)
	t.DoneUts = nullInt(done)
	t.CancelledUts = nullInt(cancelled)
	// Always a slice, never null, and a malformed value is an empty list rather than an
	// error: a task whose member list cannot be parsed is still a task the desk must see.
	t.MemberIDs = []types.MemberID{}
	if memberIDs.String != "" {
		_ = json.Unmarshal([]byte(memberIDs.String), &t.MemberIDs)
		if t.MemberIDs == nil {
			t.MemberIDs = []types.MemberID{}
		}
	}
	return &t, nil
}

func scanTour(row scanner) (*Tour, error) {
	var (
		t         Tour
		departure sql.NullInt64
		underway  sql.NullInt64
		completed sql.NullInt64
		cancelled sql.NullInt64
		notes     sql.NullString
	)
	if err := row.Scan(&t.ID, &t.YearSlug, &t.SectionSlug, &departure, &notes, &t.State,
		&t.CreatedUts, &underway, &completed, &cancelled, &t.CancelReason); err != nil {
		return nil, err
	}
	t.Notes = notes.String
	t.DepartureUts = nullInt(departure)
	t.UnderwayUts = nullInt(underway)
	t.CompletedUts = nullInt(completed)
	t.CancelledUts = nullInt(cancelled)
	t.Stops = []TourStop{}
	return &t, nil
}

func nullInt(v sql.NullInt64) *int64 {
	if !v.Valid {
		return nil
	}
	n := v.Int64
	return &n
}

func placeholders(n int) string {
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}
