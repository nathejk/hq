package searchperson

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// PhoneRole says whose number matched.
//
// It exists because the answer is operationally load-bearing rather than decorative: an
// operator about to speak to somebody must know whether the number belongs to the scout, to
// their guardian, or to the adult who signed the team up. A hit that silently presented a
// mother's number as her child's would have the operator open with the wrong sentence.
type PhoneRole string

const (
	PhoneRoleNone    PhoneRole = ""
	PhoneRoleOwn     PhoneRole = "own"
	PhoneRoleParent  PhoneRole = "parent"
	PhoneRoleContact PhoneRole = "contact"
)

// StatusKind says which vocabulary a Result's Status is drawn from.
//
// Two different axes end up in one field, and a client that could not tell them apart would
// have to guess: `racing` is a fact about a person, `PAID` is a fact about their team. Naming
// the vocabulary keeps the UI honest without duplicating the status itself.
type StatusKind string

const (
	// StatusKindUnknown is an ordinary outcome, not an error: member statuses begin at
	// signup-time `registered`, and older years predate the spejderstatus table entirely.
	StatusKindUnknown StatusKind = ""

	// StatusKindMember is a lifecycle status from spejderstatus: racing, waiting, transit,
	// sheltered, reunited, released, finished, seated, registered.
	StatusKindMember StatusKind = "member"

	// StatusKindTeam is a team's signup status: NEW, HOLD, PAY, SEMIPAID, PAID, STARTED, OUT.
	StatusKindTeam StatusKind = "team"
)

// Result is one person as search returns them.
//
// Deliberately narrow: no address, no birthday, no notes. Those live on the detail pages a
// result links to, so search is a way to *find* a person rather than a way to export the
// population — which matters because this is the first endpoint in hq that returns contact
// details for minors aggregated across the whole event.
type Result struct {
	Kind Kind   `json:"kind"`
	ID   string `json:"id"`
	Year string `json:"year"`
	Name string `json:"name"`

	// Phone as entered, because that is what an operator reads back to a caller, and
	// PhoneRole to say whose it is.
	Phone     string    `json:"phone"`
	PhoneRole PhoneRole `json:"phoneRole"`
	Email     string    `json:"email,omitempty"`

	// Where they belong, resolved by join rather than stored: a patrol or klan name, a
	// personnel section, a crew section label.
	TeamID     string `json:"teamId,omitempty"`
	TeamName   string `json:"teamName,omitempty"`
	TeamNumber string `json:"teamNumber,omitempty"`

	Status     string     `json:"status,omitempty"`
	StatusKind StatusKind `json:"statusKind,omitempty"`

	// Removed reports that this person was taken off a roster: "udmeldt".
	//
	// A separate field from Status, and it must stay separate. Being removed before the
	// event and going home during it are different facts \u2014 "udmeldt" is not "released" \u2014
	// and collapsing them server-side would deny the UI the ability to say which.
	Removed bool `json:"removed"`
}

// The one SELECT, with every join search needs.
//
// Six LEFT JOINs, each of which may legitimately miss:
//
//   - patrulje / klan resolve a team name. Joined on teamId **alone**, not on teamId and
//     year: patrulje's own projection writes its year from msg.Time().Year() while
//     search_person takes the subject's year token. The two agree in practice, but teamId is
//     unique so there is no reason to depend on that.
//   - spejderstatus is where a scout's lifecycle status lives. Joined rather than projected so
//     PRD 006's state machine stays in one place \u2014 a copy here could disagree with it \u2014 and so
//     this projection needs no status subscriptions at all.
//   - personnel and crewmember supply context for the populations whose events carry no team.
//   - section turns a crew member's sectionSlug into something readable.
//
// LEFT throughout, and no COALESCE onto a default: a missing status must arrive as missing so
// the UI can say "ukendt" instead of asserting `registered`, which would be a lie.
const selectResult = `SELECT
		sp.kind, sp.id, sp.year, sp.name, sp.email,
		sp.phone, sp.phoneNormalized, sp.phoneParent, sp.phoneParentNormalized,
		sp.teamId, sp.deleted,
		COALESCE(p.name, ''), COALESCE(p.teamNumber, ''), COALESCE(p.signupStatus, ''),
		COALESCE(k.name, ''), COALESCE(k.signupStatus, ''),
		COALESCE(ss.status, ''),
		COALESCE(pe.klan, ''), COALESCE(pe.groupName, ''), COALESCE(pe.signupStatus, ''),
		COALESCE(se.label, ''), COALESCE(cm.groupName, '')
	FROM search_person sp
	LEFT JOIN patrulje p ON p.teamId = sp.teamId
	LEFT JOIN klan k ON k.teamId = sp.teamId
	LEFT JOIN spejderstatus ss ON ss.id = sp.id AND ss.year = sp.year
	LEFT JOIN personnel pe ON pe.userId = sp.id AND pe.year = sp.year
	LEFT JOIN crewmember cm ON cm.userId = sp.id AND cm.year = sp.year
	LEFT JOIN section se ON se.slug = cm.sectionSlug AND se.year = sp.year`

