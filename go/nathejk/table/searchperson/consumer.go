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
		cqrs.SubjectFromStr("NATHEJK.*.senior.*.updated"),
		cqrs.SubjectFromStr("NATHEJK.*.gøgler.*.signedup"),
		cqrs.SubjectFromStr("NATHEJK.*.gøgler.*.updated"),
		cqrs.SubjectFromStr("NATHEJK.*.friend.*.signedup"),
		cqrs.SubjectFromStr("NATHEJK.*.friend.*.updated"),
		cqrs.SubjectFromStr("NATHEJK.*.crewmember.*.registered"),
		cqrs.SubjectFromStr("NATHEJK.*.crewmember.*.updated"),
		// A crew signup is what mints a crew member — the crewmember projection treats it
		// that way, with userId == teamId. Without it, crew are unfindable until somebody
		// edits them.
		cqrs.SubjectFromStr("NATHEJK.*.crew.*.signedup"),

		// The contact persons. Named entity by entity rather than with the wildcard
		// `NATHEJK:*.*.*.signedup` that the signup projection uses, which matters: a
		// wildcard in the *entity* position is what makes live.EntitySet.Exhaustive false,
		// so naming them keeps the advertised token set complete and lets the SPA go on
		// warning about dependencies nothing can satisfy. We know which team types have a
		// contact person, so there is nothing to discover at runtime.
		cqrs.SubjectFromStr("NATHEJK.*.patrulje.*.signedup"),
		cqrs.SubjectFromStr("NATHEJK.*.patrulje.*.updated"),
		cqrs.SubjectFromStr("NATHEJK.*.klan.*.signedup"),
		// klan.updated carries the klan's contact person, and *nothing* projects it today:
		// the klan table has no contact columns and signup only keeps what the signup event
		// carried. So a corrected klan contact currently exists nowhere in the read model,
		// and after this it exists here.
		cqrs.SubjectFromStr("NATHEJK.*.klan.*.updated"),
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
	case msg.Subject().Match("NATHEJK.*.senior.*.updated"):
		return t.handleSeniorUpdated(msg)
	case msg.Subject().Match("NATHEJK.*.gøgler.*.signedup"),
		msg.Subject().Match("NATHEJK.*.friend.*.signedup"),
		msg.Subject().Match("NATHEJK.*.crew.*.signedup"):
		return t.handleSignedUp(msg)
	case msg.Subject().Match("NATHEJK.*.gøgler.*.updated"),
		msg.Subject().Match("NATHEJK.*.friend.*.updated"):
		return t.handlePersonnelUpdated(msg)
	case msg.Subject().Match("NATHEJK.*.crewmember.*.registered"),
		msg.Subject().Match("NATHEJK.*.crewmember.*.updated"):
		return t.handleCrewMember(msg)
	case msg.Subject().Match("NATHEJK.*.patrulje.*.signedup"):
		return t.handleTeamSignedUp(msg, KindPatruljeContact)
	case msg.Subject().Match("NATHEJK.*.klan.*.signedup"):
		return t.handleTeamSignedUp(msg, KindKlanContact)
	case msg.Subject().Match("NATHEJK.*.patrulje.*.updated"):
		return t.handleTeamUpdated(msg, KindPatruljeContact)
	case msg.Subject().Match("NATHEJK.*.klan.*.updated"):
		return t.handleTeamUpdated(msg, KindKlanContact)
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

	id := firstNonEmpty(string(body.MemberID), string(legacy.MemberID), subjectEntityID(msg.Subject()))
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

// handleSeniorUpdated indexes a klan member.
//
// Same two-phase decode as the spejder branch, and for the same reason: the senior-updated
// shape carries the editable fields, the legacy added shape carries the team.
//
// Note there is no phoneParent here. Seniors are adults; the column stays empty rather than
// being filled with something that is not a guardian's number.
func (t *Table) handleSeniorUpdated(msg cqrs.Message) error {
	var body messages.NathejkSeniorUpdated
	if err := msg.Body(&body); err != nil {
		return err
	}
	var legacy messages.NathejkMemberAdded
	if err := msg.Body(&legacy); err != nil {
		return err
	}

	id := firstNonEmpty(string(body.MemberID), string(legacy.MemberID), subjectEntityID(msg.Subject()))
	if id == "" {
		return fmt.Errorf("searchperson: senior.updated with no memberId")
	}

	return t.upsert(person{
		Kind:   KindSenior,
		ID:     id,
		Year:   subjectYear(msg.Subject()),
		TeamID: string(legacy.TeamID),
		Name:   body.Name,
		Email:  string(body.Email),
		Phone:  string(body.Phone),
		At:     datetime(msg.Time()),
	})
}

// handleSignedUp indexes whoever just signed up, for the entity types whose signup *is* the
// person: gøgler, friend and crew.
//
// One handler for three subjects because the payload is the same shape and the only
// difference is the kind, which is read off the subject. The alternative — three near-identical
// handlers — is where a copy-paste error would put a friend's details under a gøgler's kind.
//
// This is emphatically **not** the branch that handles a patrulje or klan signup: those signups
// describe a *team*, and the person in them is a contact person, which is a different kind with
// a different key (task 176).
func (t *Table) handleSignedUp(msg cqrs.Message) error {
	var body messages.NathejkTeamSignedUp
	if err := msg.Body(&body); err != nil {
		return err
	}

	// The signup's teamId *is* the person's id for these types: a crew signup mints the crew
	// member with userId == teamId, and personnel are keyed the same way.
	id := firstNonEmpty(string(body.TeamID), subjectEntityID(msg.Subject()))
	if id == "" {
		// Not an error: an id-less signup cannot be indexed, but it is the signup pipeline's
		// problem, not something to dead-letter the replay over.
		return nil
	}

	kind := Kind(subjectPart(msg.Subject(), 2))
	if kind == "crew" {
		// Already the right token; named here only so the mapping is visible.
		kind = KindCrew
	}

	return t.upsert(person{
		Kind:  kind,
		ID:    id,
		Year:  subjectYear(msg.Subject()),
		Name:  body.Name,
		Email: string(body.Email),
		Phone: string(body.Phone),
		At:    datetime(msg.Time()),
	})
}

