package dispatch

import "github.com/nathejk/shared-go/types"

// SectionDispatchableSet marks an organisation section as being (or no longer being)
// a dispatch unit: a subsection of logistics that holds a vehicle, a driver and
// possibly a co-driver, and that tours may be assigned to.
//
// This is a fact about kørsel rather than about the section, which is why it is a
// dispatch event and a dispatch table instead of a column on shared-go's section —
// the same decision PRD 001 took for the nødtelefon's assignable sections, for the
// same reason. Published on NATHEJK.{year}.dispatch.section.{slug}.dispatchable, with
// the new state in the body rather than as two subjects: it is one fact with two
// values, and a consumer matching two subjects can handle one and silently miss the
// other.
type SectionDispatchableSet struct {
	SectionSlug  types.Slug `json:"sectionSlug"`
	Dispatchable bool       `json:"dispatchable"`
}
