package table

import (
	"fmt"
	"log"

	"github.com/jrgensen/cqrs"
	"github.com/jrgensen/stream"
	"github.com/jrgensen/stream/subject"
	"github.com/nathejk/shared-go/messages"
	"github.com/nathejk/shared-go/types"

	_ "embed"
)

type Pincode struct {
	TeamID  types.TeamID `sql:"teamId"`
	Pincode string       `sql:"pincode"`
}

type pincode struct {
	w cqrs.Writer
}

func NewPincode(w cqrs.Writer) *pincode {
	table := &pincode{w: w}
	if err := w.Consume(table.CreateTableSql()); err != nil {
		log.Printf("Error creating table %q", err)
	}
	return table
}

//go:embed pincode.sql
var pincodeSchema string

func (t *pincode) CreateTableSql() string {
	return pincodeSchema
}

func (c *pincode) Consumes() (subjs []stream.Subject) {
	return []stream.Subject{
		subject.FromStr("nathejk"),
	}
}

func (c *pincode) HandleMessage(msg stream.Message) error {
	switch msg.Subject().Subject() {
	case "nathejk:patrulje.signedup", "nathejk:klan.signedup":
		var body messages.NathejkTeamSignedUp
		if err := msg.Body(&body); err != nil {
			return err
		}
		return c.w.Consume(fmt.Sprintf("INSERT INTO pincode SET teamId=%q, pincode=%q ON DUPLICATE KEY UPDATE pincode=VALUES(pincode)", body.TeamID, body.Pincode))
	}
	return nil
}
