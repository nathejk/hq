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
		// The probe body has to satisfy every handler's "is there anything here to index?"
		// guard, or this test would pass by matching a handler that then declined to write.
		msg := message(t, concrete, map[string]any{
			"memberId":    "member-1",
			"userId":      "member-1",
			"teamId":      "team-1",
			"toTeamId":    "team-1",
			"name":        "Prober Probersen",
			"contactName": "Prober Probersen",
			"phone":       "12345678",
		})
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
	// A scan, which is the `qr` entity and carries no person at all.
	msg := message(t, "NATHEJK.2026.qr.scan-1.scanned", map[string]any{})
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

// One round trip per source. Table-driven because the interesting part is that all the
// populations land in one table under the right kind — what would break silently is a
// copy-pasted handler filing a friend under a gøgler's kind.
func TestEachSourceProducesItsOwnKind(t *testing.T) {
	cases := []struct {
		name    string
		subject string
		body    map[string]any
		want    map[string]string
	}{{
		name:    "senior",
		subject: "NATHEJK.2026.senior.member-2.updated",
		body: map[string]any{
			"memberId": "member-2",
			"teamId":   "klan-9",
			"name":     "Bente Bentsen",
			"phone":    "22334455",
			"mail":     "bente@example.dk",
		},
		want: map[string]string{
			"kind":            "senior",
			"id":              "member-2",
			"teamId":          "klan-9",
			"name":            "Bente Bentsen",
			"phoneNormalized": "22334455",
		},
	}, {
		name:    "gøgler signup",
		subject: "NATHEJK.2026.gøgler.user-3.signedup",
		body: map[string]any{
			"teamId": "user-3",
			"name":   "Carl Carlsen",
			"phone":  "+45 33 44 55 66",
			"email":  "carl@example.dk",
		},
		want: map[string]string{
			"kind":            "gøgler",
			"id":              "user-3",
			"name":            "Carl Carlsen",
			"phoneNormalized": "33445566",
			"email":           "carl@example.dk",
		},
	}, {
		name:    "gøgler update",
		subject: "NATHEJK.2026.gøgler.user-3.updated",
		body:    map[string]any{"userId": "user-3", "name": "Carl Carlsen", "phone": "33445566"},
		want:    map[string]string{"kind": "gøgler", "id": "user-3"},
	}, {
		name:    "friend signup",
		subject: "NATHEJK.2026.friend.user-4.signedup",
		body:    map[string]any{"teamId": "user-4", "name": "Dorte Dortesen", "phone": "44556677"},
		want:    map[string]string{"kind": "friend", "id": "user-4"},
	}, {
		// The event that makes a crew signup into a crew member. Without this branch, crew
		// are unfindable until somebody happens to edit them.
		name:    "crew signup",
		subject: "NATHEJK.2026.crew.user-5.signedup",
		body:    map[string]any{"teamId": "user-5", "name": "Erik Eriksen", "phone": "55667788"},
		want:    map[string]string{"kind": "crew", "id": "user-5", "phoneNormalized": "55667788"},
	}, {
		name:    "crewmember registered",
		subject: "NATHEJK.2026.crewmember.user-6.registered",
		body:    map[string]any{"userId": "user-6", "name": "Frida Fridasen", "phone": "66778899"},
		want:    map[string]string{"kind": "crew", "id": "user-6", "name": "Frida Fridasen"},
	}, {
		name:    "crewmember updated",
		subject: "NATHEJK.2026.crewmember.user-6.updated",
		body:    map[string]any{"userId": "user-6", "name": "Frida Fridasen", "phone": "66778899"},
		want:    map[string]string{"kind": "crew", "id": "user-6"},
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			table, w := newTable(t)
			if err := table.HandleMessage(message(t, tc.subject, tc.body)); err != nil {
				t.Fatalf("HandleMessage: %v", err)
			}
			if len(w.Statements) != 1 {
				t.Fatalf("expected one statement, got %v", w.Statements)
			}
			for col, val := range tc.want {
				if want := col + "=" + quote(val); !strings.Contains(w.Last(), want) {
					t.Errorf("statement missing %q:\n%s", want, w.Last())
				}
			}
			if !strings.Contains(w.Last(), "year='2026'") {
				t.Errorf("year not taken from the subject:\n%s", w.Last())
			}
		})
	}
}

