package spejdernote

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jrgensen/stream"
	"github.com/jrgensen/stream/subject"
	"github.com/nathejk/shared-go/messages"
	"github.com/nathejk/shared-go/types"
)

// MaxNoteLength caps a note.
//
// 2000 characters: long enough for a phone call's worth of detail — who was rung, what was agreed,
// what must not happen — and short enough that a row snippet and a VARCHAR column stay sane. A note
// longer than this is a sign somebody is using the trail as a document, which it is not.
const MaxNoteLength = 2000

var (
	// ErrEmptyNote is returned rather than publishing a blank note. A note that says nothing is
	// not a record of anything, and it would still show up as a count on the row.
	ErrEmptyNote = errors.New("note is required")

	// ErrNoteTooLong is returned for a note over MaxNoteLength.
	ErrNoteTooLong = errors.New("note is too long")

	// ErrWrongMember is returned when a correction names a note that belongs to somebody else.
	//
	// Its own error rather than a not-found, because the two are different problems: a wrong id is
	// a client bug, and a note belonging to another member is either a client bug or an attempt to
	// edit a record from the wrong screen. Both refuse, and an operator should not be told "no such
	// note" about a note that plainly exists.
	ErrWrongMember = errors.New("note belongs to another member")
)

// Commands is the write surface for the trail.
//
// The Actor is passed in by the caller rather than read from the request context, matching every
// other table package: it keeps this package free of nathejk.dk/internal/requestctx.
//
// # What these deliberately do not enforce
//
// Editing is not restricted to the note's author. There is no identity to restrict it with — every
// actor is anonymous until HQ has login (PRD 001 §6) — so an ownership check today would compare
// two empty strings and permit everything, which is worse than no check because it would look like
// one. Accepted for a crew of colleagues on one site (PRD 008 §8), and **worth revisiting the day
// accounts exist**: at that point "anybody may rewrite anybody's note about a child" stops being a
// consequence of the auth gap and becomes a decision.
type Commands interface {
	Comment(ctx context.Context, actor Actor, year types.YearSlug, member types.MemberID, note string) (NoteID, error)
	UpdateComment(ctx context.Context, actor Actor, year types.YearSlug, member types.MemberID, id NoteID, note string) error
}

type commander struct {
	p stream.Publisher
	q Queries
}

// Comment adds a note to a scout's trail and returns its id.
//
// The id is minted here rather than accepted from the client, as sos comment ids are: a client
// cannot collide with an id it has not seen, and — more to the point — cannot choose an id that
// already belongs to another member's note.
//
// No dirty-check, and no attempt to detect a duplicate note. Two identical notes are a legitimate
// thing to write: "ringet til mor, intet svar" twice is two facts, an hour apart. Suppressing the
// second would lose the more interesting one.
func (c commander) Comment(ctx context.Context, actor Actor, year types.YearSlug, member types.MemberID, note string) (NoteID, error) {
	note = strings.TrimSpace(note)
	if err := validateNote(note); err != nil {
		return "", err
	}
	if member == "" {
		return "", ErrRecordNotFound
	}
	id := NewNoteID()
	body := &Commented{NoteID: id, MemberID: member, Note: note, Actor: actor}
	if err := c.publish(actor, year, member, "commented", body); err != nil {
		return "", err
	}
	return id, nil
}

// UpdateComment corrects a note's text.
//
// Two checks before publishing, and the second is the one that matters: the note must exist, and it
// must belong to the member the request names. Without the ownership check a client could amend any
// note by id — the same hole sos.UpdateComment closes for cases, and the reason the projection
// scopes its UPDATE by member as well.
//
// Unchanged text publishes nothing. A re-submitted edit therefore does not put a second version in
// the stream, which keeps the history worth reading: every version in it is a version somebody
// actually wrote.
func (c commander) UpdateComment(ctx context.Context, actor Actor, year types.YearSlug, member types.MemberID, id NoteID, note string) error {
	note = strings.TrimSpace(note)
	if err := validateNote(note); err != nil {
		return err
	}
	if id == "" {
		return ErrRecordNotFound
	}
	current, err := c.q.GetByID(ctx, year, id)
	if err != nil {
		return err
	}
	if current.MemberID != member {
		return ErrWrongMember
	}
	if current.Note == note {
		return nil
	}
	body := &CommentUpdated{NoteID: id, MemberID: member, Note: note, Actor: actor}
	return c.publish(actor, year, member, "comment.updated", body)
}

// validateNote applies the two rules both commands share.
//
// Length counted in runes, not bytes: a note about "Sofija, må sove i hallen — bange for
// edderkopper" is shorter than len() thinks, and a Danish crew must not have a note refused because
// of its æ's. The same trap as shelter.MaxPlacementLength.
func validateNote(note string) error {
	if note == "" {
		return ErrEmptyNote
	}
	if len([]rune(note)) > MaxNoteLength {
		return ErrNoteTooLong
	}
	return nil
}

func (c commander) publish(actor Actor, year types.YearSlug, member types.MemberID, event string, body any) error {
	msg := c.p.MessageFunc()(subject.FromStr(fmt.Sprintf("NATHEJK.%s.spejder.%s.%s", year, member, event)))
	if err := msg.SetBody(body); err != nil {
		return err
	}
	if err := msg.SetMeta(&messages.Metadata{UserID: actor.UserID}); err != nil {
		return err
	}
	return c.p.Publish(msg)
}