// handlePersonnelUpdated re-indexes a gøgler or friend after an edit.
//
// The kind comes from the subject rather than the body, which carries no notion of which
// population the person belongs to. That also means an update cannot move somebody between
// kinds — correct, since the subject is what the broker routed on.
func (t *Table) handlePersonnelUpdated(msg cqrs.Message) error {
	var body messages.NathejkPersonnelUpdated
	if err := msg.Body(&body); err != nil {
		return err
	}

	id := firstNonEmpty(string(body.UserID), subjectEntityID(msg.Subject()))
	if id == "" {
		return fmt.Errorf("searchperson: personnel updated with no userId")
	}

	return t.upsert(person{
		Kind:  Kind(subjectPart(msg.Subject(), 2)),
		ID:    id,
		Year:  subjectYear(msg.Subject()),
		Name:  body.Name,
		Email: string(body.Email),
		Phone: string(body.Phone),
		At:    datetime(msg.Time()),
	})
}

// handleCrewMember indexes a crew member, registered or edited.
//
// Both events are folded by one handler because both are upserts of the same three fields.
// The registered/updated distinction matters to the crewmember projection, which carries more
// columns; here it does not.
func (t *Table) handleCrewMember(msg cqrs.Message) error {
	var body messages.NathejkCrewMemberUpdated
	if err := msg.Body(&body); err != nil {
		return err
	}

	id := firstNonEmpty(string(body.UserID), subjectEntityID(msg.Subject()))
	if id == "" {
		return fmt.Errorf("searchperson: crewmember event with no userId")
	}

	return t.upsert(person{
		Kind:  KindCrew,
		ID:    id,
		Year:  subjectYear(msg.Subject()),
		Name:  body.Name,
		Email: string(body.Email),
		Phone: string(body.Phone),
		At:    datetime(msg.Time()),
	})
}

// handleTeamSignedUp indexes the person who submitted a team.
//
// The signup event's Name/Phone/Email describe a *human*, not the team: the team's own name
// arrives later on `updated`. That is exactly why this is its own kind rather than a field on
// the team — and why a klan contact is findable at all, since `klan` has nowhere to put them.
//
// Keyed by teamId, since a contact person has no id. `kind` in the primary key is what keeps
// that from colliding with a member who happens to share the string.
func (t *Table) handleTeamSignedUp(msg cqrs.Message, kind Kind) error {
	var body messages.NathejkTeamSignedUp
	if err := msg.Body(&body); err != nil {
		return err
	}

	teamID := firstNonEmpty(string(body.TeamID), subjectEntityID(msg.Subject()))
	if teamID == "" {
		return nil
	}

	// One phone, not two.
	//
	// `signup` holds the number in two columns, phonePending and phone, and it is tempting to
	// read that as two numbers. It is not: verification does `SET phone = phonePending`, so
	// they are one number in two *states*. Indexing it once therefore finds the contact
	// whether or not they ever verified — which is the behaviour wanted, since an unverified
	// contact still needs finding.
	//
	// A signup that names no human at all is not a contact person: a row with no name and no
	// number can be found by neither, so it would only pad the table.
	if body.Name == "" && body.Phone == "" && body.Email == "" {
		return nil
	}

	return t.upsert(person{
		Kind:   kind,
		ID:     teamID,
		Year:   subjectYear(msg.Subject()),
		TeamID: teamID,
		Name:   body.Name,
		Email:  string(body.Email),
		Phone:  string(body.Phone),
		At:     datetime(msg.Time()),
	})
}

// handleTeamUpdated follows a correction to a team's contact person.
//
// The contact fields on this event are prefixed (ContactName rather than Name) because the
// unprefixed ones describe the team. Reading the wrong pair would file the *patrol's* name as
// a person — a search that returned "Ørnene" as a human being.
func (t *Table) handleTeamUpdated(msg cqrs.Message, kind Kind) error {
	var body messages.NathejkTeamUpdated
	if err := msg.Body(&body); err != nil {
		return err
	}

	teamID := firstNonEmpty(string(body.TeamID), subjectEntityID(msg.Subject()))
	if teamID == "" {
		return nil
	}

	// A team update that names no contact person at all is not a contact person being
	// deleted; it is an event about something else on the team. Writing it would blank a
	// perfectly good phone number, and an operator would find nobody.
	if body.ContactName == "" && body.ContactPhone == "" && body.ContactEmail == "" {
		return nil
	}

	return t.upsert(person{
		Kind:   kind,
		ID:     teamID,
		Year:   subjectYear(msg.Subject()),
		TeamID: teamID,
		Name:   body.ContactName,
		Email:  string(body.ContactEmail),
		Phone:  string(body.ContactPhone),
		At:     datetime(msg.Time()),
	})
}

// firstNonEmpty returns the first non-empty string, for the body-then-subject fallback every
// handler needs.
func firstNonEmpty(vs ...string) string {
	for _, v := range vs {
		if v != "" {
			return v
		}
	}
	return ""
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
