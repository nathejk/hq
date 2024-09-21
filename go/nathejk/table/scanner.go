package table

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"nathejk.dk/nathejk/messages"
	"nathejk.dk/nathejk/types"
	"nathejk.dk/pkg/stream/entity"
	"nathejk.dk/pkg/streaminterface"
	"nathejk.dk/pkg/tablerow"

	_ "embed"
)

type Scanner struct {
	UserID     types.UserID
	Year       string
	Name       string
	Number     int
	Phone      types.PhoneNumber
	TeamID     types.TeamID
	Department string
	Created    time.Time
	Updated    time.Time
}
type ScannerTableEvent struct {
	UserID types.UserID
}

type scanner struct {
	w tablerow.Consumer
	p entity.EntityChangedPublisher
}

func NewScanner(w tablerow.Consumer, p streaminterface.Publisher) *scanner {
	table := &scanner{w: w, p: entity.NewEntityChangedPublisher(p, "personnel")}
	if err := w.Consume(table.CreateTableSql()); err != nil {
		log.Fatalf("Error creating table %q", err)
	}
	return table
}

//go:embed scanner.sql
var scannerSchema string

func (t *scanner) CreateTableSql() string {
	return scannerSchema
}

func (c *scanner) Consumes() (subjs []streaminterface.Subject) {
	return []streaminterface.Subject{
		streaminterface.SubjectFromStr("monolith:nathejk_member"),
		streaminterface.SubjectFromStr("nathejk"),
	}
}

func (c *scanner) HandleMessage(msg streaminterface.Message) error {
	switch msg.Subject().Subject() {
	case "nathejk:personnel.updated":
		var body messages.NathejkPersonnelUpdated
		if err := msg.Body(&body); err != nil {
			return err
		}
		bandit := map[string]int{"bhq": 1, "bandit": 1, "lok1": 1, "lok2": 1, "lok3": 1, "lok4": 1, "lok5": 1}
		err := c.w.Consume(fmt.Sprintf("INSERT INTO scanner (userId, name, bandit,  phone, department, createdAt, updatedAt) VALUES (%q,%q,%d,%q,%q,%q,%q) ON DUPLICATE KEY UPDATE  name=VALUES(name), bandit=VALUES(bandit), phone=VALUES(phone), teamId=VALUES(teamId), department=VALUES(department), updatedAt=VALUES(updatedAt)", body.UserID, body.Name, bandit[body.Department], body.Phone, body.Department, msg.Time(), msg.Time()))
		if err != nil {
			log.Fatalf("Error consuming sql %q", err)
		}
		c.p.Changed(&ScannerTableEvent{UserID: body.UserID})

	case "nathejk:personnel.deleted":
		var body messages.NathejkPersonnelDeleted
		if err := msg.Body(&body); err != nil {
			return err
		}
		err := c.w.Consume(fmt.Sprintf("DELETE FROM personnel WHERE userId=%q", body.UserID))
		if err != nil {
			log.Fatalf("Error consuming sql %q", err)
		}
		c.p.Deleted(&ScannerTableEvent{UserID: body.UserID})

	case "monolith:nathejk_member":
		var body messages.MonolithNathejkMember
		if err := msg.Body(&body); err != nil {
			return err
		}
		if body.Entity.Number == "" {
			break
		}

		createdAt := time.Time{}
		year := ""
		if body.Entity.CreatedUts.Time() != nil {
			createdAt = *body.Entity.CreatedUts.Time()
			year = fmt.Sprintf("%d", createdAt.Year())
		}
		query := "INSERT INTO scanner SET userId=%q, year=%q, bandit=%d, teamId=%q, name=%q, department=%q, phone=%q, createdAt=%q, updatedAt=%q ON DUPLICATE KEY UPDATE name=VALUES(name), bandit=VALUES(bandit), phone=VALUES(phone), department=VALUES(department), updatedAt=VALUES(updatedAt)"
		banditNumber, _ := strconv.Atoi(body.Entity.Number)
		if banditNumber == 0 {
			banditNumber = 1
		}
		args := []any{
			body.Entity.ID,
			year,
			banditNumber,
			body.Entity.TeamID,
			body.Entity.Title,
			"bandit",
			body.Entity.Phone,
			createdAt,
			"",
		}

		sql := fmt.Sprintf(query, args...)
		if err := c.w.Consume(sql); err != nil {
			log.Printf("Error consuming sql %q", err)
		}
	}
	return nil
}
