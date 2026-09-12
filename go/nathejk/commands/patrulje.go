package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/jrgensen/stream"
	"github.com/jrgensen/stream/subject"
	"github.com/nathejk/shared-go/messages"
	"github.com/nathejk/shared-go/types"
	"nathejk.dk/nathejk/table/patrulje"
)

// The patrulje write operations hq owns, as opposed to the signup edits the Team
// commands publish on tilmelding's behalf.
//
// There is one: the operational note for banditter and postmandskab. It is HQ's own
// annotation on a team — nobody outside HQ writes it and the team never sees it — which
// is why it is here rather than folded into UpdatePatrulje, whose events are published
// as `tilmelding-api` and describe what the *team* submitted.
type patruljeRemarkQuerier interface {
	GetByID(context.Context, types.TeamID) (*patrulje.Patrulje, error)
}

type patruljeCommander struct {
	p stream.Publisher
	q patruljeRemarkQuerier
}

func NewPatrulje(p stream.Publisher, q patruljeRemarkQuerier) *patruljeCommander {
	return &patruljeCommander{p: p, q: q}
}

// ErrRemarkSeverityInvalid reports a severity outside the set an operator may choose.
var ErrRemarkSeverityInvalid = errors.New("invalid remark severity")

// ErrRemarkUnchanged reports a save that would store the note a patrol already has.
//
// Silent rather than an error at the HTTP edge — see the handler — but distinguished here
// so the command never publishes an event that changes nothing, and therefore never makes
// every open patrol page revalidate for a no-op save.
var ErrRemarkUnchanged = errors.New("patrulje already has that remark")

// SetRemark stores the note shown to banditter and postmandskab.
//
// The note is published in full rather than as a patch (see patrulje.RemarkSet), and the
// year comes from the patrol's own row rather than from a configured current year: an
// operator annotating a past year's team must not publish onto this year's subject, where
// the projection would apply it to whichever team happens to hold that id now.
func (c *patruljeCommander) SetRemark(ctx context.Context, teamID types.TeamID, remark, severity string) error {
	if !patrulje.ValidRemarkSeverity(severity) {
		return ErrRemarkSeverityInvalid
	}
	team, err := c.q.GetByID(ctx, teamID)
	if err != nil {
		return err
	}
	if team.Remark == remark && team.RemarkSeverity == severity {
		return ErrRemarkUnchanged
	}

	msg := c.p.MessageFunc()(subject.FromStr(fmt.Sprintf("NATHEJK.%s.patrulje.%s.remark.set", team.Year, teamID)))
	msg.SetBody(&patrulje.RemarkSet{
		TeamID:   teamID,
		Remark:   remark,
		Severity: severity,
	})
	// Ours, not tilmelding's: this is an HQ operator annotating a team, and the trail
	// should say so.
	msg.SetMeta(&messages.Metadata{Producer: "hq-api"})
	return c.p.Publish(msg)
}

// ClearRemark removes the note entirely, returning the patrol to the state it was in before
// anybody wrote one.
//
// Its own method rather than SetRemark("", ""), because an empty severity is not something an
// operator may choose — SetRemark rejects it, which is what stops a note being saved with no
// severity at all. Deleting is a different act with a different meaning, so it says so.
//
// Distinct from severity `inactive`: that keeps the text on file so it can be put back into
// force, whereas this discards it.
func (c *patruljeCommander) ClearRemark(ctx context.Context, teamID types.TeamID) error {
	team, err := c.q.GetByID(ctx, teamID)
	if err != nil {
		return err
	}
	if team.Remark == "" && team.RemarkSeverity == "" {
		return ErrRemarkUnchanged
	}

	msg := c.p.MessageFunc()(subject.FromStr(fmt.Sprintf("NATHEJK.%s.patrulje.%s.remark.set", team.Year, teamID)))
	msg.SetBody(&patrulje.RemarkSet{TeamID: teamID})
	msg.SetMeta(&messages.Metadata{Producer: "hq-api"})
	return c.p.Publish(msg)
}
