package table

import (
	"database/sql"
	"fmt"
	"log"

	"nathejk.dk/nathejk/messages"
	"nathejk.dk/nathejk/types"
	"nathejk.dk/pkg/stream/entity"
	"nathejk.dk/pkg/streaminterface"
	"nathejk.dk/pkg/tablerow"

	_ "embed"
)

type Department struct {
	DepartmentID types.DepartmentID
	Name         string
	HelloText    string
}
type DepartmentTableEvent struct {
	DepartmentID types.DepartmentID
}

type department struct {
	w tablerow.Consumer
	p entity.EntityChangedPublisher
}

func NewDepartment(w tablerow.Consumer, p streaminterface.Publisher) *department {
	table := &department{w: w, p: entity.NewEntityChangedPublisher(p, "department")}
	if err := w.Consume(table.CreateTableSql()); err != nil {
		log.Fatalf("Error creating table %q", err)
	}
	return table
}

//go:embed department.sql
var departmentSchema string

func (t *department) CreateTableSql() string {
	return departmentSchema
}

func (t *department) Consumes() []streaminterface.Subject {
	return []streaminterface.Subject{
		streaminterface.SubjectFromStr("nathejk"),
	}
}

func (t *department) HandleMessage(msg streaminterface.Message) error {
	switch msg.Subject().Subject() {
	case "nathejk:department.created":
		var body messages.NathejkDepartmentCreated
		if err := msg.Body(&body); err != nil {
			return err
		}
		sql := fmt.Sprintf("INSERT INTO department (departmentId, name, helloText) VALUES (%q, %q, %q) ON DUPLICATE KEY UPDATE name=VALUES(name), helloText=VALUES(helloText)", body.DepartmentID, body.Name, body.HelloText)
		if err := t.w.Consume(sql); err != nil {
			return nil
		}
		t.p.Changed(&DepartmentTableEvent{DepartmentID: body.DepartmentID})

	case "nathejk:department.updated":
		var body messages.NathejkDepartmentUpdated
		if err := msg.Body(&body); err != nil {
			return err
		}
		sql := fmt.Sprintf("UPDATE department (name, helloText) VALUES (%q, %q) WHERE departmentId=%q", body.Name, body.HelloText, body.DepartmentID)
		if err := t.w.Consume(sql); err != nil {
			return nil
		}
		t.p.Changed(&DepartmentTableEvent{DepartmentID: body.DepartmentID})

	case "nathejk:department.deleted":
		var body messages.NathejkDepartmentDeleted
		if err := msg.Body(&body); err != nil {
			return err
		}
		sql := fmt.Sprintf("DELETE FROM  department WHERE departmentId=%q", body.DepartmentID)
		if err := t.w.Consume(sql); err != nil {
			return nil
		}
		t.p.Deleted(&DepartmentTableEvent{DepartmentID: body.DepartmentID})
	}
	return nil
}

type departmentQueries struct {
	department *sql.Stmt
}

func DepartmentQuerier(db *sql.DB) *departmentQueries {
	prepare := func(query string) *sql.Stmt {
		stmt, err := db.Prepare(query)
		if err != nil {
			log.Fatal(err)
		}
		return stmt
	}
	return &departmentQueries{
		department: prepare("SELECT departmentId, name, helloText FROM department WHERE departmentId = ?"),
	}
}
func (q *departmentQueries) Department(departmentID types.DepartmentID) (*Department, error) {
	var row Department
	err := q.department.QueryRow(departmentID).Scan(&row.DepartmentID, &row.Name, &row.HelloText)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}
