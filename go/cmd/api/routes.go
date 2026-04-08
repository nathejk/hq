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

	router.HandlerFunc(http.MethodPost, "/api/signup", app.signupHandler)
	router.HandlerFunc(http.MethodPost, "/api/signup/pincode", app.signupPincodeHandler)
	router.HandlerFunc(http.MethodGet, "/api/signup/:id", app.showSignupHandler)
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
	router.HandlerFunc(http.MethodGet, "/api/mail/recipients", app.mailRecipientsHandler)
	/*
		ctrlgrp := NewCrudRoute(NewControlGroupCmd(app.jetstream), &CreateRequest{}, &ReadRequest{}, &UpdateRequest{}, &DeleteRequest{})
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
	mux.Handle("/api/", app.Metrics(app.authenticate(router)))
	mux.Handle("/confirm/", router)
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

func (app *application) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		/*
			// Add the "Vary: Authorization" header to the response. This indicates to any
			// caches that the response may vary based on the value of the Authorization
			// header in the request.
			w.Header().Add("Vary", "Authorization")
			// Retrieve the value of the Authorization header from the request. This will
			// return the empty string "" if there is no such header found.
			authorizationHeader := r.Header.Get("Authorization")
			// If there is no Authorization header found, use the contextSetUser() helper
			// that we just made to add the AnonymousUser to the request context. Then we
			// call the next handler in the chain and return without executing any of the
			// code below.
			if authorizationHeader == "" {
				r = app.contextSetUser(r, data.AnonymousUser)
				next.ServeHTTP(w, r)
				return
			}
			// Otherwise, we expect the value of the Authorization header to be in the format
			// "Bearer <token>". We try to split this into its constituent parts, and if the
			// header isn't in the expected format we return a 401 Unauthorized response
			// using the invalidAuthenticationTokenResponse() helper (which we will create
			// in a moment).
			headerParts := strings.Split(authorizationHeader, " ")
			if len(headerParts) != 2 || headerParts[0] != "Bearer" {
				app.invalidAuthenticationTokenResponse(w, r)
				return
			}
			// Extract the actual authentication token from the header parts.
			token := headerParts[1]
			// Validate the token to make sure it is in a sensible format.
			v := validator.New()
			// If the token isn't valid, use the invalidAuthenticationTokenResponse()
			// helper to send a response, rather than the failedValidationResponse() helper
			// that we'd normally use.
			if data.ValidateTokenPlaintext(v, token); !v.Valid() {
				app.invalidAuthenticationTokenResponse(w, r)
				return
			}
			// Retrieve the details of the user associated with the authentication token,
			// again calling the invalidAuthenticationTokenResponse() helper if no
			// matching record was found. IMPORTANT: Notice that we are using
			// ScopeAuthentication as the first parameter here.
			user, err := app.models.Users.GetForToken(data.ScopeAuthentication, token)
			if err != nil {
				switch {
				case errors.Is(err, data.ErrRecordNotFound):
					app.invalidAuthenticationTokenResponse(w, r)
				default:
					app.serverErrorResponse(w, r, err)
				}
				return
			}
			// Call the contextSetUser() helper to add the user information to the request
			// context.
			r = app.contextSetUser(r, user)
		*/

		ctx := requestctx.WithUser(r.Context(), &requestctx.User{ID: types.UserID(""), Name: "anonymous"})

		// Call the next handler in the chain.
		next.ServeHTTP(w, r.WithContext(ctx))

		//		next.ServeHTTP(w, r)
	})
}
