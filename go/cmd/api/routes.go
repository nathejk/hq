package main

import (
	"database/sql"
	"expvar"
	"log"
	"net/http"
	"os"

	"github.com/julienschmidt/httprouter"
	"nathejk.dk/cmd/api/handlers"
	"nathejk.dk/cmd/api/user"
	"nathejk.dk/nathejk/commands"
	"nathejk.dk/nathejk/table"
)

func (app *application) routes() http.Handler {
	router := httprouter.New()

	router.NotFound = http.HandlerFunc(app.NotFoundResponse)
	router.MethodNotAllowed = http.HandlerFunc(app.MethodNotAllowedResponse)

	router.HandlerFunc(http.MethodGet, "/api/v1/healthcheck", app.HealthcheckHandler)

	cmd := NewCommander(app.publisher)
	api := NewApi(cmd)

	db, err := sql.Open("mysql", app.config.db.dsn)
	if err != nil {
		log.Fatal(err)
		panic("Can't connect to " + app.config.db.dsn)
		return nil
	}
	log.Printf("Connected to " + app.config.db.dsn)

	ctrlgrp := NewCrudRoute(NewControlGroupCmd(app.publisher), &CreateRequest{}, &ReadRequest{}, &UpdateRequest{}, &DeleteRequest{})
	sos := NewSosRoutes(NewSosCmd(app.publisher, app.state, app.sms))

	//s := server{router: http.NewServeMux()}
	//mux := http.NewServeMux()
	//mux := chi.NewRouter()
	//mux.Use(middleware.Logger)

	router.HandlerFunc(http.MethodGet, "/api/base", NewBaseHandler())
	router.HandlerFunc(http.MethodGet, "/api/user", user.ShowUserFromCookieHandler(os.Getenv("JWT_COOKIE_NAME"), os.Getenv("AUTH_BASEURL")))
	router.HandlerFunc(http.MethodGet, "/api/cgstatus", NewControlgroupStatusHandler(db))
	router.HandlerFunc(http.MethodGet, "/api/patrulje/", patruljeHandler(app.state))
	router.HandlerFunc(http.MethodGet, "/api/teams", monolithHandler)
	router.HandlerFunc(http.MethodGet, "/api/teams/", monolithTeamHandler)
	router.HandlerFunc(http.MethodDelete, "/api/personnel", api.HandleUser)
	router.HandlerFunc(http.MethodPost, "/api/personnel", api.HandleUser)
	router.HandlerFunc(http.MethodGet, "/api/controlgroup", ctrlgrp.Handler)
	router.HandlerFunc(http.MethodPut, "/api/controlgroup", ctrlgrp.Handler)
	router.HandlerFunc(http.MethodPost, "/api/controlgroup", ctrlgrp.Handler)
	router.HandlerFunc(http.MethodDelete, "/api/controlgroup", ctrlgrp.Handler)
	router.HandlerFunc(http.MethodPut, "/api/sos", sos.Handler)
	router.HandlerFunc(http.MethodPost, "/api/sos", sos.Handler)
	router.HandlerFunc(http.MethodDelete, "/api/sos", sos.Handler)
	router.HandlerFunc(http.MethodPut, "/api/sos/:cmd", sos.Handler)
	router.HandlerFunc(http.MethodPost, "/api/sos/:cmd", sos.Handler)
	router.HandlerFunc(http.MethodDelete, "/api/sos/:cmd", sos.Handler)

	router.HandlerFunc(http.MethodGet, "/api/checkgroups", app.listCheckgroupsHandler)
	router.HandlerFunc(http.MethodGet, "/api/patruljer", app.listPatruljerHandler)
	router.HandlerFunc(http.MethodGet, "/api/patrulje/:id", app.viewPatruljerHandler)

	router.HandlerFunc(http.MethodGet, "/api/years", app.listYearsHandler)

	depQuerier := table.DepartmentQuerier(db)
	depCmd := commands.NewDepartment(depQuerier, app.publisher)
	router.HandlerFunc(http.MethodPut, "/api/department", handlers.CreateDepartment(depCmd))
	router.HandlerFunc(http.MethodPost, "/api/department", handlers.UpdateDepartment(depCmd))
	router.HandlerFunc(http.MethodDelete, "/api/department", handlers.DeleteDepartment(depCmd))

	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(SpaFileSystem(http.Dir(app.config.webroot))))
	mux.Handle("/api/", app.Metrics(router))
	mux.Handle("/ws", auth(app.state))
	mux.Handle("/debug/vars", expvar.Handler())

	return mux
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
