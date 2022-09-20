package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
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

	db, err := sql.Open("mysql", os.Getenv("DB_DSN"))
	if err != nil {
		log.Fatal(err)
		panic("Can't connect to " + os.Getenv("DB_DSN"))
		return nil
	}
	log.Printf("Connected to " + os.Getenv("DB_DSN"))

	ctrlgrp := NewCrudRoute(NewControlGroupCmd(publisher), &CreateRequest{}, &ReadRequest{}, &UpdateRequest{}, &DeleteRequest{})
	sos := NewSosRoutes(NewSosCmd(publisher, state, sms))

	//s := server{router: http.NewServeMux()}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/base", NewBaseHandler())
	mux.HandleFunc("/api/cgstatus", NewControlgroupStatusHandler(db))
	mux.HandleFunc("/api/patrulje/", patruljeHandler(state))
	mux.HandleFunc("/api/teams", monolithHandler)
	mux.HandleFunc("/api/teams/", monolithTeamHandler)
	mux.HandleFunc("/api/personnel", api.HandleUser)
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
	//mux.Handle("", StaticFileHandler("/www/index.html"))

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
func NewControlgroupStatusHandler(con *sql.DB) http.HandlerFunc {
	type response struct {
		ControlGroups []json.RawMessage
	}
	cgIDs := func() (IDs []types.ControlGroupID) {
		rows, err := con.Query("SELECT controlGroupId FROM controlpoint GROUP BY controlGroupId")
		if err != nil {
			log.Fatal(err)
		}
		defer rows.Close()

		var cgID types.ControlGroupID
		for rows.Next() {
			if err := rows.Scan(&cgID); err != nil {
				log.Fatal(err)
			}
			IDs = append(IDs, cgID)
		}
		return
	}
	startedTeamIDs := func() []types.TeamID {
		teamIDs := []types.TeamID{}
		rows, err := con.Query("SELECT teamId FROM patruljestatus WHERE startedUts > 0 AND teamId >= 2022000")
		if err != nil {
			log.Fatal(err)
		}
		defer rows.Close()

		var teamID types.TeamID
		for rows.Next() {
			if err := rows.Scan(&teamID); err != nil {
				log.Fatal(err)
			}
			teamIDs = append(teamIDs, teamID)
		}
		return teamIDs
	}
	inactiveTeamIDs := func() []types.TeamID {
		teamIDs := []types.TeamID{}
		rows, err := con.Query("SELECT DISTINCT m.teamId FROM patruljemerged m JOIN patruljestatus s ON m.teamId = s.teamId WHERE s.startedUts > 0 AND m.teamId > 2022000")
		if err != nil {
			log.Fatal(err)
		}
		defer rows.Close()

		var teamID types.TeamID
		for rows.Next() {
			if err := rows.Scan(&teamID); err != nil {
				log.Fatal(err)
			}
			teamIDs = append(teamIDs, teamID)
		}
		return teamIDs
	}
	type scan struct {
		TeamID         types.TeamID
		TeamNumber     string
		Uts            int
		UserID         string
		ControlGroupID types.ControlGroupID
		ControlIndex   int
		OnTime         bool
	}
	allScans := func() (ss []scan) {
		rows, err := con.Query(`select teamId, teamNumber, uts, userId, cp.controlGroupId, cp.controlIndex, (openFromUts - 60*minusMinutes <= uts AND uts <= openUntilUts + 60*plusMinutes) AS ontime from scan 
  JOIN controlgroup_user cgu ON scan.scannerId = cgu.userId AND startUts <= uts AND uts <= endUts 
  JOIN controlpoint cp ON cgu.controlGroupId = cp.controlGroupId AND cgu.controlIndex = cp.controlIndex`)
		if err != nil {
			log.Fatalf("Query: %v", err)
		}
		for rows.Next() {
			var s scan
			err = rows.Scan(&s.TeamID, &s.TeamNumber, &s.Uts, &s.UserID, &s.ControlGroupID, &s.ControlIndex, &s.OnTime)
			if err != nil {
				log.Fatalf("Scan: %v", err)
			}
			ss = append(ss, s)
		}
		return
	}
	type counts struct {
		NotArrived types.TeamIDs
		OnTime     types.TeamIDs
		OverTime   types.TeamIDs
		Inactive   types.TeamIDs
	}

	type scans map[types.TeamID]types.UserID

	type cgCount struct {
		ontime  scans
		delayed scans
	}
	/*

	   controlCount := func (cgID types.ControlGroupID) (cc  cgCount) {

	*/
	return func(w http.ResponseWriter, r *http.Request) {
		allTeamIDs := startedTeamIDs()
		controlGroups := map[types.ControlGroupID]counts{}
		for _, cgID := range cgIDs() {
			controlGroups[cgID] = counts{
				//NotArrived: startedTeamIDs(),
				NotArrived: types.TeamIDs{},
				OnTime:     types.TeamIDs{},
				OverTime:   types.TeamIDs{},
				Inactive:   types.TeamIDs{},
			}
		}
		for _, scan := range allScans() {
			cg := controlGroups[scan.ControlGroupID]
			log.Printf("ADDING TeamID %q to %q", scan.TeamID, scan.ControlGroupID)
			if scan.OnTime {
				if !cg.OnTime.Exists(scan.TeamID) {
					cg.OnTime = append(cg.OnTime, scan.TeamID)
				}
			} else {
				if !cg.OverTime.Exists(scan.TeamID) {
					cg.OverTime = append(cg.OverTime, scan.TeamID)
				}
			}
			controlGroups[scan.ControlGroupID] = cg
		}

		cgs := map[types.ControlGroupID]counts{}
		for cgID, cg := range controlGroups {
			inactive := types.DiffTeamID(inactiveTeamIDs(), cg.OnTime, cg.OverTime)
			notArrived := types.DiffTeamID(allTeamIDs, cg.OnTime, cg.OverTime, inactive)
			cgs[cgID] = counts{
				NotArrived: notArrived,
				OnTime:     cg.OnTime,
				OverTime:   types.DiffTeamID(cg.OverTime, cg.OnTime),
				Inactive:   inactive,
			}
		}

		/*
		           for cgID, cg := range controlGroups {

		           }
		   		/*	rows, err := con.Query("select * from patruljestatus ps left join scan on ps.teamId = scan.teamId left join controlgroup_user cgu on scan.scannerId = cgu.userId")
		   			if err != nil {
		   				panic(err.Error())
		   			}

		   			defer rows.Close()
		   			resp := &response{}
		   			for _, j := range jsonify.Jsonify(rows) {
		   				log.Print(j)
		   				resp.ControlGroups = append(resp.ControlGroups, json.RawMessage(j))
		   			}*/
		json.NewEncoder(w).Encode(map[string]interface{}{
			"startedCount": len(allTeamIDs),
			//"scans":         allScans(),
			"controlGroups":  cgs,
			"startedTeamIds": allTeamIDs,
		})
		//w.Write([]byte(jsonify.Jsonify(rows)[0]))
	}
}
