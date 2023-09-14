package handlers

import (
	"encoding/json"
	"net/http"

	"nathejk.dk/nathejk/types"
)

type CreateDepartmentCommands interface {
	CreateDepartment(string, string) (types.DepartmentID, error)
}

func CreateDepartment(c CreateDepartmentCommands) http.HandlerFunc {

	type request struct {
		Name      string `json:"name"`
		HelloText string `json:"helloText"`
	}
	type response struct {
		DepartmentID types.DepartmentID `json:"departmentId"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		departmentID, err := c.CreateDepartment(req.Name, req.HelloText)
		if err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response{
			DepartmentID: departmentID,
		})
	}
}

type UpdateDepartmentCommands interface {
	UpdateDepartment(types.DepartmentID, string, string) error
}

func UpdateDepartment(c UpdateDepartmentCommands) http.HandlerFunc {

	type request struct {
		DepartmentID types.DepartmentID `json:"departmentId"`
		Name         string             `json:"name"`
		HelloText    string             `json:"helloText"`
	}
	type response struct {
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err := c.UpdateDepartment(req.DepartmentID, req.Name, req.HelloText)
		if err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response{})
	}
}

type DeleteDepartmentCommands interface {
	DeleteDepartment(types.DepartmentID) error
}

func DeleteDepartment(c DeleteDepartmentCommands) http.HandlerFunc {

	type request struct {
		DepartmentID types.DepartmentID `json:"departmentId"`
	}
	type response struct {
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err := c.DeleteDepartment(req.DepartmentID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response{})
	}
}