// row is the raw join, before the per-kind choices are made.
//
// The alternative was a SQL CASE per output column, which would have put the domain rule
// "a spejder's status comes from spejderstatus, a klan member's from their klan" inside a
// string. Deciding in Go keeps it testable and readable.
type row struct {
	kind                  Kind
	id                    string
	year                  string
	name                  string
	email                 string
	phone                 string
	phoneNormalized       string
	phoneParent           string
	phoneParentNormalized string
	teamID                string
	deleted               bool

	patruljeName   string
	patruljeNumber string
	patruljeStatus string
	klanName       string
	klanStatus     string
	memberStatus   string
	personnelKlan  string
	personnelGroup string
	personnelStat  string
	sectionLabel   string
	crewGroup      string
}

func scanRow(rows *sql.Rows) (row, error) {
	var r row
	err := rows.Scan(
		&r.kind, &r.id, &r.year, &r.name, &r.email,
		&r.phone, &r.phoneNormalized, &r.phoneParent, &r.phoneParentNormalized,
		&r.teamID, &r.deleted,
		&r.patruljeName, &r.patruljeNumber, &r.patruljeStatus,
		&r.klanName, &r.klanStatus,
		&r.memberStatus,
		&r.personnelKlan, &r.personnelGroup, &r.personnelStat,
		&r.sectionLabel, &r.crewGroup,
	)
	return r, err
}

// result turns one joined row into what a client sees, applying the per-kind rules.
//
// role is passed in rather than derived here: which number matched is a property of the query
// that found the row, not of the row itself.
func (r row) result(role PhoneRole) Result {
	out := Result{
		Kind:      r.kind,
		ID:        r.id,
		Year:      r.year,
		Name:      r.name,
		Email:     r.email,
		TeamID:    r.teamID,
		Removed:   r.deleted,
		Phone:     r.phone,
		PhoneRole: role,
	}
	if role == PhoneRoleParent {
		out.Phone = r.phoneParent
	}

	switch r.kind {
	case KindSpejder:
		out.TeamName, out.TeamNumber = r.patruljeName, r.patruljeNumber
		// The lifecycle status, which is the only one that describes the *person*. Falls
		// back to the team's signup status when there is no status row \u2014 which is the
		// ordinary state before the race \u2014 rather than to nothing at all.
		if r.memberStatus != "" {
			out.Status, out.StatusKind = r.memberStatus, StatusKindMember
		} else if r.patruljeStatus != "" {
			out.Status, out.StatusKind = r.patruljeStatus, StatusKindTeam
		}

	case KindSenior:
		out.TeamName = r.klanName
		if r.klanStatus != "" {
			out.Status, out.StatusKind = r.klanStatus, StatusKindTeam
		}

	case KindPatruljeContact:
		out.TeamName, out.TeamNumber = r.patruljeName, r.patruljeNumber
		if r.patruljeStatus != "" {
			out.Status, out.StatusKind = r.patruljeStatus, StatusKindTeam
		}

	case KindKlanContact:
		out.TeamName = r.klanName
		if r.klanStatus != "" {
			out.Status, out.StatusKind = r.klanStatus, StatusKindTeam
		}

	case KindGoegler, KindFriend:
		// A gøgler's klan is free text on the personnel row and is the more specific of the
		// two, so it wins where it is filled in.
		out.TeamName = firstNonEmpty(r.personnelKlan, r.personnelGroup)
		if r.personnelStat != "" {
			out.Status, out.StatusKind = r.personnelStat, StatusKindTeam
		}

	case KindCrew:
		// The section they were assigned to, which is the only context crew have. Neither
		// their events nor their table carry a status, so they have none — reported as
		// unknown rather than invented.
		out.TeamName = firstNonEmpty(r.sectionLabel, r.crewGroup)
	}

	return out
}

// query runs selectResult with a caller-supplied predicate.
//
// Unexported and predicate-shaped because the classification that builds those predicates is a
// separate concern (see search.go); this is the part that knows how to read the join.
//
// roleOf decides, per row, whose number matched — which cannot be known from the predicate
// alone: `phoneNormalized = ? OR phoneParentNormalized = ?` matches either way, and the
// difference is what the operator needs told.
func (t *Table) query(ctx context.Context, where string, args []any, roleOf func(row) PhoneRole, order string, limit int) ([]Result, error) {
	if t.r == nil {
		return nil, fmt.Errorf("searchperson: no Reader")
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	stmt := selectResult + "\n\tWHERE " + where
	if order != "" {
		stmt += "\n\tORDER BY " + order
	}
	if limit > 0 {
		stmt += fmt.Sprintf("\n\tLIMIT %d", limit)
	}

	rows, err := t.r.QueryContext(ctx, stmt, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Result{}
	for rows.Next() {
		r, err := scanRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r.result(roleOf(r)))
	}
	return out, rows.Err()
}

// Lookup returns one person by kind and id.
//
// Its purpose is verification rather than the UI: it exercises every join with a predicate
// simple enough that a failure is unambiguous.
func (t *Table) Lookup(ctx context.Context, year string, kind Kind, id string) (*Result, error) {
	results, err := t.query(ctx,
		"sp.kind = ? AND sp.id = ? AND sp.year = ?",
		[]any{string(kind), id, year},
		nameRole, "", 1)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, nil
	}
	return &results[0], nil
}
