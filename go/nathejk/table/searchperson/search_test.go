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

// A complete number must be an equality match, because that is the case the whole projection
// exists to make an index seek. A LIKE here would quietly turn every emergency lookup into a
// scan and nothing would report it.
func TestExactPhoneQueryUsesEquality(t *testing.T) {
	where, args := classify("+45 12 34 56 78").predicate()

	if !strings.Contains(where, "sp.phoneNormalized = ?") {
		t.Errorf("expected an equality match, got %q", where)
	}
	if strings.Contains(where, "LIKE") {
		t.Errorf("an 8-digit query must not use LIKE, got %q", where)
	}
	// The guardian's number is searched as readily as the scout's own: it is very often the
	// parent who rings.
	if !strings.Contains(where, "sp.phoneParentNormalized = ?") {
		t.Errorf("the guardian's number is not searched, got %q", where)
	}
	if len(args) != 2 || args[0] != "12345678" || args[1] != "12345678" {
		t.Errorf("args = %v, want the normalized number twice", args)
	}
}

// A partial number is a prefix match, which the same index still serves as a range scan.
func TestPartialPhoneQueryUsesPrefix(t *testing.T) {
	where, args := classify("1234").predicate()

	if !strings.Contains(where, "LIKE ?") {
		t.Errorf("expected a prefix match, got %q", where)
	}
	for _, a := range args {
		s, ok := a.(string)
		if !ok {
			t.Fatalf("unexpected arg type %T", a)
		}
		if strings.HasPrefix(s, "%") {
			t.Errorf("a leading wildcard cannot use the index: %q", s)
		}
		if s != "1234%" {
			t.Errorf("arg = %q, want %q", s, "1234%")
		}
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
		"(sp.year = ?) DESC",            // this year before any other
		"(sp.phoneNormalized = ?) DESC", // their own number before one that merely reaches them
		"sp.deleted ASC",                // still involved before departed
	} {
		if !strings.Contains(order, want) {
			t.Errorf("order missing %q: %s", want, order)
		}
	}
	// A stable tiebreak, or two identical searches could disagree about the order.
	if !strings.Contains(order, "sp.id ASC") {
		t.Errorf("no stable tiebreak: %s", order)
	}
	if len(args) != 2 {
		t.Errorf("args = %v, want the year and the number", args)
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

// writerStub accepts the schema statement and nothing else is needed for query-shaping tests.
type writerStub struct{}

func (writerStub) Consume(string) error { return nil }
