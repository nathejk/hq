package searchperson

import (
	"strings"
	"testing"
)

// Classification decides whether an operator typed a number or a name, and getting it wrong
// means finding nobody. Table-driven because the interesting cases are all real inputs.
func TestClassify(t *testing.T) {
	cases := []struct {
		in         string
		wantDigits string
		wantPhone  bool
		wantName   bool
		wantExact  bool
	}{
		// A complete Danish number, in the formats the register actually holds.
		{"12345678", "12345678", true, false, true},
		{"12 34 56 78", "12345678", true, false, true},
		{"+45 12 34 56 78", "12345678", true, false, true},
		{"0045 12345678", "12345678", true, false, true},

		// A fragment, read back badly over the phone: prefix, not equality.
		{"1234", "1234", true, false, false},
		{"123456", "123456", true, false, false},

		// Too little of a number to be worth scanning for.
		{"123", "123", false, false, false},
		{"1", "1", false, false, false},

		// Names.
		{"Anders", "", false, true, false},
		{"Bo", "", false, false, false}, // below MinNameLength
		{"Søren Ø", "", false, true, false},

		// Both, and this is the case a letters-then-stop rule would lose: an operator
		// pasting a number out of a note, with its label attached.
		{"tlf. 12 34 56 78", "12345678", true, true, true},
		{"Anders 12345678", "12345678", true, true, true},

		// Letters plus too few digits: a name search, and the digits are ignored.
		{"Patrulje 42", "42", false, true, false},

		// Nothing at all.
		{"", "", false, false, false},
		{"   ", "", false, false, false},
	}

	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got := classify(tc.in)
			if got.Digits != tc.wantDigits {
				t.Errorf("Digits = %q, want %q", got.Digits, tc.wantDigits)
			}
			if got.MatchPhone != tc.wantPhone {
				t.Errorf("MatchPhone = %v, want %v", got.MatchPhone, tc.wantPhone)
			}
			if got.MatchName != tc.wantName {
				t.Errorf("MatchName = %v, want %v", got.MatchName, tc.wantName)
			}
			if got.MatchPhone && got.Exact() != tc.wantExact {
				t.Errorf("Exact = %v, want %v", got.Exact(), tc.wantExact)
			}
		})
	}
}

// A whole number is matched as a substring, not an equality — the change made by task 187.
//
// The reason is a real pattern in the register: a guardian field often holds two numbers as free
// text, `mor 22 79 01 52 eller Far 22110715`, which normalizes to one 16-digit string. Under an
// equality match neither parent was findable, which defeated the feature's premise in exactly
// the field an inbound call is most likely to match.
func TestPhoneQueryMatchesAsASubstring(t *testing.T) {
	where, args := classify("+45 12 34 56 78").predicate()

	if !strings.Contains(where, "sp.phoneNormalized LIKE ?") {
		t.Errorf("expected a substring match, got %q", where)
	}
	// The guardian's number is searched as readily as the scout's own: it is very often the
	// parent who rings.
	if !strings.Contains(where, "sp.phoneParentNormalized LIKE ?") {
		t.Errorf("the guardian's number is not searched, got %q", where)
	}
	for _, a := range args {
		if a != "%12345678%" {
			t.Errorf("arg = %v, want the normalized number wrapped in wildcards", a)
		}
	}
}

// The case this looseness exists for, at the level where it is decided.
func TestCollapsedTwoNumberFieldIsFindableByEitherHalf(t *testing.T) {
	// `mor 22 79 01 52 eller Far 22110715` as the projection stores it.
	const collapsed = "2279015222110715"

	for _, needle := range []string{"22790152", "22110715"} {
		c := classify(needle)

		_, args := c.predicate()
		pattern, _ := args[0].(string)
		if !strings.Contains(collapsed, needle) {
			t.Fatalf("test premise wrong: %q is not inside %q", needle, collapsed)
		}
		if pattern != "%"+needle+"%" {
			t.Errorf("pattern = %q, want it to match mid-string", pattern)
		}

		// And the hit must be labelled as the guardian's, not the person's own. Equality
		// here would fall through to nameRole and claim the number was theirs.
		if got := c.roleOf(row{phoneParentNormalized: collapsed}); got != PhoneRoleParent {
			t.Errorf("roleOf for %q = %q, want parent", needle, got)
		}
	}
}

// A number pasted in twice — `+452244565222445652` in the live register — becomes findable by the
// same mechanism, without any rule of its own.
func TestDoubledNumberIsFindable(t *testing.T) {
	c := classify("22445652")
	if got := c.roleOf(row{phoneNormalized: "452244565222445652"}); got != PhoneRoleOwn {
		t.Errorf("roleOf = %q, want own", got)
	}
}

