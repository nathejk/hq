package table

import (
	"fmt"
	"log"

	"nathejk.dk/nathejk/messages"
	"nathejk.dk/nathejk/types"
	"nathejk.dk/pkg/streaminterface"
	"nathejk.dk/pkg/tablerow"

	_ "embed"
)

type PatruljeMerged struct {
	TeamID       types.TeamID `sql:"teamId"`
	ParentTeamID types.TeamID `sql:"teamId"`
}

type patruljeMerged struct {
	w tablerow.Consumer
}

func NewPatruljeMerged(w tablerow.Consumer) *patruljeMerged {
	table := &patruljeMerged{w: w}
	if err := w.Consume(table.CreateTableSql()); err != nil {
		log.Fatalf("Error creating table %q", err)
	}
	return table
}

//go:embed patruljemerged.sql
var patruljeMergedSchema string

func (t *patruljeMerged) CreateTableSql() string {
	return patruljeMergedSchema
}

func (c *patruljeMerged) Consumes() (subjs []streaminterface.Subject) {
	return []streaminterface.Subject{
		streaminterface.SubjectFromStr("nathejk"),
	}
}

func (c *patruljeMerged) HandleMessage(msg streaminterface.Message) error {
	switch msg.Subject().Subject() {
	case "nathejk:team.merged":
		var body messages.NathejkTeamMerged
		if err := msg.Body(&body); err != nil {
			return err
		}
		sql := fmt.Sprintf("INSERT INTO patruljemerged SET teamId=%q, parentTeamId=%q ON DUPLICATE KEY UPDATE teamId=VALUES(teamId)", body.TeamID, body.ParentTeamID)
		if err := c.w.Consume(sql); err != nil {
			log.Fatalf("Error consuming sql %q", err)
		}
	case "nathejk:team.splited":
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
