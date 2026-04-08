package checkgroup

import (
	"fmt"
	"log"

	"github.com/nathejk/shared-go/messages"
	"nathejk.dk/pkg/tablerow"
	"nathejk.dk/superfluids/streaminterface"

	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/mysql"
)

type consumer struct {
	w tablerow.Consumer
}

func (c *consumer) Consumes() (subjs []streaminterface.Subject) {
	return []streaminterface.Subject{
		streaminterface.SubjectFromStr("NATHEJK.*.checkgroup.*.created"),
		streaminterface.SubjectFromStr("NATHEJK.*.checkgroup.*.updated"),
		streaminterface.SubjectFromStr("NATHEJK.*.checkgroup.*.deleted"),
		streaminterface.SubjectFromStr("NATHEJK.*.checkgroups.sorted"),
	}
}

func (c *consumer) HandleMessage(msg streaminterface.Message) error {
	//dialect := mysql.Dialect()
	var dialect = goqu.Dialect("mysql")

	switch true {
	case msg.Subject().Match("NATHEJK.*.checkgroup.*.created"):
		row := goqu.Record{
			"id":   msg.Subject().Parts()[3],
			"year": msg.Subject().Parts()[1],
		}
		sqlStr, _, err := dialect.Insert("checkgroup").OnConflict(goqu.DoNothing()).Rows(row).ToSQL()
		if err != nil {
			return err
		}
		return c.w.Consume(sqlStr)

	case msg.Subject().Match("NATHEJK.*.checkgroup.*.updated"):
		var body messages.NathejkCheckgroupUpdated
		if err := msg.Body(&body); err != nil {
			return err
		}

		insert := goqu.Record{
			"id":   body.CheckgroupID,
			"year": msg.Time().Year(),
		}
		update := goqu.Record{}
		if body.Name != nil {
			insert["name"] = *body.Name
			update["name"] = goqu.L("VALUES(name)")
		}
		if body.ShowOnMap != nil {
			insert["showOnMap"] = *body.ShowOnMap
			update["showOnMap"] = goqu.L("VALUES(showOnMap)")
		}
		if body.Mandatory != nil {
			insert["mandatory"] = *body.Mandatory
			update["mandatory"] = goqu.L("VALUES(mandatory)")
		}
		if body.Scheme != nil {
			insert["scheme"] = *body.Scheme
			update["scheme"] = goqu.L("VALUES(scheme)")
		}
		if body.RelativeCheckgroupID != nil {
			insert["relativeCheckgroupId"] = *body.RelativeCheckgroupID
			update["relativeCheckgroupId"] = goqu.L("VALUES(relativeCheckgroupId)")
		}
		ds := dialect.Insert("checkgroup").Rows(insert).OnConflict(goqu.DoUpdate("id", update))
		sqlStr, _, _ := ds.ToSQL()

		//	year := msg.Time().Year()
		//	sql := fmt.Sprintf("INSERT INTO controlgroup SET id=%q, name=%q, year=\"%d\" ON DUPLICATE KEY UPDATE name=VALUES(name)", body.CheckgroupID, body.Name, year)
		if err := c.w.Consume(sqlStr); err != nil {
			log.Fatalf("Error consuming sql %q", err)
		}
		//c.p.Changed(&ControlGroupTableEvent{})

	case msg.Subject().Match("NATHEJK.*.checkgroup.*.deleted"):
		var body messages.NathejkCheckgroupDeleted
		if err := msg.Body(&body); err != nil {
			return err
		}
		err := c.w.Consume(fmt.Sprintf("DELETE FROM checkgroup WHERE id=%q", body.CheckgroupID))
		if err != nil {
			log.Fatalf("Error consuming sql %q", err)
		}
		//c.p.Deleted(&ControlGroupTableEvent{})

	case msg.Subject().Match("NATHEJK.*.checkgroups.sorted"):
		var body messages.NathejkCheckgroupsSorted
		if err := msg.Body(&body); err != nil {
			return err
		}
		for i, cgID := range body.SortedCheckgroupIDs {
			sqlStr := fmt.Sprintf("UPDATE checkgroup SET sortOrder=%d WHERE id=%q", i, cgID)
			if err := c.w.Consume(sqlStr); err != nil {
				log.Printf("Error updating sort order for checkgroup %s: %v", cgID, err)
			}
		}

	}
	return nil
}

/*
type CheckGroupScan struct {
	TeamID         types.TeamID
	TeamNumber     string
	Uts            int
	UserID         string
	ControlGroupID types.ControlGroupID
	ControlIndex   int
	OnTime         bool
}

func (cg *controlgroup) AllScans(year string) []CheckGroupScan {
	rows, err := cg.db.Query(`
SELECT
	s.teamId,
	s.teamNumber,
	s.uts,
	s.userId,
	cp.controlGroupId,
	cp.controlIndex,
	(cp.openFromUts - 60*cp.minusMinutes <= s.uts AND s.uts <= cp.openUntilUts + 60*plusMinutes) AS ontime
FROM scan
  JOIN controlgroup_user cgu ON scan.scannerId = cgu.userId AND startUts <= uts AND uts <= endUts
  JOIN controlpoint cp ON cgu.controlGroupId = cp.controlGroupId AND cgu.controlIndex = cp.controlIndex`)
	if err != nil {
		log.Fatalf("Query: %v", err)
	}
	ss := []CheckGroupScan{}
	for rows.Next() {
		var s CheckGroupScan
		err = rows.Scan(&s.TeamID, &s.TeamNumber, &s.Uts, &s.UserID, &s.ControlGroupID, &s.ControlIndex, &s.OnTime)
		if err != nil {
			log.Fatalf("Scan: %v", err)
		}
		ss = append(ss, s)
	}
	return ss
}*/
