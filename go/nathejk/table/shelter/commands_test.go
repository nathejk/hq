package shelter

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jrgensen/stream"
	"github.com/nathejk/shared-go/types"

	"nathejk.dk/nathejk/table/spejderstatus"
)

// --- fakes ---

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

// stubPlacements serves what this package holds about a member.
type stubPlacements struct {
	rows map[types.MemberID]Placement
}

func (s stubPlacements) GetByMemberIDs(_ context.Context, _ types.YearSlug, ids []types.MemberID) (map[types.MemberID]Placement, error) {
	out := map[types.MemberID]Placement{}
	for _, id := range ids {
		if p, ok := s.rows[id]; ok {
			out[id] = p
		}
	}
	return out, nil
}

func (s stubPlacements) DistinctPlacements(context.Context, types.YearSlug) ([]Zone, error) {
	return nil, nil
}

// stubStatus serves one member's lifecycle status, which is what SetPlacement checks against.
type stubStatus struct {
	status types.MemberStatus
	team   types.TeamID
}

func (s stubStatus) GetByMemberID(_ context.Context, _ types.YearSlug, id types.MemberID) (*spejderstatus.SpejderStatus, error) {
	return &spejderstatus.SpejderStatus{MemberID: id, CurrentTeamID: s.team, Status: s.status}, nil
}

func (s stubStatus) GetByMemberIDs(context.Context, types.YearSlug, []types.MemberID) (map[types.MemberID]spejderstatus.SpejderStatus, error) {
	return nil, nil
}

func (s stubStatus) GetByStatuses(context.Context, types.YearSlug, []types.MemberStatus) ([]spejderstatus.SpejderStatus, error) {
	return nil, nil
}

func (s stubStatus) GetByTeam(context.Context, spejderstatus.Filter) ([]spejderstatus.SpejderStatus, error) {
	return nil, nil
}

func (s stubStatus) GetHistory(context.Context, types.YearSlug, types.MemberID) ([]spejderstatus.StatusEvent, error) {
	return nil, nil
}

func (s stubStatus) InOurCare(context.Context, types.YearSlug) (*spejderstatus.Care, error) {
	return &spejderstatus.Care{}, nil
}

// Unused by shelter, present because spejderstatus.Queries is one interface per table (PRD 011
// added TeamMemberships for the patrol track map).
func (s stubStatus) TeamMemberships(context.Context, types.YearSlug, types.TeamID) ([]spejderstatus.Membership, error) {
	return nil, nil
}

func newCommander(status types.MemberStatus, held map[types.MemberID]Placement) (commander, *recordingPublisher) {
	p := &recordingPublisher{}
	return commander{
		p:      p,
		q:      stubPlacements{rows: held},
		status: stubStatus{status: status, team: "team-1"},
	}, p
}

// --- tests ---

func TestSetPlacementPublishesThePlacering(t *testing.T) {
	c, p := newCommander(types.MemberStatusSheltered, nil)

	if err := c.SetPlacement(context.Background(), spejderstatus.Actor{}, "2026", "m-1", "Telt 4"); err != nil {
		t.Fatalf("SetPlacement: %v", err)
	}
	if len(p.subjects) != 1 || !strings.HasSuffix(p.subjects[0], ".shelter.placed") {
		t.Fatalf("expected one shelter.placed, got %v", p.subjects)
	}
	body, ok := p.bodies[0].(*spejderstatus.ShelterPlaced)
	if !ok {
		t.Fatalf("unexpected body type %T", p.bodies[0])
	}
	if body.Placement != "Telt 4" {
		t.Errorf("Placement = %q, want %q", body.Placement, "Telt 4")
	}
	// The team travels on the event so the projection can create a row for a member it has
	// never seen — see the consumer's upsert.
	if body.TeamID != "team-1" {
		t.Errorf("TeamID = %q, want the member's current team", body.TeamID)
	}
}

// A placering means "this child is asleep here", so recording one for a scout still in a car
// would put them in two places at once and send the crew to a tent nobody is in.
func TestSetPlacementRefusesAMemberNotInTheShelter(t *testing.T) {
	for _, status := range []types.MemberStatus{
		types.MemberStatusRacing,
		types.MemberStatusWaiting,
		types.MemberStatusTransit,
		types.MemberStatusReleased,
	} {
		t.Run(string(status), func(t *testing.T) {
			c, p := newCommander(status, nil)

			err := c.SetPlacement(context.Background(), spejderstatus.Actor{}, "2026", "m-1", "Telt 4")
			if !errors.Is(err, ErrNotSheltered) {
				t.Errorf("err = %v, want ErrNotSheltered", err)
			}
			if len(p.subjects) != 0 {
				t.Errorf("published despite refusing: %v", p.subjects)
			}
		})
	}
}

