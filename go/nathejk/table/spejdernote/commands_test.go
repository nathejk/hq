package spejdernote

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

// stubQueries serves one note, which is all UpdateComment reads.
type stubQueries struct {
	note *Note
	err  error
}

func (s stubQueries) GetByID(context.Context, types.YearSlug, NoteID) (*Note, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.note == nil {
		return nil, ErrRecordNotFound
	}
	return s.note, nil
}

func (s stubQueries) GetByMember(context.Context, types.YearSlug, types.MemberID) ([]Note, error) {
	return nil, nil
}

func (s stubQueries) SummaryByMembers(context.Context, types.YearSlug, []types.MemberID) (map[types.MemberID]Summary, error) {
	return nil, nil
}

func newCommander(existing *Note) (commander, *recordingPublisher) {
	p := &recordingPublisher{}
	return commander{p: p, q: stubQueries{note: existing}}, p
}

const written = "Ringet til mor 01.20. Hun henter kl. 06."

// --- tests ---

func TestCommentPublishesOnTheScoutsSubject(t *testing.T) {
	c, p := newCommander(nil)

	id, err := c.Comment(context.Background(), Actor{}, "2026", "m-1", written)
	if err != nil {
		t.Fatalf("Comment: %v", err)
	}
	if id == "" {
		t.Fatal("expected a minted note id")
	}
	// The subject is what carries the live signal's entity token, so this is what makes notes
	// arrive on `spejder` — which every member screen already depends on. A subject of our own
	// would have needed every client to declare a new dependency.
	if len(p.subjects) != 1 || p.subjects[0] != "NATHEJK.2026.spejder.m-1.commented" {
		t.Fatalf("subjects = %v", p.subjects)
	}
	body, ok := p.bodies[0].(*Commented)
	if !ok {
		t.Fatalf("unexpected body type %T", p.bodies[0])
	}
	if body.NoteID != id {
		t.Errorf("the published id %q is not the one returned %q", body.NoteID, id)
	}
	if body.Note != written {
		t.Errorf("note = %q, want %q", body.Note, written)
	}
}

// Ids are minted server-side so a client cannot collide with one it has not seen — or name another
// member's note.
func TestCommentMintsAFreshIDEveryTime(t *testing.T) {
	c, _ := newCommander(nil)

	first, _ := c.Comment(context.Background(), Actor{}, "2026", "m-1", written)
	second, _ := c.Comment(context.Background(), Actor{}, "2026", "m-1", written)

	if first == second {
		t.Error("two notes were given the same id")
	}
}

// Two identical notes are a legitimate thing to write: "ringet til mor, intet svar" twice is two
// facts, an hour apart. Suppressing the second would lose the more interesting one.
func TestCommentDoesNotSuppressADuplicate(t *testing.T) {
	c, p := newCommander(nil)

	_, _ = c.Comment(context.Background(), Actor{}, "2026", "m-1", written)
	_, _ = c.Comment(context.Background(), Actor{}, "2026", "m-1", written)

	if len(p.subjects) != 2 {
		t.Errorf("expected both notes published, got %v", p.subjects)
	}
}

func TestCommentRefusesAnEmptyNote(t *testing.T) {
	for _, in := range []string{"", "   ", "\t\n"} {
		c, p := newCommander(nil)

		if _, err := c.Comment(context.Background(), Actor{}, "2026", "m-1", in); !errors.Is(err, ErrEmptyNote) {
			t.Errorf("err = %v for %q, want ErrEmptyNote", err, in)
		}
		if len(p.subjects) != 0 {
			t.Errorf("published an empty note: %v", p.subjects)
		}
	}
}

// Counted in runes: a note about "Sofija, må sove i hallen — bange for edderkopper" is shorter than
// len() thinks, and a Danish crew must not have a note refused because of its æ's.
func TestNoteLengthIsCountedInRunes(t *testing.T) {
	c, p := newCommander(nil)

	justFits := strings.Repeat("æ", MaxNoteLength)
	if _, err := c.Comment(context.Background(), Actor{}, "2026", "m-1", justFits); err != nil {
		t.Errorf("a %d-rune note was refused: %v", MaxNoteLength, err)
	}
	if len(p.subjects) != 1 {
		t.Errorf("expected the note to publish, got %v", p.subjects)
	}

	c2, p2 := newCommander(nil)
	tooLong := strings.Repeat("a", MaxNoteLength+1)
	if _, err := c2.Comment(context.Background(), Actor{}, "2026", "m-1", tooLong); !errors.Is(err, ErrNoteTooLong) {
		t.Errorf("err = %v, want ErrNoteTooLong", err)
	}
	if len(p2.subjects) != 0 {
		t.Errorf("published an over-long note: %v", p2.subjects)
	}
}

