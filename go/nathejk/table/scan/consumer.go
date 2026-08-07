package scan

import (
	"fmt"
	"log"

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
		sql := "INSERT IGNORE INTO scan SET qrId=%q, teamId=%q, teamNumber=%q, scannerId=%q, scannerPhone=%q, uts=%d, latitude=%q, longitude=%q"
		args := []any{body.QrID, body.TeamID, body.TeamNumber, body.ScannerID, body.ScannerPhone, msg.Time().Unix(), body.Location.Latitude, body.Location.Longitude}
		return c.w.Consume(fmt.Sprintf(sql, args...))

	default:
		log.Printf("Unhandled message %q", msg.Subject().Subject())

	}
	return nil
}
