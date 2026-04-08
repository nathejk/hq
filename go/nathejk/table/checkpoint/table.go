package checkpoint

import (
	"database/sql"
	"log"
	"time"

	"github.com/nathejk/shared-go/types"
	"nathejk.dk/nathejk/table/checkpersonnel"
	"nathejk.dk/pkg/tablerow"
	"nathejk.dk/superfluids/streaminterface"

	_ "embed"
)

type Checkpoint struct {
	ID           types.CheckpointID `json:"id"`
	CheckgroupID types.CheckgroupID `json:"checkgroupId"`
	Year         types.YearSlug     `json:"year"`
	Name         string             `json:"name"`
	SortOrder    int                `json:"sortOrder"`
	OpenFrom     time.Time          `json:"openFrom"`
	OpenUntil    time.Time          `json:"openUntil"`
	OpenDuration time.Duration      `json:"openDuration"`
	Latitude     *float32           `json:"latitude"`
	Longitude    *float32           `json:"longitude"`
	Address      string             `json:"address"`
	Description  string             `json:"description"`

	Scheme               types.CheckgroupScheme
	RelativeCheckgroupID types.CheckgroupID
	Plus                 int                             `json:"plus"`
	Minus                int                             `json:"minus"`
	OnTime               types.TeamIDs                   `json:"onTimeTeamIds"`
	OverTime             types.TeamIDs                   `json:"overTimeTeamIds"`
	Scanners             []checkpersonnel.Checkpersonnel `json:"scanners"`
}
type Control struct {
	ControlGroupID   types.CheckgroupID
	ControlGroupName string
	ControlIndex     int
	ControlName      string
	OpenFrom         time.Time
	OpenUntil        time.Time
	Plus             int
	Minus            int
}

type table struct {
	commander
	consumer
	querier
}

func New(p streaminterface.Publisher, w tablerow.Consumer, r *sql.DB) *table {
	q := querier{db: r}
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
