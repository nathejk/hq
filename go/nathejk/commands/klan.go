package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/jrgensen/stream"
	"github.com/jrgensen/stream/subject"
	"github.com/nathejk/shared-go/messages"
	sharedklan "github.com/nathejk/shared-go/tables/klan"
	"github.com/nathejk/shared-go/types"
)

// Klan write operations hq needs and the shared-go entity does not offer.
//
// There is exactly one: setting the signup status directly. Everywhere else the
// status is a *consequence* — a signup puts a klan on hold, an order being paid
// moves it to PAID — which is correct, and is why no such command existed.
//
// It exists because the money does not always arrive the way the system expects.
// A klan that pays by bank transfer, in cash at a meeting, or as part of another
// group's payment has genuinely paid, but no MobilePay callback will ever say so,
// so the read model stays on PAY and the klan is treated all weekend as if it had
// not paid. The alternative to an override is HQ inventing a payment that did not
// happen through the provider, which corrupts the payment trail to fix a display.
//
// Deliberately *not* a payment: this command changes the status and nothing else.
// The orders and payments a klan has remain exactly as they were, so the override
// is visible as what it is — an administrative decision that the status no longer
// follows from the payments — rather than as forged evidence.
type klanQuerier interface {
	GetByID(context.Context, types.TeamID) (*sharedklan.Klan, error)
}

type klanCommander struct {
	p stream.Publisher
	q klanQuerier

	// The shared-go entity, for the operations it already owns. Delete publishes
	// the same status.changed event this file does, and duplicating that here
	// would be two definitions of what deleting a klan means.
	entity sharedklan.Commands
}

func NewKlan(p stream.Publisher, q klanQuerier, entity sharedklan.Commands) *klanCommander {
	return &klanCommander{p: p, q: q, entity: entity}
}

// ErrKlanStatusUnchanged reports an override that would set the status a klan
// already has.
//
// An error rather than a silent no-op, unlike the dirty-checks elsewhere: those
// guard against redundant events on a bulk save, whereas this is a deliberate,
// single act by an operator. Told nothing, they cannot tell "already correct"
// from "did not work".
var ErrKlanStatusUnchanged = errors.New("klan already has that status")

// SetStatus overrides a klan's signup status.
//
// The year comes from the klan's own row rather than a configured current year:
// an operator correcting last year's books must not publish an event onto this
// year's subject, where the projection would apply it to a klan that is not the
// one they were looking at.
func (c *klanCommander) SetStatus(ctx context.Context, teamID types.TeamID, status types.SignupStatus) error {
	klan, err := c.q.GetByID(ctx, teamID)
	if err != nil {
		return err
	}
	if klan.Status == status {
		return ErrKlanStatusUnchanged
	}

	msg := c.p.MessageFunc()(subject.FromStr(fmt.Sprintf("NATHEJK.%s.klan.%s.status.changed", klan.Year, teamID)))
	msg.SetBody(&messages.NathejkKlanStatusChanged{
		TeamID: teamID,
		Status: status,
	})
	// Marked as ours so the trail shows an HQ operator did this, not the payment
	// flow — the one thing that makes an out-of-band status readable afterwards.
	msg.SetMeta(&messages.Metadata{Producer: "hq-api"})
	return c.p.Publish(msg)
}

// Delete withdraws a klan, delegating to the shared-go entity that owns what
// that means (a status.changed to "deleted", so the row survives for the trail).
func (c *klanCommander) Delete(ctx context.Context, teamID types.TeamID) error {
	return c.entity.Delete(ctx, teamID)
}
