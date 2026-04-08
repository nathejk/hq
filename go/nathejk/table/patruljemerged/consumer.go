package patruljemerged

import (
	"fmt"
	"log"

	"github.com/nathejk/shared-go/messages"
	"nathejk.dk/pkg/tablerow"
	"nathejk.dk/superfluids/streaminterface"
)

type consumer struct {
	w tablerow.Consumer
}

func (c *consumer) Consumes() (subjs []streaminterface.Subject) {
	return []streaminterface.Subject{
		streaminterface.SubjectFromStr("NATHEJK:*.patrulje.*.merged"),
		streaminterface.SubjectFromStr("NATHEJK:*.patrulje.*.splited"),
	}
}

func (c *consumer) HandleMessage(msg streaminterface.Message) error {
	switch true {
	case msg.Subject().Match("NATHEJK.*.patrulje.*.merged"):
		var body messages.NathejkTeamMerged
		if err := msg.Body(&body); err != nil {
			return err
		}
		sql := fmt.Sprintf("INSERT INTO patruljemerged SET teamId=%q, parentTeamId=%q ON DUPLICATE KEY UPDATE teamId=VALUES(teamId)", body.TeamID, body.ParentTeamID)
		if err := c.w.Consume(sql); err != nil {
			log.Fatalf("Error consuming sql %q", err)
		}
	case msg.Subject().Match("NATHEJK.*.patrulje.*.splited"):
		var body messages.NathejkTeamSplited
		if err := msg.Body(&body); err != nil {
			return err
		}
		sql := fmt.Sprintf("DELETE FROM patruljemerged WHERE teamId=%q OR parentTeamId=%q", body.TeamID, body.TeamID)
		if err := c.w.Consume(sql); err != nil {
			log.Fatalf("Error consuming sql %q", err)
		}
	}
	return nil
}
