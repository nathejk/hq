package main

import (
	"context"
	"net/http"
	"sort"
	"strings"

	"github.com/nathejk/shared-go/types"
	jsonapi "nathejk.dk/cmd/api/app"
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
	// Which window depends on the line's scheme — see checkgroupWindow.
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
	// DeadlineUts is when this team was due here. Per team, because on a relative line
	// the clock starts when that team left the previous line — so two teams an hour
	// apart have deadlines an hour apart. Omitted when there is none to state.
	DeadlineUts int64 `json:"deadlineUts,omitempty"`
	// Distance is how far the team is from this line, for a team that has not arrived.
	//
	// Set only for teams with no scan here: for one that came through, "how far away"
	// is a question about the rest of its night and not about this post. Nil whenever
	// HQ does not know where the team is, which is the majority case and not an error
	// — see distanceToLine.
	Distance *teamDistance `json:"distance,omitempty"`
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

// checkgroupWindow is a line's opening scheme, as far as timeliness is concerned.
//
// A postlinje either opens at wall-clock times (`fixed`), or gives every team its own
// deadline measured from when that team came through an earlier line (`relative`), or
// keeps no time at all (`none`).
type checkgroupWindow struct {
	Scheme types.CheckgroupScheme
	// RelativeTo is the line the deadline is measured from, for the relative scheme.
	RelativeTo types.CheckgroupID
}

// rawScan is one attributed scan: a team seen at one checkpoint of one line.
type rawScan struct {
	CheckgroupID types.CheckgroupID
	TeamID       types.TeamID
	Uts          int64
	// OpenFromUts/OpenUntilUts are the checkpoint's wall-clock window, meaningful
	// under the fixed scheme only; both are 0 for a relative line.
	OpenFromUts  int64
	OpenUntilUts int64
	// OpenDurationMinutes is the checkpoint's allowance under the relative scheme:
	// the "+N minutter" the postlinje dialog edits.
	OpenDurationMinutes int64
}

// scanWasOnTime judges one scan against its line's scheme.
//
// baseUts is the team's own reference time at the line this one is relative to, and 0
// when the team has not been seen there. That case reads as on time on purpose: with
// no reference scan there is no deadline to have missed, and the alternative — the
// behaviour this replaced — marks a team late for a window that was never computed.
func scanWasOnTime(s rawScan, window checkgroupWindow, baseUts int64) bool {
	switch window.Scheme {
	case types.CheckgroupSchemeRelative:
		if baseUts == 0 {
			return true
		}
		// No lower bound: arriving early is not an offence, and a post cannot be
		// reached before the team left the line it is measured from anyway.
		return s.Uts <= baseUts+s.OpenDurationMinutes*60
	case types.CheckgroupSchemeNone:
		return true
	default:
		// Fixed, and the empty scheme older records carry, which query.go also reads
		// as fixed. An unset window means nothing to be late for.
		if s.OpenFromUts == 0 && s.OpenUntilUts == 0 {
			return true
		}
		return s.Uts >= s.OpenFromUts && s.Uts <= s.OpenUntilUts
	}
}

// lastScanByTeam is each team's latest scan per line.
//
// This is the reference a relative line measures from: the moment the team left the
// line it points to, which is its last scan there rather than its first.
func lastScanByTeam(scans []rawScan) map[types.CheckgroupID]map[types.TeamID]int64 {
	base := map[types.CheckgroupID]map[types.TeamID]int64{}
	for _, s := range scans {
		if base[s.CheckgroupID] == nil {
			base[s.CheckgroupID] = map[types.TeamID]int64{}
		}
		if s.Uts > base[s.CheckgroupID][s.TeamID] {
			base[s.CheckgroupID][s.TeamID] = s.Uts
		}
	}
	return base
}

// aggregateScans folds attributed scans into one verdict per team per line.
//
// A team counts as on time at a line if *any* of its scans there was in time, and the
// arrival shown is its earliest scan — the same rule for every scheme.
func aggregateScans(scans []rawScan, windows map[types.CheckgroupID]checkgroupWindow) map[types.CheckgroupID]map[types.TeamID]checkgroupScan {
	// A relative line needs every team's time at the line it points to, which is why
	// the scans are fetched for those lines as well.
	base := lastScanByTeam(scans)

	out := map[types.CheckgroupID]map[types.TeamID]checkgroupScan{}
	for _, s := range scans {
		window := windows[s.CheckgroupID]
		var baseUts int64
		if window.Scheme == types.CheckgroupSchemeRelative {
			baseUts = base[window.RelativeTo][s.TeamID]
		}
		if out[s.CheckgroupID] == nil {
			out[s.CheckgroupID] = map[types.TeamID]checkgroupScan{}
		}
		agg, seen := out[s.CheckgroupID][s.TeamID]
		if !seen {
			agg = checkgroupScan{TeamID: s.TeamID, FirstUts: s.Uts}
		}
		if s.Uts < agg.FirstUts {
			agg.FirstUts = s.Uts
		}
		agg.OnTime = agg.OnTime || scanWasOnTime(s, window, baseUts)
		out[s.CheckgroupID][s.TeamID] = agg
	}
	return out
}

// checkgroupWindows reads the schemes of the given lines, plus the lines any of them
// measure from, so a relative deadline can be resolved without a second round trip.
//
// The returned ID slice is what the scan query must cover: the requested lines and
// their references. A reference chain is followed one hop only — the dialog lets a line
// be relative to another line, not to a chain, and a cycle would not terminate.
func (app *application) checkgroupWindows(ctx context.Context, cgIDs []types.CheckgroupID) (map[types.CheckgroupID]checkgroupWindow, []types.CheckgroupID, error) {
	query := `SELECT id, scheme, relativeCheckgroupId FROM checkgroup WHERE id IN (?` + strings.Repeat(",?", len(cgIDs)-1) + `)`
	args := make([]any, len(cgIDs))
	for i, id := range cgIDs {
		args[i] = string(id)
	}
	rows, err := app.db.DB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	windows := map[types.CheckgroupID]checkgroupWindow{}
	needed := map[types.CheckgroupID]bool{}
	for _, id := range cgIDs {
		needed[id] = true
	}
	for rows.Next() {
		var id types.CheckgroupID
		var scheme types.CheckgroupScheme
		var relativeTo types.CheckgroupID
		if err := rows.Scan(&id, &scheme, &relativeTo); err != nil {
			return nil, nil, err
		}
		windows[id] = checkgroupWindow{Scheme: scheme, RelativeTo: relativeTo}
		if scheme == types.CheckgroupSchemeRelative && relativeTo != "" {
			needed[relativeTo] = true
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	all := make([]types.CheckgroupID, 0, len(needed))
	for id := range needed {
		all = append(all, id)
	}
	sort.Slice(all, func(i, j int) bool { return all[i] < all[j] })
	return windows, all, nil
}

// checkgroupLimit is how long a line stays open, taken across all its posts.
//
// The widest of them, because that is the moment after which a team can no longer come
// through anywhere on the line — the deadline an operator is working against, and the
// same rule for both schemes so the two read alike.
type checkgroupLimit struct {
	// ClosesAtUts is the latest wall-clock closing time, for a fixed line.
	ClosesAtUts int64
	// AllowanceSeconds is the largest "+N minutter", for a relative line.
	AllowanceSeconds int64
}

// checkgroupLimits reads the opening hours of the lines' posts.
//
// Read from the checkpoint table rather than from the scan rows: a post nobody has
// scanned at yet still has opening hours, and it is precisely the teams with no scan
// whose deadline the operator wants to see.
func (app *application) checkgroupLimits(ctx context.Context, cgIDs []types.CheckgroupID) (map[types.CheckgroupID]checkgroupLimit, error) {
	query := `SELECT checkgroupId, MAX(openUntilUts), MAX(openDuration) FROM checkpoint
		WHERE checkgroupId IN (?` + strings.Repeat(",?", len(cgIDs)-1) + `) GROUP BY checkgroupId`
	args := make([]any, len(cgIDs))
	for i, id := range cgIDs {
		args[i] = string(id)
	}
	rows, err := app.db.DB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	limits := map[types.CheckgroupID]checkgroupLimit{}
	for rows.Next() {
		var id types.CheckgroupID
		var closesAt, durationMinutes int64
		if err := rows.Scan(&id, &closesAt, &durationMinutes); err != nil {
			return nil, err
		}
		limits[id] = checkgroupLimit{ClosesAtUts: closesAt, AllowanceSeconds: durationMinutes * 60}
	}
	return limits, rows.Err()
}

// teamDeadline is when this team has to have been through this line, or 0 when that is
// not knowable.
//
// Zero is a real answer, not a failure: a relative line has no deadline for a team that
// has not reached the line it measures from, and a line with no opening hours has none
// at all. The client renders it as "no deadline" rather than as a time.
func teamDeadline(window checkgroupWindow, limit checkgroupLimit, baseUts int64) int64 {
	switch window.Scheme {
	case types.CheckgroupSchemeRelative:
		if baseUts == 0 || limit.AllowanceSeconds == 0 {
			return 0
		}
		return baseUts + limit.AllowanceSeconds
	case types.CheckgroupSchemeNone:
		return 0
	default:
		return limit.ClosesAtUts
	}
}

// checkgroupTiming is everything the post list and its dialog need to know about time
// at a set of lines: who was seen when, and by when each team was due.
//
// One value so the counts and the rows behind them are computed from the same reads.
type checkgroupTiming struct {
	windows map[types.CheckgroupID]checkgroupWindow
	limits  map[types.CheckgroupID]checkgroupLimit
	// lastScan covers the referenced lines too, which is what a relative deadline is
	// measured from.
	lastScan map[types.CheckgroupID]map[types.TeamID]int64
	scans    map[types.CheckgroupID]map[types.TeamID]checkgroupScan
}

// Scans is the per-team verdict at one line; never nil, so callers need no guard.
func (t *checkgroupTiming) Scans(cgID types.CheckgroupID) map[types.TeamID]checkgroupScan {
	if byTeam := t.scans[cgID]; byTeam != nil {
		return byTeam
	}
	return map[types.TeamID]checkgroupScan{}
}

// Deadline is when the given team was due at the given line, or 0 if there is none.
func (t *checkgroupTiming) Deadline(cgID types.CheckgroupID, teamID types.TeamID) int64 {
	window := t.windows[cgID]
	var baseUts int64
	if window.Scheme == types.CheckgroupSchemeRelative {
		baseUts = t.lastScan[window.RelativeTo][teamID]
	}
	return teamDeadline(window, t.limits[cgID], baseUts)
}

// ClosesAt is the line's own closing time, which exists only for a fixed line: on a
// relative one every team has its own, so there is no single moment to show.
func (t *checkgroupTiming) ClosesAt(cgID types.CheckgroupID) int64 {
	if t.windows[cgID].Scheme == types.CheckgroupSchemeRelative {
		return 0
	}
	return teamDeadline(t.windows[cgID], t.limits[cgID], 0)
}

// checkgroupTimings loads the schemes, opening hours and attributed scans for the given
// lines.
//
// A scan carries no checkpoint: the only link is who scanned it and when, so a scan
// counts for a post if the scanner was on a registered shift there at that moment.
// That makes the postmandskab rota load-bearing for these numbers — with no shifts
// recorded, no scan can be attributed and every team reads as missing.
//
// Timeliness is decided in Go rather than in the SQL it used to be, because a relative
// line's deadline is per team: it depends on when *that* team came through the line it
// is measured from, which one CASE over the checkpoint's columns cannot express.
func (app *application) checkgroupTimings(ctx context.Context, cgIDs []types.CheckgroupID) (*checkgroupTiming, error) {
	timing := &checkgroupTiming{
		windows:  map[types.CheckgroupID]checkgroupWindow{},
		limits:   map[types.CheckgroupID]checkgroupLimit{},
		lastScan: map[types.CheckgroupID]map[types.TeamID]int64{},
		scans:    map[types.CheckgroupID]map[types.TeamID]checkgroupScan{},
	}
	if len(cgIDs) == 0 {
		return timing, nil
	}

	windows, queryIDs, err := app.checkgroupWindows(ctx, cgIDs)
	if err != nil {
		return nil, err
	}
	timing.windows = windows

	limits, err := app.checkgroupLimits(ctx, queryIDs)
	if err != nil {
		return nil, err
	}
	timing.limits = limits

	query := `
		SELECT cpt.checkgroupId, s.teamId, s.uts, cpt.openFromUts, cpt.openUntilUts, cpt.openDuration
		FROM scan s
		JOIN checkpersonnel cpn ON s.scannerId = cpn.userId AND s.uts >= cpn.startUts AND s.uts <= cpn.endUts
		JOIN checkpoint cpt ON cpn.checkpointId = cpt.id
		WHERE cpt.checkgroupId IN (?` + strings.Repeat(",?", len(queryIDs)-1) + `)`
	args := make([]any, len(queryIDs))
	for i, id := range queryIDs {
		args[i] = string(id)
	}

	rows, err := app.db.DB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var scans []rawScan
	for rows.Next() {
		var s rawScan
		if err := rows.Scan(&s.CheckgroupID, &s.TeamID, &s.Uts, &s.OpenFromUts, &s.OpenUntilUts, &s.OpenDurationMinutes); err != nil {
			return nil, err
		}
		scans = append(scans, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	timing.lastScan = lastScanByTeam(scans)
	aggregated := aggregateScans(scans, windows)
	// Only the lines that were asked for: the extra lines were read to resolve
	// deadlines, and returning them would count teams at a line nobody asked about.
	for _, id := range cgIDs {
		if byTeam, ok := aggregated[id]; ok {
			timing.scans[id] = byTeam
		}
	}
	return timing, nil
}

// checkgroupStats counts the buckets for several checkgroups at once.
//
// Shares checkgroupTimings and resolveTeamStatuses with the dialog endpoint, so the
// number on the page and the rows behind it cannot disagree.
func (app *application) checkgroupStats(ctx context.Context, year types.YearSlug, cgIDs []types.CheckgroupID) ([]CheckgroupStats, int, error) {
	teams, err := app.startedTeams(ctx, year)
	if err != nil {
		return nil, 0, err
	}

	timing, err := app.checkgroupTimings(ctx, cgIDs)
	if err != nil {
		return nil, 0, err
	}

	stats := make([]CheckgroupStats, 0, len(cgIDs))
	for _, cgID := range cgIDs {
		stats = append(stats, countStatuses(cgID, resolveTeamStatuses(teams, timing.Scans(cgID))))
	}
	return stats, len(teams), nil
}

// CheckpointStats is how many distinct teams were seen at one post.
//
// Per post, not per line: a line's four numbers say whether a team came through it at
// all, which on a multi-post line hides that one post is carrying everybody and another
// has seen nobody — either because its shift is unmanned or because the teams are not
// finding it.
type CheckpointStats struct {
	CheckgroupID types.CheckgroupID `json:"checkgroupId"`
	CheckpointID types.CheckpointID `json:"checkpointId"`
	TeamCount    int                `json:"teamCount"`
}

// checkpointStats counts the distinct teams scanned at each post of the given lines.
//
// Attribution is the same join as checkgroupTimings — a scan belongs to the post whose
// registered shift the scanner was on at that moment — so these numbers cannot describe
// a different set of scans than the line's own. Timeliness plays no part: the question
// here is where the scans landed, not whether they were in time.
//
// Posts with no scans are absent from the result; the caller knows the full post list
// and reads a missing entry as zero.
func (app *application) checkpointStats(ctx context.Context, cgIDs []types.CheckgroupID) ([]CheckpointStats, error) {
	if len(cgIDs) == 0 {
		return []CheckpointStats{}, nil
	}
	query := `
		SELECT cpt.checkgroupId, cpt.id, COUNT(DISTINCT s.teamId)
		FROM scan s
		JOIN checkpersonnel cpn ON s.scannerId = cpn.userId AND s.uts >= cpn.startUts AND s.uts <= cpn.endUts
		JOIN checkpoint cpt ON cpn.checkpointId = cpt.id
		WHERE cpt.checkgroupId IN (?` + strings.Repeat(",?", len(cgIDs)-1) + `)
		GROUP BY cpt.checkgroupId, cpt.id`
	args := make([]any, len(cgIDs))
	for i, id := range cgIDs {
		args[i] = string(id)
	}
	rows, err := app.db.DB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := []CheckpointStats{}
	for rows.Next() {
		var s CheckpointStats
		if err := rows.Scan(&s.CheckgroupID, &s.CheckpointID, &s.TeamCount); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return stats, nil
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

	timing, err := app.checkgroupTimings(ctx, []types.CheckgroupID{cgID})
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}
	statuses := resolveTeamStatuses(teams, timing.Scans(cgID))

	// Where the ones who have not arrived are. Read only if this line has a located post to
	// measure to, since two year-wide aggregates for an answer nothing can use is waste on a
	// dialog that also revalidates on every scan while it is open.
	posts, err := app.linePosts(ctx, cgID)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}
	positions := map[string]positionFix{}
	if len(posts) > 0 {
		positions, err = app.lastKnownPositions(ctx, cg.YearSlug)
		if err != nil {
			app.ServerErrorResponse(w, r, err)
			return
		}
	}

	rows := make([]TeamAtCheckgroup, 0, len(started))
	for _, p := range started {
		s := statuses[p.TeamID]
		row := TeamAtCheckgroup{
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
			DeadlineUts:       timing.Deadline(cgID, p.TeamID),
		}
		if row.ScannedAtUts == 0 {
			fix, known := positions[string(p.TeamID)]
			row.Distance = distanceToLine(fix, known, posts)
		}
		rows = append(rows, row)
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

	// When the line shuts. Only a fixed line has one: on a relative line each team has
	// its own deadline (see TeamAtCheckgroup.DeadlineUts), and averaging or maximising
	// them would state a time that is nobody's. Zero, and the client omits it.
	closesAtUts := timing.ClosesAt(cgID)

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
