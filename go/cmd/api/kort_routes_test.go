package main

import (
	"net/http"
	"testing"

	"github.com/julienschmidt/httprouter"
)

// The kort routes are asserted here, in the composition root, because their failure mode is
// startup and not a request.
//
// httprouter panics when a static segment sits beside a wildcard at the same level, so
// `/api/kortsaet/sorted` next to `/api/kortsaet/:id` does not answer 404 or misroute — it stops
// the API booting at all, for every endpoint. There is no runtime symptom to notice and no
// response to inspect, which is exactly the kind of thing that needs a test rather than a
// comment. `/api/checkgroups/sorted` avoids the collision with an English plural; "kort" and
// "kortsæt" are their own plurals in Danish, so the collection carries the order instead.

// kortRoutes registers the same paths as routes(), in the same order, with nil handlers.
//
// Only the paths matter to httprouter's tree, so this needs no application — which keeps the test
// free of the whole dependency graph main() builds, and means it still guards the tree when a
// handler's signature changes.
func kortRoutes(t *testing.T) *httprouter.Router {
	t.Helper()
	router := httprouter.New()
	h := func(w http.ResponseWriter, r *http.Request) {}

	router.HandlerFunc(http.MethodPost, "/api/kortsaet", h)
	router.HandlerFunc(http.MethodPut, "/api/kortsaet", h)
	router.HandlerFunc(http.MethodPut, "/api/kortsaet/:id", h)
	router.HandlerFunc(http.MethodDelete, "/api/kortsaet/:id", h)
	router.HandlerFunc(http.MethodPut, "/api/kortsaet/:id/kort", h)

	router.HandlerFunc(http.MethodGet, "/api/kort", h)
	router.HandlerFunc(http.MethodPost, "/api/kort", h)
	router.HandlerFunc(http.MethodPut, "/api/kort/:id", h)
	router.HandlerFunc(http.MethodDelete, "/api/kort/:id", h)
	router.HandlerFunc(http.MethodPut, "/api/kort/:id/checkpoints", h)

	return router
}

func TestKortRoutesDoNotConflict(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("route registration panicked, so the API would not boot: %v", r)
		}
	}()
	kortRoutes(t)
}

// A set id must reach the record handler, and the collection must not be read as a set whose id
// is empty.
func TestKortsaetRoutesResolve(t *testing.T) {
	router := kortRoutes(t)

	for _, tc := range []struct {
		method, path string
		wantParam    string
	}{
		{http.MethodPost, "/api/kortsaet", ""},
		{http.MethodPut, "/api/kortsaet", ""},
		{http.MethodPut, "/api/kortsaet/kortsaet-1", "kortsaet-1"},
		{http.MethodDelete, "/api/kortsaet/kortsaet-1", "kortsaet-1"},
		{http.MethodPut, "/api/kortsaet/kortsaet-1/kort", "kortsaet-1"},
		{http.MethodGet, "/api/kort", ""},
		{http.MethodPost, "/api/kort", ""},
		{http.MethodPut, "/api/kort/kort-1", "kort-1"},
		{http.MethodDelete, "/api/kort/kort-1", "kort-1"},
		{http.MethodPut, "/api/kort/kort-1/checkpoints", "kort-1"},
	} {
		handler, params, _ := router.Lookup(tc.method, tc.path)
		if handler == nil {
			t.Errorf("%s %s: no handler", tc.method, tc.path)
			continue
		}
		if got := params.ByName("id"); got != tc.wantParam {
			t.Errorf("%s %s: id = %q, want %q", tc.method, tc.path, got, tc.wantParam)
		}
	}
}

// There is deliberately no per-sheet GET: the whole year is a handful of records, `GET /api/kort`
// returns all of it, and both the modal and the hej-app work from that one cached response.
func TestThereIsNoSingleMapRead(t *testing.T) {
	router := kortRoutes(t)

	if handler, _, _ := router.Lookup(http.MethodGet, "/api/kort/kort-1"); handler != nil {
		t.Error("GET /api/kort/:id exists; it would be a second read path with no caller")
	}
}

// The sort route is a PUT on the collection, so a set that happens to be *called* "sorted" is
// still an ordinary set and reachable by id. This is the property the rejected
// `/api/kortsaet/sorted` design could not have.
func TestSetNamedSortedIsStillAddressable(t *testing.T) {
	router := kortRoutes(t)

	handler, params, _ := router.Lookup(http.MethodPut, "/api/kortsaet/sorted")
	if handler == nil {
		t.Fatal("no handler for a set id of \"sorted\"")
	}
	if got := params.ByName("id"); got != "sorted" {
		t.Errorf("id = %q, want \"sorted\"", got)
	}
}
