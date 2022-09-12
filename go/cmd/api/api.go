package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/davecgh/go-spew/spew"

	"nathejk.dk/nathejk/types"
)

type ApiCommander interface {
	SaveUser(PostUserRequest) error
	DeleteUser(DeleteUserRequest) error
}

type api struct {
	cmd ApiCommander
}

func NewApi(cmd ApiCommander) *api {
	return &api{
		cmd: cmd,
	}
}

type PostUserRequest struct {
	UserID   types.UserID      `json:"userId"`
	Name     string            `json:"name"`
	Phone    types.PhoneNumber `json:"phone"`
	HqAccess bool              `json:"hqAccess"`
	Group    string            `json:"group"`
}
type DeleteUserRequest struct {
	UserID types.UserID `json:"userId"`
}
type UserResponse struct {
	Greeting string `json:"greeting"`
}

func (a *api) HandleUser(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		var req PostUserRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := a.cmd.SaveUser(req); err != nil {
			http.Error(w, err.Error(), http.StatusFailedDependency)
			return
		}
	}
	if r.Method == "DELETE" {
		var req DeleteUserRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			spew.Dump(r.Body, req)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := a.cmd.DeleteUser(req); err != nil {
			http.Error(w, err.Error(), http.StatusFailedDependency)
			return
		}
	}
	fmt.Fprint(w, "ok")
	return
	/*
		fmt.Fprintf(w, "format %s", "World")
		w.Header().Set("Content-Type", "application/json")
		//w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(UserResponse{
			Greeting: "userlist",
		})*/
}
