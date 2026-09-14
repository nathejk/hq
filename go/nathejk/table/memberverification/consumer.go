// Package memberverification records which phone numbers a member has verified themselves.
//
// The fact originates with the member, in the hej app, and is published as
// `NATHEJK:{year}.spejder.{memberId}.verified` (see messages.NathejkMemberVerified). hq
// keeps it so the patrol's start row can say which numbers the counter no longer has to
// ask about: check-in establishes every member's own and emergency contact number at the
// counter while a queue forms, and each number already verified is a conversation saved
// (hej, PRD 015).
//
// It is a projection of its own rather than two columns on `spejder`, because it says
// something different from that table. `spejder` holds what the *register* has — what the
// patrol leader typed at signup, and what an organizer may correct afterwards — while this
// holds what the *member* proved. Folding the two together would lose exactly the
// distinction the shield in the UI draws: an operator may edit the register's number, and
// the moment they do, the verification no longer vouches for what is on screen.
package memberverification

import (
	"fmt"
	"log"

	"github.com/jrgensen/cqrs"
	"github.com/jrgensen/stream"
	"github.com/jrgensen/stream/subject"
	"github.com/nathejk/shared-go/messages"
)

type consumer struct {
	w cqrs.Writer
}

func (c *consumer) Consumes() []stream.Subject {
	return []stream.Subject{
		subject.FromStr("NATHEJK:*.spejder.*.verified"),
	}
}

func (c *consumer) HandleMessage(msg stream.Message) error {
	switch true {
	case msg.Subject().Match("NATHEJK.*.spejder.*.verified"):
		var body messages.NathejkMemberVerified
		if err := msg.Body(&body); err != nil {
			return err
		}
		// The member is the third subject token, but trust the body's id when it has one —
		// it is what the publisher named — and fall back to the subject otherwise.
		memberID := string(body.MemberID)
		if memberID == "" {
			memberID = msg.Subject().Parts()[3]
		}
		if memberID == "" {
			return nil
		}
		// A verification carrying neither number says nothing; there is no row to write.
		phone, phoneContact := body.Phone.Normalize(), body.PhoneContact.Normalize()
		if phone == "" && phoneContact == "" {
			return nil
		}

		year := msg.Subject().Parts()[1]
		uts := body.VerifiedAt.Unix()
		// The timestamp belongs to whichever number the event actually carries, so an
		// absent number carries no time either — which is what makes the "newer wins"
		// comparison below per-column rather than per-event.
		phoneUts, contactUts := int64(0), int64(0)
		if phone != "" {
			phoneUts = uts
		}
		if phoneContact != "" {
			contactUts = uts
		}

		// Each column is written only by an event that both names it and is not older than
		// what is stored. That makes the row order-independent — a replay delivering an
		// older verification after a newer one must not move a number back — and it is why
		// a plain login (phone only) cannot erase an already-verified contact number.
		sql := fmt.Sprintf(
			"INSERT INTO memberverification SET year=%q, memberId=%q, "+
				"phone=%q, phoneUts=%d, phoneContact=%q, phoneContactUts=%d "+
				"ON DUPLICATE KEY UPDATE "+
				"phone=IF(VALUES(phone) <> '' AND VALUES(phoneUts) >= phoneUts, VALUES(phone), phone), "+
				"phoneUts=IF(VALUES(phone) <> '' AND VALUES(phoneUts) >= phoneUts, VALUES(phoneUts), phoneUts), "+
				"phoneContact=IF(VALUES(phoneContact) <> '' AND VALUES(phoneContactUts) >= phoneContactUts, VALUES(phoneContact), phoneContact), "+
				"phoneContactUts=IF(VALUES(phoneContact) <> '' AND VALUES(phoneContactUts) >= phoneContactUts, VALUES(phoneContactUts), phoneContactUts)",
			year, memberID, phone, phoneUts, phoneContact, contactUts,
		)
		return c.w.Consume(sql)

	default:
		log.Printf("Unhandled message %q", msg.Subject().Subject())
	}
	return nil
}
