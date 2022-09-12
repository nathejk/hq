package main

import (
	"encoding/json"
	"net/http"
)

type sosRoutes struct {
	cmd SosCommander
}

func NewSosRoutes(cmd SosCommander) *sosRoutes {
	return &sosRoutes{
		cmd: cmd,
	}
}

func (routes *sosRoutes) Handler(w http.ResponseWriter, r *http.Request) {

	user := NewUserFromRequest(r)

	switch r.Method + ":" + r.URL.Path {
	case "PUT:/api/sos":
		var req SosRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		res, err := routes.cmd.Create(req, user)
		if err != nil {
			http.Error(w, err.Error(), http.StatusFailedDependency)
			return
		}
		json.NewEncoder(w).Encode(res)

	case "PUT:/api/sos/comment":
		var req SosCommentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		res, err := routes.cmd.AddComment(req, user)
		if err != nil {
			http.Error(w, err.Error(), http.StatusFailedDependency)
			return
		}
		json.NewEncoder(w).Encode(res)

	case "POST:/api/sos/headline":
		var req SosRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		res, err := routes.cmd.UpdateHeadline(req, user)
		if err != nil {
			http.Error(w, err.Error(), http.StatusFailedDependency)
			return
		}
		json.NewEncoder(w).Encode(res)

	case "POST:/api/sos/close":
		var req SosRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		res, err := routes.cmd.Close(req, user)
		if err != nil {
			http.Error(w, err.Error(), http.StatusFailedDependency)
			return
		}
		json.NewEncoder(w).Encode(res)

	case "POST:/api/sos/reopen":
		var req SosRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		res, err := routes.cmd.Reopen(req, user)
		if err != nil {
			http.Error(w, err.Error(), http.StatusFailedDependency)
			return
		}
		json.NewEncoder(w).Encode(res)

	case "POST:/api/sos/team":
		var req SosTeamRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		res, err := routes.cmd.AssociateTeam(req, user)
		if err != nil {
			http.Error(w, err.Error(), http.StatusFailedDependency)
			return
		}
		json.NewEncoder(w).Encode(res)

	case "DELETE:/api/sos/team":
		var req SosTeamRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		res, err := routes.cmd.DisassociateTeam(req, user)
		if err != nil {
			http.Error(w, err.Error(), http.StatusFailedDependency)
			return
		}
		json.NewEncoder(w).Encode(res)

	case "POST:/api/sos/member":
		var req SosMemberRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		res, err := routes.cmd.MemberStatusChange(req, user)
		if err != nil {
			http.Error(w, err.Error(), http.StatusFailedDependency)
			return
		}
		json.NewEncoder(w).Encode(res)

	case "POST:/api/sos/severity":
		var req SosSeverityRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		res, err := routes.cmd.SetSeverity(req, user)
		if err != nil {
			http.Error(w, err.Error(), http.StatusFailedDependency)
			return
		}
		json.NewEncoder(w).Encode(res)

	case "POST:/api/sos/assign":
		var req SosAssignRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		res, err := routes.cmd.Assign(req, user)
		if err != nil {
			http.Error(w, err.Error(), http.StatusFailedDependency)
			return
		}
		json.NewEncoder(w).Encode(res)

	case "POST:/api/sos/sms":
		var req SosMemberRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		res, err := routes.cmd.SendPositionSms(req, user)
		if err != nil {
			http.Error(w, err.Error(), http.StatusFailedDependency)
			return
		}
		json.NewEncoder(w).Encode(res)

	case "POST:/api/sos/merge":
		var req SosTeamRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		res, err := routes.cmd.MergeTeams(req, user)
		if err != nil {
			http.Error(w, err.Error(), http.StatusFailedDependency)
			return
		}
		json.NewEncoder(w).Encode(res)

	case "POST:/api/sos/split":
		var req SosTeamRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		res, err := routes.cmd.SplitTeam(req, user)
		if err != nil {
			http.Error(w, err.Error(), http.StatusFailedDependency)
			return
		}
		json.NewEncoder(w).Encode(res)

	default:
		http.Error(w, "Not implemented", http.StatusNotImplemented)
		return
	}
	// PUT: /api/sos
	// POST:/api/sos
	// DELETE:/api/sos
	// PUT: /api/sos/comment
	// POST:/api/sos/comment
	// POST:/api/sos/severity
	// POST:/api/sos/assign
	// POST:/api/sos/close
	// POST:/api/sos/reopen
	w.Header().Set("Content-Type", "application/json")
	//w.WriteHeader(http.StatusCreated)
	/*
		if r.Method == "PUT" {
			if err := json.NewDecoder(r.Body).Decode(&routes.put); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			res, err := routes.cmd.Create(routes.put)
			if err != nil {
				http.Error(w, err.Error(), http.StatusFailedDependency)
				return
			}
			json.NewEncoder(w).Encode(res)
		}
		if r.Method == "GET" {
			if err := json.NewDecoder(r.Body).Decode(&routes.get); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			res, err := routes.cmd.Read(routes.get)
			if err != nil {
				http.Error(w, err.Error(), http.StatusFailedDependency)
				return
			}
			json.NewEncoder(w).Encode(res)
		}
		if r.Method == "POST" {
			if err := json.NewDecoder(r.Body).Decode(&routes.post); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if _, err := routes.cmd.Update(routes.post); err != nil {
				http.Error(w, err.Error(), http.StatusFailedDependency)
				return
			}
		}
		if r.Method == "DELETE" {
			if err := json.NewDecoder(r.Body).Decode(&routes.delete); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if _, err := routes.cmd.Delete(routes.delete); err != nil {
				http.Error(w, err.Error(), http.StatusFailedDependency)
				return
			}
		}
	*/
}
