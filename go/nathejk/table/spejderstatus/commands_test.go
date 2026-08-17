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
type stubQueries struct {
	member *SpejderStatus
	team   []SpejderStatus
	err    error
}

func (q stubQueries) GetByMemberID(context.Context, types.YearSlug, types.MemberID) (*SpejderStatus, error) {
	if q.err != nil {
		return nil, q.err
	}
	return q.member, nil
}

func (q stubQueries) GetByTeam(context.Context, Filter) ([]SpejderStatus, error) {
	return q.team, nil
}

func (q stubQueries) InOurCare(context.Context, types.YearSlug) (*Care, error) {
	return &Care{}, nil
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
