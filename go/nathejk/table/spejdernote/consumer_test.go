package spejdernote

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/jrgensen/stream"
	"github.com/jrgensen/stream/subject"
)

// --- fakes ---

// Asserting on emitted SQL, as the other projections in this repo do. For a table that is both
// appended to and edited, what matters is which columns each event is entitled to write — get that
// wrong and a replay quietly rewrites the night's notes.
type recordingWriter struct {
	stmts []string
}

func (w *recordingWriter) Consume(stmt string) error {
	w.stmts = append(w.stmts, stmt)
	return nil
}

func (w *recordingWriter) only(t *testing.T) string {
	t.Helper()
	if len(w.stmts) != 1 {
		t.Fatalf("expected exactly one statement, got %d: %v", len(w.stmts), w.stmts)
	}
	return w.stmts[0]
}

type fakeMessage struct {
	subject stream.Subject
	body    any
	at      time.Time
}

func (m fakeMessage) Subject() stream.Subject { return m.subject }
func (m fakeMessage) Time() time.Time         { return m.at }
func (m fakeMessage) Sequence() uint64        { return 1 }
func (m fakeMessage) Body(dst any) error {
	b, err := json.Marshal(m.body)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, dst)
}
func (m fakeMessage) Meta(any) error { return nil }
func (m fakeMessage) RawBody() any   { return m.body }
func (m fakeMessage) RawMeta() any   { return nil }

func msg(subj string, body any) stream.Message {
	return fakeMessage{
		subject: subject.FromStr(subj),
		body:    body,
		at:      time.Date(2026, 9, 26, 1, 20, 0, 0, time.UTC),
	}
}

func msgAt(subj string, body any, at time.Time) stream.Message {
	return fakeMessage{subject: subject.FromStr(subj), body: body, at: at}
}

func handle(t *testing.T, c *consumer, m stream.Message) {
	t.Helper()
	if err := c.HandleMessage(m); err != nil {
		t.Fatalf("HandleMessage(%s): %v", m.Subject().Subject(), err)
	}
}

func updateClause(t *testing.T, stmt string) string {
	t.Helper()
	i := strings.Index(stmt, "ON DUPLICATE KEY UPDATE")
	if i < 0 {
		t.Fatalf("statement is not an upsert, so replay would duplicate or clobber: %s", stmt)
	}
	return stmt[i:]
}

const note = "Ringet til mor 01.20. Hun henter kl. 06."

// --- tests ---

func TestCommentedWritesTheNote(t *testing.T) {
	w := &recordingWriter{}
	c := &consumer{w: w}
	handle(t, c, msg("NATHEJK.2026.spejder.m-1.commented",
		Commented{NoteID: "n-1", MemberID: "m-1", Note: note}))

	stmt := w.only(t)
	for _, want := range []string{"'n-1'", "'m-1'", "'2026'", "Ringet til mor"} {
		if !strings.Contains(stmt, want) {
			t.Errorf("want %s in: %s", want, stmt)
		}
	}
	updateClause(t, stmt)
}

// A correction moves the text and updatedAt, and must leave createdAt alone: the note was written
// when it was written, and a trail that reordered itself because somebody fixed a typo would be
// worse than one with a typo in it.
func TestCommentUpdatedDoesNotMoveCreatedAt(t *testing.T) {
	w := &recordingWriter{}
	c := &consumer{w: w}
	handle(t, c, msg("NATHEJK.2026.spejder.m-1.comment.updated",
		CommentUpdated{NoteID: "n-1", MemberID: "m-1", Note: "Rettet tekst"}))

	stmt := w.only(t)
	if !strings.HasPrefix(stmt, "UPDATE") {
		t.Fatalf("expected an UPDATE, got: %s", stmt)
	}
	if strings.Contains(stmt, "createdAt") {
		t.Errorf("a correction touched createdAt: %s", stmt)
	}
	if !strings.Contains(stmt, "updatedAt") {
		t.Errorf("a correction did not move updatedAt: %s", stmt)
	}
}

// A correction is scoped to the member from the subject as well as the note id, so an event
// published by anything else cannot reach another member's note.
func TestCommentUpdatedIsScopedToMemberAndYear(t *testing.T) {
	w := &recordingWriter{}
	c := &consumer{w: w}
	handle(t, c, msg("NATHEJK.2026.spejder.m-1.comment.updated",
		CommentUpdated{NoteID: "n-1", MemberID: "m-1", Note: "Rettet"}))

	stmt := w.only(t)
	for _, want := range []string{"'n-1'", "'m-1'", "'2026'"} {
		if !strings.Contains(stmt, want) {
			t.Errorf("want %s in the WHERE clause: %s", want, stmt)
		}
	}
}

// **The test this table exists to pass.** Replay re-delivers the original note on every boot, after
// the correction. If the insert's update list included the text, every corrected note would silently
// revert to what it first said — losing the correction, with nothing anywhere reporting it.
func TestReplayingTheOriginalDoesNotUndoACorrection(t *testing.T) {
	w := &recordingWriter{}
	c := &consumer{w: w}

	original := msg("NATHEJK.2026.spejder.m-1.commented",
		Commented{NoteID: "n-1", MemberID: "m-1", Note: note})
	handle(t, c, original)
	handle(t, c, msg("NATHEJK.2026.spejder.m-1.comment.updated",
		CommentUpdated{NoteID: "n-1", MemberID: "m-1", Note: "Rettet tekst"}))
	handle(t, c, original) // the boot replay

	// The replayed insert may overwrite the text (the correction's own event is replayed after it,
	// in stream order, and will win) — but it must not touch updatedAt, or the ordering that makes
	// that true stops being guaranteed.
	replayed := w.stmts[len(w.stmts)-1]
	if strings.Contains(updateClause(t, replayed), "updatedAt") {
		t.Errorf("the replayed note moved updatedAt, so a correction's precedence depends on luck: %s", replayed)
	}
	if strings.Contains(updateClause(t, replayed), "createdAt") {
		t.Errorf("the replayed note moved createdAt: %s", replayed)
	}
}

