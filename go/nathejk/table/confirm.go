package table

import (
	"fmt"
	"log"

	"github.com/jrgensen/cqrs"
	"github.com/jrgensen/stream"
	"github.com/jrgensen/stream/subject"
	"github.com/nathejk/shared-go/messages"
	"github.com/nathejk/shared-go/types"
)

type confirm struct {
	w cqrs.Writer
}

func NewConfirm(w cqrs.Writer) *confirm {
	table := &confirm{w: w}
	if err := w.Consume(table.CreateTableSql()); err != nil {
		log.Printf("Error creating table %q", err)
	}
	return table
}

func (t *confirm) CreateTableSql() string {
	return `
CREATE TABLE IF NOT EXISTS confirm (
    teamId VARCHAR(99) NOT NULL,
    emailPending VARCHAR(99) NOT NULL,
    secret VARCHAR(99),
    PRIMARY KEY (teamId)
);
`
}

func (t *confirm) Consumes() []stream.Subject {
	return []stream.Subject{
		subject.FromStr(fmt.Sprintf("NATHEJK:%s.*.*.mail.%s.sent", "2024", types.PingTypeSignup)),
	}
}

func (t *confirm) HandleMessage(msg stream.Message) error {
	switch msg.Subject().Subject() {
	//case "NATHEJK.year.created":
	default:
		var body messages.NathejkMailSent
		if err := msg.Body(&body); err != nil {
			return err
		}
		var meta messages.Metadata
		if err := msg.Meta(&meta); err != nil {
			return err
		}
		sql := "INSERT INTO confirm SET teamId=%q, emailPending=%q, secret=%q ON DUPLICATE KEY UPDATE emailPending=VALUES(emailPending), secret=VALUES(secret)"
		args := []any{
			body.TeamID,
			body.Recipient,
			meta.Phase,
		}
		if err := t.w.Consume(fmt.Sprintf(sql, args...)); err != nil {
			return err
		}
		//default:
		//	return fmt.Errorf("unhandled subject %q", msg.Subject().Subject())
	}
	return nil
}

/*
func escapeNull(s *time.Time) string {
	if s == nil {
		return "NULL"
	}
	return fmt.Sprintf("%q", s)
}*/
