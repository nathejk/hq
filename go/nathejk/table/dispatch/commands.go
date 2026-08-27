package dispatch

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jrgensen/stream"
	"github.com/jrgensen/stream/subject"
	"github.com/nathejk/shared-go/messages"
	"github.com/nathejk/shared-go/tables"
	"github.com/nathejk/shared-go/types"
)

// Commands is the write surface, as the application sees it.
//
// Every method takes an Actor rather than reading the acting user out of the request
// context: the handler is already the layer that knows about HTTP, and this package must
// not import nathejk.dk/internal/requestctx.
//
// The state transitions are separate methods rather than fields on a patch, and the
// endpoints in front of them are separate routes rather than one PATCH. That is deliberate
// (PRD 009 §8): the driver's screen will be another app in another repo, and
// "stop reached" / "people aboard" are exactly what it will call. Folding them into a
// general patch would mean building them twice.
type Commands interface {
	SetSectionDispatchable(ctx context.Context, actor Actor, year types.YearSlug, slug types.Slug, dispatchable bool) error

	// TourCommands is embedded rather than listed here: the tour transitions are a coherent
	// surface of their own (tours.go), and the driver's app will use that half and almost none
	// of this one.
	TourCommands

	CreateTask(ctx context.Context, actor Actor, year types.YearSlug, cmd CreateTaskCommand) (TaskID, error)
	PatchTask(ctx context.Context, actor Actor, year types.YearSlug, id TaskID, cmd PatchTaskCommand) error

	// MarkPickedUp records people aboard. The member transitions to `transit` are *not*
	// published here: they belong to spejderstatus (task 118), and this package may not
	// import it. The handler publishes both, in that order.
	MarkPickedUp(ctx context.Context, actor Actor, year types.YearSlug, id TaskID, unit types.Slug, atUts int64) error

	CancelTask(ctx context.Context, actor Actor, year types.YearSlug, id TaskID, reason string) error

	// Duty windows (task 115). SetDuty both creates and edits: an id given is an edit, an empty
	// one is a new window, because the editor's two gestures are the same gesture to the roster.
	SetDuty(ctx context.Context, actor Actor, year types.YearSlug, id DutyID, unit types.Slug, startUts, endUts int64) (DutyID, error)
	RemoveDuty(ctx context.Context, actor Actor, year types.YearSlug, id DutyID) error
}

// CreateTaskCommand opens a task.
//
// Almost everything is optional, and that is a feature rather than laxity: PRD 009 §8 names
// desk discipline as the biggest risk to the whole board, and the mitigation is that writing
// a job down must be faster than not writing it down. A kind and a description are the
// minimum, because a task with neither is a row nobody can interpret at handover.
type CreateTaskCommand struct {
	Kind        Kind
	Priority    Priority
	Description string
	SpaceNeeds  string
	Pickup      Place
	Dropoff     Place

	// CreatedUts is the waiting clock. Zero means "now", which is the ordinary case; an
	// operator may backdate it, because a patrol that rang twenty minutes ago has been
	// waiting twenty minutes.
	CreatedUts   int64
	NotBeforeUts *int64
	DeadlineUts  *int64

	SosID     types.SosID
	TeamID    types.TeamID
	MemberIDs []types.MemberID
}

// PatchTaskCommand is a partial update: a nil field is absent, a non-nil field is a new
// value — including an intentionally empty one.
//
// Pointers rather than a map or "changed" flags, matching updateYearHandler and
// patchKlanHandler, because the distinction between "not mentioned" and "cleared" is the
// entire point of PATCH. Two pointers deep for the times (**int64) for the same reason one
// level down: `"deadlineUts": null` clears the deadline and an absent field leaves it alone,
// and those must not be the same request.
type PatchTaskCommand struct {
	Kind        *Kind
	Priority    *Priority
	Description *string
	SpaceNeeds  *string
	Pickup      *Place
	Dropoff     *Place

	NotBeforeUts **int64
	DeadlineUts  **int64
}