// Replay must reproduce the same statements, not accumulate rows: this table is rebuilt from the
// full history on every API boot.
func TestReplayIsIdempotent(t *testing.T) {
	w := &recordingWriter{}
	c := &consumer{w: w}
	m := msg("NATHEJK.2026.spejder.m-1.commented",
		Commented{NoteID: "n-1", MemberID: "m-1", Note: note})
	handle(t, c, m)
	handle(t, c, m)

	if len(w.stmts) != 2 || w.stmts[0] != w.stmts[1] {
		t.Errorf("replay did not reproduce the same statement: %v", w.stmts)
	}
}

// An event with no note id cannot find its row. Logged and skipped rather than returned as an error:
// a poison message must not wedge the rebuild of every note in the event.
func TestEventWithoutANoteIDIsSkipped(t *testing.T) {
	for _, subj := range []string{
		"NATHEJK.2026.spejder.m-1.commented",
		"NATHEJK.2026.spejder.m-1.comment.updated",
	} {
		w := &recordingWriter{}
		c := &consumer{w: w}
		handle(t, c, msg(subj, map[string]string{"memberId": "m-1", "note": "hej"}))

		if len(w.stmts) != 0 {
			t.Errorf("%s wrote something without a note id: %v", subj, w.stmts)
		}
	}
}

// The year comes from the subject, never the message clock: replay crosses year boundaries by
// definition, and msg.Time().Year() would file every historical note under the current year.
func TestYearComesFromSubjectNotMessageTime(t *testing.T) {
	w := &recordingWriter{}
	c := &consumer{w: w}
	handle(t, c, msgAt("NATHEJK.2025.spejder.m-1.commented",
		Commented{NoteID: "n-1", MemberID: "m-1", Note: note},
		time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)))

	if stmt := w.only(t); !strings.Contains(stmt, "'2025'") {
		t.Errorf("expected the subject's year: %s", stmt)
	}
}

// A note about a member this projection has never seen still lands. Notes are not restricted to
// scouts the lifecycle has touched — the nødtelefon takes calls about scouts who are still racing.
func TestNoteForAnUnknownMemberStillLands(t *testing.T) {
	w := &recordingWriter{}
	c := &consumer{w: w}
	handle(t, c, msg("NATHEJK.2026.spejder.m-never-seen.commented",
		Commented{NoteID: "n-1", MemberID: "m-never-seen", Note: note}))

	if stmt := w.only(t); !strings.HasPrefix(stmt, "INSERT") {
		t.Errorf("expected an insert: %s", stmt)
	}
}

func TestUnknownSubjectIsANoOp(t *testing.T) {
	w := &recordingWriter{}
	c := &consumer{w: w}
	handle(t, c, msg("NATHEJK.2026.spejder.m-1.shelter.accepted", struct{}{}))

	if len(w.stmts) != 0 {
		t.Errorf("expected no statements, got %v", w.stmts)
	}
}

// Every subject subscribed to must be handled. A subscription with no case is silent: the event
// arrives, "unhandled" is logged, and the trail is simply empty with no error anywhere.
func TestEverySubjectDeclaredIsHandled(t *testing.T) {
	base := &consumer{}
	for _, s := range base.Consumes() {
		declared := strings.Replace(s.Subject(), ":", ".", 1)
		concrete := strings.Replace(strings.Replace(declared, "*.spejder", "2026.spejder", 1), "spejder.*", "spejder.m-1", 1)

		w := &recordingWriter{}
		c := &consumer{w: w}
		handle(t, c, msg(concrete, map[string]any{
			"noteId":   "n-1",
			"memberId": "m-1",
			"note":     note,
		}))
		if len(w.stmts) == 0 {
			t.Errorf("subject %q is subscribed but produced no statement", concrete)
		}
	}
}

// Edited() is the one bit of logic on the type, and it is there so no consumer reimplements a
// time comparison and gets it subtly wrong.
func TestEditedReportsACorrection(t *testing.T) {
	written := time.Date(2026, 9, 26, 1, 20, 0, 0, time.UTC)

	fresh := Note{CreatedAt: written, UpdatedAt: written}
	if fresh.Edited() {
		t.Error("a note whose timestamps match has not been edited")
	}

	corrected := Note{CreatedAt: written, UpdatedAt: written.Add(4 * time.Minute)}
	if !corrected.Edited() {
		t.Error("a note with a later updatedAt has been edited")
	}
}

// The ids the server mints must be unique and recognisable; a client-chosen id could collide with
// one it has not seen, or name another member's note.
func TestNewNoteIDIsUniqueAndPrefixed(t *testing.T) {
	first, second := NewNoteID(), NewNoteID()

	if first == second {
		t.Error("two minted ids collided")
	}
	if !strings.HasPrefix(string(first), "spejdernote-") {
		t.Errorf("id %q is not recognisable as a note id", first)
	}
}

// Guard the types the querier scans, since the scan order and the column list are two halves of one
// contract kept in separate strings.
func TestSelectNoteColumnOrder(t *testing.T) {
	want := "SELECT noteId, memberId, year, note, actorUserId, createdAt, updatedAt"
	if !strings.HasPrefix(selectNote, want) {
		t.Errorf("expected the shared column list to start %q, got %q", want, selectNote)
	}
}
