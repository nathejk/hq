package patruljemerged

import (
	"fmt"

	"github.com/jrgensen/stream"
	"github.com/jrgensen/stream/subject"
	"github.com/nathejk/shared-go/messages"
	"nathejk.dk/pkg/tablerow"
)

type consumer struct {
	w tablerow.Consumer
}

func (c *consumer) Consumes() (subjs []stream.Subject) {
	return []stream.Subject{
		subject.FromStr("NATHEJK:*.patrulje.*.merged"),
		subject.FromStr("NATHEJK:*.patrulje.*.splited"),
	}
}

func (c *consumer) HandleMessage(msg stream.Message) error {
	switch true {
	case msg.Subject().Match("NATHEJK.*.patrulje.*.merged"):
		var body messages.NathejkTeamMerged
		if err := msg.Body(&body); err != nil {
			return err
		}
		sql := fmt.Sprintf("INSERT INTO patruljemerged SET teamId=%q, parentTeamId=%q ON DUPLICATE KEY UPDATE teamId=VALUES(teamId)", body.TeamID, body.ParentTeamID)
		return c.w.Consume(sql)
	case msg.Subject().Match("NATHEJK.*.patrulje.*.splited"):
		var body messages.NathejkTeamSplited
		if err := msg.Body(&body); err != nil {
			return err
		}
		sql := fmt.Sprintf("DELETE FROM patruljemerged WHERE teamId=%q OR parentTeamId=%q", body.TeamID, body.TeamID)
		return c.w.Consume(sql)
	}
	return nil
}
