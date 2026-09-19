package scan

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
		subject.FromStr("NATHEJK.*.qr.*.scanned"),
	}
}

func (c *consumer) HandleMessage(msg stream.Message) error {
	switch true {
	case msg.Subject().Match("NATHEJK.*.qr.*.scanned"):
		var body messages.NathejkQrScanned
		if err := msg.Body(&body); err != nil {
			return err
		}
		// year is taken from the subject, which is the only place it exists — the event body
		// does not carry it.
		//
		// It was never written, and the column being NULL on every row made *both* of this
		// table's indexes unusable: they lead with year (see table.sql), so every query
		// against scans was a full table scan, and any WHERE year = ? matched nothing at all
		// while looking like a legitimate empty result. That cost the bingo table half a
		// second at 100k scans and would grow with every year the table accumulates.
		//
		// ON DUPLICATE KEY UPDATE rather than the INSERT IGNORE this replaces, so a replay
		// backfills rows projected before this fix. With IGNORE the existing rows would keep
		// their NULL year forever and the indexes would stay dead on any database that is not
		// dropped. The other columns are restated for the same reason: a replay is the only
		// repair mechanism this read model has, and one that cannot correct a row it already
		// has is not a repair mechanism.
		sql := "INSERT INTO scan SET qrId=%q, year=%q, teamId=%q, teamNumber=%q, scannerId=%q, scannerPhone=%q, uts=%d, latitude=%q, longitude=%q" +
			" ON DUPLICATE KEY UPDATE year=VALUES(year), teamId=VALUES(teamId), teamNumber=VALUES(teamNumber), scannerId=VALUES(scannerId), scannerPhone=VALUES(scannerPhone), latitude=VALUES(latitude), longitude=VALUES(longitude)"
		args := []any{body.QrID, msg.Subject().Parts()[1], body.TeamID, body.TeamNumber, body.ScannerID, body.ScannerPhone, msg.Time().Unix(), body.Location.Latitude, body.Location.Longitude}
		return c.w.Consume(fmt.Sprintf(sql, args...))

	default:
		log.Printf("Unhandled message %q", msg.Subject().Subject())

	}
	return nil
}
