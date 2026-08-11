package sos

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jrgensen/stream"
	"github.com/nathejk/shared-go/tables"
	"github.com/nathejk/shared-go/types"
)

// --- fakes ---

// recordingPublisher captures the subjects published, which is what the
// dirty-check tests are really about: how many events a patch produces, and which.
type recordingPublisher struct {
	subjects []string
	bodies   []any
}

func (p *recordingPublisher) MessageFunc() stream.MessageFunc {
	return func(s stream.Subject) stream.MutableMessage {
		return &mutableMessage{subject: s, publisher: p}
	}
}

func (p *recordingPublisher) Publish(m stream.Message) error {
	p.subjects = append(p.subjects, m.Subject().Subject())
	p.bodies = append(p.bodies, m.RawBody())
	return nil
}

type mutableMessage struct {
	subject   stream.Subject
	body      any
	meta      any
	at        time.Time
	publisher *recordingPublisher
}

func (m *mutableMessage) Subject() stream.Subject     { return m.subject }
func (m *mutableMessage) Time() time.Time             { return m.at }
func (m *mutableMessage) Sequence() uint64            { return 0 }
func (m *mutableMessage) Body(any) error              { return nil }
func (m *mutableMessage) Meta(any) error              { return nil }
func (m *mutableMessage) RawBody() any                { return m.body }
func (m *mutableMessage) RawMeta() any                { return m.meta }
func (m *mutableMessage) SetSubject(s stream.Subject) { m.subject = s }
func (m *mutableMessage) SetBody(b any) error         { m.body = b; return nil }
func (m *mutableMessage) SetMeta(v any) error         { m.meta = v; return nil }
func (m *mutableMessage) SetTime(t time.Time) error   { m.at = t; return nil }

// stubQueries returns a fixed case, standing in for the read model the commander
// dirty-checks against.
type stubQueries struct {
	sos        *Sos
	assignable []types.Slug
	err        error
}

func (q stubQueries) GetAll(context.Context, Filter) ([]*Sos, error) { return nil, q.err }
func (q stubQueries) GetByID(context.Context, types.SosID) (*Sos, error) {
	if q.err != nil {
		return nil, q.err
	}
	return q.sos, nil
}
func (q stubQueries) GetByTeam(context.Context, types.YearSlug, types.TeamID) ([]*Sos, error) {
	return nil, q.err
}
func (q stubQueries) AssignableSections(context.Context, types.YearSlug) ([]types.Slug, error) {
	return q.assignable, q.err
}

func existingCase() *Sos {
	return &Sos{
		ID:                  "sos-1",
		YearSlug:            "2026",
		Headline:            "Forstuvet ankel",
		Description:         "Ringer fra stien",
		Status:              StatusOpen,
		Severity:            SeverityYellow,
		AssigneeSectionSlug: "samarit",
		Teams:               []Team{{TeamID: "team-1"}},
		Timeline:            []Activity{{Type: ActivityCommented, ActivityID: "c1"}},
	}
}

func newCommander(s *Sos) (commander, *recordingPublisher) {
	p := &recordingPublisher{}
	return commander{p: p, q: stubQueries{sos: s}}, p
}

func str(s string) *string          { return &s }
func sev(s Severity) *Severity      { return &s }
func stat(s Status) *Status         { return &s }
func slug(s types.Slug) *types.Slug { return &s }

// --- tests ---

func TestPatchPublishesOnlyChangedFields(t *testing.T) {
	c, p := newCommander(existingCase())

	err := c.Patch(context.Background(), Actor{}, "sos-1", PatchCommand{
		Headline: str("Forstuvet ankel ved post 4"), // changed
		Severity: sev(SeverityYellow),               // same as current
		Status:   stat(StatusClosed),                // changed
	})
	if err != nil {
		t.Fatalf("Patch: %v", err)
	}

	want := []string{
		"NATHEJK.2026.sos.sos-1.headline.updated",
		"NATHEJK.2026.sos.sos-1.closed",
	}
	if len(p.subjects) != len(want) {
		t.Fatalf("published %v, want %v", p.subjects, want)
	}
	for i, s := range want {
		if p.subjects[i] != s {
			t.Errorf("published[%d] = %q, want %q", i, p.subjects[i], s)
		}
	}
}

