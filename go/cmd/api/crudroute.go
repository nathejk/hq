package main

import (
	"encoding/json"
	"net/http"
)

type CrudCommander interface {
	Create(interface{}) (interface{}, error)
	Read(interface{}) (interface{}, error)
	Update(interface{}) (interface{}, error)
	Delete(interface{}) (interface{}, error)
}

type route struct {
	cmd    CrudCommander
	put    interface{}
	get    interface{}
	post   interface{}
	delete interface{}
}

func NewCrudRoute(cmd CrudCommander, put, get, post, del interface{}) *route {
	return &route{
		cmd:    cmd,
		put:    put,
		get:    get,
		post:   post,
		delete: del,
	}
}

func (route *route) Handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	//w.WriteHeader(http.StatusCreated)

	if r.Method == "PUT" {
		if err := json.NewDecoder(r.Body).Decode(&route.put); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		res, err := route.cmd.Create(route.put)
		if err != nil {
			http.Error(w, err.Error(), http.StatusFailedDependency)
			return
		}
		json.NewEncoder(w).Encode(res)
	}
	if r.Method == "GET" {
		if err := json.NewDecoder(r.Body).Decode(&route.get); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		res, err := route.cmd.Read(route.get)
		if err != nil {
			http.Error(w, err.Error(), http.StatusFailedDependency)
			return
		}
		json.NewEncoder(w).Encode(res)
	}
	if r.Method == "POST" {
		if err := json.NewDecoder(r.Body).Decode(&route.post); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if _, err := route.cmd.Update(route.post); err != nil {
			http.Error(w, err.Error(), http.StatusFailedDependency)
			return
		}
	}
	if r.Method == "DELETE" {
		if err := json.NewDecoder(r.Body).Decode(&route.delete); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if _, err := route.cmd.Delete(route.delete); err != nil {
			http.Error(w, err.Error(), http.StatusFailedDependency)
			return
		}
	}
}
