package spejderstatus

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jrgensen/stream"
	"github.com/nathejk/shared-go/types"
)

// --- fakes ---

// recordingPublisher captures the subjects and bodies a command publishes, which is
// the whole observable behaviour of a write side that owns no state.
type recordingPublisher struct {
	subjects []string
	bodies   []any
}

func (p *recordingPublisher) MessageFunc() stream.MessageFunc {
	return func(s stream.Subject) stream.MutableMessage {
		return &recordedMessage{p: p, subject: s}
	}
}

func (p *recordingPublisher) Publish(msg stream.Message) error {
	m := msg.(*recordedMessage)
	p.subjects = append(p.subjects, m.subject.Subject())
	p.bodies = append(p.bodies, m.body)
	return nil
}

type recordedMessage struct {
	p       *recordingPublisher
	subject stream.Subject
	body    any
}

func (m *recordedMessage) Subject() stream.Subject     { return m.subject }
func (m *recordedMessage) SetBody(b any) error         { m.body = b; return nil }
func (m *recordedMessage) SetMeta(any) error           { return nil }
func (m *recordedMessage) Body(any) error              { return nil }
func (m *recordedMessage) Meta(any) error              { return nil }
func (m *recordedMessage) RawBody() any                { return m.body }
func (m *recordedMessage) RawMeta() any                { return nil }
func (m *recordedMessage) Time() time.Time             { return time.Time{} }
func (m *recordedMessage) Sequence() uint64            { return 0 }
func (m *recordedMessage) SetSubject(s stream.Subject) { m.subject = s }
func (m *recordedMessage) SetTime(time.Time) error     { return nil }

// stubQueries serves one member and their team, which is all any command reads.
//
// GetByMemberID resolves the **requested** id against the team rows when it can, falling
// back to the single `member` field. That matters for the bulk commands: a stub that
// returned the same member for every id would make a per-member loop look like it was
// handling one member three times, and the strength assertions would pass for the wrong
// reason.
type stubQueries struct {
	member *SpejderStatus
	team   []SpejderStatus
	err    error
}

func (q stubQueries) GetByMemberID(_ context.Context, _ types.YearSlug, id types.MemberID) (*SpejderStatus, error) {
	if q.err != nil {
		return nil, q.err
	}
	for i := range q.team {
		if q.team[i].MemberID == id {
			found := q.team[i]
			return &found, nil
		}
	}
	return q.member, nil
}

func (q stubQueries) GetByTeam(context.Context, Filter) ([]SpejderStatus, error) {
	return q.team, nil
}

func (q stubQueries) InOurCare(context.Context, types.YearSlug) (*Care, error) {
	return &Care{}, nil
}

func (q stubQueries) GetHistory(context.Context, types.YearSlug, types.MemberID) ([]StatusEvent, error) {
	return nil, nil
}

func newCommander(member *SpejderStatus, team []SpejderStatus) (commander, *recordingPublisher) {
	p := &recordingPublisher{}
	return commander{p: p, q: stubQueries{member: member, team: team}}, p
}

func racing(id types.MemberID, team types.TeamID) SpejderStatus {
	return SpejderStatus{MemberID: id, CurrentTeamID: team, Status: types.MemberStatusRacing}
}

// --- the self-carrying boundary ---

// The rule the whole design rests on: an operator owns the transitions where the
// member is still on their own legs, and nothing beyond.
func TestWithdrawalOnlyFromRacing(t *testing.T) {
	for _, status := range []types.MemberStatus{
		types.MemberStatusTransit,
		types.MemberStatusSheltered,
		types.MemberStatusReleased,
		types.MemberStatusReunited,
	} {
		c, p := newCommander(&SpejderStatus{MemberID: "m-1", CurrentTeamID: "t-1", Status: status}, nil)
		_, err := c.RequestWithdrawal(context.Background(), Actor{}, "2026", "m-1")
		if !errors.Is(err, ErrNotSelfCarrying) {
			t.Errorf("from %q: err = %v, want ErrNotSelfCarrying", status, err)
		}
		if len(p.subjects) != 0 {
			t.Errorf("from %q: published %v despite refusing", status, p.subjects)
		}
	}
}

