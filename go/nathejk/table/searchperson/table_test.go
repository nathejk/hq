package searchperson

import (
	"strings"
	"testing"

	"github.com/jrgensen/cqrs"
	"github.com/jrgensen/cqrs/cqrstest"
	"github.com/nathejk/shared-go/messages"
)

func newTable(t *testing.T) (*Table, *cqrstest.Writer) {
	t.Helper()
	w := &cqrstest.Writer{}
	table, err := New(w, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	// The schema statement is not interesting to the assertions that follow.
	w.Reset()
	return table, w
}

func message(t *testing.T, subj string, body any) cqrs.Message {
	t.Helper()
	msg := cqrstest.NewMessage(cqrs.SubjectFromStr(subj))
	if err := msg.SetBody(body); err != nil {
		t.Fatalf("SetBody: %v", err)
	}
	return msg
}

func TestNewCreatesSchema(t *testing.T) {
	w := &cqrstest.Writer{}
	if _, err := New(w, nil); err != nil {
		t.Fatalf("New: %v", err)
	}
	if len(w.Statements) != 1 || !strings.Contains(w.Statements[0], "CREATE TABLE IF NOT EXISTS search_person") {
		t.Fatalf("expected the schema to be created, got %v", w.Statements)
	}
	// kind belongs in the key because the id spaces do not merge. Without it a team's
	// contact person and a member sharing an id would silently overwrite each other.
	if !strings.Contains(w.Statements[0], "PRIMARY KEY (kind, id, year)") {
		t.Error("the primary key must include kind, or two id spaces collide")
	}
	// The whole justification for this projection over a query-time UNION.
	for _, idx := range []string{"idx_search_phone", "idx_search_phone_parent"} {
		if !strings.Contains(w.Statements[0], idx) {
			t.Errorf("missing %s: a phone lookup would scan", idx)
		}
	}
}

func TestNewRequiresWriter(t *testing.T) {
	if _, err := New(nil, nil); err == nil {
		t.Fatal("expected an error without a Writer")
	}
}

// A subject this projection subscribes with must be matched by the pattern its handler
// switches on. These are two independent strings that have to agree, and nothing at
// runtime would report it if they stopped agreeing: the projection would simply never
// write a row, and an empty index looks exactly like an event with no participants.
func TestConsumedSubjectsReachAHandler(t *testing.T) {
	table, w := newTable(t)
	for _, subj := range table.Consumes() {
		concrete := strings.Replace(subj.Subject(), "*", "2026", 1)
		concrete = strings.Replace(concrete, "*", "member-1", 1)

		w.Reset()
		msg := message(t, concrete, map[string]any{"memberId": "member-1", "toTeamId": "team-1"})
		if err := table.HandleMessage(msg); err != nil {
			t.Fatalf("HandleMessage(%s): %v", concrete, err)
		}
		if len(w.Statements) == 0 {
			t.Errorf("subscribed to %q but no handler matched it", concrete)
		}
	}
}

// The trap this package's comments warn about, pinned. subject.FromStr normalizes the
// first colon to a dot, so a legacy-form subject must still reach a handler; a Match
// pattern written with a colon would match nothing.
func TestLegacyColonSubjectStillMatches(t *testing.T) {
	table, w := newTable(t)
	msg := message(t, "NATHEJK:2026.spejder.member-1.updated", messages.NathejkScoutUpdated{
		MemberID: "member-1",
		Name:     "Anders Andersen",
	})
	if err := table.HandleMessage(msg); err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if len(w.Statements) != 1 {
		t.Fatalf("legacy colon subject did not reach a handler, got %v", w.Statements)
	}
	if !strings.Contains(w.Last(), "year='2026'") {
		t.Errorf("year not taken from the normalized subject: %s", w.Last())
	}
}

func TestSpejderUpdatedWritesARow(t *testing.T) {
	table, w := newTable(t)
	msg := message(t, "NATHEJK.2026.spejder.member-1.updated", map[string]any{
		"memberId":     "member-1",
		"teamId":       "team-42",
		"name":         "Anders Andersen",
		"mail":         "anders@example.dk",
		"phone":        "+45 12 34 56 78",
		"phoneContact": "87654321",
	})
	if err := table.HandleMessage(msg); err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if len(w.Statements) != 1 {
		t.Fatalf("expected one statement, got %v", w.Statements)
	}
	stmt := w.Last()
	for _, want := range []string{
		"INSERT INTO search_person SET",
		"kind='spejder'",
		"id='member-1'",
		"year='2026'",
		"teamId='team-42'",
		"name='Anders Andersen'",
		"phone='+45 12 34 56 78'",
		// The point of the projection: the searchable form is digits only.
		"phoneNormalized='12345678'",
		"phoneParent='87654321'",
		"phoneParentNormalized='87654321'",
		"ON DUPLICATE KEY UPDATE",
	} {
		if !strings.Contains(stmt, want) {
			t.Errorf("statement missing %q:\n%s", want, stmt)
		}
	}
}

// Normalization is the difference between finding somebody and not. All of these forms are
// in the source tables today, and they must all reduce to the same searchable digits.
func TestPhoneNormalizationIsFormatIndependent(t *testing.T) {
	for _, entered := range []string{
		"+45 12 34 56 78",
		"12 34 56 78",
		"12345678",
		"12-34-56-78",
		"(+45) 12345678",
		"0045 12345678",
		"tlf. 12 34 56 78",
	} {
		table, w := newTable(t)
		msg := message(t, "NATHEJK.2026.spejder.member-1.updated", map[string]any{
			"memberId": "member-1",
			"phone":    entered,
		})
		if err := table.HandleMessage(msg); err != nil {
			t.Fatalf("HandleMessage(%q): %v", entered, err)
		}
		if !strings.Contains(w.Last(), "phoneNormalized='12345678'") {
			t.Errorf("%q did not normalize to 12345678:\n%s", entered, w.Last())
		}
		// The as-entered form survives, because it is what an operator reads back.
		if !strings.Contains(w.Last(), "phone="+quote(entered)) {
			t.Errorf("%q was not stored as entered:\n%s", entered, w.Last())
		}
	}
}

// The country-code strip must not eat a real number. 45 is a genuine Danish prefix, so an
// 8-digit number beginning with it belongs to somebody — stripping it would make every
// 45-prefixed subscriber in the event silently unfindable.
func TestNormalizationKeeps45PrefixedNationalNumbers(t *testing.T) {
	cases := map[string]string{
		"45123456":     "45123456",   // a real number, left alone
		"+45 45123456": "45123456",   // the same number, internationally
		"4512345678":   "12345678",   // country code plus eight digits
		"":             "",           // nothing is nothing, not a match-everything
		"ukendt":       "",           // free text with no digits at all
		"123456":       "123456",     // a partial, for prefix search
		"+47 12345678": "4712345678", // not Danish: left as dialled rather than mangled
	}
	for in, want := range cases {
		if got := normalizePhone(in); got != want {
			t.Errorf("normalizePhone(%q) = %q, want %q", in, got, want)
		}
	}
}

// A replay re-delivers every event on every boot, so the same event twice must produce
// one row. Asserted on the statement rather than a database because the upsert is the
// mechanism: an INSERT IGNORE or a plain INSERT would fail this.
func TestSpejderUpdatedIsIdempotent(t *testing.T) {
	table, w := newTable(t)
	body := map[string]any{"memberId": "member-1", "name": "Anders Andersen", "phone": "12345678"}

	msg := message(t, "NATHEJK.2026.spejder.member-1.updated", body)
	if err := table.HandleMessage(msg); err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	first := w.Last()
	if err := table.HandleMessage(msg); err != nil {
		t.Fatalf("HandleMessage (replay): %v", err)
	}
	if second := w.Last(); second != first {
		t.Errorf("replay produced a different statement:\n%s\n%s", first, second)
	}
	if !strings.Contains(first, "name=VALUES(name)") {
		t.Error("a replay must be able to correct the row, so the update list must carry name")
	}
}

// The one field that is protected from being blanked. An event that carries a person's
// details but not their team is ordinary, and a result row with no patrol is barely a
// search result at all.
func TestUpdateWithoutTeamDoesNotBlankIt(t *testing.T) {
	table, w := newTable(t)
	msg := message(t, "NATHEJK.2026.spejder.member-1.updated", map[string]any{
		"memberId": "member-1",
		"name":     "Anders Andersen",
	})
	if err := table.HandleMessage(msg); err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if !strings.Contains(w.Last(), "teamId=IF(VALUES(teamId)='', teamId, VALUES(teamId))") {
		t.Errorf("teamId is not protected against a payload that omits it:\n%s", w.Last())
	}
}

func TestSpejderUpdatedWithoutMemberIDFallsBackToSubject(t *testing.T) {
	table, w := newTable(t)
	msg := message(t, "NATHEJK.2026.spejder.member-7.updated", map[string]any{
		"name": "Navnløs",
	})
	if err := table.HandleMessage(msg); err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if !strings.Contains(w.Last(), "id='member-7'") {
		t.Errorf("id not taken from the subject:\n%s", w.Last())
	}
}

func TestSpejderReassignedMovesOnlyTheTeam(t *testing.T) {
	table, w := newTable(t)
	msg := message(t, "NATHEJK.2026.spejder.member-1.reassigned", messages.NathejkMemberReassigned{
		MemberID:   "member-1",
		FromTeamID: "team-1",
		ToTeamID:   "team-2",
	})
	if err := table.HandleMessage(msg); err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	stmt := w.Last()
	if !strings.HasPrefix(stmt, "UPDATE search_person SET teamId='team-2'") {
		t.Errorf("expected a teamId-only update, got:\n%s", stmt)
	}
	for _, forbidden := range []string{"name=", "phone="} {
		if strings.Contains(stmt, forbidden) {
			t.Errorf("a reassignment must not touch %s:\n%s", forbidden, stmt)
		}
	}
	// Year-scoped because member ids are not: without it, a reassignment in one season
	// would rewrite the same member's row in another.
	if !strings.Contains(stmt, "year='2026'") {
		t.Errorf("the update is not year-scoped:\n%s", stmt)
	}
}

func TestSpejderReassignedWithoutDestinationIsIgnored(t *testing.T) {
	table, w := newTable(t)
	msg := message(t, "NATHEJK.2026.spejder.member-1.reassigned", messages.NathejkMemberReassigned{
		MemberID: "member-1",
	})
	if err := table.HandleMessage(msg); err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if len(w.Statements) != 0 {
		t.Errorf("expected no write for a destination-less reassignment, got %v", w.Statements)
	}
}

// An unrelated subject must not be an error: the mux hands over what it routes, and
// failing here would dead-letter another projection's event.
func TestUnhandledSubjectIsNotAnError(t *testing.T) {
	table, w := newTable(t)
	msg := message(t, "NATHEJK.2026.klan.team-1.signedup", map[string]any{})
	if err := table.HandleMessage(msg); err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if len(w.Statements) != 0 {
		t.Errorf("expected no write, got %v", w.Statements)
	}
}

// Names are user input and end up inside a string literal, so the quoting has to hold.
func TestQuotingSurvivesAwkwardNames(t *testing.T) {
	table, w := newTable(t)
	msg := message(t, "NATHEJK.2026.spejder.member-1.updated", map[string]any{
		"memberId": "member-1",
		"name":     `O'Brien \ "Bobby" ` + "\n",
	})
	if err := table.HandleMessage(msg); err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	stmt := w.Last()
	if strings.Contains(stmt, "\n") {
		t.Errorf("a newline reached the statement unescaped:\n%q", stmt)
	}
	if !strings.Contains(stmt, `O\'Brien`) {
		t.Errorf("apostrophe not escaped:\n%s", stmt)
	}
}

// A malformed subject must not panic a projection mid-replay: that would take the API's
// boot down with it, and the cause would be one bad message years ago.
func TestShortSubjectDoesNotPanic(t *testing.T) {
	table, _ := newTable(t)
	msg := message(t, "NATHEJK.2026", map[string]any{})
	if err := table.HandleMessage(msg); err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
}

func TestTruncateClipsToColumnWidth(t *testing.T) {
	if got := truncate(strings.Repeat("a", 250), 199); len(got) != 199 {
		t.Errorf("got %d bytes, want 199", len(got))
	}
	if got := truncate("kort", 199); got != "kort" {
		t.Errorf("got %q, want it untouched", got)
	}
}
