package main

import (
	"context"
	"net/http"
	"sort"
	"strings"

	"github.com/nathejk/shared-go/types"
	jsonapi "nathejk.dk/cmd/api/app"
	"nathejk.dk/nathejk/table/checkpoint"
	"nathejk.dk/nathejk/table/patrulje"
)

// A started team's standing at one checkgroup.
//
// The same four tokens serve the counts on the post list and the rows in the dialog
// behind them, deliberately: an operator clicks a number expecting to see exactly
// that many teams, so the number and the list must be one computation with one
// vocabulary. Two code paths would agree only until the next change.
const (
	// TeamAtCheckgroupOnTime — scanned inside the post's open window.
	TeamAtCheckgroupOnTime = "onTime"
	// TeamAtCheckgroupLate — scanned, but outside the window.
	TeamAtCheckgroupLate = "late"
	// TeamAtCheckgroupRetired — udgået: nobody left racing, and no scan here.
	TeamAtCheckgroupRetired = "retired"
	// TeamAtCheckgroupMissing — started, still racing, not seen here.
	TeamAtCheckgroupMissing = "missing"
)

// startedTeam is the part of a patrol this computation needs.
type startedTeam struct {
	TeamID types.TeamID
	// ActiveMemberCount is the canonical test for udgået: zero means nobody is left
	// racing. Owned by the spejderstatus projection, so it is read rather than
	// re-derived here (see patrulje.Patrulje.ActiveMemberCount).
	ActiveMemberCount int
}

// checkgroupScan is what the stream knows about a team passing a checkgroup.
type checkgroupScan struct {
	TeamID types.TeamID
	// OnTime is true when at least one of the team's scans fell inside the window.
	OnTime bool
	// FirstUts is the earliest scan, which is what an operator wants to read: when
	// this team came through.
	FirstUts int64
}

// TeamAtCheckgroup is one row of the dialog behind a number.
type TeamAtCheckgroup struct {
	TeamID            types.TeamID      `json:"teamId"`
	TeamNumber        string            `json:"teamNumber"`
	Name              string            `json:"name"`
	Group             string            `json:"group"`
	ContactName       string            `json:"contactName"`
	ContactPhone      types.PhoneNumber `json:"contactPhone"`
	MemberCount       int               `json:"memberCount"`
	ActiveMemberCount int               `json:"activeMemberCount"`
	Status            string            `json:"status"`
	// ScannedAtUts is 0 for a team that was never seen here; omitted from the JSON so
	// the client cannot mistake the epoch for a time.
	ScannedAtUts int64 `json:"scannedAtUts,omitempty"`
}

// resolveTeamStatus places one team in exactly one bucket.
//
// # The order of the tests, which is the whole design
//
// A scan wins over being udgået, and being udgået wins over missing:
//
//   - A team that came through and *later* withdrew passed this post. Calling it
//     retired here would erase a fact the post staff recorded, and would make the
//     numbers disagree with the paper on the table.
//   - A team that withdrew before reaching the post is not missing. This is the case
//     the feature exists for: with a checkgroup about to close, the missing list is
//     who to chase, and it must not send anybody after a patrol that went home hours
//     ago.
//
// So per checkgroup a team's standing moves as the night goes on — on time at posts
// 1 and 2, retired at 3 and after — which is what makes each post's numbers about
// that post rather than about the race in general.
func resolveTeamStatus(team startedTeam, scan *checkgroupScan) (string, int64) {
	if scan != nil {
		if scan.OnTime {
			return TeamAtCheckgroupOnTime, scan.FirstUts
		}
		return TeamAtCheckgroupLate, scan.FirstUts
	}
	if team.ActiveMemberCount == 0 {
		return TeamAtCheckgroupRetired, 0
	}
	return TeamAtCheckgroupMissing, 0
}

// resolveTeamStatuses partitions every started team, whether or not it was ever seen.
//
// Driven by the team list rather than by the scans, which is the fix for the numbers
// not adding up: the old count started `missing` at the number of started teams and
// decremented per scanning team, so a scan by a team that had not started (or had
// been deleted) pushed the total off, and nothing ever landed in the retired bucket
// at all because nothing computed it.
func resolveTeamStatuses(started []startedTeam, scans map[types.TeamID]checkgroupScan) map[types.TeamID]struct {
	Status string
	Uts    int64
} {
	out := make(map[types.TeamID]struct {
		Status string
		Uts    int64
	}, len(started))
	for _, team := range started {
		var scan *checkgroupScan
		if s, ok := scans[team.TeamID]; ok {
			scan = &s
		}
		status, uts := resolveTeamStatus(team, scan)
		out[team.TeamID] = struct {
			Status string
			Uts    int64
		}{status, uts}
	}
	return out
}