func TestPatchThatChangesNothingPublishesNothing(t *testing.T) {
	// This is the property that keeps other operators' screens still: no event
	// means no live signal, so a no-op save does not make every open page refetch.
	c, p := newCommander(existingCase())

	err := c.Patch(context.Background(), Actor{}, "sos-1", PatchCommand{
		Headline:            str("Forstuvet ankel"),
		Description:         str("Ringer fra stien"),
		Severity:            sev(SeverityYellow),
		AssigneeSectionSlug: slug("samarit"),
		Status:              stat(StatusOpen),
	})
	if err != nil {
		t.Fatalf("Patch: %v", err)
	}
	if len(p.subjects) != 0 {
		t.Errorf("published %v, want nothing", p.subjects)
	}
}

func TestClosingAClosedCaseIsANoOp(t *testing.T) {
	// PRD 001 §5: idempotent, not an error. Falls out of treating status as a field
	// rather than as two verbs.
	s := existingCase()
	s.Status = StatusClosed
	c, p := newCommander(s)

	if err := c.Patch(context.Background(), Actor{}, "sos-1", PatchCommand{Status: stat(StatusClosed)}); err != nil {
		t.Fatalf("Patch: %v", err)
	}
	if len(p.subjects) != 0 {
		t.Errorf("published %v, want nothing", p.subjects)
	}
}

func TestReopenPublishesReopened(t *testing.T) {
	s := existingCase()
	s.Status = StatusClosed
	c, p := newCommander(s)

	if err := c.Patch(context.Background(), Actor{}, "sos-1", PatchCommand{Status: stat(StatusOpen)}); err != nil {
		t.Fatalf("Patch: %v", err)
	}
	if len(p.subjects) != 1 || p.subjects[0] != "NATHEJK.2026.sos.sos-1.reopened" {
		t.Errorf("published %v, want a single reopened event", p.subjects)
	}
}

func TestCreateRequiresHeadlineAndDescription(t *testing.T) {
	c, p := newCommander(nil)

	for _, tc := range []struct{ headline, description string }{
		{"", "beskrivelse"},
		{"overskrift", ""},
		{"   ", "beskrivelse"},
		{"overskrift", "  "},
	} {
		_, err := c.Create(context.Background(), Actor{}, "2026", tc.headline, tc.description)
		if !errors.Is(err, ErrEmptyField) {
			t.Errorf("Create(%q, %q) error = %v, want ErrEmptyField", tc.headline, tc.description, err)
		}
	}
	if len(p.subjects) != 0 {
		t.Errorf("published %v, want nothing", p.subjects)
	}
}

func TestCreatePublishesCreatedWithAMintedID(t *testing.T) {
	c, p := newCommander(nil)

	id, err := c.Create(context.Background(), Actor{}, "2026", " Forstuvet ankel ", " Ringer fra stien ")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if id == "" {
		t.Fatal("Create returned an empty id")
	}
	if len(p.subjects) != 1 {
		t.Fatalf("published %v, want one event", p.subjects)
	}
	if want := "NATHEJK.2026.sos." + string(id) + ".created"; p.subjects[0] != want {
		t.Errorf("subject = %q, want %q", p.subjects[0], want)
	}
	// Trimmed, so a stray space does not become part of the headline shown in the list.
	body, ok := p.bodies[0].(*Created)
	if !ok {
		t.Fatalf("body is %T, want *Created", p.bodies[0])
	}
	if body.Headline != "Forstuvet ankel" || body.Description != "Ringer fra stien" {
		t.Errorf("body = %+v, want trimmed values", body)
	}
}

