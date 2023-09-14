package table

import (
	"database/sql"
	"fmt"
	"log"

	"nathejk.dk/nathejk/messages"
	"nathejk.dk/nathejk/types"
	"nathejk.dk/pkg/stream/entity"
	"nathejk.dk/pkg/streaminterface"
	"nathejk.dk/pkg/tablerow"

	_ "embed"
)

type ControlGroupTableEvent struct {
	ID types.ControlGroupID
}

type ControlGroup struct {
	ID       types.ControlGroupID
	YearSlug string
	Name     string
}

type controlgroup struct {
	w  tablerow.Consumer
	p  entity.EntityChangedPublisher
	db *sql.DB
}

func NewControlGroup(w tablerow.Consumer, p streaminterface.Publisher) *controlgroup {
	table := &controlgroup{w: w, p: entity.NewEntityChangedPublisher(p, "controlgroup")}
	if err := w.Consume(table.CreateTableSql()); err != nil {
		log.Fatalf("Error creating table %q", err)
	}
	return table
}

//go:embed controlgroup.sql
var controlgroupSchema string

func (t *controlgroup) CreateTableSql() string {
	return controlgroupSchema
}

func (c *controlgroup) Consumes() (subjs []streaminterface.Subject) {
	return []streaminterface.Subject{
		streaminterface.SubjectFromStr("nathejk"),
	}
}

func (c *controlgroup) HandleMessage(msg streaminterface.Message) error {
	switch msg.Subject().Subject() {
	case "nathejk:controlgroup.updated":
		var body messages.NathejkControlGroupUpdated
		if err := msg.Body(&body); err != nil {
			return err
		}
		year := msg.Time().Year()
		sql := fmt.Sprintf("INSERT INTO controlgroup SET id=%q, name=%q, year=\"%d\" ON DUPLICATE KEY UPDATE name=VALUES(name)", body.ControlGroupID, body.Name, year)
		if err := c.w.Consume(sql); err != nil {
			log.Fatalf("Error consuming sql %q", err)
		}
		c.p.Changed(&ControlGroupTableEvent{})

	case "nathejk:controlgroup.deleted":
		var body messages.NathejkControlGroupDeleted
		if err := msg.Body(&body); err != nil {
			return err
		}
		err := c.w.Consume(fmt.Sprintf("DELETE FROM controlgroup WHERE id=%q", body.ControlGroupID))
		if err != nil {
			log.Fatalf("Error consuming sql %q", err)
		}
		c.p.Deleted(&ControlGroupTableEvent{})

	}
	return nil
}

type CheckGroupScan struct {
	TeamID         types.TeamID
	TeamNumber     string
	Uts            int
	UserID         string
	ControlGroupID types.ControlGroupID
	ControlIndex   int
	OnTime         bool
}

func (cg *controlgroup) AllScans(year string) []CheckGroupScan {
	rows, err := cg.db.Query(`
SELECT
	s.teamId,
	s.teamNumber,
	s.uts,
	s.userId,
	cp.controlGroupId,
	cp.controlIndex,
	(cp.openFromUts - 60*cp.minusMinutes <= s.uts AND s.uts <= cp.openUntilUts + 60*plusMinutes) AS ontime
FROM scan
  JOIN controlgroup_user cgu ON scan.scannerId = cgu.userId AND startUts <= uts AND uts <= endUts
  JOIN controlpoint cp ON cgu.controlGroupId = cp.controlGroupId AND cgu.controlIndex = cp.controlIndex`)
	if err != nil {
		log.Fatalf("Query: %v", err)
	}
	ss := []CheckGroupScan{}
	for rows.Next() {
		var s CheckGroupScan
		err = rows.Scan(&s.TeamID, &s.TeamNumber, &s.Uts, &s.UserID, &s.ControlGroupID, &s.ControlIndex, &s.OnTime)
		if err != nil {
			log.Fatalf("Scan: %v", err)
		}
		ss = append(ss, s)
	}
	return ss
}
