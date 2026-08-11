package sos

import (
	"github.com/nathejk/shared-go/types"
)

// Filter narrows a case query. Year is always set in practice — every read path
// in the SPA sends X-YearSlug — but it is a field rather than a positional
// argument so that adding a filter later does not change every call site.
type Filter struct {
	YearSlug types.YearSlug
	TeamID   types.TeamID
	Status   Status

	// IncludeDeleted is deliberately awkward to reach for: soft-deleted cases are
	// excluded everywhere in the product, and the flag exists for diagnostics
	// rather than for a screen.
	IncludeDeleted bool
}
