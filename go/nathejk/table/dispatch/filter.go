package dispatch

import (
	"github.com/nathejk/shared-go/types"
)

// Filter narrows a task query. Year is always set in practice — every read path in the
// SPA sends X-YearSlug — but it is a field rather than a positional argument so that
// adding a filter later does not change every call site.
type Filter struct {
	YearSlug types.YearSlug

	// States restricts the query to the given states. Empty means every state, which is
	// what the board wants: the queue, the tours' tasks and the night's finished work are
	// one payload, because the desk switches between them constantly and a second round
	// trip per pane is a slower screen for no benefit at this volume.
	States []TaskState

	// SosID scopes to one case's tasks, for the nødtelefon operator who wants the
	// expected time without opening the dispatch board.
	SosID types.SosID
}

// TourFilter narrows a tour query.
type TourFilter struct {
	YearSlug types.YearSlug

	// States restricts the query. Empty means every state including cancelled ones, so a
	// caller that wants the board's tour pane should ask for planned and underway.
	States []TourState

	// SectionSlug scopes to one unit.
	SectionSlug types.Slug
}
