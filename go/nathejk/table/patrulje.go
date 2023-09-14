package table

import (
	"fmt"
	"log"
	"strconv"

	"github.com/davecgh/go-spew/spew"
	"nathejk.dk/nathejk/messages"
	"nathejk.dk/nathejk/types"
	"nathejk.dk/pkg/streaminterface"
	"nathejk.dk/pkg/tablerow"

	_ "embed"
)

type Patrulje struct {
	TeamID       types.TeamID       `sql:"teamId"`
	Year         string             `sql:"year"`
	Name         string             `sql:"name"`
	GroupName    string             `sql:"groupName"`
	Korps        string             `sql:"korps"`
	ContactName  string             `sql:"contactName"`
	ContactPhone types.PhoneNumber  `sql:"contactPhone"`
	ContactEmail types.Email        `sql:"contactEmail"`
	ContactRole  string             `sql:"contactRole"`
	SignupStatus types.SignupStatus `sql:"signupStatus"`
}

type patrulje struct {
	w tablerow.Consumer
}

func NewPatrulje(w tablerow.Consumer) *patrulje {
	table := &patrulje{w: w}
	if err := w.Consume(table.CreateTableSql()); err != nil {
		log.Fatalf("Error creating table %q", err)
	}
	return table
}

//go:embed patrulje.sql
var patruljeSchema string

func (t *patrulje) CreateTableSql() string {
	return patruljeSchema
}

func (c *patrulje) Consumes() (subjs []streaminterface.Subject) {
	return []streaminterface.Subject{
		streaminterface.SubjectFromStr("monolith:nathejk_team"),
		streaminterface.SubjectFromStr("nathejk"),
	}
}

func (c *patrulje) HandleMessage(msg streaminterface.Message) error {
	switch msg.Subject().Subject() {
	case "nathejk:patrulje.signedup":
		var body messages.NathejkTeamSignedUp
		if err := msg.Body(&body); err != nil {
			return err
		}
		if body.TeamID == "" {
			return nil
		}
		sql := fmt.Sprintf("INSERT INTO patrulje SET teamId=%q, year=\"%d\", contactName=%q, contactPhone=%q, contactEmail=%q ON DUPLICATE KEY UPDATE contactName=VALUES(contactName), contactPhone=VALUES(contactPhone), contactEmail=VALUES(contactEmail)", body.TeamID, msg.Time().Year(), body.Name, body.Phone, body.Email)
		if err := c.w.Consume(sql); err != nil {
			log.Fatalf("Error consuming sql %q", err)
		}

	case "nathejk:patrulje.updated":
		var body messages.NathejkTeamUpdated
		if err := msg.Body(&body); err != nil {
			return err
		}
		err := c.w.Consume(fmt.Sprintf("UPDATE patrulje SET name=%q, groupName=%q, korps=%q, contactName=%q, contactPhone=%q, contactEmail=%q, contactRole=%q WHERE teamId=%q", body.Name, body.GroupName, body.Korps, body.ContactName, body.ContactPhone, body.ContactEmail, body.ContactRole, body.TeamID))
		if err != nil {
			log.Fatalf("Error consuming sql %q", err)
		}

	case "nathejk:patrulje.status.changed":
		var body messages.NathejkPatruljeStatusChanged
		if err := msg.Body(&body); err != nil {
			return err
		}
		err := c.w.Consume(fmt.Sprintf("UPDATE klan SET signupStatus=%q WHERE teamId=%q", body.Status, body.TeamID))
		if err != nil {
			log.Fatalf("Error consuming sql %q", err)
		}

	case "monolith:nathejk_team":
		var body messages.MonolithNathejkTeam
		if err := msg.Body(&body); err != nil {
			spew.Dump(msg)
			log.Print(err)
			return nil
		}
		if body.Entity.TypeName != types.TeamTypePatrulje {
			return nil
		}
		var sql string
		if body.Entity.DeletedUts.Time() == nil {
			//spew.Dump(body, body.Entity.CreatedUts.Time())
			if body.Entity.CreatedUts.Time() == nil {
				return nil
			}
			var memberCount int64
			if body.Entity.MemberCount != "" {
				memberCount, _ = strconv.ParseInt(body.Entity.MemberCount, 10, 64)
			}

			query := "INSERT INTO patrulje SET teamId=%q, year=\"%d\", teamNumber=%q, name=%q, groupName=%q, korps=%q, memberCount=%d, contactName=%q, contactPhone=%q, contactEmail=%q, signupStatus=%q  ON DUPLICATE KEY UPDATE teamNumber=VALUES(teamNumber), name=VALUES(name), groupName=VALUES(groupName), korps=VALUES(korps), memberCount=VALUES(memberCount), contactName=VALUES(contactName), contactPhone=VALUES(contactPhone), contactEmail=VALUES(contactEmail), signupStatus=VALUES(signupStatus)"
			args := []any{
				body.Entity.ID,
				body.Entity.CreatedUts.Time().Year(),
				body.Entity.TeamNumber,
				body.Entity.Title,
				body.Entity.Gruppe,
				body.Entity.Korps,
				memberCount,
				body.Entity.ContactTitle,
				body.Entity.ContactPhone,
				body.Entity.ContactMail,
				body.Entity.SignupStatusTypeName,
			}

			sql = fmt.Sprintf(query, args...)
		} else {
			sql = fmt.Sprintf("DELETE FROM patrulje WHERE teamId=%q", body.Entity.ID)
		}
		if err := c.w.Consume(sql); err != nil {
			log.Printf("Error consuming sql %q", err)
		}
	}
	return nil
}
