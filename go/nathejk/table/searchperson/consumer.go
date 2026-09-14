package searchperson

import (
	"fmt"
	"strings"

	"github.com/jrgensen/cqrs"
	"github.com/nathejk/shared-go/messages"
)

// Consumes declares the subjects this projection folds in.
//
// Only events that carry identity — a name or a phone number. Notably absent is
// `bandit.*.armNumber.assigned`, which both the senior and personnel projections consume
// and which carries neither; subscribing to it because the neighbours do would add replay
// cost and a live signal for nothing.
//
// The `NATHEJK:` / `NATHEJK.` inconsistency across the existing consumers is cosmetic:
// subject.FromStr replaces the first colon with a dot, so both forms produce the same
// subscription. It is the Match patterns in HandleMessage that must use dots — see there.
func (t *Table) Consumes() []cqrs.Subject {
	return []cqrs.Subject{
		cqrs.SubjectFromStr("NATHEJK.*.spejder.*.updated"),
		cqrs.SubjectFromStr("NATHEJK.*.spejder.*.reassigned"),
	}
}

// HandleMessage folds one event into the index.
//
// Every branch is idempotent, as the cqrs.Consumer contract requires: the writes are
// upserts and absolute assignments, never increments, so a replay lands on the same row
// content whatever order it happens in.
//
// Match patterns are spelled with dots and never with a colon. subject.FromStr normalized
// the subject's first separator to a dot before the mux ever routed it, whereas
// Subject.Match escapes `.` and leaves `:` alone — so `Match("NATHEJK:…")` matches nothing
// at all, silently. Match is also case-insensitive, which is why the neighbouring
// consumers get away with lowercase patterns.
func (t *Table) HandleMessage(msg cqrs.Message) error {
	switch {
	case msg.Subject().Match("NATHEJK.*.spejder.*.updated"):
		return t.handleSpejderUpdated(msg)
	case msg.Subject().Match("NATHEJK.*.spejder.*.reassigned"):
		return t.handleSpejderReassigned(msg)
	default:
		// Not an error. The mux may hand over a subject this projection does not care
		// about, and failing would dead-letter somebody else's event.
		return nil
	}
}

func (t *Table) handleSpejderUpdated(msg cqrs.Message) error {
	var body messages.NathejkScoutUpdated
	if err := msg.Body(&body); err != nil {
		return err
	}

	// The same payload decoded a second time, for teamId alone.
	//
	// NathejkScoutUpdated does not carry a team — the roster's own projection leaves
	// teamId untouched on this event, so that a pre-race reassignment is the single way a
	// member's team changes. But the event as actually published also carries the legacy
	// added-shape's fields, and the spejder projection reads them the same way. Without
	// this a scout would have no team until they were reassigned, and every result row
	// would be missing the one piece of context that identifies them.
	var legacy messages.NathejkMemberAdded
	if err := msg.Body(&legacy); err != nil {
		return err
	}

	id := string(body.MemberID)
	if id == "" {
		id = string(legacy.MemberID)
	}
	if id == "" {
		// From the subject, which is the more trustworthy source anyway: the broker
		// matched on it. A row with an empty id would collide with every other
		// id-less row of its kind.
		id = subjectEntityID(msg.Subject())
	}
	if id == "" {
		return fmt.Errorf("searchperson: spejder.updated with no memberId")
	}

	return t.upsert(person{
		Kind:        KindSpejder,
		ID:          id,
		Year:        subjectYear(msg.Subject()),
		TeamID:      string(legacy.TeamID),
		Name:        body.Name,
		Email:       string(body.Email),
		Phone:       string(body.Phone),
		PhoneParent: string(body.PhoneContact),
		At:          datetime(msg.Time()),
	})
}

// handleSpejderReassigned follows a member to their new team before the race.
//
// teamId only. The reassignment says nothing about who the member is, and overwriting
// name or phone from it would blank them.
func (t *Table) handleSpejderReassigned(msg cqrs.Message) error {
	var body messages.NathejkMemberReassigned
	if err := msg.Body(&body); err != nil {
		return err
	}
	if body.MemberID == "" || body.ToTeamID == "" {
		// A reassignment that names no destination cannot be applied, and is not worth
		// dead-lettering the stream over.
		return nil
	}

	// Absolute, not relative, so a replay is a no-op rather than a second move. Scoped by
	// year as well as id because member ids are not year-scoped: a reassignment in one
	// season would otherwise rewrite the same member's row in another.
	return t.w.Consume(fmt.Sprintf(
		"UPDATE search_person SET teamId=%s, updatedAt=%s WHERE kind=%s AND id=%s AND year=%s",
		quote(truncate(string(body.ToTeamID), 99)),
		datetime(msg.Time()),
		quote(string(KindSpejder)),
		quote(truncate(string(body.MemberID), 99)),
		quote(subjectYear(msg.Subject())),
	))
}

