package patrulje

import (
	"strings"

	"github.com/nathejk/shared-go/types"
)

// The operational note HQ can attach to a patrulje for banditter and postmandskab.
//
// # Why severity carries "not in force"
//
// The note and its severity are one field pair, and `inactive` is a severity rather than a
// separate boolean. A note plus an enabled-flag can disagree — an active flag on an empty
// note, a filled note nobody notices is switched off — and both readings then have to be
// defended everywhere the note is displayed. With one field there is exactly one question:
// what does severity say?
//
// "" is the ordinary state: no note has ever been written. It is distinct from `inactive`,
// which means somebody wrote one and stood it down; keeping them apart is what lets the UI
// stay collapsed for the many patrols that never use this at all.
const (
	// RemarkSeverityInformation — worth knowing, does not change what anyone does.
	RemarkSeverityInformation = "information"

	// RemarkSeverityStop — "Fuld stop": the patrol must not be sent on.
	RemarkSeverityStop = "stop"

	// RemarkSeverityInactive — filed but not in force. The text is kept so it can be put
	// back into force without being retyped.
	RemarkSeverityInactive = "inactive"
)

// RemarkSeverities is the severities an operator may choose, in the order they are offered.
//
// Deliberately excludes "": that is the absence of a note, which is expressed by never
// having set one, not by choosing it from a list.
var RemarkSeverities = []string{
	RemarkSeverityInformation,
	RemarkSeverityStop,
	RemarkSeverityInactive,
}

// ValidRemarkSeverity reports whether a severity is one an operator may set.
func ValidRemarkSeverity(s string) bool {
	for _, v := range RemarkSeverities {
		if v == s {
			return true
		}
	}
	return false
}

// RemarkSet is the event body: the note as it now stands, in full.
//
// State, not a delta. The note is a single small piece of text that one operator edits at a
// time, so replaying "here is the note now" converges on the same answer whatever order the
// log arrives in — whereas a patch would need the previous text to make sense of.
type RemarkSet struct {
	TeamID   types.TeamID `json:"teamId"`
	Remark   string       `json:"remark"`
	Severity string       `json:"severity"`
}

// PhotoConsentSet is the event body for "Fototilladelse": who on the patrol has refused
// public photographs of themselves after the race, in full.
//
// Accepting is the default, so a patrol that never had this set is one where everybody
// accepts. Refusal is recorded at one of two resolutions, because HQ is often told "one of
// them doesn't want to be in the pictures" without being told which:
//
//   - TeamRefused: somebody on the patrol refused, and we do not know who. The whole
//     patrol must then be treated as refusing, so MemberIDs is meaningless and is sent empty.
//   - MemberIDs: exactly these members refused; the rest of the patrol accepts.
//
// State, not a delta, for the same reason as RemarkSet.
type PhotoConsentSet struct {
	TeamID      types.TeamID     `json:"teamId"`
	TeamRefused bool             `json:"teamRefused"`
	MemberIDs   []types.MemberID `json:"memberIds"`
}

// JoinMemberIDs is how PhotoConsentSet.MemberIDs is stored in its column.
func JoinMemberIDs(ids []types.MemberID) string {
	s := make([]string, 0, len(ids))
	for _, id := range ids {
		if id != "" {
			s = append(s, string(id))
		}
	}
	return strings.Join(s, ",")
}

// SplitMemberIDs is JoinMemberIDs' inverse. Never nil, so it serialises as [].
func SplitMemberIDs(s string) []types.MemberID {
	ids := []types.MemberID{}
	for _, id := range strings.Split(s, ",") {
		if id != "" {
			ids = append(ids, types.MemberID(id))
		}
	}
	return ids
}
