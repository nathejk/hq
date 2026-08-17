package spejderstatus

import (
	"context"
	"errors"
	"fmt"

	"github.com/jrgensen/stream"
	"github.com/jrgensen/stream/subject"
	"github.com/nathejk/shared-go/messages"
	"github.com/nathejk/shared-go/types"
)

// Commands is the write surface for the member lifecycle, as the application sees
// it.
//
// Every method takes an Actor rather than reading the acting user out of the
// request context, for the same reason the sos package does: reading the context
// here would mean importing nathejk.dk/internal/requestctx and turning the
// eventual lift into a rewrite (task 083).
//
// # What these do not do
//
// They publish the **member** events only. The summarising case event that puts the
// operation on a timeline is the sos package's, and this package may not import it
// — so the *handler* composes the two, which is the layer that is allowed to know
// about both. Each method therefore returns what it changed, in enough detail for
// the caller to build that summary without re-reading anything.
//
// They also do not validate the destination of a move beyond "not where the member
// already is". Whether a target patrol exists, started, and is still racing is a
// question about the patrulje entity, which this package equally may not import;
// the handler checks it before calling. Documented rather than silently assumed,
// because it is the one place a caller could pass something meaningless.
//
// # What they refuse
//
// The self-carrying boundary, and only that. A member is on their own legs up to
// and including waiting, so those are the transitions this interface owns; from the
// car door onwards the events belong to the car and shelter interfaces. The
// requirement that a patrol keep three racing members is deliberately **not**
// enforced here — see PRD 006 §8: the member is leaving regardless, and refusing to
// record it would only make the data wrong.
type Commands interface {
	RequestWithdrawal(ctx context.Context, actor Actor, year types.YearSlug, id types.MemberID) (*Change, error)
	CancelWithdrawal(ctx context.Context, actor Actor, year types.YearSlug, id types.MemberID) (*Change, error)
	OverrideStatus(ctx context.Context, actor Actor, year types.YearSlug, id types.MemberID, to types.MemberStatus) (*Change, error)
	MoveTeam(ctx context.Context, actor Actor, year types.YearSlug, id types.MemberID, to types.TeamID) (*Move, error)
	CollectTeam(ctx context.Context, actor Actor, year types.YearSlug, team types.TeamID) ([]Change, error)
}

var (
	// ErrAlreadyCollected is returned when an operator tries to put a member back
	// into the race after a car has accepted them.
	//
	// Its own error rather than a generic conflict because the operator can act on
	// it: it means "allerede hentet", and the answer is to tell the caller the
	// scout is on their way to HQ, not to retry. The race it guards is real and
	// narrow — the operator presses resume in the same moment the driver accepts
	// the member aboard — and the acceptance wins, because it reflects a member
	// physically sitting in a car.
	ErrAlreadyCollected = errors.New("member has already been collected")

	// ErrNotWaiting is returned when a resume is attempted on a member who never
	// asked to leave.
	ErrNotWaiting = errors.New("member is not waiting")

	// ErrNotSelfCarrying is returned when a withdrawal is requested for a member
	// who is no longer on their own legs. Requesting one is meaningless once a car
	// has them: they have already left the route.
	ErrNotSelfCarrying = errors.New("member is no longer self-carrying")

	// ErrCannotFinish is returned when a correction tries to mark a member
	// finished. Only walking the route unaided earns that, so no override may
	// confer it — see types.MemberStatus.CanFinish.
	ErrCannotFinish = errors.New("finished cannot be set by hand")

	// ErrInvalidStatus is returned for a status this version of the code does not
	// know.
	ErrInvalidStatus = errors.New("unknown member status")

	// ErrSameTeam is returned when a member is moved to the team they are already
	// on. Not an error the operator caused so much as one the UI should have
	// prevented, but publishing it would put a meaningless line on a timeline.
	ErrSameTeam = errors.New("member is already on that team")
)

// Change is what a status-changing command did, for the caller to summarise.
//
// From is included because the timeline entry has to say "fra racing til waiting"
// and keep saying it: the summary is stored, not re-derived, so the previous status
// has to be captured at the moment it stopped being true.
//
// TeamStrength is the team's racing count *after* this change, computed here rather
// than read back from patrulje.activeMemberCount. That is not an optimisation: the
// projection has not seen the event yet at this point, so the column still holds
// the old number. Computing it from the members this command can see is the only
// way to record the resulting strength without waiting for a projection to catch
// up.
type Change struct {
	MemberID     types.MemberID
	TeamID       types.TeamID
	From         types.MemberStatus
	To           types.MemberStatus
	TeamStrength int
}

// Move is what a team move did. Two strengths, because a move changes both teams.
type Move struct {
	MemberID         types.MemberID
	FromTeamID       types.TeamID
	ToTeamID         types.TeamID
	FromTeamStrength int
	ToTeamStrength   int
}

