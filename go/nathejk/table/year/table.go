package year

import (
	"database/sql"
	"log"
	"time"

	"github.com/nathejk/shared-go/types"
	"nathejk.dk/pkg/tablerow"
	"nathejk.dk/superfluids/streaminterface"

	_ "embed"
)

type Year struct {
	Slug      types.YearSlug `json:"slug"`
	CreatedAt time.Time      `json:"created_at"`
	Version   uint64         `json:"version"`

	Headline        string     `json:"headline"`
	Description     string     `json:"description"`
	CityDeparture   string     `json:"cityDeparture"`
	CityDestination string     `json:"cityDestination"`
	DateStart       types.Date `json:"dateStart"`
	DateEnd         types.Date `json:"dateEnd"`
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
