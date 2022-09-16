package table

import (
	"fmt"
	"log"
	"time"

	"nathejk.dk/nathejk/messages"
	"nathejk.dk/nathejk/types"
	"nathejk.dk/pkg/stream/entity"
	"nathejk.dk/pkg/streaminterface"
	"nathejk.dk/pkg/tablerow"

	_ "embed"
)

type Scan struct {
	QrID         types.QrID
	ControlIndex int
	UserID       types.UserID
	StartDate    time.Time
	Ts           time.Time
}
type ScanTableEvent struct {
	UserID types.UserID
}

type scan struct {
	w tablerow.Consumer
	p entity.EntityChangedPublisher
}

func NewScan(w tablerow.Consumer, p streaminterface.Publisher) *scan {
	table := &scan{w: w, p: entity.NewEntityChangedPublisher(p, "scan")}
	if err := w.Consume(table.CreateTableSql()); err != nil {
		log.Fatalf("Error creating table %q", err)
	}
	return table
}

//go:embed scan.sql
var scanSchema string

func (t *scan) CreateTableSql() string {
	return scanSchema
}

func (c *scan) Consumes() (subjs []streaminterface.Subject) {
	return []streaminterface.Subject{
		streaminterface.SubjectFromStr("nathejk"),
	}
}

func (c *scan) HandleMessage(msg streaminterface.Message) error {
	switch msg.Subject().Subject() {
	case "nathejk:qr.scanned":
		var body messages.NathejkQrScanned
		if err := msg.Body(&body); err != nil {
			return err
		}
		sql := fmt.Sprintf("INSERT INTO scan (qrId, teamId, teamNumber, scannerId, scannerPhone, uts) VALUES (%q, %q, %q, %q, %q, %d) ON DUPLICATE KEY UPDATE qrId=VALUES(qrId)", body.QrID, body.TeamID, body.TeamNumber, body.ScannerID, body.ScannerPhone, msg.Time().Unix())
		if err := c.w.Consume(sql); err != nil {
			return nil
		}
		//c.p.ChangedScanEvent{})
	}
	return nil
}