type commander struct {
	p stream.Publisher
	q Queries
}

// RequestWithdrawal records that a member wants to leave the race and is waiting to
// be collected.
//
// Idempotent: a member who is already waiting produces no event, so a double click
// or a retried request does not put two lines on a timeline. That is the house
// pattern — a no-op write publishes nothing and therefore emits no live signal
// either.
func (c commander) RequestWithdrawal(ctx context.Context, actor Actor, year types.YearSlug, id types.MemberID) (*Change, error) {
	current, err := c.q.GetByMemberID(ctx, year, id)
	if err != nil {
		return nil, err
	}
	if current.Status == types.MemberStatusWaiting {
		return nil, nil
	}
	// Only from racing. Anything past waiting means a car already has them, and a
	// member in a car has not "asked to leave" — they have left.
	if current.Status != types.MemberStatusRacing {
		return nil, ErrNotSelfCarrying
	}
	change, err := c.change(ctx, year, current, types.MemberStatusWaiting)
	if err != nil {
		return nil, err
	}
	body := &WithdrawalRequested{MemberID: id, TeamID: current.CurrentTeamID, Actor: actor}
	if err := c.publish(actor, year, id, "withdrawal.requested", body); err != nil {
		return nil, err
	}
	return change, nil
}

// CancelWithdrawal puts a member who changed their mind back into the race.
//
// Valid only while they are still self-carrying, and enforced here rather than only
// in the UI: the operator's screen may be a moment stale, and that moment is exactly
// when a car is accepting the member.
func (c commander) CancelWithdrawal(ctx context.Context, actor Actor, year types.YearSlug, id types.MemberID) (*Change, error) {
	current, err := c.q.GetByMemberID(ctx, year, id)
	if err != nil {
		return nil, err
	}
	if current.Status == types.MemberStatusRacing {
		return nil, nil
	}
	if current.Status != types.MemberStatusWaiting {
		// Distinguish the two so the operator gets an answer they can use. "Already
		// collected" is a fact about where the scout is; "not waiting" means the
		// screen was wrong about them.
		if current.Status.InOurCare() {
			return nil, ErrAlreadyCollected
		}
		return nil, ErrNotWaiting
	}
	change, err := c.change(ctx, year, current, types.MemberStatusRacing)
	if err != nil {
		return nil, err
	}
	body := &WithdrawalCancelled{MemberID: id, TeamID: current.CurrentTeamID, Actor: actor}
	if err := c.publish(actor, year, id, "withdrawal.cancelled", body); err != nil {
		return nil, err
	}
	return change, nil
}

// OverrideStatus corrects a member's status by hand.
//
// **Deliberately lenient about ordering** (decided 2026-08-17): any valid status is
// accepted from any other, without enforcing the documented sequence. This is the
// out-of-sync repair tool — its whole purpose is recording a reality that did not
// follow the diagram, and a version that rejected racing → sheltered because no
// pickup was logged would refuse precisely the correction it exists to make. The
// timeline shows what happened, which is the honest record.
//
// The one exception is `finished`, which no correction may confer.
func (c commander) OverrideStatus(ctx context.Context, actor Actor, year types.YearSlug, id types.MemberID, to types.MemberStatus) (*Change, error) {
	if to == types.MemberStatusFinished {
		return nil, ErrCannotFinish
	}
	if !to.Valid() {
		return nil, ErrInvalidStatus
	}
	current, err := c.q.GetByMemberID(ctx, year, id)
	if err != nil {
		return nil, err
	}
	if current.Status == to {
		return nil, nil
	}
	change, err := c.change(ctx, year, current, to)
	if err != nil {
		return nil, err
	}
	body := &StatusOverridden{MemberID: id, TeamID: current.CurrentTeamID, To: to, Actor: actor}
	if err := c.publish(actor, year, id, "status.overridden", body); err != nil {
		return nil, err
	}
	return change, nil
}

// MoveTeam moves a member to another patrol.
//
// The member keeps racing and keeps their initial team; only currentTeamId changes.
// Whether the destination is a real, started, still-racing patrol is the caller's to
// check — see the note on Commands.
func (c commander) MoveTeam(ctx context.Context, actor Actor, year types.YearSlug, id types.MemberID, to types.TeamID) (*Move, error) {
	current, err := c.q.GetByMemberID(ctx, year, id)
	if err != nil {
		return nil, err
	}
	if current.CurrentTeamID == to {
		return nil, ErrSameTeam
	}

	// Both teams' resulting strengths, computed before the event is published for
	// the same reason as Change.TeamStrength: the projection has not run yet.
	from, err := c.strengthAfter(ctx, year, current.CurrentTeamID, id, types.MemberStatusNone)
	if err != nil {
		return nil, err
	}
	dest, err := c.strengthAfter(ctx, year, to, id, types.MemberStatusRacing)
	if err != nil {
		return nil, err
	}

	body := &TeamMoved{MemberID: id, FromTeamID: current.CurrentTeamID, ToTeamID: to, Actor: actor}
	if err := c.publish(actor, year, id, "team.moved", body); err != nil {
		return nil, err
	}
	return &Move{
		MemberID:         id,
		FromTeamID:       current.CurrentTeamID,
		ToTeamID:         to,
		FromTeamStrength: from,
		ToTeamStrength:   dest,
	}, nil
}

