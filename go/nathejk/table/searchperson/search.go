package searchperson

import (
	"context"
	"fmt"
	"strings"
	"unicode"
)

// How short a query may be before it is refused.
//
// Both limits exist to stop the first keystroke asking for the whole event. They are also the
// difference between "searched and found nobody" and "not really searched yet", which the UI has
// to be able to say differently — an operator who is told "ingen match" after one character
// would believe it.
const (
	MinNameLength  = 3
	MinPhoneDigits = 4

	// DefaultLimit caps a response. Stated in the payload when it bites, so the UI can say
	// so rather than silently truncating and letting the operator scroll to a false bottom.
	DefaultLimit = 50

	// nationalNumberLength is when a Danish number is complete, and so when a lookup becomes
	// an equality match on the index rather than a prefix scan.
	nationalNumberLength = 8
)

// Query is one search as asked for.
type Query struct {
	// Text is what the operator typed, unmodified.
	Text string

	// Year is required. Not defaulted, not inferred: a widened year scope must be
	// impossible to reach by omission, so an empty Year is an error rather than "all
	// years". See ErrNoYear.
	Year string

	// IncludeOtherYears drops the year predicate.
	//
	// Named "other" rather than "previous" because the data is not only historical: the dev
	// stream carries fixtures under year 9999, and nothing stops a future event edition
	// existing before the current one closes. Off by default, and the server never turns it
	// on by itself — not on an empty result, not on an exact match. An operator who has not
	// asked for other years must be able to trust that what they are reading is this one.
	IncludeOtherYears bool

	// Limit overrides DefaultLimit. Zero means the default.
	Limit int
}

// Results is what a search returns.
type Results struct {
	Results []Result `json:"results"`

	// Truncated reports that the cap was reached and there are more matches.
	Truncated bool `json:"truncated"`

	// TooShort reports that nothing was searched for, because the query was below the
	// minimum. Distinct from an empty Results, which means we looked and found nobody.
	TooShort bool `json:"tooShort"`

	// MatchedPhone and MatchedName say how the query was interpreted, so the UI can explain
	// itself when the interpretation surprises the operator.
	MatchedPhone bool `json:"matchedPhone"`
	MatchedName  bool `json:"matchedName"`
}

// ErrNoYear is returned when a Query carries no year.
//
// An error rather than a silent widening to every year, because the year scope is the one
// default in this feature with a privacy dimension: search returns contact details for minors,
// and "this year" is a much smaller disclosure than "every year we have ever run".
var ErrNoYear = fmt.Errorf("searchperson: a year is required")

// classification is how one query string was understood.
type classification struct {
	// Digits is the query reduced to national form, empty when it holds no usable number.
	Digits string

	// Name is the trimmed text, for substring matching.
	Name string

	MatchPhone bool
	MatchName  bool
}

// Exact reports whether the query holds a whole number rather than a fragment.
//
// No longer decides *how* the match is done — phone matching is a substring in every case, so
// that a guardian field holding two numbers is findable by either (task 187). It survives
// because the distinction still means something to a reader and to the tests: eight digits is
// somebody's number, four is part of one, and only the former can be expected to identify a
// single person.
func (c classification) Exact() bool { return len(c.Digits) >= nationalNumberLength }

// classify decides whether a query is a number, a name, or both.
//
// Both is a real case, not a hedge: an operator who pastes "tlf. 12 34 56 78" out of a note has
// typed letters *and* a complete number, and a rule that read the letters and stopped would find
// nobody. So the number is used whenever there is enough of one, and the name is used as well
// whenever there are letters. The digits-only case is unambiguous and skips the name scan.
func classify(text string) classification {
	trimmed := strings.TrimSpace(text)
	digits := normalizePhone(trimmed)

	hasLetter := false
	for _, r := range trimmed {
		if unicode.IsLetter(r) {
			hasLetter = true
			break
		}
	}

	return classification{
		Digits:     digits,
		Name:       trimmed,
		MatchPhone: len(digits) >= MinPhoneDigits,
		MatchName:  hasLetter && len([]rune(trimmed)) >= MinNameLength,
	}
}

