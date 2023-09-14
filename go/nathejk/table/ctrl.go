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

type Control struct {
	ControlGroupID   types.ControlGroupID
	ControlGroupName string
	ControlIndex     int
	ControlName      string
	OpenFrom         time.Time
	OpenUntil        time.Time
	Plus             int
	Minus            int
}
type ControlTableEvent struct {
	UserID types.UserID
}

type controlPoint struct {
	w tablerow.Consumer
	p entity.EntityChangedPublisher
}

func NewControlPoint(w tablerow.Consumer, p streaminterface.Publisher) *controlPoint {
	table := &controlPoint{w: w, p: entity.NewEntityChangedPublisher(p, "controlPoint")}
	if err := w.Consume(table.CreateTableSql()); err != nil {
		log.Fatalf("Error creating table %q", err)
	}
	return table
}

//go:embed ctrl.sql
var controlPointSchema string

func (t *controlPoint) CreateTableSql() string {
	return controlPointSchema
}

func (c *controlPoint) Consumes() (subjs []streaminterface.Subject) {
	return []streaminterface.Subject{
		streaminterface.SubjectFromStr("nathejk"),
	}
}

func (c *controlPoint) HandleMessage(msg streaminterface.Message) error {
	switch msg.Subject().Subject() {
	case "nathejk:controlgroup.updated":
		var body messages.NathejkControlGroupUpdated
		if err := msg.Body(&body); err != nil {
			return err
		}
		if err := c.w.Consume(fmt.Sprintf("DELETE FROM controlpoint WHERE controlGroupId=%q", body.ControlGroupID)); err != nil {
			return err
		}
		for index, control := range body.Controls {
			//for _, scanner := range control.Scanners {
			args := []any{
				body.ControlGroupID,
				body.Name,
				index,
				control.Name,
				control.Scheme,
				control.RelativeCheckgroupID,
				control.DateRange.StartDate.Unix(),
				control.DateRange.EndDate.Unix(),
				control.Plus,
				control.Minus,
			}

			sql := fmt.Sprintf("INSERT INTO controlpoint (controlGroupId, controlGroupName, controlIndex, controlName, scheme, relativeControlGroupId, openFromUts, openUntilUts, plusMinutes, minusMinutes) VALUES (%q, %q, %d, %q, %q, %q, %d, %d, %d, %d)", args...)
			if err := c.w.Consume(sql); err != nil {
				return nil
			}
			c.p.Changed(&PersonnelTableEvent{})
			//}
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
