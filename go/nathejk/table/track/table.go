// Package track is where people were: positions reported by the hej-app and projected into a read
// model HQ can put on a map (PRD 011).
//
// # Why HQ keeps a copy at all
//
// The stream is the entire integration surface — there is no API between the services — so
// anything HQ wants to show, it must have projected. That cuts both ways: there is no "ask the
// app" fallback for a missing name or a gap in a route, and no backfill. If a fact is not on the
// stream, HQ does not have it, and the UI degrades rather than fetching.
//
// # What this package deliberately does not do
//
// **It does not validate points.** The producer's `track.Clean` already drops what is not a
// position at all — NaN, exactly (0,0) (Null Island: a failed fix reported as a success), clocks
// before 2020 or more than a day ahead, accuracy beyond 100 km — and deliberately *keeps*
// poor-but-real fixes, because a multi-kilometre cell-tower fix is still the only evidence of
// where someone was. Re-validating here would drift from that decision and silently discard data
// the producer chose to keep. HQ stores what it is given.
//
// **It does not interpret gaps.** Nobody records unbroken for a 30-hour race — phones lock, apps
// are killed, batteries die — so gaps of hours are the normal shape of this data, not an anomaly.
// Turning a point sequence into segments a map can draw honestly is a read concern (task 145), not
// a storage one: the raw points are what arrived, and that is what is kept.
//
// **It does not resolve people.** A `personId` is either a memberID or a crewmemberID, and both
// are opaque. This package never joins to a name, which is what lets one small presence query
// serve every people list in HQ.
package track

import (
	"database/sql"
	"log"

	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/mysql"
	"github.com/jrgensen/cqrs"

	_ "embed"
)

// Point is one recorded position.
//
// Field names and units match what the producer sends, which in turn matches what the client
// stores in IndexedDB, so nothing is converted on the way through. `Ts` is epoch milliseconds and
// is half a point's identity — see table.sql for why it is an integer rather than a formatted
// date.
type Point struct {
	Ts       int64   `json:"ts"`
	Lat      float64 `json:"lat"`
	Lng      float64 `json:"lng"`
	Accuracy float64 `json:"accuracy"`
}

// Latest is the last known position of one person.
//
// PersonType is the role held when that position was reported, not the role held now: see
// table.sql.
type Latest struct {
	PersonID   string  `json:"personId"`
	PersonType string  `json:"personType"`
	Year       string  `json:"year"`
	Lat        float64 `json:"lat"`
	Lng        float64 `json:"lng"`
	Accuracy   float64 `json:"accuracy"`
	Ts         int64   `json:"ts"`
}

type table struct {
	consumer
	querier
}

func New(w cqrs.Writer, r *sql.DB) *table {
	table := &table{
		consumer: consumer{w: w},
		querier:  querier{db: r, r: goqu.New("mysql", r)},
	}
	if err := w.Consume(table.CreateTableSql()); err != nil {
		log.Printf("Error creating table %q", err)
	}
	if err := w.Consume(pointSchema); err != nil {
		log.Printf("Error creating table %q", err)
	}
	return table
}

//go:embed table.sql
var tableSchema string

//go:embed point.sql
var pointSchema string

func (t *table) CreateTableSql() string {
	return tableSchema
}
