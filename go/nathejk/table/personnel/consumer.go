package personnel

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/jrgensen/stream"
	"github.com/jrgensen/stream/subject"
	"github.com/nathejk/shared-go/messages"
	"nathejk.dk/pkg/tablerow"

	_ "embed"
)

type consumer struct {
	w tablerow.Consumer
}

func (*consumer) Consumes() []stream.Subject {
	return []stream.Subject{
		subject.FromStr("NATHEJK.*.gøgler.*.signedup"),
		subject.FromStr("NATHEJK.*.gøgler.*.updated"),
		subject.FromStr("NATHEJK.*.gøgler.*.status.changed"),
		subject.FromStr("NATHEJK.*.friend.*.signedup"),
		subject.FromStr("NATHEJK.*.friend.*.updated"),
		subject.FromStr("NATHEJK.*.friend.*.status.changed"),
		subject.FromStr("NATHEJK.*.bandit.*.armNumber.assigned"),
	}
}

func (c *consumer) HandleMessage(msg stream.Message) error {
	switch true {

	case msg.Subject().Match("NATHEJK.*.*.*.signedup"):
		var body messages.NathejkTeamSignedUp
		if err := msg.Body(&body); err != nil {
			return err
		}
		if body.TeamID == "" {
			return nil
		}
		subject := msg.Subject().Parts()
		args := []any{body.TeamID, subject[2], subject[1], body.Name, body.Phone, body.Email}
		sql := fmt.Sprintf("INSERT IGNORE INTO personnel SET userId=%q, userType=%q, year=%q, name=%q, phone=%q, email=%q", args...)
		return c.w.Consume(sql)

	case msg.Subject().Match("NATHEJK.*.*.*.updated"):
		var body messages.NathejkPersonnelUpdated
		if err := msg.Body(&body); err != nil {
			return err
		}
		additionals, _ := json.Marshal(body.Additionals)
		msg.Subject().Parts()
		query := "UPDATE personnel SET name=%q, groupName=%q, korps=%q, klan=%q, phone=%q, email=%q, tshirtSize=%q, additionals=%q  WHERE userId=%q"
		args := []any{body.Name, body.Group, string(body.Corps), body.Klan, body.Phone, body.Email, body.TshirtSize, additionals, body.UserID}

		return c.w.Consume(fmt.Sprintf(query, args...))

	case msg.Subject().Match("NATHEJK.*.bandit.*.armNumber.assigned"):
		var body messages.NathejkLokArmNumberAssigned
		if err := msg.Body(&body); err != nil {
			return err
		}
		query := "UPDATE personnel SET armNumber=%q WHERE userId=%q"
		args := []any{
			body.ArmNumber,
			msg.Subject().Parts()[3],
		}
		return c.w.Consume(fmt.Sprintf(query, args...))
		/*
			case msg.Subject().Match("NATHEJK.*.staff.*.status.changed"):
				var body messages.NathejkStaffStatusChanged
				if err := msg.Body(&body); err != nil {
					return err
				}
				err := c.w.Consume(fmt.Sprintf("UPDATE staff SET signupStatus=%q WHERE staffId=%q", body.Status, body.StaffID))
				if err != nil {
					log.Fatalf("Error consuming sql %q", err)
				}
		*/
	default:
		log.Printf("Unhandled message %q", msg.Subject().Subject())
	}
	return nil
}