// Errors the handler maps onto 422s. Danish phrasing lives in the handler; these say what
// the domain refuses.
var (
	// ErrEmptyDescription: a task nobody can interpret at handover is worse than a refused
	// form. The one required field, alongside a kind.
	ErrEmptyDescription = errors.New("description is required")
	ErrInvalidKind      = errors.New("unknown task kind")
	ErrInvalidPriority  = errors.New("unknown priority")
	// ErrReasonRequired: a cancelled task with no explanation is the one thing a handover
	// cannot recover from.
	ErrReasonRequired = errors.New("a reason is required to cancel")
	// ErrNotPickup: only people are picked up. Enforced because `pickedup` is the custody
	// transition, and a delivery of dinner claiming custody of nobody would put a
	// meaningless entry in the one log the shelter trusts.
	ErrNotPickup = errors.New("only a pickup task can record people aboard")
	// ErrTaskFinished: a done or cancelled task is history. Editing it would rewrite what
	// the desk promised, and the timeline is the record.
	ErrTaskFinished = errors.New("the task is already finished")
)

// MaxDescriptionLength is generous on purpose: this is prose typed at 3am, and a truncated
// instruction is worse than a long one. The limit exists so a paste accident cannot put a
// megabyte on the event stream.
const MaxDescriptionLength = 2000

type commander struct {
	p stream.Publisher
	q Queries
}

// CreateTask opens a task. The server mints the id, so a client cannot choose one.
func (c commander) CreateTask(ctx context.Context, actor Actor, year types.YearSlug, cmd CreateTaskCommand) (TaskID, error) {
	if !cmd.Kind.Valid() {
		return "", ErrInvalidKind
	}
	if cmd.Priority != "" && !cmd.Priority.Valid() {
		return "", ErrInvalidPriority
	}
	description := strings.TrimSpace(cmd.Description)
	if description == "" {
		return "", ErrEmptyDescription
	}
	if len(description) > MaxDescriptionLength {
		return "", fmt.Errorf("description longer than %d characters", MaxDescriptionLength)
	}
	createdUts := cmd.CreatedUts
	if createdUts == 0 {
		createdUts = nowUts()
	}
	id := NewTaskID()
	body := &TaskCreated{
		TaskID:       id,
		Kind:         cmd.Kind,
		Priority:     cmd.Priority,
		Description:  description,
		SpaceNeeds:   strings.TrimSpace(cmd.SpaceNeeds),
		Pickup:       cmd.Pickup,
		Dropoff:      cmd.Dropoff,
		CreatedUts:   createdUts,
		NotBeforeUts: cmd.NotBeforeUts,
		DeadlineUts:  cmd.DeadlineUts,
		SosID:        cmd.SosID,
		TeamID:       cmd.TeamID,
		MemberIDs:    cmd.MemberIDs,
	}
	if err := c.publishTask(actor, year, id, "created", body); err != nil {
		return "", err
	}
	return id, nil
}