// Trimmed before publishing, so the stored text does not carry whatever whitespace a textarea left
// behind — and so a "correction" that only adds a trailing newline is recognised as a no-op below.
func TestCommentTrims(t *testing.T) {
	c, p := newCommander(nil)

	if _, err := c.Comment(context.Background(), Actor{}, "2026", "m-1", "  "+written+"\n"); err != nil {
		t.Fatalf("Comment: %v", err)
	}
	if body := p.bodies[0].(*Commented); body.Note != written {
		t.Errorf("note = %q, want it trimmed", body.Note)
	}
}

func TestUpdateCommentPublishesTheCorrection(t *testing.T) {
	c, p := newCommander(&Note{NoteID: "n-1", MemberID: "m-1", Note: written})

	if err := c.UpdateComment(context.Background(), Actor{}, "2026", "m-1", "n-1", "Rettet tekst"); err != nil {
		t.Fatalf("UpdateComment: %v", err)
	}
	if len(p.subjects) != 1 || p.subjects[0] != "NATHEJK.2026.spejder.m-1.comment.updated" {
		t.Fatalf("subjects = %v", p.subjects)
	}
	body, ok := p.bodies[0].(*CommentUpdated)
	if !ok {
		t.Fatalf("unexpected body type %T", p.bodies[0])
	}
	// The whole text, not a diff: the projection holds current text and the stream holds history, so
	// each version has to stand alone. Replaying the third correction must not depend on the first two.
	if body.Note != "Rettet tekst" {
		t.Errorf("note = %q", body.Note)
	}
}

// **The check that matters.** Without it a client could amend any note by id — the same hole
// sos.UpdateComment closes for cases. It is refused with its own error, because telling an operator
// "no such note" about a note plainly on their screen would send them hunting a bug that is not there.
func TestUpdateCommentRefusesAnotherMembersNote(t *testing.T) {
	c, p := newCommander(&Note{NoteID: "n-1", MemberID: "m-somebody-else", Note: written})

	err := c.UpdateComment(context.Background(), Actor{}, "2026", "m-1", "n-1", "Rettet")
	if !errors.Is(err, ErrWrongMember) {
		t.Errorf("err = %v, want ErrWrongMember", err)
	}
	if len(p.subjects) != 0 {
		t.Errorf("published despite refusing: %v", p.subjects)
	}
}

// Unchanged text publishes nothing, so a re-submitted edit does not put a second version in the
// stream. That keeps the history worth reading: every version in it is one somebody actually wrote.
func TestUpdateCommentIsIdempotentOnTheText(t *testing.T) {
	c, p := newCommander(&Note{NoteID: "n-1", MemberID: "m-1", Note: written})

	if err := c.UpdateComment(context.Background(), Actor{}, "2026", "m-1", "n-1", "  "+written+"  "); err != nil {
		t.Fatalf("expected a no-op rather than an error, got %v", err)
	}
	if len(p.subjects) != 0 {
		t.Errorf("whitespace made identical text look like a correction: %v", p.subjects)
	}
}

func TestUpdateCommentRequiresAnExistingNote(t *testing.T) {
	c, p := newCommander(nil) // GetByID reports not found

	if err := c.UpdateComment(context.Background(), Actor{}, "2026", "m-1", "n-missing", "Rettet"); !errors.Is(err, ErrRecordNotFound) {
		t.Errorf("err = %v, want ErrRecordNotFound", err)
	}
	if len(p.subjects) != 0 {
		t.Errorf("published for a note that does not exist: %v", p.subjects)
	}
}

func TestUpdateCommentRefusesAnEmptyCorrection(t *testing.T) {
	c, p := newCommander(&Note{NoteID: "n-1", MemberID: "m-1", Note: written})

	if err := c.UpdateComment(context.Background(), Actor{}, "2026", "m-1", "n-1", "   "); !errors.Is(err, ErrEmptyNote) {
		t.Errorf("err = %v, want ErrEmptyNote", err)
	}
	if len(p.subjects) != 0 {
		t.Errorf("published an empty correction: %v", p.subjects)
	}
}

// A note needs somebody to be about. Refused before publishing, since a note on the empty member id
// would land on the subject NATHEJK.2026.spejder..commented and belong to nobody.
func TestCommentRequiresAMember(t *testing.T) {
	c, p := newCommander(nil)

	if _, err := c.Comment(context.Background(), Actor{}, "2026", "", written); !errors.Is(err, ErrRecordNotFound) {
		t.Errorf("err = %v, want ErrRecordNotFound", err)
	}
	if len(p.subjects) != 0 {
		t.Errorf("published a note about nobody: %v", p.subjects)
	}
}
