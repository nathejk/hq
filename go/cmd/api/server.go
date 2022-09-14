package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strconv"

	"github.com/rs/cors"

	"nathejk.dk/nathejk/aggregate/team"
	"nathejk.dk/nathejk/types"
	"nathejk.dk/pkg/notification"
	"nathejk.dk/pkg/streaminterface"
)

type server struct {
	//    db     *someDatabase
	//router *apiRouter
	//    email  EmailSender
	mux *http.ServeMux
}

type StateReader interface {
	http.Handler
	Read(string, string, interface{}) error
}

func NewServer(publisher streaminterface.Publisher, state StateReader, sms notification.SmsSender) *server {
	cmd := NewCommander(publisher)
	api := NewApi(cmd)

	ctrlgrp := NewCrudRoute(NewControlGroupCmd(publisher), &CreateRequest{}, &ReadRequest{}, &UpdateRequest{}, &DeleteRequest{})
	sos := NewSosRoutes(NewSosCmd(publisher, state, sms))

	//s := server{router: http.NewServeMux()}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/base", NewBaseHandler())
	mux.HandleFunc("/api/patrulje/", patruljeHandler(state))
	mux.HandleFunc("/api/teams", monolithHandler)
	mux.HandleFunc("/api/teams/", monolithTeamHandler)
	mux.HandleFunc("/api/user", api.HandleUser)
	mux.HandleFunc("/api/controlgroup", ctrlgrp.Handler)
	mux.HandleFunc("/api/sos", sos.Handler)
	mux.HandleFunc("/api/sos/", sos.Handler)

	mux.Handle("/ws", auth(state))

	mux.Handle("/js/", http.FileServer(http.Dir("/www/")))
	mux.Handle("/css/", http.FileServer(http.Dir("/www/")))
	mux.Handle("/img/", http.FileServer(http.Dir("/www/")))
	mux.Handle("/app.bundle.js", StaticFileHandler("/www/app.bundle.js"))
	mux.Handle("/app.bundle.js.map", StaticFileHandler("/www/app.bundle.js.map"))
	mux.Handle("/config.js", StaticFileHandler("/www/config.js"))

	mux.Handle("/assets/", http.FileServer(http.Dir("/www/")))
	mux.Handle("/favicon.ico", StaticFileHandler("/www/favicon.ico"))

	mux.Handle("/index.html", StaticFileHandler("/www/index.html"))
	mux.Handle("/", StaticFileHandler("/www/index.html"))
	mux.Handle("", StaticFileHandler("/www/index.html"))

	return &server{mux: mux}
}

func (s *server) ListenAndServe(addr string) error {
	// Start listening for incoming messages to websocket
	go handleMessages()

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://*.nathejk.dk", "https://*.nathejk.dk"},
		AllowCredentials: true,
		AllowedMethods:   []string{"PUT", "GET", "POST", "DELETE"},
		Debug:            true,
	})
	return http.ListenAndServe(addr, c.Handler(s.mux))
}

func StaticFileHandler(entrypoint string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, entrypoint)
	})
}

func userHandler(w http.ResponseWriter, r *http.Request) {
	_, err := r.Cookie(os.Getenv("JWT_COOKIE_NAME"))
	if err != nil {
		fmt.Fprintf(w, "Unauth")
		return
	}
}
func patruljeHandler(state StateReader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		teamID := r.URL.Path[len("/api/patrulje/"):]
		var t team.TeamMembersAggregate
		state.Read("patrulje", teamID, &t)
		json.NewEncoder(w).Encode(t)
	}
}

func auth(next http.Handler) http.HandlerFunc {
	// $jwt = $_COOKIE[getenv('JWT_COOKIE_NAME')] ?? "none";
	// $json = @file_get_contents(getenv('AUTH_BASEURL')."/token/" . $jwt);
	// if ($USER = json_decode($json)) {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, _ := r.Cookie(os.Getenv("JWT_COOKIE_NAME"))
		resp, err := http.Get(os.Getenv("AUTH_BASEURL") + "/token/" + cookie.Value)
		if err != nil || resp.StatusCode != http.StatusOK {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	}
}
func monolithTeamHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(os.Getenv("JWT_COOKIE_NAME"))
	if err != nil {
		fmt.Fprintf(w, "Unauth")
		return
	}
	b, e := HTTPwithCookies(os.Getenv("MONOLITH_BASEURL")+"/api/team.php?id="+path.Base(r.URL.Path), cookie)
	if e != nil {
		panic(e)
	}

	fmt.Fprintf(w, string(b))
	//fmt.Println(string(b))
}
func monolithHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(os.Getenv("JWT_COOKIE_NAME"))
	if err != nil {
		fmt.Fprintf(w, "Unauth")
		return
	}
	b, e := HTTPwithCookies(os.Getenv("MONOLITH_BASEURL")+"/api/team.php", cookie)
	if e != nil {
		panic(e)
	}
	//resp.Body.Close()

	fmt.Fprintf(w, string(b))
	//fmt.Println(string(b))
}
func HTTPwithCookies(url string, cookie *http.Cookie) (b []byte, err error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}
	req.AddCookie(cookie)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		err = errors.New(url +
			"\nresp.StatusCode: " + strconv.Itoa(resp.StatusCode))
		return
	}

	return ioutil.ReadAll(resp.Body)
}

func IndexHandler(entrypoint string) func(w http.ResponseWriter, r *http.Request) {
	fn := func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, entrypoint)
	}
	return http.HandlerFunc(fn)
}
func NewBaseHandler() http.HandlerFunc {
	type corps struct {
		Slug  types.CorpsSlug
		Label string
	}
	type response struct {
		Build   string  `json:"build"`
		Corpses []corps `json:"corpses"`
	}
	corpses := []corps{}
	for _, slug := range types.CorpsSlugs {
		corpses = append(corpses, corps{Slug: slug, Label: slug.String()})
	}
	return func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(&response{
			Build:   "dev",
			Corpses: corpses,
		})
	}
}
