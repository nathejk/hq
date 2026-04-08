package lok

import (
	"fmt"
	"log"
	"strings"

	"github.com/nathejk/shared-go/messages"
	"nathejk.dk/pkg/tablerow"
	"nathejk.dk/superfluids/streaminterface"
)

type consumer struct {
	w tablerow.Consumer
}

func (c *consumer) Consumes() []streaminterface.Subject {
	return []streaminterface.Subject{
		streaminterface.SubjectFromStr("NATHEJK.*.lok.*.updated"),
		streaminterface.SubjectFromStr("NATHEJK.*.lok.*.deleted"),
	}
}

func (c *consumer) HandleMessage(msg streaminterface.Message) error {
	switch true {
	case msg.Subject().Match("NATHEJK.*.lok.*.updated"):
		var body messages.NathejkLokUpdated
		if err := msg.Body(&body); err != nil {
			return err
		}
		var userIDs, teamIDs []string
		for _, ID := range body.UserIDs {
			userIDs = append(userIDs, string(ID))
		}
		for _, ID := range body.TeamIDs {
			teamIDs = append(teamIDs, string(ID))
		}
		sql := "INSERT INTO lok SET lokId=%q, year=%q, name=%q, sortOrder=%d, userIds=%q, teamIds=%q ON DUPLICATE KEY UPDATE name=VALUES(name), sortOrder=VALUES(sortOrder), userIds=VALUES(userIds), teamIds=VALUES(teamIds)"
		args := []any{body.LokID, msg.Subject().Parts()[1], body.Name, body.SortOrder, strings.Join(userIDs, ","), strings.Join(teamIDs, ",")}
		if err := c.w.Consume(fmt.Sprintf(sql, args...)); err != nil {
			log.Fatalf("Error consuming sql %q", err)
		}
	case msg.Subject().Match("NATHEJK.*.lok.*.deleted"):
		var body messages.NathejkLokDeleted
		if err := msg.Body(&body); err != nil {
			return err
		}
		sql := "DELETE FROM lok WHERE lokId=%q"
		args := []any{body.LokID}
		if err := c.w.Consume(fmt.Sprintf(sql, args...)); err != nil {
			log.Fatalf("Error consuming sql %q", err)
		}

	default:
		log.Printf("Unhandled message %q", msg.Subject().Subject())

	}
	return nil
}
