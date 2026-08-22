package shelter

import (
	"context"
	"strings"
	"testing"
)

// The suggestion query is the whole zone model, so its shape is worth pinning without a
// database — the same approach the projection's tests take with emitted SQL.

// Ordering is what makes a typo harmless rather than permanent. "Telt 4" with four scouts in
// it must sit above a fat-fingered "Telt 44" with one, and the name tiebreak keeps a list of
// equal counts from reshuffling under the cursor between two loads.
func TestDistinctPlacementsOrdersByUseThenName(t *testing.T) {
	if !strings.Contains(distinctPlacementsQuery, "ORDER BY c DESC, placement ASC") {
		t.Errorf("expected most-used first with a stable name tiebreak, got %q", distinctPlacementsQuery)
	}
}

// A scout accepted but not yet bedded down has an empty placering. That is not evidence of a
// zone called "", and suggesting it would be offering to un-place somebody.
func TestDistinctPlacementsExcludesTheEmptyPlacering(t *testing.T) {
	if !strings.Contains(distinctPlacementsQuery, "placement <> ''") {
		t.Errorf("expected empty placeringer to be excluded, got %q", distinctPlacementsQuery)
	}
}

// Year-scoped, like every read in the platform: two races must not share a vocabulary of
// tents, and the year is a placeholder rather than interpolated.
func TestDistinctPlacementsIsYearScoped(t *testing.T) {
	if !strings.Contains(distinctPlacementsQuery, "year = ?") {
		t.Errorf("expected a year constraint, got %q", distinctPlacementsQuery)
	}
}

// An empty id set must not reach the database: `id IN ()` is a MySQL syntax error, so a screen
// with nobody sheltered would get a database error where it meant an empty answer.
func TestGetByMemberIDsEmptySetIssuesNoQuery(t *testing.T) {
	// A nil *sql.DB is the assertion: any attempt to query would panic.
	q := &querier{}

	got, err := q.GetByMemberIDs(context.Background(), "2026", nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got == nil || len(got) != 0 {
		t.Errorf("expected an empty map, got %v", got)
	}
}

// The column list is shared so the scan order cannot drift from it.
func TestSelectPlacementColumnOrder(t *testing.T) {
	want := "SELECT id, year, teamId, placement, acceptedAt, placedAt"
	if !strings.HasPrefix(selectPlacement, want) {
		t.Errorf("expected the shared column list to start %q, got %q", want, selectPlacement)
	}
}
