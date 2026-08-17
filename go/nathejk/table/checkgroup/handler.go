package checkgroup

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"github.com/nathejk/shared-go/types"
)

func NewControlgroupStatusHandler(con *sql.DB) http.HandlerFunc {
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
		// Was: SELECT DISTINCT m.teamId FROM patruljemerged m JOIN patruljestatus s
		//      ON m.teamId = s.teamId WHERE s.startedUts > 0 AND m.teamId > 2022000
		//
		// The patruljemerged table is gone (PRD 006, task 069). Teams are no longer
		// merged and split; members are moved, and a team that started and has no
		// racing members left is discontinued — readable as
		// patrulje.activeMemberCount = 0 AND patrulje.signupStatus = 'STARTED'.
		//
		// Returning empty rather than rewriting the query, because this whole handler
		// is unreachable: it is only wired from the commented-out /api/cgstatus route
		// in cmd/api/routes.go, and everything checkgroup-related is explicitly out of
		// scope for PRD 006 (§4). Whoever revives this screen should use the predicate
		// above — and note the "and started" half is not optional: without it every
		// team in a year that has not raced yet counts as inactive.
		return []types.TeamID{}
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
			log.Printf("Query: %v", err)
			return ss
		}
		for rows.Next() {
			var s scan
			err = rows.Scan(&s.TeamID, &s.TeamNumber, &s.Uts, &s.UserID, &s.ControlGroupID, &s.ControlIndex, &s.OnTime)
			if err != nil {
				log.Printf("Scan: %v", err)
				continue
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
