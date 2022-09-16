package table

import (
	"fmt"
	"log"
	"time"

	"nathejk.dk/nathejk/messages"
	"nathejk.dk/nathejk/types"
	"nathejk.dk/pkg/stream/entity"
	"nathejk.dk/pkg/streaminterface"
	"nathejk.dk/pkg/tablerow"

	_ "embed"
)

type ControlGroupUser struct {
	ControlGroupID types.ControlGroupID
	ControlIndex   int
	UserID         types.UserID
	StartDate      time.Time
	EndDate        time.Time
}
type ControlGroupUserTableEvent struct {
	UserID types.UserID
}

type controlGroupUser struct {
	w tablerow.Consumer
	p entity.EntityChangedPublisher
}

func NewControlGroupUser(w tablerow.Consumer, p streaminterface.Publisher) *controlGroupUser {
	table := &controlGroupUser{w: w, p: entity.NewEntityChangedPublisher(p, "controlGroupUser")}
	if err := w.Consume(table.CreateTableSql()); err != nil {
		log.Fatalf("Error creating table %q", err)
	}
	return table
}

//go:embed ctrlgrpuser.sql
var controlGroupUserSchema string

func (t *controlGroupUser) CreateTableSql() string {
	return controlGroupUserSchema
}

func (c *controlGroupUser) Consumes() (subjs []streaminterface.Subject) {
	return []streaminterface.Subject{
		streaminterface.SubjectFromStr("nathejk"),
	}
}

func (c *controlGroupUser) HandleMessage(msg streaminterface.Message) error {
	switch msg.Subject().Subject() {
	case "nathejk:controlgroup.updated":
		var body messages.NathejkControlGroupUpdated
		if err := msg.Body(&body); err != nil {
			return err
		}
		if err := c.w.Consume(fmt.Sprintf("DELETE FROM controlgroup_user WHERE controlGroupId=%q", body.ControlGroupID)); err != nil {
			return err
		}
		for index, control := range body.Controls {
			for _, scanner := range control.Scanners {
				sql := fmt.Sprintf("INSERT INTO controlgroup_user (controlGroupId, controlIndex, userId, startUts, endUts) VALUES (%q, %d, %q, %d, %d)", body.ControlGroupID, index, scanner.UserID, scanner.DateRange.StartDate.Unix(), scanner.DateRange.EndDate.Unix())
				if err := c.w.Consume(sql); err != nil {
					return nil
				}
				c.p.Changed(&PersonnelTableEvent{UserID: scanner.UserID})
			}
		}

	case "nathejk:controlgroup.deleted":
		var body messages.NathejkControlGroupDeleted
		if err := msg.Body(&body); err != nil {
			return err
		}
		err := c.w.Consume(fmt.Sprintf("DELETE FROM controlgroup_user WHERE controlGroupId=%q", body.ControlGroupID))
		if err != nil {
			log.Fatalf("Error consuming sql %q", err)
		}
		c.p.Deleted(&ControlGroupUserTableEvent{})

	}
	return nil
}
