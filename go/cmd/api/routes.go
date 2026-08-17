package main

import (
	"expvar"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/nathejk/shared-go/types"
	"nathejk.dk/internal/requestctx"
)

func (app *application) routes() http.Handler {
	router := httprouter.New()

	router.NotFound = http.HandlerFunc(app.NotFoundResponse)
	router.MethodNotAllowed = http.HandlerFunc(app.MethodNotAllowedResponse)

	router.HandlerFunc(http.MethodGet, "/api/home", app.homeHandler)

	router.HandlerFunc(http.MethodGet, "/api/years", app.listYearHandler)
	router.HandlerFunc(http.MethodPost, "/api/year/:slug", app.createYearHandler)
	router.HandlerFunc(http.MethodGet, "/api/year/:slug", app.yearHandler)
	router.HandlerFunc(http.MethodPatch, "/api/year/:slug", app.updateYearHandler)
	router.HandlerFunc(http.MethodDelete, "/api/year/:slug", app.deleteYearHandler)

	router.HandlerFunc(http.MethodGet, "/api/checkgroups", app.listCheckgroupsHandler)
	router.HandlerFunc(http.MethodPost, "/api/checkgroup", app.createCheckgroupHandler)
	router.HandlerFunc(http.MethodGet, "/api/checkgroup/:id", app.checkgroupHandler)
	router.HandlerFunc(http.MethodPut, "/api/checkgroup/:id", app.updateCheckgroupHandler)
	router.HandlerFunc(http.MethodDelete, "/api/checkgroup/:id", app.deleteCheckgroupHandler)
	router.HandlerFunc(http.MethodPut, "/api/checkgroups/sorted", app.sortCheckgroupsHandler)

	router.HandlerFunc(http.MethodPost, "/api/checkpersonnel", app.createCheckpersonnelHandler)
	router.HandlerFunc(http.MethodPut, "/api/checkpersonnel/:id", app.updateCheckpersonnelHandler)
	router.HandlerFunc(http.MethodDelete, "/api/checkpersonnel/:id", app.deleteCheckpersonnelHandler)

	router.HandlerFunc(http.MethodGet, "/api/personnel", app.listPersonnelHandler)
	router.HandlerFunc(http.MethodGet, "/api/orders", app.listOrdersHandler)
	router.HandlerFunc(http.MethodGet, "/api/order/:id", app.showOrderHandler)

	router.HandlerFunc(http.MethodGet, "/api/patrulje", app.showPatruljeListHandler)
	router.HandlerFunc(http.MethodGet, "/api/patrulje/:id", app.showPatruljeHandler)
	router.HandlerFunc(http.MethodPut, "/api/patrulje/:id", app.updatePatruljeHandler)
	router.HandlerFunc(http.MethodPut, "/api/patrulje/:id/start", app.startPatruljeHandler)
	router.HandlerFunc(http.MethodGet, "/api/patrulje/:id/scans", app.scansPatruljeHandler)
	router.HandlerFunc(http.MethodGet, "/api/lok/:id", app.showLokHandler)
	router.HandlerFunc(http.MethodPatch, "/api/lok/:id", app.updateLokHandler)
	router.HandlerFunc(http.MethodGet, "/api/lok", app.showLoksHandler)
	router.HandlerFunc(http.MethodPut, "/api/lok", app.updateLoksHandler)
	router.HandlerFunc(http.MethodDelete, "/api/lok/:id", app.deleteLokHandler)
	router.HandlerFunc(http.MethodGet, "/api/klan", app.showKlanListHandler)
	router.HandlerFunc(http.MethodGet, "/api/klan/:id", app.showKlanHandler)
	router.HandlerFunc(http.MethodPut, "/api/klan/:id", app.updateKlanHandler)
	router.HandlerFunc(http.MethodPatch, "/api/klan/:id", app.patchKlanHandler)
	router.HandlerFunc(http.MethodGet, "/api/badut", app.showBadutListHandler)

	// Members in our care (PRD 006). Event-wide rather than per case: a member we
	// are responsible for is our problem whether or not anybody opened a case.
	//
	// Plural `/api/members/care` rather than `/api/member/care`, because httprouter builds
	// one tree per method and cannot hold a static segment where a sibling route has a
	// wildcard — `GET /api/member/:memberId` below would conflict with it and panic the
	// router at boot. Plural is also the truer name: this is a fact about the population,
	// not about a member.
	router.HandlerFunc(http.MethodGet, "/api/members/care", app.showMemberCareHandler)

	// One member in full, for the detail modal on the case card: contact details, address,
	// birthday and the whole status history. Its own endpoint so the case payload does not
	// carry all of that for every member of every associated patrol.
	router.HandlerFunc(http.MethodGet, "/api/member/:memberId", app.showMemberHandler)

	// The member lifecycle write surface (PRD 006). Every one of these requires a
	// sosId: nothing changes a member's status or team without a case explaining why.
	//
	// Registered on the member rather than nested under /api/sos because a member's
	// status is a fact about the member; the case is why an operator was looking.
	router.HandlerFunc(http.MethodPut, "/api/member/:memberId/waiting", app.requestWaitingHandler)
	router.HandlerFunc(http.MethodPut, "/api/member/:memberId/racing", app.resumeRacingHandler)
	router.HandlerFunc(http.MethodPut, "/api/member/:memberId/status", app.overrideMemberStatusHandler)
	router.HandlerFunc(http.MethodPut, "/api/member/:memberId/team", app.moveMemberTeamHandler)

	// Collecting a whole patrol is one action on the *case*, not four on members:
	// three separate calls could half-succeed and split a team across two states.
	router.HandlerFunc(http.MethodPost, "/api/sos/:id/team/:teamId/collect", app.collectTeamHandler)
	// Likewise moving a below-strength patrol's remnants: one decision, one request, one
	// timeline entry.
	router.HandlerFunc(http.MethodPost, "/api/sos/:id/team/:teamId/move", app.moveMembersHandler)
	router.HandlerFunc(http.MethodGet, "/api/mail/recipients", app.mailRecipientsHandler)

	// Nødtelefon / SOS (PRD 001)
	router.HandlerFunc(http.MethodGet, "/api/sos", app.listSosHandler)
	router.HandlerFunc(http.MethodPost, "/api/sos", app.createSosHandler)
	router.HandlerFunc(http.MethodGet, "/api/sos/:id", app.showSosHandler)
	router.HandlerFunc(http.MethodPatch, "/api/sos/:id", app.patchSosHandler)
	router.HandlerFunc(http.MethodDelete, "/api/sos/:id", app.deleteSosHandler)
	router.HandlerFunc(http.MethodPost, "/api/sos/:id/comment", app.commentSosHandler)
	router.HandlerFunc(http.MethodPatch, "/api/sos/:id/comment/:commentId", app.updateSosCommentHandler)
	router.HandlerFunc(http.MethodPut, "/api/sos/:id/team/:teamId", app.associateSosTeamHandler)
	router.HandlerFunc(http.MethodDelete, "/api/sos/:id/team/:teamId", app.disassociateSosTeamHandler)

	// Organisation (sections + crew members + vehicles)
	router.HandlerFunc(http.MethodGet, "/api/organisation", app.showOrganisationHandler)
	router.HandlerFunc(http.MethodPost, "/api/organisation/copy-from/:sourceYear", app.copySectionsFromYearHandler)
	router.HandlerFunc(http.MethodPost, "/api/section", app.createSectionHandler)
	router.HandlerFunc(http.MethodPut, "/api/sections/sorted", app.sortSectionsHandler)
	router.HandlerFunc(http.MethodPatch, "/api/section/:slug", app.updateSectionHandler)
	router.HandlerFunc(http.MethodPut, "/api/section/:slug/parent", app.moveSectionHandler)
	router.HandlerFunc(http.MethodPut, "/api/section/:slug/sos-assignable", app.setSectionSosAssignableHandler)
	router.HandlerFunc(http.MethodDelete, "/api/section/:slug", app.deleteSectionHandler)
	router.HandlerFunc(http.MethodPost, "/api/crewmember", app.registerCrewMemberHandler)
	router.HandlerFunc(http.MethodPatch, "/api/crewmember/:userId", app.updateCrewMemberHandler)
	router.HandlerFunc(http.MethodDelete, "/api/crewmember/:userId", app.deleteCrewMemberHandler)
	router.HandlerFunc(http.MethodPut, "/api/crewmember/:userId/section", app.assignCrewMemberSectionHandler)
	router.HandlerFunc(http.MethodPost, "/api/vehicle", app.registerVehicleHandler)
	router.HandlerFunc(http.MethodPatch, "/api/vehicle/:vehicleId", app.updateVehicleHandler)
	router.HandlerFunc(http.MethodDelete, "/api/vehicle/:vehicleId", app.deleteVehicleHandler)
	router.HandlerFunc(http.MethodPut, "/api/vehicle/:vehicleId/section", app.assignVehicleSectionHandler)
	/*
		ctrlgrp := NewCrudRoute(NewControlGroupCmd(app.publisher), &CreateRequest{}, &ReadRequest{}, &UpdateRequest{}, &DeleteRequest{})
		router.HandlerFunc(http.MethodGet, "/api/cgstatus", checkgroup.NewControlgroupStatusHandler(app.db.DB()))
		router.HandlerFunc(http.MethodGet, "/api/controlgroup", ctrlgrp.Handler)
		router.HandlerFunc(http.MethodPost, "/api/controlgroup", ctrlgrp.Handler)
		router.HandlerFunc(http.MethodPut, "/api/controlgroup", ctrlgrp.Handler)
		router.HandlerFunc(http.MethodDelete, "/api/controlgroup", ctrlgrp.Handler)
	*/
	router.HandlerFunc(http.MethodGet, "/api/excel/klan", app.excelKlanHandler)
	router.HandlerFunc(http.MethodGet, "/api/excel/patrulje", app.excelPatruljeHandler)
	router.HandlerFunc(http.MethodGet, "/api/excel/personnel", app.excelPersonnelHandler)
	/*
		router.HandlerFunc(http.MethodPut, "/api/*filepath", app.cleo.ProxyHandler)
		router.HandlerFunc(http.MethodGet, "/api/*filepath", app.cleo.ProxyHandler)
		router.HandlerFunc(http.MethodPost, "/api/*filepath", app.cleo.ProxyHandler)
		router.HandlerFunc(http.MethodDelete, "/api/*filepath", app.cleo.ProxyHandler)
		router.HandlerFunc(http.MethodPatch, "/api/*filepath", app.cleo.ProxyHandler)
	*/
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(SpaFileSystem(http.Dir(app.config.webroot))))
	mux.HandleFunc("/api/v1/healthcheck", app.HealthcheckHandler)
	// Deliberately outside app.Metrics: metricsResponseWriter does not implement
	// http.Flusher, and a stream held open for hours would also be recorded as one
	// multi-hour request. ServeMux prefers the longer pattern, so this wins over
	// "/api/" below.
	mux.Handle("/api/stream", app.authenticate(app.streamHandler()))
	mux.Handle("/api/", app.Metrics(app.authenticate(router)))
	mux.Handle("/debug/vars", expvar.Handler())

	return mux
}

func (app *application) YearSlug(r *http.Request) types.YearSlug {
	yearSlug := types.YearSlug(r.Header.Get("X-YearSlug"))
	if len(yearSlug) > 0 {
		return yearSlug
	}
	return types.YearSlug(fmt.Sprintf("%d", time.Now().Year()))
}

type spaFileSystem struct {
	root http.FileSystem
}

func (fs *spaFileSystem) Open(name string) (http.File, error) {
	f, err := fs.root.Open(name)
	if os.IsNotExist(err) {
		return fs.root.Open("index.html")
	}
	return f, err
}
func SpaFileSystem(fs http.FileSystem) *spaFileSystem {
	return &spaFileSystem{root: fs}
}

// authenticate populates the request context with the acting user.
//
// Authentication itself lives in an external service (AUTH_BASEURL,
// lukmigind.nathejk.dk), which is why there is no token handling here: every
// request is currently attributed to an anonymous user.
func (app *application) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := requestctx.WithUser(r.Context(), &requestctx.User{ID: types.UserID(""), Name: "anonymous"})

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
