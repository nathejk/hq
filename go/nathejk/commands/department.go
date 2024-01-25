package commands

import (
	"fmt"

	"nathejk.dk/nathejk/messages"
	"nathejk.dk/nathejk/table"
	"nathejk.dk/nathejk/types"
	"nathejk.dk/pkg/streaminterface"
)

type DepartmentQueries interface {
	Department(types.DepartmentID) (*table.Department, error)
}

type departmentCommands struct {
	q DepartmentQueries
	p streaminterface.Publisher
}

func NewDepartment(q DepartmentQueries, p streaminterface.Publisher) *departmentCommands {
	c := &departmentCommands{
		q: q,
		p: p,
	}
	return c
}

func (c *departmentCommands) CreateDepartment(name string, helloText string) (types.DepartmentID, error) {
	body := messages.NathejkDepartmentCreated{
		DepartmentID: types.NewDepartmentID(),
		Name:         name,
		HelloText:    helloText,
	}
	msg := c.p.MessageFunc()(streaminterface.SubjectFromStr("nathejk:department.updated"))
	msg.SetBody(body)
	meta := messages.Metadata{Producer: "hq-api"}
	msg.SetMeta(&meta)

	if err := c.p.Publish(msg); err != nil {
		return "", err
	}
	return body.DepartmentID, nil
}

func (c *departmentCommands) UpdateDepartment(departmentID types.DepartmentID, name string, helloText string) error {
	department, err := c.q.Department(departmentID)
	if err != nil {
		return err
	}
	if department == nil {
		return fmt.Errorf("Error updating department %q does not exist", departmentID)
	}
	body := messages.NathejkDepartmentUpdated{
		DepartmentID: departmentID,
		Name:         name,
		HelloText:    helloText,
	}
	msg := c.p.MessageFunc()(streaminterface.SubjectFromStr("nathejk:department.updated"))
	msg.SetBody(body)
	meta := messages.Metadata{Producer: "hq-api"}
	msg.SetMeta(&meta)

	return c.p.Publish(msg)
}

func (c *departmentCommands) DeleteDepartment(departmentID types.DepartmentID) error {
	department, err := c.q.Department(departmentID)
	if err != nil {
		return err
	}
	if department == nil {
		return fmt.Errorf("Error deleting department %q does not exist", departmentID)
	}
	body := messages.NathejkDepartmentDeleted{
		DepartmentID: departmentID,
	}
	msg := c.p.MessageFunc()(streaminterface.SubjectFromStr("nathejk:department.deleted"))
	msg.SetBody(body)
	meta := messages.Metadata{Producer: "hq-api"}
	msg.SetMeta(&meta)

	return c.p.Publish(msg)
}