// CheckgroupStats counts the four buckets. They sum to the number of started teams,
// by construction.
type CheckgroupStats struct {
	CheckgroupID types.CheckgroupID `json:"checkgroupId"`
	OnTime       int                `json:"onTime"`
	Late         int                `json:"late"`
	// Retired is udgået. Named for what it means; it was `expired` and always zero.
	Retired int `json:"retired"`
	Missing int `json:"missing"`
}

func countStatuses(cgID types.CheckgroupID, statuses map[types.TeamID]struct {
	Status string
	Uts    int64
}) CheckgroupStats {
	stats := CheckgroupStats{CheckgroupID: cgID}
	for _, s := range statuses {
		switch s.Status {
		case TeamAtCheckgroupOnTime:
			stats.OnTime++
		case TeamAtCheckgroupLate:
			stats.Late++
		case TeamAtCheckgroupRetired:
			stats.Retired++
		default:
			stats.Missing++
		}
	}
	return stats
}

// startedTeams lists the patrols on the route, with the strength that decides whether
// they are still on it.
//
// The narrow query, not GetAll: this runs on every revalidation of the post list, which
// during the race means every scan. See patrulje.StartedTeam.
func (app *application) startedTeams(ctx context.Context, year types.YearSlug) ([]startedTeam, error) {
	rows, err := app.models.Patrulje.GetStartedTeams(ctx, patrulje.Filter{YearSlug: year})
	if err != nil {
		return nil, err
	}
	teams := make([]startedTeam, 0, len(rows))
	for _, r := range rows {
		teams = append(teams, startedTeam{TeamID: r.TeamID, ActiveMemberCount: r.ActiveMemberCount})
	}
	return teams, nil
}

// startedPatruljer is the same set with the details the dialog lists — names, groups and
// the contact to ring about a missing patrol.
//
// The expensive query is confined to here on purpose: the dialog is opened by hand, a few
// times a night, while the counts are recomputed continuously.
func (app *application) startedPatruljer(ctx context.Context, year types.YearSlug) ([]patrulje.Patrulje, error) {
	all, err := app.models.Patrulje.GetAll(ctx, patrulje.Filter{YearSlug: year})
	if err != nil {
		return nil, err
	}
	started := make([]patrulje.Patrulje, 0, len(all))
	for _, p := range all {
		if p.SignupStatus == types.SignupStatusStarted {
			started = append(started, p)
		}
	}
	return started, nil
}

// scansByCheckgroup reports which teams have been scanned at which checkgroup.
//
// A scan carries no checkpoint: the only link is who scanned it and when, so a scan
// counts for a post if the scanner was on a registered shift there at that moment.
// That makes the postmandskab rota load-bearing for these numbers — with no shifts
// recorded, no scan can be attributed and every team reads as missing.
func (app *application) scansByCheckgroup(ctx context.Context, cgIDs []types.CheckgroupID) (map[types.CheckgroupID]map[types.TeamID]checkgroupScan, error) {
	out := map[types.CheckgroupID]map[types.TeamID]checkgroupScan{}
	if len(cgIDs) == 0 {
		return out, nil
	}

	query := `
		SELECT cpt.checkgroupId, s.teamId,
			MAX(CASE WHEN s.uts >= cpt.openFromUts AND s.uts <= cpt.openUntilUts THEN 1 ELSE 0 END) AS wasOnTime,
			MIN(s.uts) AS firstUts
		FROM scan s
		JOIN checkpersonnel cpn ON s.scannerId = cpn.userId AND s.uts >= cpn.startUts AND s.uts <= cpn.endUts
		JOIN checkpoint cpt ON cpn.checkpointId = cpt.id
		WHERE cpt.checkgroupId IN (?` + strings.Repeat(",?", len(cgIDs)-1) + `)
		GROUP BY cpt.checkgroupId, s.teamId`
	args := make([]any, len(cgIDs))
	for i, id := range cgIDs {
		args[i] = string(id)
	}

	rows, err := app.db.DB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var cgID types.CheckgroupID
		var teamID types.TeamID
		var wasOnTime int
		var firstUts int64
		if err := rows.Scan(&cgID, &teamID, &wasOnTime, &firstUts); err != nil {
			return nil, err
		}
		if out[cgID] == nil {
			out[cgID] = map[types.TeamID]checkgroupScan{}
		}
		out[cgID][teamID] = checkgroupScan{TeamID: teamID, OnTime: wasOnTime == 1, FirstUts: firstUts}
	}
	return out, rows.Err()
}