// Search finds people by number or by name.
//
// Removed people are **included**, flagged rather than filtered: a scout taken off a roster in
// August still has a guardian who rings in September, and "ingen match" is an answer an operator
// cannot distinguish from "never involved" (PRD 014 §5). Nothing in this function filters on
// `deleted`, deliberately, and a test asserts as much.
func (t *Table) Search(ctx context.Context, q Query) (Results, error) {
	if q.Year == "" && !q.IncludeOtherYears {
		return Results{}, ErrNoYear
	}

	c := classify(q.Text)
	out := Results{
		Results:      []Result{},
		MatchedPhone: c.MatchPhone,
		MatchedName:  c.MatchName,
	}
	if !c.MatchPhone && !c.MatchName {
		out.TooShort = true
		return out, nil
	}

	limit := q.Limit
	if limit <= 0 {
		limit = DefaultLimit
	}

	where, args := q.scope()

	match, matchArgs := c.predicate()
	where = append(where, "("+match+")")
	args = append(args, matchArgs...)

	order, orderArgs := c.order(q.Year)
	args = append(args, orderArgs...)

	// One row more than the cap, so truncation is detected rather than guessed at.
	results, err := t.query(ctx, strings.Join(where, " AND "), args, c.roleOf, order, limit+1)
	if err != nil {
		return out, err
	}

	if len(results) > limit {
		out.Truncated = true
		results = results[:limit]
	}
	out.Results = results
	return out, nil
}

// scope builds the year predicate.
//
// Its own function so the rule can be tested without a database, because it is the one default
// in this feature with a privacy dimension: a search returns minors' contact details, and "this
// year" is a far smaller disclosure than "every year we have ever run". The predicate is present
// unless the caller explicitly opted out of it, and nothing in Search can widen it — not an
// empty result, not an exact match.
func (q Query) scope() ([]string, []any) {
	if q.IncludeOtherYears {
		return []string{}, []any{}
	}
	return []string{"sp.year = ?"}, []any{q.Year}
}

// predicate builds the match condition and its arguments.
func (c classification) predicate() (string, []any) {
	terms, args := []string{}, []any{}

	if c.MatchPhone {
		// One substring match, covering both columns, whatever the length of the needle.
		//
		// This is deliberately looser than an equality or a prefix, and the reason is a real
		// pattern in the register: a guardian field very often holds *two* numbers as free
		// text — `mor 22 79 01 52 eller Far 22110715` — which normalizes to one 16-digit
		// string. Under an equality match neither parent could be found, which defeated the
		// feature's premise for 33 of 1459 rows, in exactly the field an inbound call is most
		// likely to match. A substring finds both halves, and the same trick recovers the
		// occasional number that was pasted in twice.
		//
		// The cost is a leading wildcard, so this no longer uses the phone indexes and scans.
		// Measured before committing to it (tasks 186 and 187): a scan of this table costs
		// ~5ms at 73k rows, sixteen times its current size, against a 50ms target.
		//
		// The second cost is false positives, which cannot be designed away: an 8-digit needle
		// can match across the join between two collapsed numbers, and a 4-digit one matches
		// mid-number. Mitigated by ordering rather than by exclusion — exact matches sort
		// first, see order — and accepted, because a spurious row an operator can see and
		// dismiss is far cheaper than a missing row they cannot know about.
		terms = append(terms, "sp.phoneNormalized LIKE ?", "sp.phoneParentNormalized LIKE ?")
		args = append(args, "%"+c.Digits+"%", "%"+c.Digits+"%")
	}

	if c.MatchName {
		// A leading wildcard, so this cannot use an index and scans. Accepted: a name is
		// looked up mid-string ("Koch" in "Rakel A. Koch") often enough that a prefix
		// match would be the wrong feature, and the row counts are in the thousands.
		//
		// LIKE is case-insensitive under the column's utf8mb4_general_ci collation, which
		// covers æ/ø/å as well.
		terms = append(terms, "sp.name LIKE ?")
		args = append(args, "%"+escapeLike(c.Name)+"%")
	}

	return strings.Join(terms, " OR "), args
}

