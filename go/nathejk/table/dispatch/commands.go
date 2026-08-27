package dispatch

import (
	"context"
	"fmt"

	"github.com/jrgensen/stream"
	"github.com/jrgensen/stream/subject"
	"github.com/nathejk/shared-go/messages"
	"github.com/nathejk/shared-go/types"
)

// Commands is the write surface, as the application sees it.
//
// Every method takes an Actor rather than reading the acting user out of the request
// context: the handler is already the layer that knows about HTTP, and this package
// must not import nathejk.dk/internal/requestctx.
type Commands interface {
	SetSectionDispatchable(ctx context.Context, actor Actor, year types.YearSlug, slug types.Slug, dispatchable bool) error
}

type commander struct {
	p stream.Publisher
	q Queries
}

// SetSectionDispatchable marks an organisation section as being (or no longer being) a
// dispatch unit.
//
// Dirty-checked against the current list, so a toggle that changes nothing publishes
// nothing — the Organisation page can send the state it wants without first working
// out whether that is already the case. Note the consequence (PRD 009 §8): a command
// that publishes nothing emits no live signal, so the toggle is optimistic in the UI
// rather than waiting to be told.
func (c commander) SetSectionDispatchable(ctx context.Context, actor Actor, year types.YearSlug, slug types.Slug, dispatchable bool) error {
	if !slug.Valid() {
		return fmt.Errorf("invalid section slug %q", slug)
	}
	current, err := c.q.DispatchableSections(ctx, year)
	if err != nil {
		return err
	}
	already := false
	for _, s := range current {
		if s == slug {
			already = true
			break
		}
	}
	if already == dispatchable {
		return nil
	}

	msg := c.p.MessageFunc()(subject.FromStr(fmt.Sprintf("NATHEJK.%s.dispatch.section.%s.dispatchable", year, slug)))
	if err := msg.SetBody(&SectionDispatchableSet{SectionSlug: slug, Dispatchable: dispatchable}); err != nil {
		return err
	}
	if err := msg.SetMeta(&messages.Metadata{UserID: actor.UserID}); err != nil {
		return err
	}
	return c.p.Publish(msg)
}