// PatchTask edits a task's fields.
//
// Dirty-checked field by field against the current row, so a form resubmitted unchanged
// publishes nothing. Note the consequence the repo learned on the klan status override: a
// command that publishes nothing emits no live signal, so a UI relying on the signal alone
// to confirm a save must also refresh.
func (c commander) PatchTask(ctx context.Context, actor Actor, year types.YearSlug, id TaskID, cmd PatchTaskCommand) error {
	current, err := c.q.GetTask(ctx, year, id)
	if err != nil {
		return err
	}
	if current == nil {
		return tables.ErrRecordNotFound
	}
	if current.State == TaskStateDone || current.State == TaskStateCancelled {
		return ErrTaskFinished
	}

	next := TaskUpdated{
		TaskID:       id,
		Kind:         current.Kind,
		Priority:     current.Priority,
		Description:  current.Description,
		SpaceNeeds:   current.SpaceNeeds,
		Pickup:       current.Pickup,
		Dropoff:      current.Dropoff,
		NotBeforeUts: current.NotBeforeUts,
		DeadlineUts:  current.DeadlineUts,
	}
	changed := []string{}
	if cmd.Kind != nil && *cmd.Kind != current.Kind {
		if !cmd.Kind.Valid() {
			return ErrInvalidKind
		}
		next.Kind, changed = *cmd.Kind, append(changed, "kind")
	}
	if cmd.Priority != nil && *cmd.Priority != current.Priority {
		if *cmd.Priority != "" && !cmd.Priority.Valid() {
			return ErrInvalidPriority
		}
		next.Priority, changed = *cmd.Priority, append(changed, "priority")
	}
	if cmd.Description != nil {
		description := strings.TrimSpace(*cmd.Description)
		if description == "" {
			return ErrEmptyDescription
		}
		if len(description) > MaxDescriptionLength {
			return fmt.Errorf("description longer than %d characters", MaxDescriptionLength)
		}
		if description != current.Description {
			next.Description, changed = description, append(changed, "description")
		}
	}
	if cmd.SpaceNeeds != nil {
		if space := strings.TrimSpace(*cmd.SpaceNeeds); space != current.SpaceNeeds {
			next.SpaceNeeds, changed = space, append(changed, "spaceNeeds")
		}
	}
	if cmd.Pickup != nil && *cmd.Pickup != current.Pickup {
		next.Pickup, changed = *cmd.Pickup, append(changed, "pickup")
	}
	if cmd.Dropoff != nil && *cmd.Dropoff != current.Dropoff {
		next.Dropoff, changed = *cmd.Dropoff, append(changed, "dropoff")
	}
	if cmd.NotBeforeUts != nil && !sameUts(*cmd.NotBeforeUts, current.NotBeforeUts) {
		next.NotBeforeUts, changed = *cmd.NotBeforeUts, append(changed, "notBeforeUts")
	}
	if cmd.DeadlineUts != nil && !sameUts(*cmd.DeadlineUts, current.DeadlineUts) {
		next.DeadlineUts, changed = *cmd.DeadlineUts, append(changed, "deadlineUts")
	}
	if len(changed) == 0 {
		return nil
	}
	next.Changed = changed
	return c.publishTask(actor, year, id, "updated", &next)
}

// MarkPickedUp records that the people a pickup is for are in the car.
//
// Idempotent: a dispatcher who presses Hentet twice — plausible on a phone, at night, with a
// driver still talking — must not put two custody changes on the log.
func (c commander) MarkPickedUp(ctx context.Context, actor Actor, year types.YearSlug, id TaskID, unit types.Slug, atUts int64) error {
	current, err := c.q.GetTask(ctx, year, id)
	if err != nil {
		return err
	}
	if current == nil {
		return tables.ErrRecordNotFound
	}
	if current.Kind != KindPickup {
		return ErrNotPickup
	}
	if current.State == TaskStateCancelled || current.State == TaskStateDone {
		return ErrTaskFinished
	}
	if current.PickedUpUts != nil {
		return nil
	}
	if atUts == 0 {
		atUts = nowUts()
	}
	return c.publishTask(actor, year, id, "pickedup", &TaskPickedUp{
		TaskID:      id,
		SectionSlug: unit,
		MemberIDs:   current.MemberIDs,
		AtUts:       atUts,
	})
}

// CancelTask withdraws a task, with a reason.
func (c commander) CancelTask(ctx context.Context, actor Actor, year types.YearSlug, id TaskID, reason string) error {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return ErrReasonRequired
	}
	current, err := c.q.GetTask(ctx, year, id)
	if err != nil {
		return err
	}
	if current == nil {
		return tables.ErrRecordNotFound
	}
	// Already cancelled is success, not a conflict: two operators pressing the same button
	// is a race the desk should not have to think about. Already *done* is a refusal, because
	// cancelling a completed job means somebody has misread the board.
	if current.State == TaskStateCancelled {
		return nil
	}
	if current.State == TaskStateDone {
		return ErrTaskFinished
	}
	return c.publishTask(actor, year, id, "cancelled", &TaskCancelled{
		TaskID: id, Reason: reason, AtUts: nowUts(),
	})
}