// A fragment matches mid-number too. Worth pinning as intended rather than incidental: an
// operator with six digits of a number read back badly should find it wherever those digits sit.
func TestPartialPhoneQueryMatchesMidNumber(t *testing.T) {
	_, args := classify("1234").predicate()
	for _, a := range args {
		if a != "%1234%" {
			t.Errorf("arg = %v, want %q", a, "%1234%")
		}
	}

	if got := classify("4565").roleOf(row{phoneNormalized: "22445652"}); got != PhoneRoleOwn {
		t.Error("a fragment in the middle of a number did not match")
	}
}

// A name search is deliberately a mid-string match: "Koch" has to find "Rakel A. Koch". That
// cannot use an index and scans, which is accepted — but it must not be silently turned into
// something else either.
func TestNameQueryMatchesMidString(t *testing.T) {
	where, args := classify("Koch").predicate()
	if !strings.Contains(where, "sp.name LIKE ?") {
		t.Errorf("expected a name match, got %q", where)
	}
	if args[0] != "%Koch%" {
		t.Errorf("arg = %q, want %q", args[0], "%Koch%")
	}
}

// A query of "%" must not return the entire population. Not a injection risk — the value is
// bound — but a search that quietly lists every minor in the event is not something to leave to
// chance.
func TestWildcardsInUserTextAreEscaped(t *testing.T) {
	// A name query, since a digits-only string is never matched against the name column.
	_, args := classify("Anders%").predicate()
	got, _ := args[0].(string)
	if !strings.Contains(got, `\%`) {
		t.Errorf("the user's %% was not escaped: %q", got)
	}

	for _, in := range []string{"%", "_", `\`} {
		if escaped := escapeLike(in); !strings.HasPrefix(escaped, `\`) {
			t.Errorf("escapeLike(%q) = %q, want it escaped", in, escaped)
		}
	}
}

// Whose number matched is decided per row, because the SQL matched either column and only a
// comparison can say which. An operator about to ring somebody must know if a parent answers.
func TestRoleOfDistinguishesOwnFromGuardian(t *testing.T) {
	c := classify("12345678")

	own := row{phone: "12345678", phoneNormalized: "12345678", phoneParentNormalized: "87654321"}
	if got := c.roleOf(own); got != PhoneRoleOwn {
		t.Errorf("got %q, want own", got)
	}

	parent := row{phone: "11111111", phoneNormalized: "11111111", phoneParent: "12345678", phoneParentNormalized: "12345678"}
	if got := c.roleOf(parent); got != PhoneRoleParent {
		t.Errorf("got %q, want parent", got)
	}
}

// A name hit falls back to the guardian's number when the person has none of their own — and
// says so. A result whose only useful field is blank is barely a result.
func TestNameHitFallsBackToGuardianNumber(t *testing.T) {
	c := classify("Anders")

	if got := c.roleOf(row{phoneParent: "87654321", phoneParentNormalized: "87654321"}); got != PhoneRoleParent {
		t.Errorf("got %q, want parent", got)
	}
	if got := c.roleOf(row{}); got != PhoneRoleNone {
		t.Errorf("got %q, want none for somebody with no number at all", got)
	}
}

// The year scope is the one default here with a privacy dimension, so an omitted year is an
// error rather than a silent widening to every edition of the event.
func TestSearchRequiresAYear(t *testing.T) {
	table, err := New(&writerStub{}, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := table.Search(t.Context(), Query{Text: "12345678"}); err != ErrNoYear {
		t.Errorf("got %v, want ErrNoYear", err)
	}
}

// Ordering has to be explainable: an operator who cannot predict it cannot trust that the
// person they want is not further down. No relevance scoring, and every rule is one they would
// state themselves.
func TestOrderPutsThisYearAndLiveRecordsFirst(t *testing.T) {
	order, args := classify("12345678").order("2026")

	for _, want := range []string{
		"(sp.year = ?) DESC", // this year before any other
		// An exact hit before a coincidental substring — what makes the loose match tolerable.
		"(sp.phoneNormalized = ? OR sp.phoneParentNormalized = ?) DESC",
		"(sp.phoneNormalized LIKE ?) DESC", // their own number before one that merely reaches them
		"sp.deleted ASC",                   // still involved before departed
	} {
		if !strings.Contains(order, want) {
			t.Errorf("order missing %q: %s", want, order)
		}
	}
	// A stable tiebreak, or two identical searches could disagree about the order.
	if !strings.Contains(order, "sp.id ASC") {
		t.Errorf("no stable tiebreak: %s", order)
	}
	if len(args) != 4 {
		t.Errorf("args = %v, want the year, the number twice and the pattern", args)
	}
}

// Exactness is expressed in the ordering, since it is no longer expressed in the predicate. If
// this regresses, the person whose number was actually dialled can sit below a coincidence.
func TestExactMatchesSortAboveSubstringMatches(t *testing.T) {
	order, _ := classify("12345678").order("2026")

	exact := strings.Index(order, "= ? OR sp.phoneParentNormalized = ?) DESC")
	deleted := strings.Index(order, "sp.deleted ASC")
	if exact < 0 {
		t.Fatalf("no exactness term: %s", order)
	}
	if exact > deleted {
		t.Errorf("exactness must outrank the other tiebreaks: %s", order)
	}
}

// Nothing anywhere may filter on `deleted`. This is the retention requirement PRD 014 flags as
// the one most likely to be quietly broken, so it is asserted at the query layer too.
func TestSearchNeverFiltersOutRemovedPeople(t *testing.T) {
	for _, text := range []string{"12345678", "1234", "Anders"} {
		where, _ := classify(text).predicate()
		if strings.Contains(where, "deleted") {
			t.Errorf("query for %q filters on deleted: %s", text, where)
		}
	}
}

// A too-short query is not an empty result. The UI has to say "keep typing" rather than
// "ingen match", which an operator would believe.
func TestTooShortIsDistinctFromNoMatch(t *testing.T) {
	table, err := New(&writerStub{}, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	got, err := table.Search(t.Context(), Query{Text: "ab", Year: "2026"})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if !got.TooShort {
		t.Error("expected TooShort")
	}
	if len(got.Results) != 0 {
		t.Errorf("expected no results, got %d", len(got.Results))
	}
	if got.MatchedPhone || got.MatchedName {
		t.Error("nothing was searched, so neither interpretation should be reported")
	}
}

// The year predicate is present unless it was explicitly opted out of. Tested directly because
// it is the one default here with a privacy dimension, and because "the server never widens the
// scope on its own" is a claim, not an implementation detail.
func TestScopeIsYearBoundUnlessOptedOut(t *testing.T) {
	where, args := Query{Year: "2026"}.scope()
	if len(where) != 1 || where[0] != "sp.year = ?" {
		t.Fatalf("where = %v, want a year predicate", where)
	}
	if len(args) != 1 || args[0] != "2026" {
		t.Errorf("args = %v, want the year", args)
	}

	wide, wideArgs := Query{Year: "2026", IncludeOtherYears: true}.scope()
	if len(wide) != 0 || len(wideArgs) != 0 {
		t.Errorf("opting out should drop the predicate, got %v / %v", wide, wideArgs)
	}
}

// Coverage across every source, including a removed person: the requirement PRD 014 names as the
// one most likely to be quietly broken by a later "clean up the deleted rows" change.
//
// Asserted from event to queryable row via the statements the projection emits, per kind, so that
// a source dropped from Consumes() or a handler that stops writing shows up here rather than as a
// population that silently cannot be found.
func TestEverySourceIsIndexedAndFindable(t *testing.T) {
	events := []struct {
		kind    string
		subject string
		body    map[string]any
	}{
		{"spejder", "NATHEJK.2026.spejder.m-1.updated", map[string]any{"memberId": "m-1", "name": "A", "phone": "11111111", "phoneContact": "99999999"}},
		{"senior", "NATHEJK.2026.senior.m-2.updated", map[string]any{"memberId": "m-2", "name": "B", "phone": "22222222"}},
		{"gøgler", "NATHEJK.2026.gøgler.u-3.signedup", map[string]any{"teamId": "u-3", "name": "C", "phone": "33333333"}},
		{"friend", "NATHEJK.2026.friend.u-4.signedup", map[string]any{"teamId": "u-4", "name": "D", "phone": "44444444"}},
		{"crew", "NATHEJK.2026.crew.u-5.signedup", map[string]any{"teamId": "u-5", "name": "E", "phone": "55555555"}},
		{"patruljekontakt", "NATHEJK.2026.patrulje.t-6.signedup", map[string]any{"teamId": "t-6", "name": "F", "phone": "66666666"}},
		{"klankontakt", "NATHEJK.2026.klan.t-7.signedup", map[string]any{"teamId": "t-7", "name": "G", "phone": "77777777"}},
	}

	table, w := newTable(t)
	for _, e := range events {
		if err := table.HandleMessage(message(t, e.subject, e.body)); err != nil {
			t.Fatalf("%s: %v", e.kind, err)
		}
		stmt := w.Last()
		if !strings.Contains(stmt, "kind="+quote(e.kind)) {
			t.Errorf("%s produced no row of its own kind:\n%s", e.kind, stmt)
		}
		// Findable by name and by number, which is the whole contract.
		if !strings.Contains(stmt, "name=") || !strings.Contains(stmt, "phoneNormalized=") {
			t.Errorf("%s is not findable by name and number:\n%s", e.kind, stmt)
		}
	}

	// And the removed person stays, flagged.
	if err := table.HandleMessage(message(t, "NATHEJK.2026.spejder.m-1.deleted", map[string]any{"memberId": "m-1"})); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if removal := w.Last(); !strings.Contains(removal, "deleted=1") ||
		strings.Contains(strings.ToUpper(removal), "DELETE FROM") {
		t.Errorf("a removed person must be kept and flagged:\n%s", removal)
	}
}

// writerStub accepts the schema statement and nothing else is needed for query-shaping tests.
type writerStub struct{}

func (writerStub) Consume(string) error { return nil }
