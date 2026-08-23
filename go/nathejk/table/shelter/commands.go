package shelter

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jrgensen/stream"
	"github.com/jrgensen/stream/subject"
	"github.com/nathejk/shared-go/messages"
	"github.com/nathejk/shared-go/types"

	"nathejk.dk/nathejk/table/spejderstatus"
)

// The write side of the placering.
//
// # Why this is not on spejderstatus's commander
//
// PRD 007's task list put all three shelter commands there, and one of them cannot go there:
// SetPlacement has to dirty-check the *current placering*, which lives in this package's
// table. spejderstatus may not import hq packages at all (it lifts to shared-go verbatim,
// task 083), and it could not import this one in any case — this package imports it, so the
// dependency would be a cycle.
//
// The split that resolves it is also the better design, which is why it was taken rather than
// worked around: a command belongs with the read model it checks against. Status transitions
// are spejderstatus's, the placering is this package's. The handler composes the two, exactly
// as it already composes member events with the sos summary.

// MaxPlacementLength caps the placering.
//
// 64 characters, matching the column. It is a label like "Telt 4" or "Sovesalen, ved vinduet",
// not a description: anything longer will not fit the column on screen either, and a crew
// member typing a paragraph into it is trying to record something this field is not for.
const MaxPlacementLength = 64

var (
	// ErrNotSheltered is returned when a placering is set for somebody who is not in the
	// shelter.
	//
	// A placering means "this child is asleep here", so recording one for a scout still in a
	// car would put them in two places at once — and the crew would look for them in a tent
	// they have never been in. The fix is to accept them first, which is one button away.
	ErrNotSheltered = errors.New("member is not in the shelter")

	// ErrEmptyPlacement is returned for a blank placering.
	//
	// Clearing one is deliberately not offered. "Nowhere" is not a fact about a child in our
	// care; if they have moved, the answer is where they moved to, and if they have left, the
	// answer is a handover.
	ErrEmptyPlacement = errors.New("placering is required")

	// ErrPlacementTooLong is returned for a placering over MaxPlacementLength.
	ErrPlacementTooLong = errors.New("placering is too long")
)

// Commands is the write surface for the placering.
//
// The Actor is passed in by the caller rather than read from the request context, matching
// every other table package: it keeps this package free of nathejk.dk/internal/requestctx.
type Commands interface {
	SetPlacement(ctx context.Context, actor spejderstatus.Actor, year types.YearSlug, id types.MemberID, placement string) error
}

type commander struct {
	p stream.Publisher
	q Queries

	// status is read to answer "is this member actually in the shelter?". The shelter table
	// alone cannot answer it: a member with no row here might be in a car, or might be
	// sheltered by an acceptance this projection has not applied yet.
	status spejderstatus.Queries
}

// SetPlacement records where in the shelter a member is, or moves them.
//
// Idempotent on the value: setting the tent a scout is already in publishes nothing, so a
// double submit puts one line on the timeline rather than two. That is the house pattern, and
// it is also what stops a re-render of the editor from looking like the child was moved.
//
// The trimmed value is what is published. A placering typed with a trailing space would
// otherwise become a second zone in the suggestion list, one pixel different from the first —
// which is precisely the drift the suggestions exist to prevent.
func (c commander) SetPlacement(ctx context.Context, actor spejderstatus.Actor, year types.YearSlug, id types.MemberID, placement string) error {
	placement = strings.TrimSpace(placement)
	if placement == "" {
		return ErrEmptyPlacement
	}
	// Counted in runes, not bytes: "Sovesalen ved døren" is shorter than len() thinks, and a
	// Danish crew should not be told their tent name is too long because of an æ.
	if len([]rune(placement)) > MaxPlacementLength {
		return ErrPlacementTooLong
	}

	current, err := c.status.GetByMemberID(ctx, year, id)
	if err != nil {
		return err
	}
	if current.Status != types.MemberStatusSheltered {
		return ErrNotSheltered
	}

	// Dirty-check against the placering we hold. A missing row is not a reason to refuse:
	// the member is sheltered per the status above, so their shelter row is either being
	// written right now or was lost, and either way recording where they are is the useful
	// thing to do.
	existing, err := c.q.GetByMemberIDs(ctx, year, []types.MemberID{id})
	if err != nil {
		return err
	}
	if p, ok := existing[id]; ok && p.Placement == placement {
		return nil
	}

	body := &spejderstatus.ShelterPlaced{
		MemberID:  id,
		TeamID:    current.CurrentTeamID,
		Placement: placement,
		Actor:     actor,
	}
	return c.publish(actor, year, id, "shelter.placed", body)
}

func (c commander) publish(actor spejderstatus.Actor, year types.YearSlug, id types.MemberID, event string, body any) error {
	msg := c.p.MessageFunc()(subject.FromStr(fmt.Sprintf("NATHEJK.%s.spejder.%s.%s", year, id, event)))
	if err := msg.SetBody(body); err != nil {
		return err
	}
	if err := msg.SetMeta(&messages.Metadata{UserID: actor.UserID}); err != nil {
		return err
	}
	return c.p.Publish(msg)
}
