package table

import (
	_ "embed"
	"fmt"
	"log"

	"nathejk.dk/nathejk/messages"
	"nathejk.dk/nathejk/types"
	"nathejk.dk/pkg/streaminterface"
	"nathejk.dk/pkg/tablerow"
)

type SosAssociation struct {
	SosID  types.SosID  `json:"sosId"`
	TeamID types.TeamID `json:"teamId"`
}

type sosassoc struct {
	w tablerow.Consumer
}

func NewSosAssociation(w tablerow.Consumer) *sosassoc {
	table := &sosassoc{w: w}
	if err := w.Consume(table.CreateTableSql()); err != nil {
		log.Fatalf("Error creating table %q", err)
	}
	return table
}

//go:embed sosassoc.sql
var sosAssociationSchema string

func (t *sosassoc) CreateTableSql() string {
	return sosAssociationSchema
}

func (c *sosassoc) Consumes() (subjs []streaminterface.Subject) {
	strings := []string{
		"nathejk:sos.team.associated",
		"nathejk:sos.team.disassociated",
	}
	subjects := []streaminterface.Subject{}
	for _, subject := range strings {
		subjects = append(subjects, streaminterface.SubjectFromStr(subject))
	}
	return subjects
}
func (m *sosassoc) HandleMessage(msg streaminterface.Message) error {
	switch msg.Subject().Subject() {
	case "nathejk:sos.team.associated":
		var body messages.NathejkSosTeamAssociated
		if err := msg.Body(&body); err != nil {
			return err
		}
		if body.SosID == "" || body.TeamID == "" {
			return nil
		}
		query := "INSERT IGNORE INTO sos_team SET sosId=%q, teamId=%q"
		args := []any{
			body.SosID,
			body.TeamID,
		}
		sql := fmt.Sprintf(query, args...)
		if err := m.w.Consume(sql); err != nil {
			log.Fatalf("Error consuming sql %q", err)
		}

	case "nathejk:sos.team.disassociated":
		var body messages.NathejkSosTeamDisassociated
		if err := msg.Body(&body); err != nil {
			return err
		}
		query := "DELETE FROM sos_team WHERE sosId=%q AND teamId=%q"
		args := []any{
			body.SosID,
			body.TeamID,
		}
		sql := fmt.Sprintf(query, args...)
		if err := m.w.Consume(sql); err != nil {
			log.Fatalf("Error consuming sql %q", err)
		}

	default:
		log.Printf("Unsupported message type %q", msg.Subject().Subject())
	}
	return nil
}