// Setting the tent a scout is already in publishes nothing, so a double submit puts one line on
// the timeline rather than two — and a re-render of the editor does not look like the child was
// moved.
func TestSetPlacementIsIdempotentOnTheValue(t *testing.T) {
	c, p := newCommander(types.MemberStatusSheltered, map[types.MemberID]Placement{
		"m-1": {MemberID: "m-1", Placement: "Telt 4"},
	})

	if err := c.SetPlacement(context.Background(), spejderstatus.Actor{}, "2026", "m-1", "Telt 4"); err != nil {
		t.Fatalf("expected a no-op rather than an error, got %v", err)
	}
	if len(p.subjects) != 0 {
		t.Errorf("a no-op published %v; it must emit no event and therefore no live signal", p.subjects)
	}
}

// A real move does publish, obviously — but it is worth pinning next to the idempotence test,
// because a dirty-check that compared the wrong thing would silently swallow every move.
func TestSetPlacementPublishesAMove(t *testing.T) {
	c, p := newCommander(types.MemberStatusSheltered, map[types.MemberID]Placement{
		"m-1": {MemberID: "m-1", Placement: "Telt 4"},
	})

	if err := c.SetPlacement(context.Background(), spejderstatus.Actor{}, "2026", "m-1", "Telt 7"); err != nil {
		t.Fatalf("SetPlacement: %v", err)
	}
	if len(p.subjects) != 1 {
		t.Fatalf("expected the move to publish, got %v", p.subjects)
	}
}

// Trimmed before both the comparison and the publish. A tent typed with a trailing space would
// otherwise become a second zone in the suggestion list, one pixel different from the first —
// exactly the drift the suggestions exist to prevent.
func TestSetPlacementTrimsBeforeComparingAndPublishing(t *testing.T) {
	c, p := newCommander(types.MemberStatusSheltered, map[types.MemberID]Placement{
		"m-1": {MemberID: "m-1", Placement: "Telt 4"},
	})

	if err := c.SetPlacement(context.Background(), spejderstatus.Actor{}, "2026", "m-1", "  Telt 4  "); err != nil {
		t.Fatalf("SetPlacement: %v", err)
	}
	if len(p.subjects) != 0 {
		t.Errorf("whitespace made an identical placering look like a move: %v", p.subjects)
	}

	c2, p2 := newCommander(types.MemberStatusSheltered, nil)
	if err := c2.SetPlacement(context.Background(), spejderstatus.Actor{}, "2026", "m-1", " Telt 9 "); err != nil {
		t.Fatalf("SetPlacement: %v", err)
	}
	if body := p2.bodies[0].(*spejderstatus.ShelterPlaced); body.Placement != "Telt 9" {
		t.Errorf("Placement = %q, want it trimmed", body.Placement)
	}
}

// Clearing a placering is deliberately not offered: "nowhere" is not a fact about a child in
// our care. If they moved, the answer is where to; if they left, the answer is a handover.
func TestSetPlacementRefusesABlankPlacering(t *testing.T) {
	for _, in := range []string{"", "   ", "\t"} {
		c, p := newCommander(types.MemberStatusSheltered, nil)

		if err := c.SetPlacement(context.Background(), spejderstatus.Actor{}, "2026", "m-1", in); !errors.Is(err, ErrEmptyPlacement) {
			t.Errorf("err = %v for %q, want ErrEmptyPlacement", err, in)
		}
		if len(p.subjects) != 0 {
			t.Errorf("published a blank placering: %v", p.subjects)
		}
	}
}

// Counted in runes, not bytes. A Danish crew must not be told their tent name is too long
// because of an æ — which is what len() on a string would do.
func TestSetPlacementLengthIsCountedInRunes(t *testing.T) {
	c, p := newCommander(types.MemberStatusSheltered, nil)

	// 64 runes, comfortably over 64 bytes in UTF-8.
	justFits := strings.Repeat("æ", MaxPlacementLength)
	if err := c.SetPlacement(context.Background(), spejderstatus.Actor{}, "2026", "m-1", justFits); err != nil {
		t.Errorf("a 64-rune placering was refused: %v", err)
	}
	if len(p.subjects) != 1 {
		t.Errorf("expected the 64-rune placering to publish, got %v", p.subjects)
	}

	c2, p2 := newCommander(types.MemberStatusSheltered, nil)
	tooLong := strings.Repeat("a", MaxPlacementLength+1)
	if err := c2.SetPlacement(context.Background(), spejderstatus.Actor{}, "2026", "m-1", tooLong); !errors.Is(err, ErrPlacementTooLong) {
		t.Errorf("err = %v, want ErrPlacementTooLong", err)
	}
	if len(p2.subjects) != 0 {
		t.Errorf("published an over-long placering: %v", p2.subjects)
	}
}

// A sheltered member with no row here still gets their placering recorded. The status says they
// are in the shelter, so the row is either being written this moment or was lost — and in both
// cases recording where they are is the useful thing to do.
func TestSetPlacementWorksWithNoExistingRow(t *testing.T) {
	c, p := newCommander(types.MemberStatusSheltered, map[types.MemberID]Placement{})

	if err := c.SetPlacement(context.Background(), spejderstatus.Actor{}, "2026", "m-1", "Telt 4"); err != nil {
		t.Fatalf("SetPlacement: %v", err)
	}
	if len(p.subjects) != 1 {
		t.Errorf("expected a publish for a member with no shelter row, got %v", p.subjects)
	}
}