func TestAssociateTeamIsDirtyChecked(t *testing.T) {
	// The projection would upsert happily; the point of the check is the timeline,
	// which should not collect duplicate "patrulje tilknyttet" entries when two
	// operators on the same call both reach for the same patrol.
	c, p := newCommander(existingCase())

	if err := c.AssociateTeam(context.Background(), Actor{}, "sos-1", "team-1"); err != nil {
		t.Fatalf("AssociateTeam: %v", err)
	}
	if len(p.subjects) != 0 {
		t.Errorf("published %v for an already-associated team, want nothing", p.subjects)
	}

	if err := c.AssociateTeam(context.Background(), Actor{}, "sos-1", "team-2"); err != nil {
		t.Fatalf("AssociateTeam: %v", err)
	}
	if len(p.subjects) != 1 || p.subjects[0] != "NATHEJK.2026.sos.sos-1.team.associated" {
		t.Errorf("published %v, want one team.associated", p.subjects)
	}
}

func TestDisassociateUnassociatedTeamIsANoOp(t *testing.T) {
	c, p := newCommander(existingCase())

	if err := c.DisassociateTeam(context.Background(), Actor{}, "sos-1", "team-9"); err != nil {
		t.Fatalf("DisassociateTeam: %v", err)
	}
	if len(p.subjects) != 0 {
		t.Errorf("published %v, want nothing", p.subjects)
	}
}

func TestUpdateCommentRejectsAForeignComment(t *testing.T) {
	// A mistyped id must not amend another case's comment: the edit would be
	// recorded on a timeline nobody involved is reading.
	c, p := newCommander(existingCase())

	err := c.UpdateComment(context.Background(), Actor{}, "sos-1", "c-does-not-exist", "ny tekst")
	if !errors.Is(err, tables.ErrRecordNotFound) {
		t.Errorf("error = %v, want ErrRecordNotFound", err)
	}
	if len(p.subjects) != 0 {
		t.Errorf("published %v, want nothing", p.subjects)
	}
}

func TestUpdateCommentPublishesForAKnownComment(t *testing.T) {
	c, p := newCommander(existingCase())

	if err := c.UpdateComment(context.Background(), Actor{}, "sos-1", "c1", "ny tekst"); err != nil {
		t.Fatalf("UpdateComment: %v", err)
	}
	if len(p.subjects) != 1 || p.subjects[0] != "NATHEJK.2026.sos.sos-1.comment.updated" {
		t.Errorf("published %v, want one comment.updated", p.subjects)
	}
}

func TestCommentRequiresText(t *testing.T) {
	c, p := newCommander(existingCase())

	if _, err := c.Comment(context.Background(), Actor{}, "sos-1", "   "); !errors.Is(err, ErrEmptyComment) {
		t.Errorf("error = %v, want ErrEmptyComment", err)
	}
	if len(p.subjects) != 0 {
		t.Errorf("published %v, want nothing", p.subjects)
	}
}

func TestPatchRejectsUnknownSeverity(t *testing.T) {
	c, p := newCommander(existingCase())

	if err := c.Patch(context.Background(), Actor{}, "sos-1", PatchCommand{Severity: sev("orange")}); err == nil {
		t.Error("Patch accepted an unknown severity")
	}
	if len(p.subjects) != 0 {
		t.Errorf("published %v, want nothing", p.subjects)
	}
}

func TestPatchOnAMissingCaseFails(t *testing.T) {
	p := &recordingPublisher{}
	c := commander{p: p, q: stubQueries{err: tables.ErrRecordNotFound}}

	err := c.Patch(context.Background(), Actor{}, "sos-gone", PatchCommand{Headline: str("ny")})
	if !errors.Is(err, tables.ErrRecordNotFound) {
		t.Errorf("error = %v, want ErrRecordNotFound", err)
	}
	if len(p.subjects) != 0 {
		t.Errorf("published %v, want nothing", p.subjects)
	}
}