// person is one row on its way into the table.
//
// A struct rather than a long argument list because the six sources fill in different
// subsets of it, and a positional call would eventually swap two same-typed fields —
// phone for phoneParent being the one that would go unnoticed and matter most.
type person struct {
	Kind        Kind
	ID          string
	Year        string
	TeamID      string
	Name        string
	Email       string
	Phone       string
	PhoneParent string
	At          string
}

// upsert writes a person, replacing the identity fields if the row already exists.
//
// ON DUPLICATE KEY UPDATE rather than INSERT IGNORE: a replay has to be able to correct a
// row written by an older version of this handler. Ignoring would freeze the index at
// whatever the first code to see the event produced.
func (t *Table) upsert(p person) error {
	// teamId is protected, unlike everything else here. The events that carry a person's
	// details do not always carry their team, and blanking it would cost the row the
	// context that identifies the person — "Anders Andersen" with no patrol is barely a
	// search result. An update that knows the team still wins; one that does not leaves it
	// alone.
	//
	// The other columns are overwritten unconditionally, because for those the event *is*
	// the source of truth: a scout who removed their phone number should stop being
	// findable by it.
	set := []string{
		"kind=" + quote(string(p.Kind)),
		"id=" + quote(truncate(p.ID, 99)),
		"year=" + quote(truncate(p.Year, 99)),
		"teamId=" + quote(truncate(p.TeamID, 99)),
		"name=" + quote(truncate(p.Name, 199)),
		"email=" + quote(truncate(p.Email, 199)),
		"phone=" + quote(truncate(p.Phone, 99)),
		"phoneNormalized=" + quote(truncate(normalizePhone(p.Phone), 31)),
		"phoneParent=" + quote(truncate(p.PhoneParent, 99)),
		"phoneParentNormalized=" + quote(truncate(normalizePhone(p.PhoneParent), 31)),
		// A person whose details arrive again is on a roster, so they are not udmeldt.
		// Written explicitly rather than left alone, or a member removed and re-added
		// would stay flagged forever.
		"deleted=0",
		"updatedAt=" + p.At,
	}
	update := []string{
		"name=VALUES(name)",
		"email=VALUES(email)",
		"phone=VALUES(phone)",
		"phoneNormalized=VALUES(phoneNormalized)",
		"phoneParent=VALUES(phoneParent)",
		"phoneParentNormalized=VALUES(phoneParentNormalized)",
		"teamId=IF(VALUES(teamId)='', teamId, VALUES(teamId))",
		"deleted=0",
		"updatedAt=VALUES(updatedAt)",
	}

	// One statement, because cqrs.Writer.Consume takes exactly one.
	return t.w.Consume(fmt.Sprintf(
		"INSERT INTO search_person SET %s ON DUPLICATE KEY UPDATE %s",
		strings.Join(set, ", "), strings.Join(update, ", "),
	))
}

// subjectYear is the second subject token.
//
// From the subject and never from msg.Time(): a replay crosses year boundaries by
// definition, since that is how this table is built in the first place.
func subjectYear(s cqrs.Subject) string {
	return subjectPart(s, 1)
}

// subjectEntityID is the fourth token — the memberId, userId or teamId the event concerns.
func subjectEntityID(s cqrs.Subject) string {
	return subjectPart(s, 3)
}

// subjectPart indexes the subject defensively.
//
// Parts() splits on dots and makes no promise about length, so a malformed subject would
// panic a projection mid-replay and take the API's boot with it. An empty string instead
// lets the caller decide, which it does: an id-less event is refused, a year-less one is
// written under an empty year where it is visibly wrong rather than invisibly absent.
func subjectPart(s cqrs.Subject, i int) string {
	parts := s.Parts()
	if i >= len(parts) {
		return ""
	}
	return parts[i]
}