// A senior is an adult, so nothing may land in the guardian column. Filling it from some
// other field would make the UI annotate the hit as "forælders nummer" and mislead an
// operator about who is going to answer.
func TestSeniorHasNoParentPhone(t *testing.T) {
	table, w := newTable(t)
	msg := message(t, "NATHEJK.2026.senior.member-2.updated", map[string]any{
		"memberId": "member-2",
		"phone":    "22334455",
	})
	if err := table.HandleMessage(msg); err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if !strings.Contains(w.Last(), "phoneParent=''") {
		t.Errorf("a senior must have no guardian number:\n%s", w.Last())
	}
}

// A gøgler and a friend must not collide, and neither may be filed as "personnel" — that is
// the table's name, not an entity token, and a client depending on it would wait forever.
func TestPersonnelKindComesFromTheSubject(t *testing.T) {
	for _, kind := range []string{"gøgler", "friend"} {
		table, w := newTable(t)
		msg := message(t, "NATHEJK.2026."+kind+".user-1.updated", map[string]any{
			"userId": "user-1",
			"name":   "Samme Person",
		})
		if err := table.HandleMessage(msg); err != nil {
			t.Fatalf("HandleMessage: %v", err)
		}
		if !strings.Contains(w.Last(), "kind="+quote(kind)) {
			t.Errorf("expected kind %q:\n%s", kind, w.Last())
		}
		if strings.Contains(w.Last(), "kind='personnel'") {
			t.Error("filed under the table's name rather than the entity token")
		}
	}
}

// Nothing subscribes to bandit. A bandit is a senior with an arm number, and the only
// bandit-entity event carries neither a name nor a number — indexing it would produce empty
// rows and advertise a live dependency that never fires.
func TestBanditIsNotAPopulation(t *testing.T) {
	table, w := newTable(t)
	for _, subj := range table.Consumes() {
		if strings.Contains(subj.Subject(), "bandit") {
			t.Fatalf("subscribed to a bandit subject: %s", subj.Subject())
		}
	}
	msg := message(t, "NATHEJK.2026.bandit.member-2.armNumber.assigned", map[string]any{"armNumber": "42"})
	if err := table.HandleMessage(msg); err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if len(w.Statements) != 0 {
		t.Errorf("expected no write for an arm number, got %v", w.Statements)
	}
}

// A signup whose body omits the id still gets one from the subject, which is the more
// trustworthy source anyway: the broker matched on it. Without this fallback a real signup
// with a body-shape we did not anticipate would be silently unfindable.
func TestSignupFallsBackToTheSubjectID(t *testing.T) {
	table, w := newTable(t)
	if err := table.HandleMessage(message(t, "NATHEJK.2026.crew.user-9.signedup", map[string]any{
		"name":  "Ingen Id I Body",
		"phone": "99887766",
	})); err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if !strings.Contains(w.Last(), "id='user-9'") {
		t.Errorf("id not taken from the subject:\n%s", w.Last())
	}
}