// **The race the PRD singles out.** The operator presses resume at the same moment a
// driver accepts the member aboard; the acceptance wins, because it reflects a member
// physically sitting in a car. Enforced here rather than only by hiding a button,
// since the operator's screen is exactly one moment stale when it matters.
func TestResumeIsRefusedOnceCollected(t *testing.T) {
	c, p := newCommander(&SpejderStatus{MemberID: "m-1", CurrentTeamID: "t-1", Status: types.MemberStatusTransit}, nil)

	_, err := c.CancelWithdrawal(context.Background(), Actor{}, "2026", "m-1")
	if !errors.Is(err, ErrAlreadyCollected) {
		t.Fatalf("err = %v, want ErrAlreadyCollected", err)
	}
	if len(p.subjects) != 0 {
		t.Errorf("published %v after refusing the resume", p.subjects)
	}
}

// A resume for somebody who never asked to leave is a different mistake from a resume
// for somebody already in a car, and the operator can act on the difference.
func TestResumeDistinguishesNotWaitingFromCollected(t *testing.T) {
	c, _ := newCommander(&SpejderStatus{MemberID: "m-1", Status: types.MemberStatusRegistered}, nil)
	if _, err := c.CancelWithdrawal(context.Background(), Actor{}, "2026", "m-1"); !errors.Is(err, ErrNotWaiting) {
		t.Errorf("err = %v, want ErrNotWaiting", err)
	}
}

// --- idempotency ---

// A double click must not put two lines on a timeline. No-op writes publish nothing,
// which also means they emit no live signal — the house pattern.
func TestNoOpsPublishNothing(t *testing.T) {
	tests := []struct {
		name    string
		status  types.MemberStatus
		operate func(commander) (any, error)
	}{
		{"already waiting", types.MemberStatusWaiting, func(c commander) (any, error) {
			return c.RequestWithdrawal(context.Background(), Actor{}, "2026", "m-1")
		}},
		{"already racing", types.MemberStatusRacing, func(c commander) (any, error) {
			return c.CancelWithdrawal(context.Background(), Actor{}, "2026", "m-1")
		}},
		{"override to current status", types.MemberStatusSheltered, func(c commander) (any, error) {
			return c.OverrideStatus(context.Background(), Actor{}, "2026", "m-1", types.MemberStatusSheltered)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, p := newCommander(&SpejderStatus{MemberID: "m-1", CurrentTeamID: "t-1", Status: tt.status}, nil)
			if _, err := tt.operate(c); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(p.subjects) != 0 {
				t.Errorf("a no-op published %v", p.subjects)
			}
		})
	}
}

// --- the override ---

// Lenient about ordering by decision (2026-08-17): this is the out-of-sync repair
// tool, so racing → sheltered is accepted even though no pickup was ever logged.
// Refusing it would reject precisely the correction the tool exists to make.
func TestOverrideIsLenientAboutOrder(t *testing.T) {
	c, p := newCommander(&SpejderStatus{MemberID: "m-1", CurrentTeamID: "t-1", Status: types.MemberStatusRacing}, nil)

	change, err := c.OverrideStatus(context.Background(), Actor{}, "2026", "m-1", types.MemberStatusSheltered)
	if err != nil {
		t.Fatalf("override racing → sheltered was refused: %v", err)
	}
	if change.From != types.MemberStatusRacing || change.To != types.MemberStatusSheltered {
		t.Errorf("change = %+v, want racing → sheltered", change)
	}
	if len(p.subjects) != 1 {
		t.Fatalf("published %v, want one event", p.subjects)
	}
}

// No correction may confer a finish. Only walking the route unaided earns it, and
// this is the one door in the override that stays shut.
func TestOverrideRefusesFinished(t *testing.T) {
	c, p := newCommander(&SpejderStatus{MemberID: "m-1", Status: types.MemberStatusReunited}, nil)

	if _, err := c.OverrideStatus(context.Background(), Actor{}, "2026", "m-1", types.MemberStatusFinished); !errors.Is(err, ErrCannotFinish) {
		t.Errorf("err = %v, want ErrCannotFinish", err)
	}
	if len(p.subjects) != 0 {
		t.Errorf("published %v", p.subjects)
	}
}

