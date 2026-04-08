package patrulje

import (
	"database/sql"
	"log"

	"github.com/doug-martin/goqu/v9"
	"github.com/nathejk/shared-go/types"
	"nathejk.dk/pkg/tablerow"

	_ "embed"
)

type Patrulje struct {
	TeamID       types.TeamID       `json:"teamId"`
	TeamNumber   string             `json:"teamNumber"`
	Year         string             `json:"year"`
	Name         string             `json:"name"`
	Group        string             `json:"group"`
	Korps        string             `json:"korps"`
	Liga         string             `json:"liga"`
	ContactName  string             `json:"contactName"`
	ContactPhone types.PhoneNumber  `json:"contactPhone"`
	ContactEmail types.EmailAddress `json:"contactEmail"`
	ContactRole  string             `json:"contactRole"`
	MemberCount  int                `json:"memberCount"`
	TshirtCount  int                `json:"tshirtCount"`
	SignupStatus types.SignupStatus `json:"signupStatus"`
	PaidAmount   int                `json:"paidAmount"`
}

type table struct {
	consumer
	querier
}

func New(w tablerow.Consumer, r *sql.DB) *table {
	q := querier{db: r, r: goqu.New("mysql", r)}
	table := &table{consumer: consumer{w: w}, querier: q}
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