// order sorts results so the most likely answer is first, and does it explainably.
//
// No relevance scoring: an operator who cannot predict the order cannot trust that the person
// they want is not further down. The rules, in order, are all things they would say themselves.
func (c classification) order(currentYear string) (string, []any) {
	terms, args := []string{}, []any{}

	// This year first. A hit from a previous edition is only ever shown deliberately, and
	// must not be interleaved with the current one.
	terms = append(terms, "(sp.year = ?) DESC")
	args = append(args, currentYear)

	if c.MatchPhone {
		// Exact matches first, then the rest.
		//
		// This is what makes a substring match tolerable. The predicate returns anything
		// *containing* the digits — including a needle that straddles two collapsed numbers,
		// and every mid-number hit for a short fragment — so without this the person whose
		// number was actually dialled could sit below a coincidence. With it, they are first
		// and the coincidences are visibly below them.
		terms = append(terms,
			"(sp.phoneNormalized = ? OR sp.phoneParentNormalized = ?) DESC",
			// Then their own number, before a number that merely reaches them.
			"(sp.phoneNormalized LIKE ?) DESC",
		)
		args = append(args, c.Digits, c.Digits, "%"+c.Digits+"%")
	}

	// Somebody still involved before somebody who left.
	terms = append(terms, "sp.deleted ASC")

	// A stable tiebreak, so two consecutive identical searches cannot disagree about the
	// order — which would be its own small mystery at 3am.
	terms = append(terms, "sp.name ASC", "sp.kind ASC", "sp.id ASC")

	return strings.Join(terms, ", "), args
}

// roleOf decides, for one matched row, whose number to show and how to label it.
//
// This is why the needle is carried into the row loop rather than left in the predicate: the SQL
// matched either column, and only a comparison per row can say which — and an operator who is
// about to ring somebody must be told whether a parent will answer.
//
// Containment rather than equality, matching the predicate. A guardian field holding two numbers
// is one 16-digit string, so the needle is a *substring* of it; testing equality here would leave
// every such hit falling through to nameRole and labelled as the person's own number, which is
// precisely the lie this function exists to prevent.
//
// Own number checked first: where a number appears in both fields, it is more useful to say "this
// is their number" than "this reaches their guardian".
func (c classification) roleOf(r row) PhoneRole {
	if c.MatchPhone && c.Digits != "" {
		if strings.Contains(r.phoneNormalized, c.Digits) {
			return PhoneRoleOwn
		}
		if strings.Contains(r.phoneParentNormalized, c.Digits) {
			return PhoneRoleParent
		}
	}
	// A name hit, or a row that matched on name in a mixed query.
	return nameRole(r)
}

// nameRole picks a number to show for a hit that was not found by number.
//
// Falls through to the guardian's when the person has none of their own, because a result whose
// only useful field is blank is barely a result — and it is labelled as the guardian's, so the
// operator is not misled about who picks up.
func nameRole(r row) PhoneRole {
	switch {
	case r.phone != "":
		return PhoneRoleOwn
	case r.phoneParent != "":
		return PhoneRoleParent
	default:
		return PhoneRoleNone
	}
}

// escapeLike neutralises the wildcards in a user's own text.
//
// Without it, a query of `%` matches everybody: `LIKE '%%%'`. That is not a security hole here
// (the value is still a bound parameter) but it is a search that quietly returns the entire
// population, which for a table of minors' contact details is not something to leave to chance.
func escapeLike(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case '%', '_', '\\':
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}