// The contact persons — the population with no prior art. A patrulje's contact lives in three
// columns on the team; a klan's lives nowhere at all, which is what makes this the most
// valuable part of the projection.
func TestContactPersonSources(t *testing.T) {
	cases := []struct {
		name    string
		subject string
		body    map[string]any
		want    map[string]string
	}{{
		name:    "patrulje signup",
		subject: "NATHEJK.2026.patrulje.team-42.signedup",
		body: map[string]any{
			"teamId": "team-42",
			"name":   "Gitte Gittesen",
			"phone":  "11223344",
			"email":  "gitte@example.dk",
		},
		want: map[string]string{
			"kind":            "patruljekontakt",
			"id":              "team-42",
			"teamId":          "team-42",
			"name":            "Gitte Gittesen",
			"phoneNormalized": "11223344",
		},
	}, {
		// The case that motivates the whole projection: a klan contact person exists in no
		// table today, because klan has no contact columns and signup keeps only what the
		// signup event carried.
		name:    "klan signup",
		subject: "NATHEJK.2026.klan.klan-9.signedup",
		body: map[string]any{
			"teamId": "klan-9",
			"name":   "Henrik Henriksen",
			"phone":  "+45 22 33 44 55",
		},
		want: map[string]string{
			"kind":            "klankontakt",
			"id":              "klan-9",
			"name":            "Henrik Henriksen",
			"phoneNormalized": "22334455",
		},
	}, {
		// The prefixed fields. Reading the unprefixed ones would file the *patrol's* name as
		// a human being.
		name:    "patrulje update",
		subject: "NATHEJK.2026.patrulje.team-42.updated",
		body: map[string]any{
			"teamId":       "team-42",
			"name":         "Ørnene",
			"contactName":  "Gitte Gittesen",
			"contactPhone": "11223344",
			"contactEmail": "gitte@example.dk",
		},
		want: map[string]string{
			"kind": "patruljekontakt",
			"name": "Gitte Gittesen",
		},
	}, {
		// klan.updated carries a contact person that nothing else in the read model keeps.
		name:    "klan update",
		subject: "NATHEJK.2026.klan.klan-9.updated",
		body: map[string]any{
			"teamId":       "klan-9",
			"name":         "Klan Nordvest",
			"contactName":  "Henrik Henriksen",
			"contactPhone": "22334455",
		},
		want: map[string]string{
			"kind":            "klankontakt",
			"name":            "Henrik Henriksen",
			"phoneNormalized": "22334455",
		},
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			table, w := newTable(t)
			if err := table.HandleMessage(message(t, tc.subject, tc.body)); err != nil {
				t.Fatalf("HandleMessage: %v", err)
			}
			if len(w.Statements) != 1 {
				t.Fatalf("expected one statement, got %v", w.Statements)
			}
			for col, val := range tc.want {
				if want := col + "=" + quote(val); !strings.Contains(w.Last(), want) {
					t.Errorf("statement missing %q:\n%s", want, w.Last())
				}
			}
		})
	}
}

// A team update about something other than the contact person must not blank them. This is the
// difference between "the contact changed" and "this event was about the team's liga", and
// getting it wrong loses a phone number an operator needs.
func TestTeamUpdateWithoutContactFieldsIsIgnored(t *testing.T) {
	table, w := newTable(t)
	msg := message(t, "NATHEJK.2026.patrulje.team-42.updated", map[string]any{
		"teamId":    "team-42",
		"name":      "Ørnene",
		"groupName": "Hvidovre Gruppe",
	})
	if err := table.HandleMessage(msg); err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if len(w.Statements) != 0 {
		t.Errorf("a contact-less team update must not write: %v", w.Statements)
	}
}

// A contact person and a member may share an id string — different id spaces, no coordination
// between them. kind in the primary key is what stops one silently overwriting the other.
func TestContactAndMemberWithTheSameIDDoNotCollide(t *testing.T) {
	table, w := newTable(t)
	if err := table.HandleMessage(message(t, "NATHEJK.2026.patrulje.shared-id.signedup", map[string]any{
		"teamId": "shared-id", "name": "Kontakt Person", "phone": "11111111",
	})); err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	contact := w.Last()
	if err := table.HandleMessage(message(t, "NATHEJK.2026.spejder.shared-id.updated", map[string]any{
		"memberId": "shared-id", "name": "Et Medlem", "phone": "22222222",
	})); err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	member := w.Last()

	if !strings.Contains(contact, "kind='patruljekontakt'") || !strings.Contains(member, "kind='spejder'") {
		t.Fatalf("the two rows are not distinguished by kind:\n%s\n%s", contact, member)
	}
}

// The advertised live token set must stay complete. A wildcard in the entity position — which
// is how the signup projection subscribes — makes live.EntitySet.Exhaustive false, and the SPA
// then cannot warn about a dependency nothing can satisfy.
func TestNoWildcardInTheEntityPosition(t *testing.T) {
	table, _ := newTable(t)
	for _, subj := range table.Consumes() {
		parts := subj.Parts()
		if len(parts) < 3 {
			t.Fatalf("unexpectedly short subject %q", subj.Subject())
		}
		if parts[2] == "*" {
			t.Errorf("%q wildcards the entity, which makes the advertised token set incomplete", subj.Subject())
		}
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