// checkgroupStats counts the buckets for several checkgroups at once.
//
// Shares scansByCheckgroup and resolveTeamStatuses with the dialog endpoint, so the
// number on the page and the rows behind it cannot disagree.
func (app *application) checkgroupStats(ctx context.Context, year types.YearSlug, cgIDs []types.CheckgroupID) ([]CheckgroupStats, int, error) {
	teams, err := app.startedTeams(ctx, year)
	if err != nil {
		return nil, 0, err
	}

	scans, err := app.scansByCheckgroup(ctx, cgIDs)
	if err != nil {
		return nil, 0, err
	}

	stats := make([]CheckgroupStats, 0, len(cgIDs))
	for _, cgID := range cgIDs {
		byTeam := scans[cgID]
		if byTeam == nil {
			byTeam = map[types.TeamID]checkgroupScan{}
		}
		stats = append(stats, countStatuses(cgID, resolveTeamStatuses(teams, byTeam)))
	}
	return stats, len(teams), nil
}

// checkgroupTeamsHandler serves every started team's standing at one checkgroup.
//
// A dedicated endpoint rather than folding the rows into the post list: seventy-six
// patrols across seven lines is five hundred rows on a page that revalidates on every
// scan during the race, and they are needed only when somebody opens the dialog.
func (app *application) checkgroupTeamsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	cgID := types.CheckgroupID(app.ReadNamedParam(r, "id"))
	cg, err := app.models.Checkgroup.GetByID(ctx, cgID)
	if err != nil {
		app.NotFoundResponse(w, r)
		return
	}

	started, err := app.startedPatruljer(ctx, cg.YearSlug)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}
	teams := make([]startedTeam, 0, len(started))
	for _, p := range started {
		teams = append(teams, startedTeam{TeamID: p.TeamID, ActiveMemberCount: p.ActiveMemberCount})
	}

	scans, err := app.scansByCheckgroup(ctx, []types.CheckgroupID{cgID})
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}
	byTeam := scans[cgID]
	if byTeam == nil {
		byTeam = map[types.TeamID]checkgroupScan{}
	}
	statuses := resolveTeamStatuses(teams, byTeam)

	rows := make([]TeamAtCheckgroup, 0, len(started))
	for _, p := range started {
		s := statuses[p.TeamID]
		rows = append(rows, TeamAtCheckgroup{
			TeamID:            p.TeamID,
			TeamNumber:        p.TeamNumber,
			Name:              p.Name,
			Group:             p.Group,
			ContactName:       p.ContactName,
			ContactPhone:      p.ContactPhone,
			MemberCount:       p.MemberCount,
			ActiveMemberCount: p.ActiveMemberCount,
			Status:            s.Status,
			ScannedAtUts:      s.Uts,
		})
	}
	// By team number, as the numbers are read out and written down. Numeric where the
	// number is numeric: string order would put 10 before 2.
	sort.SliceStable(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		na, aok := teamNumberValue(a.TeamNumber)
		nb, bok := teamNumberValue(b.TeamNumber)
		if aok && bok && na != nb {
			return na < nb
		}
		if aok != bok {
			return aok
		}
		return a.TeamNumber < b.TeamNumber
	})

	// When the line shuts. Taken as the latest of its posts' windows because that is
	// the moment after which a missing team can no longer come through anywhere on the
	// line — the deadline the operator is working against.
	var closesAtUts int64
	cps, _ := app.models.Checkpoint.GetAll(ctx, checkpoint.Filter{CheckgroupIDs: []types.CheckgroupID{cgID}})
	for _, cp := range cps {
		if cp.OpenUntil.IsZero() {
			continue
		}
		if uts := cp.OpenUntil.Unix(); uts > closesAtUts {
			closesAtUts = uts
		}
	}

	envelope := jsonapi.Envelope{
		"checkgroup":  cg,
		"teams":       rows,
		"stats":       countStatuses(cgID, statuses),
		"closesAtUts": closesAtUts,
	}
	if err := app.WriteJSON(w, http.StatusOK, envelope, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// teamNumberValue parses a team number for sorting, reporting whether it is numeric.
func teamNumberValue(number string) (int, bool) {
	if number == "" {
		return 0, false
	}
	value := 0
	for _, r := range number {
		if r < '0' || r > '9' {
			return 0, false
		}
		value = value*10 + int(r-'0')
	}
	return value, true
}