// CollectTeam takes every remaining racing member of a patrol out of the race in one
// operation: the team leaves together.
//
// # One command, not N calls from the browser
//
// Three separate requests could half-succeed, and a team split across two states with
// nobody noticing is the worst available outcome — worse than the operator having to
// click three times, which is itself a way to forget two of them while on the phone.
// So the loop is here, on the server, where a partial failure is returned as a failure.
//
// # Members already out are skipped, not re-requested
//
// Collecting a patrol where one member is already in a car must not touch that member:
// they have left the route, and re-publishing a withdrawal request for them would put a
// second, false line in their history and move them backwards through the lifecycle.
//
// Returns one Change per member actually collected, in team order, for the caller to
// summarise as a single timeline entry. An empty result means there was nobody left to
// collect — not an error, just a no-op, which is what a double click produces.
func (c commander) CollectTeam(ctx context.Context, actor Actor, year types.YearSlug, team types.TeamID) ([]Change, error) {
	if team == "" {
		return nil, ErrRecordNotFound
	}
	members, err := c.q.GetByTeam(ctx, Filter{YearSlug: year, TeamID: team})
	if err != nil {
		return nil, err
	}

	var changes []Change
	for _, m := range members {
		if m.Status != types.MemberStatusRacing {
			continue
		}
		body := &WithdrawalRequested{MemberID: m.MemberID, TeamID: team, Actor: actor}
		if err := c.publish(actor, year, m.MemberID, "withdrawal.requested", body); err != nil {
			// Returned rather than swallowed: the caller must be able to tell the
			// operator that the collection did not complete, since some members are now
			// waiting and others are not. Whatever was published stays published — the
			// log is the record — and the changes so far are discarded so no summary
			// claims an operation that did not finish.
			return nil, err
		}
		changes = append(changes, Change{
			MemberID: m.MemberID,
			TeamID:   team,
			From:     types.MemberStatusRacing,
			To:       types.MemberStatusWaiting,
			// Zero by definition: every racing member of this team is being taken out
			// of the race, so none is left on the route. This is also what makes the
			// patrol discontinued, with no event of its own.
			TeamStrength: 0,
		})
	}
	return changes, nil
}

// change assembles the Change for a status transition, including the team's
// resulting strength.
func (c commander) change(ctx context.Context, year types.YearSlug, current *SpejderStatus, to types.MemberStatus) (*Change, error) {
	strength, err := c.strengthAfter(ctx, year, current.CurrentTeamID, current.MemberID, to)
	if err != nil {
		return nil, err
	}
	return &Change{
		MemberID:     current.MemberID,
		TeamID:       current.CurrentTeamID,
		From:         current.Status,
		To:           to,
		TeamStrength: strength,
	}, nil
}

// strengthAfter counts a team's racing members as they will be once the pending
// change is applied.
//
// It reads the team and substitutes the one member's new status in memory rather
// than trusting patrulje.activeMemberCount, which still holds the pre-event value
// at this point. MemberStatusNone is how "this member is leaving the team
// altogether" is expressed, since it is not racing and not a real status either.
//
// Counting in Go rather than in SQL keeps the arithmetic next to the reason for it
// and avoids a query that would have to express "…as if this row said something
// else".
func (c commander) strengthAfter(ctx context.Context, year types.YearSlug, teamID types.TeamID, member types.MemberID, to types.MemberStatus) (int, error) {
	if teamID == "" {
		return 0, nil
	}
	members, err := c.q.GetByTeam(ctx, Filter{YearSlug: year, TeamID: teamID})
	if err != nil {
		return 0, err
	}
	strength := 0
	seen := false
	for _, m := range members {
		status := m.Status
		if m.MemberID == member {
			seen = true
			status = to
		}
		if status == types.MemberStatusRacing {
			strength++
		}
	}
	// A member being moved *into* this team is not in it yet, so they were not in
	// the rows above.
	if !seen && to == types.MemberStatusRacing {
		strength++
	}
	return strength, nil
}

func (c commander) publish(actor Actor, year types.YearSlug, id types.MemberID, event string, body any) error {
	msg := c.p.MessageFunc()(subject.FromStr(fmt.Sprintf("NATHEJK.%s.spejder.%s.%s", year, id, event)))
	if err := msg.SetBody(body); err != nil {
		return err
	}
	if err := msg.SetMeta(&messages.Metadata{UserID: actor.UserID}); err != nil {
		return err
	}
	return c.p.Publish(msg)
}
