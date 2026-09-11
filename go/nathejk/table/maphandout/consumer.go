package maphandout

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

// qrRegistered is the shared registered body plus the sheet handed over.
//
// hq's copy of skan's event.QrRegistered: the map id is an additive JSON field skan adds
// to the shared body, so it is not on messages.NathejkQrRegistered and has to be read
// through a struct that embeds the shared one and names it. Absent (older events) reads as
// "" — unknown sheet, not no sheet.
type qrRegistered struct {
	messages.NathejkQrRegistered
	MapID string `json:"mapId,omitempty"`
}

func (c *consumer) Consumes() []stream.Subject {
	return []stream.Subject{
		subject.FromStr("NATHEJK.*.qr.*.registered"),
	}
}

func (c *consumer) HandleMessage(msg stream.Message) error {
	switch true {
	case msg.Subject().Match("NATHEJK.*.qr.*.registered"):
		var body qrRegistered
		if err := msg.Body(&body); err != nil {
			return err
		}
		// A binding with no team is not a handout to anyone; nothing to record.
		if body.TeamID == "" {
			return nil
		}
		year := msg.Subject().Parts()[1]
		uts := msg.Time().Unix()

		// ON DUPLICATE KEY UPDATE, not INSERT IGNORE: the same code can be bound to the
		// same team more than once (a re-scan), and the row should track the latest
		// registrar and widen [firstUts, lastUts] rather than be discarded.
		//
		//   - mapId: never overwritten with "" — the same rule skan follows, since a code
		//     re-scanned before its sheet was recorded must not lose the sheet it already
		//     names.
		//   - firstUts/lastUts: LEAST/GREATEST so the interval this team held the code is
		//     correct whatever order the log replays in.
		sql := fmt.Sprintf(
			"INSERT INTO maphandout SET year=%q, qrId=%q, teamId=%q, teamNumber=%q, "+
				"mapId=%q, registeredBy=%q, firstUts=%d, lastUts=%d "+
				"ON DUPLICATE KEY UPDATE teamNumber=VALUES(teamNumber), "+
				"mapId=IF(VALUES(mapId) = '', mapId, VALUES(mapId)), "+
				"registeredBy=VALUES(registeredBy), "+
				"firstUts=LEAST(firstUts, VALUES(firstUts)), "+
				"lastUts=GREATEST(lastUts, VALUES(lastUts))",
			year, string(body.QrID), string(body.TeamID), body.TeamNumber,
			body.MapID, body.ScannerID, uts, uts,
		)
		return c.w.Consume(sql)

	default:
		log.Printf("Unhandled message %q", msg.Subject().Subject())
	}
	return nil
}
