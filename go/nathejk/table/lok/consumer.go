package lok

import (
	"fmt"
	"log"
	"strings"

	"github.com/jrgensen/stream"
	"github.com/jrgensen/stream/subject"
	"github.com/nathejk/shared-go/messages"
	"nathejk.dk/pkg/tablerow"
)

type consumer struct {
	w tablerow.Consumer
}

func (c *consumer) Consumes() []stream.Subject {
	return []stream.Subject{
		subject.FromStr("NATHEJK.*.lok.*.updated"),
		subject.FromStr("NATHEJK.*.lok.*.deleted"),
	}
}

func (c *consumer) HandleMessage(msg stream.Message) error {
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
		return c.w.Consume(fmt.Sprintf(sql, args...))
	case msg.Subject().Match("NATHEJK.*.lok.*.deleted"):
		var body messages.NathejkLokDeleted
		if err := msg.Body(&body); err != nil {
			return err
		}
		sql := "DELETE FROM lok WHERE lokId=%q"
		args := []any{body.LokID}
		return c.w.Consume(fmt.Sprintf(sql, args...))

	default:
		log.Printf("Unhandled message %q", msg.Subject().Subject())

	}
	return nil
}
