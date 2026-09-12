package patrulje

import (
	"fmt"
	"log"

	"github.com/jrgensen/cqrs"
	"github.com/jrgensen/stream"
	"github.com/jrgensen/stream/subject"
	"github.com/nathejk/shared-go/messages"
	"github.com/nathejk/shared-go/types"
)

type consumer struct {
	w cqrs.Writer
}

func (c *consumer) Consumes() (subjs []stream.Subject) {
	return []stream.Subject{
		subject.FromStr("NATHEJK:*.patrulje.*.signedup"),
		subject.FromStr("NATHEJK:*.patrulje.*.updated"),
		subject.FromStr("NATHEJK:*.patrulje.*.numberassigned"),
		subject.FromStr("NATHEJK:*.patrulje.*.started"),
		subject.FromStr("NATHEJK:*.patrulje.*.remark.set"),
	}
}

func (c *consumer) HandleMessage(msg stream.Message) error {
	//log.Printf("patrulje.go RECEIVED %q", msg.Subject().Subject())
	switch true {
	// Checked before the four-part patterns below. This subject has six parts, so
	// `NATHEJK.*.patrulje.*.updated` would not match it — but the reverse ordering has
	// bitten this codebase before (see the spejderstatus consumer), so keep the specific
	// one first.
	case msg.Subject().Match("NATHEJK.*.patrulje.*.remark.set"):
		var body RemarkSet
		if err := msg.Body(&body); err != nil {
			return err
		}
		// The whole note is restated by every event, so this is a plain overwrite: no
		// IF(...) guard against an empty value, because clearing the note is a thing an
		// operator does deliberately and must not be silently ignored.
		query := "UPDATE patrulje SET remark=%q, remarkSeverity=%q WHERE teamId=%q"
		args := []any{body.Remark, body.Severity, body.TeamID}
		return c.w.Consume(fmt.Sprintf(query, args...))

	case msg.Subject().Match("NATHEJK.*.patrulje.*.signedup"):
		var body messages.NathejkTeamSignedUp
		if err := msg.Body(&body); err != nil {
			return err
		}
		if body.TeamID == "" {
			return nil
		}
		sql := fmt.Sprintf("INSERT INTO patrulje SET teamId=%q, year=\"%d\", contactName=%q, contactPhone=%q, contactEmail=%q ON DUPLICATE KEY UPDATE contactName=VALUES(contactName), contactPhone=VALUES(contactPhone), contactEmail=VALUES(contactEmail)", body.TeamID, msg.Time().Year(), body.Name, body.Phone, body.Email)
		return c.w.Consume(sql)

	case msg.Subject().Match("NATHEJK.*.patrulje.*.updated"):
		var body messages.NathejkTeamUpdated
		if err := msg.Body(&body); err != nil {
			return err
		}
		msg.Subject().Parts()
		query := "UPDATE patrulje SET name=%q, groupName=%q, korps=%q, liga=%q, contactName=%q, contactPhone=%q, contactEmail=%q, contactRole=%q WHERE teamId=%q"
		args := []any{body.Name, body.GroupName, body.Korps, body.AdvspejdNumber, body.ContactName, body.ContactPhone, body.ContactEmail, substr(body.ContactRole, 0, 90), body.TeamID}
		return c.w.Consume(fmt.Sprintf(query, args...))

	case msg.Subject().Match("NATHEJK.*.patrulje.*.numberassigned"):
		var body messages.NathejkPatrolNumberAssigned
		if err := msg.Body(&body); err != nil {
			return err
		}
		query := "UPDATE patrulje SET teamNumber=%q WHERE teamId=%q"
		args := []any{body.TeamNumber, body.TeamID}
		return c.w.Consume(fmt.Sprintf(query, args...))

	case msg.Subject().Match("NATHEJK.*.patrulje.*.started"):
		var body messages.NathejkTeamStarted
		if err := msg.Body(&body); err != nil {
			return err
		}
		// startedUts is the moment the patrol went on the route: the publish time of the
		// .started event. Persisted here because it lives nowhere else — patruljestatus.startedUts
		// is hardcoded to 1 on signup and never means "started" — and the patrol trail needs it
		// as the first event above the scans.
		query := "UPDATE patrulje SET signupStatus=%q, memberCount=%d, startedUts=%d WHERE teamId=%q"
		args := []any{types.SignupStatusStarted, len(body.Members), msg.Time().Unix(), body.TeamID}
		return c.w.Consume(fmt.Sprintf(query, args...))

	default:
		log.Printf("Unhandled message %q", msg.Subject().Subject())

	}
	return nil
}
func substr(input string, start int, length int) string {
	asRunes := []rune(input)

	if start >= len(asRunes) {
		return ""
	}

	if start+length > len(asRunes) {
		length = len(asRunes) - start
	}

	return string(asRunes[start : start+length])
}
