package table

import (
	_ "embed"
	"fmt"
	"log"
	"time"

	"nathejk.dk/nathejk/messages"
	"nathejk.dk/nathejk/types"
	"nathejk.dk/pkg/streaminterface"
	"nathejk.dk/pkg/tablerow"
)

type Sos struct {
	SosID       types.SosID  `json:"sosId"`
	Year        string       `json:"year"`
	Headline    string       `json:"headline"`
	Description string       `json:"description"`
	CreatedAt   time.Time    `json:"createdAt"`
	CreatedBy   types.UserID `json:"createdBy"`

	Severity types.Enum `json:"severity"`
	Status   types.Enum `json:"status"`
}

type sos struct {
	w tablerow.Consumer
}

func NewSos(w tablerow.Consumer) *sos {
	table := &sos{w: w}
	if err := w.Consume(table.CreateTableSql()); err != nil {
		log.Fatalf("Error creating table %q", err)
	}
	return table
}

//go:embed sos.sql
var sosSchema string

func (t *sos) CreateTableSql() string {
	return sosSchema
}

func (c *sos) Consumes() (subjs []streaminterface.Subject) {
	strings := []string{
		"nathejk:sos.created",
		"nathejk:sos.headline.updated",
		"nathejk:sos.description.updated",
		"nathejk:sos.severity.specified",
		"nathejk:sos.deleted",
		"nathejk:sos.closed",
		"nathejk:sos.reopened",
	}
	subjects := []streaminterface.Subject{}
	for _, subject := range strings {
		subjects = append(subjects, streaminterface.SubjectFromStr(subject))
	}
	return subjects
}
func (m *sos) HandleMessage(msg streaminterface.Message) error {
	switch msg.Subject().Subject() {
	case "nathejk:sos.created":
		var body messages.NathejkSosCreated
		if err := msg.Body(&body); err != nil {
			return err
		}
		query := "INSERT INTO sos SET id=%q, year=\"%d\", headline=%q, description=%q, createdAt=%q, createdBy=%q, status='open' ON DUPLICATE KEY UPDATE headline=VALUES(headline), description=VALUES(description)"
		args := []any{
			body.SosID,
			msg.Time().Year(),
			body.Headline,
			body.Description,
			msg.Time(),
			body.UserID,
		}
		sql := fmt.Sprintf(query, args...)
		err := m.w.Consume(sql)
		if err != nil {
			log.Fatalf("Error consuming sql %q", err)
		}

	case "nathejk:sos.headline.updated":
		var body messages.NathejkSosHeadlineUpdated
		if err := msg.Body(&body); err != nil {
			return err
		}
		query := "UPDATE sos SET headline=%q WHERE id=%q"
		args := []any{body.Headline, body.SosID}
		sql := fmt.Sprintf(query, args...)
		err := m.w.Consume(sql)
		if err != nil {
			log.Fatalf("Error consuming sql %q", err)
		}

	case "nathejk:sos.closed":
		var body messages.NathejkSosClosed
		if err := msg.Body(&body); err != nil {
			return err
		}
		query := "UPDATE sos SET status=%q WHERE id=%q"
		args := []any{"closed", body.SosID}
		sql := fmt.Sprintf(query, args...)
		err := m.w.Consume(sql)
		if err != nil {
			log.Fatalf("Error consuming sql %q", err)
		}

	case "nathejk:sos.reopened":
		var body messages.NathejkSosReopened
		if err := msg.Body(&body); err != nil {
			return err
		}
		query := "UPDATE sos SET status=%q WHERE id=%q"
		args := []any{"open", body.SosID}
		sql := fmt.Sprintf(query, args...)
		err := m.w.Consume(sql)
		if err != nil {
			log.Fatalf("Error consuming sql %q", err)
		}

	case "nathejk:sos.severity.specified":
		var body messages.NathejkSosSeveritySpecified
		if err := msg.Body(&body); err != nil {
			return err
		}
		query := "UPDATE sos SET severity=%q WHERE id=%q"
		args := []any{body.Severity, body.SosID}
		sql := fmt.Sprintf(query, args...)
		err := m.w.Consume(sql)
		if err != nil {
			log.Fatalf("Error consuming sql %q", err)
		}

	default:
		log.Printf("Unsupported message type %q", msg.Subject().Subject())
	}
	return nil
}
