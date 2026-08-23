package spejdernote

import "github.com/nathejk/shared-go/types"

// The note events, published on NATHEJK.{year}.spejder.{memberId}.{event}.
//
// # On the scout's subject, not a subject of their own
//
// A note is a fact about a member, so it goes on the member's subject. That is not only tidiness:
// the live signal's entity token is derived from the subject, so publishing here means notes arrive
// on the `spejder` token that every member screen already declares. A `spejdernote` entity would
// have been a new token, and every client would have had to add a dependency for it — the exact
// silent-no-op trap PRD 004 and task 037 record.
//
// # No sosId
//
// None of these carries a case id, for the same reason the lifecycle events do not (see
// spejderstatus/messages.go): the shelter publishes them knowing nothing about cases, and requiring
// one would either be a lie or force them to invent it. Notes stay off the case timeline entirely
// (PRD 008 §4) — the case card reads a scout's notes because it shows that scout.

// Actor is who wrote or corrected the note, resolved by the HTTP layer and passed in.
//
// Defined here rather than borrowed from another table package so this one keeps no dependency it
// does not need. Empty in practice until HQ has login (PRD 001 §6); an unsigned trail on race day
// is accepted (PRD 008 §5), and the field is recorded anyway so nothing changes when identity
// arrives.
type Actor struct {
	UserID types.UserID `json:"userId,omitempty"`
	Name   string       `json:"name,omitempty"`
}

// Commented records a new note.
//
// MemberID is carried on the body as well as being in the subject. The subject is what the stream
// routes on and the projection reads, so the two cannot disagree — but a consumer of the *body*
// alone (a future export, another service) should not have to parse a subject to learn who the note
// is about.
type Commented struct {
	NoteID   NoteID         `json:"noteId"`
	MemberID types.MemberID `json:"memberId"`
	Note     string         `json:"note"`
	Actor    Actor          `json:"actor"`
}

// CommentUpdated records a correction to an existing note.
//
// The whole text is carried rather than a diff: the projection holds current text and the stream
// holds history (PRD 008 §8), so each version has to be complete enough to stand alone. Replaying
// the third correction must not depend on having seen the first two.
//
// Editing is expected to be used for typos, and the UI nudges toward writing a new note instead —
// but nothing here enforces that, and nothing can: "fixing a typo" and "changing what it says" are
// the same operation to a machine. What keeps the trail honest is that every version stays in the
// stream.
type CommentUpdated struct {
	NoteID   NoteID         `json:"noteId"`
	MemberID types.MemberID `json:"memberId"`
	Note     string         `json:"note"`
	Actor    Actor          `json:"actor"`
}
