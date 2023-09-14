package table

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"nathejk.dk/nathejk/messages"
	"nathejk.dk/nathejk/types"
	"nathejk.dk/pkg/streaminterface"
	"nathejk.dk/pkg/tablerow"

	_ "embed"
)

type PatruljeStatus struct {
	TeamID       types.TeamID       `sql:"teamId"`
	Name         string             `sql:"name"`
	GroupName    string             `sql:"groupName"`
	Korps        string             `sql:"korps"`
	ContactName  string             `sql:"contactName"`
	ContactPhone types.PhoneNumber  `sql:"contactPhone"`
	ContactEmail types.Email        `sql:"contactEmail"`
	ContactRole  string             `sql:"contactRole"`
	SignupStatus types.SignupStatus `sql:"signupStatus"`
	Started      uint
}

type patruljeStatus struct {
	w tablerow.Consumer
}

func NewPatruljeStatus(w tablerow.Consumer) *patruljeStatus {
	table := &patruljeStatus{w: w}
	if err := w.Consume(table.CreateTableSql()); err != nil {
		log.Fatalf("Error creating table %q %q", err, table.CreateTableSql())
	}
	return table
}

//go:embed patruljestatus.sql
var patruljeStatusSchema string

func (t *patruljeStatus) CreateTableSql() string {
	return patruljeStatusSchema
}

func (c *patruljeStatus) Consumes() (subjs []streaminterface.Subject) {
	return []streaminterface.Subject{
		streaminterface.SubjectFromStr("monolith:nathejk_team"),
	}
}

func (c *patruljeStatus) HandleMessage(msg streaminterface.Message) error {
	if msg.Time().Year() != time.Now().Year() {
		// only handle messages from this year
		//return nil
	}
	switch msg.Subject().Subject() {
	case "monolith:nathejk_team":
		var body messages.Nathejk_Team
		if err := msg.Body(&body); err != nil {
			return err
		}
		//		if body.Entity.ID < "2022000" {
		//log.Printf("TeamNumber: %q %q", body.Entity.ID, body.Entity.TeamNumber)
		//			return nil
		//		}

		if body.Entity.TeamNumber == "0" {
			//log.Printf("TeamNumber: %q %q", body.Entity.ID, body.Entity.TeamNumber)
			return nil
		}
		if body.Entity.DeletedUts != "0" {
			//log.Printf("Deleted: %q %q", body.Entity.ID, body.Entity.DeletedUts)

			return c.w.Consume(fmt.Sprintf("DELETE FROM patruljestatus WHERE teamId=%q", body.Entity.ID))
		}

		uts, _ := strconv.ParseInt(body.Entity.CreatedUts, 10, 64)
		year := time.Unix(uts, 0).Year()
		startedUts, _ := strconv.Atoi(body.Entity.StartUts)
		sql := fmt.Sprintf("INSERT INTO patruljestatus SET teamId=%q, year=\"%d\", startedUts=%d ON DUPLICATE KEY UPDATE startedUts=VALUES(startedUts)", body.Entity.ID, year, startedUts)
		if err := c.w.Consume(sql); err != nil {
			log.Fatalf("Error consuming sql %q", err)
		}
	}
	return nil
}