func TestOverrideRefusesUnknownStatus(t *testing.T) {
	c, _ := newCommander(&SpejderStatus{MemberID: "m-1", Status: types.MemberStatusRacing}, nil)
	if _, err := c.OverrideStatus(context.Background(), Actor{}, "2026", "m-1", "nonsense"); !errors.Is(err, ErrInvalidStatus) {
		t.Errorf("err = %v, want ErrInvalidStatus", err)
	}
}

// --- resulting strength ---

// The summary records the strength *after* the operation, and it has to be computed
// rather than read: the projection has not seen the event yet, so
// patrulje.activeMemberCount still holds the old number at this point.
func TestStrengthIsComputedForTheStateAfterTheChange(t *testing.T) {
	team := []SpejderStatus{
		racing("m-1", "t-1"),
		racing("m-2", "t-1"),
		racing("m-3", "t-1"),
		{MemberID: "m-4", CurrentTeamID: "t-1", Status: types.MemberStatusSheltered},
	}
	c, _ := newCommander(&SpejderStatus{MemberID: "m-1", CurrentTeamID: "t-1", Status: types.MemberStatusRacing}, team)

	change, err := c.RequestWithdrawal(context.Background(), Actor{}, "2026", "m-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Three were racing; m-1 is leaving, m-4 was never counted.
	if change.TeamStrength != 2 {
		t.Errorf("TeamStrength = %d, want 2 (three racing minus the one leaving)", change.TeamStrength)
	}
}

// A move changes two teams, so it reports two strengths — the origin down one and the
// destination up one. Reporting only the destination is the plausible half-answer that
// would leave the patrol the member *left* looking stronger than it is, and therefore
// not showing as under styrke when it should.
func TestMoveReportsBothTeamStrengths(t *testing.T) {
	origin := []SpejderStatus{racing("m-1", "t-1"), racing("m-2", "t-1"), racing("m-3", "t-1")}
	c, p := newCommander(&SpejderStatus{MemberID: "m-1", CurrentTeamID: "t-1", Status: types.MemberStatusRacing}, origin)

	move, err := c.MoveTeam(context.Background(), Actor{}, "2026", "m-1", "t-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if move.FromTeamStrength != 2 {
		t.Errorf("FromTeamStrength = %d, want 2", move.FromTeamStrength)
	}
	// The stub serves the same rows for any team, so the destination count is the
	// origin's three with the mover already among them — the assertion that matters
	// is that a destination strength was computed at all and is not zero.
	if move.ToTeamStrength == 0 {
		t.Error("ToTeamStrength was not computed")
	}
	if len(p.subjects) != 1 {
		t.Fatalf("published %v, want one event", p.subjects)
	}
	if p.subjects[0] != "NATHEJK.2026.spejder.m-1.team.moved" {
		t.Errorf("subject = %q", p.subjects[0])
	}
}

// Moving a member to the team they are already on is refused rather than recorded: it
// would put a meaningless line on a timeline.
func TestMoveToSameTeamIsRefused(t *testing.T) {
	c, p := newCommander(&SpejderStatus{MemberID: "m-1", CurrentTeamID: "t-1", Status: types.MemberStatusRacing}, nil)
	if _, err := c.MoveTeam(context.Background(), Actor{}, "2026", "m-1", "t-1"); !errors.Is(err, ErrSameTeam) {
		t.Errorf("err = %v, want ErrSameTeam", err)
	}
	if len(p.subjects) != 0 {
		t.Errorf("published %v", p.subjects)
	}
}

// --- moving several members at once ---

// One event per member, published from the server so a partial failure is a failure rather
// than a patrol split between two teams with nobody told which members went.
func TestMoveMembersPublishesOneEventPerMember(t *testing.T) {
	team := []SpejderStatus{racing("m-1", "t-1"), racing("m-2", "t-1"), racing("m-3", "t-1")}
	c, p := newCommander(&SpejderStatus{MemberID: "m-1", CurrentTeamID: "t-1", Status: types.MemberStatusRacing}, team)

	moves, err := c.MoveMembers(context.Background(), Actor{}, "2026", []types.MemberID{"m-1", "m-2"}, "t-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(moves) != 2 {
		t.Fatalf("got %d moves, want 2", len(moves))
	}
	if len(p.subjects) != 2 {
		t.Fatalf("published %v, want one per member", p.subjects)
	}
	for _, s := range p.subjects {
		if !strings.HasSuffix(s, ".team.moved") {
			t.Errorf("unexpected subject %q", s)
		}
	}
}

// **The strength reported is the state after the whole operation, not after each member.**
//
// This is the reason the bulk command computes strengths itself rather than calling the
// per-member path in a loop. Moving three members out one at a time would report the origin
// at 3, then 2, then 1 — three different "resulting strengths" for what is a single step to
// 0, and the timeline entry would name whichever came last.
func TestMoveMembersReportsStrengthForTheWholeOperation(t *testing.T) {
	// Four racing; three of them are leaving.
	team := []SpejderStatus{
		racing("m-1", "t-1"), racing("m-2", "t-1"), racing("m-3", "t-1"), racing("m-4", "t-1"),
	}
	c, _ := newCommander(&SpejderStatus{MemberID: "m-1", CurrentTeamID: "t-1", Status: types.MemberStatusRacing}, team)

	moves, err := c.MoveMembers(context.Background(), Actor{}, "2026", []types.MemberID{"m-1", "m-2", "m-3"}, "t-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, m := range moves {
		if m.FromTeamStrength != 1 {
			t.Errorf("FromTeamStrength = %d, want 1 (four racing minus the three leaving)", m.FromTeamStrength)
		}
	}
	// Every move in one operation reports the same pair, because they describe one step.
	for _, m := range moves[1:] {
		if m.FromTeamStrength != moves[0].FromTeamStrength || m.ToTeamStrength != moves[0].ToTeamStrength {
			t.Errorf("strengths differ within one operation: %+v vs %+v", moves[0], m)
		}
	}
}

// Emptying a patrol reports zero, which is what makes it discontinued — and the summary
// carries that number, so the timeline can say so.
func TestMoveMembersReportsZeroWhenThePatrolIsEmptied(t *testing.T) {
	team := []SpejderStatus{racing("m-1", "t-1"), racing("m-2", "t-1")}
	c, _ := newCommander(&SpejderStatus{MemberID: "m-1", CurrentTeamID: "t-1", Status: types.MemberStatusRacing}, team)

	moves, err := c.MoveMembers(context.Background(), Actor{}, "2026", []types.MemberID{"m-1", "m-2"}, "t-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if moves[0].FromTeamStrength != 0 {
		t.Errorf("FromTeamStrength = %d, want 0", moves[0].FromTeamStrength)
	}
}

// Validation happens for **every** member before anything is published, so an operation
// that cannot legally complete publishes nothing at all. Discovering the problem mid-loop
// is the failure this command exists to avoid.
func TestMoveMembersValidatesEverybodyBeforePublishing(t *testing.T) {
	// The stub serves the same member for every id, and that member is already on t-1 —
	// so a move to t-1 must be refused before any event goes out.
	c, p := newCommander(&SpejderStatus{MemberID: "m-1", CurrentTeamID: "t-1", Status: types.MemberStatusRacing}, nil)

	if _, err := c.MoveMembers(context.Background(), Actor{}, "2026", []types.MemberID{"m-1", "m-2"}, "t-1"); !errors.Is(err, ErrSameTeam) {
		t.Errorf("err = %v, want ErrSameTeam", err)
	}
	if len(p.subjects) != 0 {
		t.Errorf("published %v despite refusing the operation", p.subjects)
	}
}

// An empty list is a no-op rather than an error, matching the collect command: a double
// click must not produce a second timeline entry.
func TestMoveMembersWithNoMembersIsANoOp(t *testing.T) {
	c, p := newCommander(nil, nil)
	moves, err := c.MoveMembers(context.Background(), Actor{}, "2026", nil, "t-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(moves) != 0 || len(p.subjects) != 0 {
		t.Errorf("moves=%v published=%v, want neither", moves, p.subjects)
	}
}

func TestMoveMembersRequiresADestination(t *testing.T) {
	c, _ := newCommander(nil, nil)
	if _, err := c.MoveMembers(context.Background(), Actor{}, "2026", []types.MemberID{"m-1"}, ""); !errors.Is(err, ErrRecordNotFound) {
		t.Errorf("err = %v, want ErrRecordNotFound", err)
	}
}

// --- collecting a whole team ---
// --- subjects ---

// Events go on the member's subject, which is what makes `spejder` the live token and
// what lets the car and shelter interfaces publish the same events later.
func TestEventsArePublishedOnTheMemberSubject(t *testing.T) {
	c, p := newCommander(&SpejderStatus{MemberID: "m-9", CurrentTeamID: "t-1", Status: types.MemberStatusRacing}, nil)

	if _, err := c.RequestWithdrawal(context.Background(), Actor{}, "2026", "m-9"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "NATHEJK.2026.spejder.m-9.withdrawal.requested"
	if p.subjects[0] != want {
		t.Errorf("subject = %q, want %q", p.subjects[0], want)
	}
}

// --- collecting a whole team ---

// One event per racing member, and the loop is on the server so a partial failure is a
// failure rather than a team split across two states with nobody noticing.
func TestCollectTeamRequestsWithdrawalForEveryRacingMember(t *testing.T) {
	team := []SpejderStatus{
		racing("m-1", "t-1"),
		racing("m-2", "t-1"),
		racing("m-3", "t-1"),
	}
	c, p := newCommander(nil, team)

	changes, err := c.CollectTeam(context.Background(), Actor{}, "2026", "t-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(changes) != 3 {
		t.Fatalf("collected %d members, want 3", len(changes))
	}
	if len(p.subjects) != 3 {
		t.Fatalf("published %d events, want one per member: %v", len(p.subjects), p.subjects)
	}
	for _, subj := range p.subjects {
		if !strings.HasSuffix(subj, ".withdrawal.requested") {
			t.Errorf("unexpected subject %q", subj)
		}
	}
	// Nobody is left on the route, which is what makes the patrol discontinued — with
	// no event of its own.
	for _, ch := range changes {
		if ch.TeamStrength != 0 {
			t.Errorf("TeamStrength = %d after collecting the whole team, want 0", ch.TeamStrength)
		}
		if ch.From != types.MemberStatusRacing || ch.To != types.MemberStatusWaiting {
			t.Errorf("change = %+v, want racing → waiting", ch)
		}
	}
}

// **The member already in a car must not be touched.**
//
// They have left the route; re-publishing a withdrawal request would put a second,
// false line in their history and walk them backwards through the lifecycle. This is
// the case that makes "collect the whole team" different from "set everyone to
// waiting".
func TestCollectTeamSkipsMembersAlreadyOutOfTheRace(t *testing.T) {
	team := []SpejderStatus{
		racing("m-1", "t-1"),
		{MemberID: "m-2", CurrentTeamID: "t-1", Status: types.MemberStatusTransit},
		{MemberID: "m-3", CurrentTeamID: "t-1", Status: types.MemberStatusWaiting},
		{MemberID: "m-4", CurrentTeamID: "t-1", Status: types.MemberStatusSheltered},
	}
	c, p := newCommander(nil, team)

	changes, err := c.CollectTeam(context.Background(), Actor{}, "2026", "t-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(changes) != 1 || changes[0].MemberID != "m-1" {
		t.Fatalf("collected %+v, want only the racing member m-1", changes)
	}
	if len(p.subjects) != 1 {
		t.Errorf("published %v, want one event", p.subjects)
	}
}

// A second click collects nobody and publishes nothing, so it cannot produce a second
// timeline entry.
func TestCollectTeamIsANoOpWhenNobodyIsRacing(t *testing.T) {
	team := []SpejderStatus{
		{MemberID: "m-1", CurrentTeamID: "t-1", Status: types.MemberStatusWaiting},
	}
	c, p := newCommander(nil, team)

	changes, err := c.CollectTeam(context.Background(), Actor{}, "2026", "t-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(changes) != 0 {
		t.Errorf("collected %+v, want none", changes)
	}
	if len(p.subjects) != 0 {
		t.Errorf("published %v", p.subjects)
	}
}
