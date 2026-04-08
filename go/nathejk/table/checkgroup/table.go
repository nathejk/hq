package checkgroup

import (
	"database/sql"
	"log"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/nathejk/shared-go/types"
	"nathejk.dk/nathejk/table/checkpoint"
	"nathejk.dk/pkg/tablerow"
	"nathejk.dk/superfluids/streaminterface"

	_ "embed"
)

type Checkgroup struct {
	ID        types.CheckgroupID `json:"id" db:"id"`
	CreatedAt time.Time          `json:"created_at"`
	Version   uint64             `json:"version"`

	YearSlug             types.YearSlug           `db:"year"`
	Name                 string                   `json:"name" db:"name"`
	ShowOnMap            bool                     `json:"showOnMap" db:"showOnMap"`
	Mandatory            bool                     `json:"mandatory" db:"mandatory"`
	Scheme               types.CheckgroupScheme   `json:"scheme" db:"scheme"`
	RelativeCheckgroupID *types.CheckgroupID      `json:"relativeCheckgroupId" db:"relativeCheckgroupId"`
	SortOrder            int                      `json:"sortOrder" db:"sortOrder"`
	Checkpoints          []*checkpoint.Checkpoint `json:"checkpoints"`
	OnTime               types.TeamIDs            `json:"onTimeTeamIds"`
	OverTime             types.TeamIDs            `json:"overTimeTeamIds"`
	NotArrived           types.TeamIDs            `json:"notArrivedTeamIds"`
	Discontinued         types.TeamIDs            `json:"discontinuedTeamIds"`
}

/*
type Checkgroup struct {
	ID       types.ControlGroupID
	YearSlug string
	Name     string
}*/

type Checkgroups []*Checkgroup

func (cgs *Checkgroups) Checkgroup(ID types.CheckgroupID) *Checkgroup {
	for _, cg := range *cgs {
		if cg.ID == ID {
			return cg
		}
	}
	return nil
}

func (cg *Checkgroup) Checkpoint(index int) *checkpoint.Checkpoint {
	for _, cp := range cg.Checkpoints {
		if cp.SortOrder == index {
			return cp
		}
	}
	return nil
}

type table struct {
	commander
	consumer
	querier
}

func New(p streaminterface.Publisher, w tablerow.Consumer, r *sql.DB) *table {
	q := querier{db: r, r: goqu.New("mysql", r)}
	table := &table{commander: commander{p: p, q: &q}, consumer: consumer{w: w}, querier: q}
	if err := w.Consume(table.CreateTableSql()); err != nil {
		log.Fatalf("Error creating table %q", err)
	}
	return table
}

//go:embed table.sql
var tableSchema string

func (t *table) CreateTableSql() string {
	return tableSchema
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
}
*/
