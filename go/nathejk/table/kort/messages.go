package kort

import (
	"github.com/nathejk/shared-go/types"
)

// The kort events, published on NATHEJK.{year}.kort.{kortId}.{event}.
//
// # Why these types live here and not in shared-go
//
// They start local and move to nathejk/shared-go once the shape has survived a season (PRD 010
// §8). The messages will very likely go before the projection does — the hej-app needs to read
// kort events long before a second service wants to materialise them — so the two moves are
// independent, and nothing here may depend on anything HQ-only. Everything below is either a
// local type or a shared-go type for exactly that reason.
//
// # Its own entity, so its own live token
//
// Unlike spejdernote, which rides on the scout's subject to reuse an existing token, a map is
// not a fact about another entity: nothing else's screen wants to refetch when a map is renamed.
// So `kort` is its own subject and its own live signal token, which the SPA declares via
// `dependsOn: ['kort', ...]` (task 126).

// Format is what was printed, and on what.
//
// A closed set rather than free text, because it drives nothing but is read by humans deciding
// what to hand over — and four values that mean something beat a column with "A4", "a4" and
// "A4 (dobbeltsidet)" in it.
type Format string

const (
	FormatA4 Format = "a4"
	FormatA3 Format = "a3"

	// FormatSkitse is a hand-drawn slip showing the next group of checkpoints.
	//
	// The awkward case, and the reason maps are modelled at all: a skitse has no QR code, so it
	// is never scanned, and usually no extent worth recording. Its checkpoint list is its only
	// trace in the system, and the hej-app reveals those checkpoints off the *previous*
	// checkpoint's scan instead (PRD 010 §8).
	FormatSkitse Format = "skitse"

	FormatAndet Format = "andet"
)

func (f Format) Valid() bool {
	switch f {
	case FormatA4, FormatA3, FormatSkitse, FormatAndet:
		return true
	}
	return false
}

// Extent is one rectangle of ground a sheet shows.
//
// Corners are north-west and south-east rather than "first" and "second": whichever two corners
// an operator clicks, they are normalised on save (task 124), so every reader can assume the
// pair is well-formed and no consumer has to guess which way round a rectangle was drawn.
type Extent struct {
	NorthWest types.Position `json:"northWest"`
	SouthEast types.Position `json:"southEast"`
}

// Created records a new sheet.
//
// Only the set and the name, matching how checkpoint.created carries almost nothing: the
// operator adds a map and then describes it, and a create that demanded a format and an extent
// would mean a half-known sheet could not be written down at all.
type Created struct {
	KortID     KortID     `json:"kortId"`
	KortsaetID KortsaetID `json:"kortsaetId"`
	Name       string     `json:"name"`
}

// Updated records a change to one or more of a sheet's fields.
//
// Every field is a pointer, following NathejkCheckpointUpdated: nil means "not part of this
// event" and is therefore left alone by the projection, which is what lets the checkpoint picker
// save a checkpoint list without restating the name, format and extents it never touched.
//
// CheckpointIDs and Extents are pointers *to slices* for the same reason, and the distinction
// matters here more than usual: `nil` means untouched, while a pointer to an empty slice means
// "this sheet now has no checkpoints" — a real edit that a plain nil slice could not express.
type Updated struct {
	KortID        KortID                `json:"kortId"`
	KortsaetID    *KortsaetID           `json:"kortsaetId,omitempty"`
	Name          *string               `json:"name,omitempty"`
	Format        *Format               `json:"format,omitempty"`
	Note          *string               `json:"note,omitempty"`
	SortOrder     *int                  `json:"sortOrder,omitempty"`
	CheckpointIDs *[]types.CheckpointID `json:"checkpointIds,omitempty"`
	Extents       *[]Extent             `json:"extents,omitempty"`
}

// Deleted records that a sheet is no longer printed.
//
// Deleting a map does not touch its checkpoints — they exist independently and are almost
// certainly on another sheet too.
type Deleted struct {
	KortID KortID `json:"kortId"`
}
