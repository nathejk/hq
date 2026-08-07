package patruljemerged

import (
	"database/sql"
	"log"

	"github.com/jrgensen/cqrs"
	"github.com/nathejk/shared-go/types"

	_ "embed"
)

type PatruljeMerged struct {
	TeamID       types.TeamID
	ParentTeamID types.TeamID
}

type table struct {
	consumer
	//querier
}

func New(w cqrs.Writer, r *sql.DB) *table {
	table := &table{consumer: consumer{w: w}} //, querier: querier{db: r}}
	if err := w.Consume(table.CreateTableSql()); err != nil {
		log.Printf("Error creating table %q", err)
	}
	return table
}

//go:embed table.sql
var tableSchema string

func (t *table) CreateTableSql() string {
	return tableSchema
}
