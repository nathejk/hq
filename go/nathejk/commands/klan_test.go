package commands

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jrgensen/stream"
	"github.com/nathejk/shared-go/messages"
	sharedklan "github.com/nathejk/shared-go/tables/klan"
	"github.com/nathejk/shared-go/types"
)

// --- fakes ---

type recordingPublisher struct {
	subjects []string
	bodies   []any
	metas    []any
}

func (p *recordingPublisher) MessageFunc() stream.MessageFunc {
	return func(s stream.Subject) stream.MutableMessage {
		return &recordedMessage{subject: s}
	}
}

func (p *recordingPublisher) Publish(m stream.Message) error {
	p.subjects = append(p.subjects, m.Subject().Subject())
	p.bodies = append(p.bodies, m.RawBody())
	p.metas = append(p.metas, m.RawMeta())
	return nil
}

type recordedMessage struct {
	subject stream.Subject
	body    any
	meta    any
	at      time.Time
}

func (m *recordedMessage) Subject() stream.Subject     { return m.subject }
func (m *recordedMessage) Time() time.Time             { return m.at }
func (m *recordedMessage) Sequence() uint64            { return 0 }
func (m *recordedMessage) Body(any) error              { return nil }
func (m *recordedMessage) Meta(any) error              { return nil }
func (m *recordedMessage) RawBody() any                { return m.body }
func (m *recordedMessage) RawMeta() any                { return m.meta }
func (m *recordedMessage) SetSubject(s stream.Subject) { m.subject = s }
func (m *recordedMessage) SetBody(b any) error         { m.body = b; return nil }
func (m *recordedMessage) SetMeta(v any) error         { m.meta = v; return nil }
func (m *recordedMessage) SetTime(t time.Time) error   { m.at = t; return nil }

type stubKlanQuerier struct {
	klan *sharedklan.Klan
	err  error
}

func (q stubKlanQuerier) GetByID(context.Context, types.TeamID) (*sharedklan.Klan, error) {
	if q.err != nil {
		return nil, q.err
	}
	return q.klan, nil
}

// stubEntity stands in for the shared-go klan entity, recording only that Delete
// was delegated to it.
type stubEntity struct {
	sharedklan.Commands
	deleted types.TeamID
	err     error
}

func (e *stubEntity) Delete(_ context.Context, teamID types.TeamID) error {
	e.deleted = teamID
	return e.err
}

func existingKlan() *sharedklan.Klan {
	return &sharedklan.Klan{
		ID:     "team-1",
		Year:   "2024",
		Name:   "Ulvene",
		Status: types.SignupStatusPay,
	}
}

// --- tests ---

// The subject's year is the load-bearing detail: an override applied to a klan
// from an earlier season must be published onto *that* season's subject, or the
// projection would apply it to whatever klan holds the id in the current year.
// A configured "current year" (as team.go uses) would get this wrong silently.
func TestSetStatusPublishesOnTheKlansOwnYear(t *testing.T) {
	p := &recordingPublisher{}
	c := NewKlan(p, stubKlanQuerier{klan: existingKlan()}, &stubEntity{})

	if err := c.SetStatus(context.Background(), "team-1", types.SignupStatusPaid); err != nil {
		t.Fatalf("SetStatus: %v", err)
	}

	if len(p.subjects) != 1 {
		t.Fatalf("expected 1 event, got %d: %v", len(p.subjects), p.subjects)
	}
	const want = "NATHEJK.2024.klan.team-1.status.changed"
	if p.subjects[0] != want {
		t.Errorf("subject = %q, want %q", p.subjects[0], want)
	}
}

func TestSetStatusCarriesTheRequestedStatus(t *testing.T) {
	p := &recordingPublisher{}
	c := NewKlan(p, stubKlanQuerier{klan: existingKlan()}, &stubEntity{})

	if err := c.SetStatus(context.Background(), "team-1", types.SignupStatusPaid); err != nil {
		t.Fatalf("SetStatus: %v", err)
	}

	body, ok := p.bodies[0].(*messages.NathejkKlanStatusChanged)
	if !ok {
		t.Fatalf("body type = %T, want *messages.NathejkKlanStatusChanged", p.bodies[0])
	}
	if body.Status != types.SignupStatusPaid {
		t.Errorf("status = %q, want %q", body.Status, types.SignupStatusPaid)
	}
	if body.TeamID != "team-1" {
		t.Errorf("teamId = %q, want team-1", body.TeamID)
	}
}

// Attributed to hq so the trail distinguishes an operator's decision from the
// payment flow's own status changes — the only thing that makes an out-of-band
// status readable after the weekend.
func TestSetStatusIsAttributedToHQ(t *testing.T) {
	p := &recordingPublisher{}
	c := NewKlan(p, stubKlanQuerier{klan: existingKlan()}, &stubEntity{})

	if err := c.SetStatus(context.Background(), "team-1", types.SignupStatusPaid); err != nil {
		t.Fatalf("SetStatus: %v", err)
	}

	meta, ok := p.metas[0].(*messages.Metadata)
	if !ok {
		t.Fatalf("meta type = %T, want *messages.Metadata", p.metas[0])
	}
	if meta.Producer != "hq-api" {
		t.Errorf("producer = %q, want hq-api", meta.Producer)
	}
}

// Reported rather than silently accepted: this is a single deliberate act, and an
// operator told nothing cannot tell "already correct" from "did not work".
func TestSetStatusRefusesTheStatusAlreadyHeld(t *testing.T) {
	p := &recordingPublisher{}
	c := NewKlan(p, stubKlanQuerier{klan: existingKlan()}, &stubEntity{})

	err := c.SetStatus(context.Background(), "team-1", types.SignupStatusPay)
	if !errors.Is(err, ErrKlanStatusUnchanged) {
		t.Fatalf("err = %v, want ErrKlanStatusUnchanged", err)
	}
	if len(p.subjects) != 0 {
		t.Errorf("published %v, want nothing", p.subjects)
	}
}

func TestSetStatusFailsWhenTheKlanIsUnknown(t *testing.T) {
	p := &recordingPublisher{}
	notFound := errors.New("no such klan")
	c := NewKlan(p, stubKlanQuerier{err: notFound}, &stubEntity{})

	if err := c.SetStatus(context.Background(), "team-1", types.SignupStatusPaid); !errors.Is(err, notFound) {
		t.Fatalf("err = %v, want %v", err, notFound)
	}
	if len(p.subjects) != 0 {
		t.Errorf("published %v, want nothing", p.subjects)
	}
}

// Delegated, not reimplemented: what deleting a klan means belongs to the entity
// that owns the klan, and a second definition here would be free to drift.
func TestDeleteDelegatesToTheEntity(t *testing.T) {
	p := &recordingPublisher{}
	entity := &stubEntity{}
	c := NewKlan(p, stubKlanQuerier{klan: existingKlan()}, entity)

	if err := c.Delete(context.Background(), "team-1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if entity.deleted != "team-1" {
		t.Errorf("delegated teamId = %q, want team-1", entity.deleted)
	}
	if len(p.subjects) != 0 {
		t.Errorf("published %v directly, want nothing (the entity publishes)", p.subjects)
	}
}
