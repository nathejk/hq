package main

import "net/http"

func (s *server) adminOnly(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		//if !currentUser(r).IsAdmin {
		//	http.NotFound(w, r)
		//	return
		//}
		h(w, r)
	}
}
