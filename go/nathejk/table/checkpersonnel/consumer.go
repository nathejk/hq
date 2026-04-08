package checkpersonnel

import (
	"fmt"
	"log"

	"github.com/nathejk/shared-go/messages"
	"nathejk.dk/pkg/tablerow"
	"nathejk.dk/superfluids/streaminterface"
)

type consumer struct {
	w tablerow.Consumer
}

func (c *consumer) Consumes() (subjs []streaminterface.Subject) {
	return []streaminterface.Subject{
		streaminterface.SubjectFromStr("NATHEJK.*.checkpersonnel.*.added"),
		streaminterface.SubjectFromStr("NATHEJK.*.checkpersonnel.*.timespecified"),
		streaminterface.SubjectFromStr("NATHEJK.*.checkpersonnel.*.removed"),
		streaminterface.SubjectFromStr("NATHEJK.*.checkgroup.*.deleted"),
		streaminterface.SubjectFromStr("NATHEJK.*.checkpoint.*.deleted"),
	}
}

func (c *consumer) HandleMessage(msg streaminterface.Message) error {
	switch true {
	case msg.Subject().Match("NATHEJK.*.checkpersonnel.*.added"):
		var body messages.NathejkCheckpersonnelAdded
		if err := msg.Body(&body); err != nil {
			return err
		}
		id := msg.Subject().Parts()[3]
		year := msg.Subject().Parts()[1]
		if body.TimeRange != nil {
			sql := fmt.Sprintf("INSERT INTO checkpersonnel (id, year, userId, checkpointId, startUts, endUts) VALUES (%q, %q, %q, %q, %d, %d)", id, year, body.UserID, body.CheckpointID, body.TimeRange.Start.Unix(), body.TimeRange.End.Unix())
			return c.w.Consume(sql)
		}
		sql := fmt.Sprintf("INSERT INTO checkpersonnel (id, year, userId, checkpointId) VALUES (%q, %q, %q, %q)", id, year, body.UserID, body.CheckpointID)
		return c.w.Consume(sql)

	case msg.Subject().Match("NATHEJK.*.checkpersonnel.*.timespecified"):
		var body messages.NathejkCheckpersonnelTimeSpecified
		if err := msg.Body(&body); err != nil {
			return err
		}
		checkpersonnelID := msg.Subject().Parts()[3]
		sql := fmt.Sprintf("UPDATE checkpersonnel SET startUts=%d, endUts=%d WHERE id=%q", body.Start.Unix(), body.End.Unix(), checkpersonnelID)
		if err := c.w.Consume(sql); err != nil {
			log.Printf("Error consuming sql %q", err)
			return err
		}

	case msg.Subject().Match("NATHEJK.*.checkpersonnel.*.removed"):
		checkpersonnelID := msg.Subject().Parts()[3]
		sql := fmt.Sprintf("DELETE FROM checkpersonnel WHERE id=%q", checkpersonnelID)
		if err := c.w.Consume(sql); err != nil {
			log.Printf("Error consuming sql %q", err)
			return err
		}

	case msg.Subject().Match("NATHEJK.*.checkgroup.*.deleted"):
		var body messages.NathejkCheckgroupDeleted
		if err := msg.Body(&body); err != nil {
			return err
		}
		sql := fmt.Sprintf("DELETE FROM checkpersonnel WHERE checkpointId IN (SELECT id FROM checkpoint WHERE checkgroupId=%q)", body.CheckgroupID)
		if err := c.w.Consume(sql); err != nil {
			log.Printf("Error consuming sql %q", err)
			return err
		}

	case msg.Subject().Match("NATHEJK.*.checkpoint.*.deleted"):
		var body messages.NathejkCheckpointDeleted
		if err := msg.Body(&body); err != nil {
			return err
		}
		sql := fmt.Sprintf("DELETE FROM checkpersonnel WHERE checkpointId=%q", body.CheckpointID)
		if err := c.w.Consume(sql); err != nil {
			log.Printf("Error consuming sql %q", err)
			return err
		}
	}
	return nil
}
