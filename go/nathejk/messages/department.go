package messages

import (
	"nathejk.dk/nathejk/types"
)

type NathejkDepartmentCreated struct {
	DepartmentID types.DepartmentID `json:"departmentId"`
	Name         string             `json:"name"`
	HelloText    string             `json:"helloText"`
}

type NathejkDepartmentUpdated struct {
	DepartmentID types.DepartmentID `json:"departmentId"`
	Name         string             `json:"name"`
	HelloText    string             `json:"helloText"`
}

type NathejkDepartmentDeleted struct {
	DepartmentID types.DepartmentID `json:"departmentId"`
}