// SetSectionDispatchable marks an organisation section as being (or no longer being) a
// dispatch unit.
//
// Dirty-checked against the current list, so a toggle that changes nothing publishes
// nothing — the Organisation page can send the state it wants without first working out
// whether that is already the case.
func (c commander) SetSectionDispatchable(ctx context.Context, actor Actor, year types.YearSlug, slug types.Slug, dispatchable bool) error {
	if !slug.Valid() {
		return fmt.Errorf("invalid section slug %q", slug)
	}
	current, err := c.q.DispatchableSections(ctx, year)
	if err != nil {
		return err
	}
	already := false
	for _, s := range current {
		if s == slug {
			already = true
			break
		}
	}
	if already == dispatchable {
		return nil
	}

	msg := c.p.MessageFunc()(subject.FromStr(fmt.Sprintf("NATHEJK.%s.dispatch.section.%s.dispatchable", year, slug)))
	if err := msg.SetBody(&SectionDispatchableSet{SectionSlug: slug, Dispatchable: dispatchable}); err != nil {
		return err
	}
	if err := msg.SetMeta(&messages.Metadata{UserID: actor.UserID}); err != nil {
		return err
	}
	return c.p.Publish(msg)
}

// ErrBadWindow is returned for a window that ends before it begins, or has no length.
//
// Overlapping windows are *not* refused: a unit rostered twice over the same hour is untidy and
// harmless — they are on duty either way — and refusing it would block an operator fixing a
// roster in the order that makes sense to them.
var ErrBadWindow = errors.New("the duty window must end after it starts")

// SetDuty records or edits a unit's duty window.
func (c commander) SetDuty(ctx context.Context, actor Actor, year types.YearSlug, id DutyID, unit types.Slug, startUts, endUts int64) (DutyID, error) {
	if unit == "" || !unit.Valid() {
		return "", ErrUnitRequired
	}
	if endUts <= startUts {
		return "", ErrBadWindow
	}
	if id == "" {
		id = NewDutyID()
	}
	body := &DutySet{DutyID: id, SectionSlug: unit, StartUts: startUts, EndUts: endUts}
	if err := c.publish(actor, fmt.Sprintf("NATHEJK.%s.dispatchduty.%s.set", year, id), body); err != nil {
		return "", err
	}
	return id, nil
}

// RemoveDuty deletes a window.
//
// Not dirty-checked against the current roster, unlike most commands here: removing a window that
// is already gone is a no-op the desk cannot distinguish from success, and reading the roster back
// first would buy nothing but a round trip.
func (c commander) RemoveDuty(ctx context.Context, actor Actor, year types.YearSlug, id DutyID) error {
	if id == "" {
		return tables.ErrRecordNotFound
	}
	return c.publish(actor, fmt.Sprintf("NATHEJK.%s.dispatchduty.%s.removed", year, id), &DutyRemoved{DutyID: id})
}

func (c commander) publishTask(actor Actor, year types.YearSlug, id TaskID, event string, body any) error {
	return c.publish(actor, fmt.Sprintf("NATHEJK.%s.dispatch.%s.%s", year, id, event), body)
}

func (c commander) publish(actor Actor, subj string, body any) error {
	msg := c.p.MessageFunc()(subject.FromStr(subj))
	if err := msg.SetBody(body); err != nil {
		return err
	}
	if err := msg.SetMeta(&messages.Metadata{UserID: actor.UserID}); err != nil {
		return err
	}
	return c.p.Publish(msg)
}

// sameUts compares two optional times by value, not by pointer.
//
// Its own function because `*a == *b` on two nil-able pointers is a panic waiting for a
// task with no deadline, and `a == b` compares addresses and is always false for two
// separately decoded JSON values — which would make every PATCH publish an event.
func sameUts(a, b *int64) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}
