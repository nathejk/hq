package main

import (
	"context"
	"net/http"
	"sort"
	"time"

	"github.com/nathejk/shared-go/types"
	jsonapi "nathejk.dk/cmd/api/app"
	"nathejk.dk/nathejk/table/checkgroup"
	"nathejk.dk/nathejk/table/patrulje"
)

// The bingo curve: how many patrols still hold a full card, minute by minute.
//
// A bingo-patrulje is one that started, has been through every obligatorisk postlinje
// inside its opening hours, and has never been caught by a bandit. That is a verdict
// about a whole night, but the interesting number is not the final one — it is how the
// field thins out: seventy-six teams start with a clean card, a deadline passes and the
// ones who did not make a post lose it, a bandit catches two more, and by Sunday morning
// whoever is left is the answer.
//
// So the series is *how many were still in the running at time t*, not how many had
// finished by t. A team is on the curve from the moment it started until the first
// moment it forfeited, and a team that never forfeits stays on it to the end. The
// alternative reading — count only teams that have completed everything — is flat zero
// for the whole night and then jumps, which says nothing an operator can act on.
//
// Forfeiting is a one-way door on purpose. A missed deadline cannot be un-missed and a
// catch cannot be un-caught, so the count is monotonically non-increasing between
// starts. The curve can still rise, but only because another team started.

// BingoPoint is one step of the curve. The series is a step function — the count holds
// until the next point — so the client must not interpolate between two points.
type BingoPoint struct {
	Uts   int64 `json:"uts"`
	Count int   `json:"count"`
}

// BingoSeries is the whole graph, axis included.
type BingoSeries struct {
	// FromUts/UntilUts are the x-axis: the first post's opening hour and the last post's
	// closing hour. Taken from the checkpoints rather than hardcoded to "friday 21:00 to
	// sunday 04:00", because those two numbers *are* the route's opening hours and
	// hardcoding them would make the graph wrong the first year the schedule moves.
	FromUts  int64 `json:"fromUts"`
	UntilUts int64 `json:"untilUts"`
	// NowUts is where the series stops, when the race is still running. Carried so the
	// client can draw the axis to the end of the night while drawing the line only as far
	// as it is actually known.
	NowUts int64 `json:"nowUts"`

	Points []BingoPoint `json:"points"`

	// StartedCount and CheckgroupCount are what the curve is measured against: how many
	// patrols could be on it, and how many obligatoriske postlinjer they must clear.
	//
	// CheckgroupCount is served because zero is a meaningful and misleading state: with no
	// line flagged obligatorisk there is nothing to fail, so every started team reads as
	// bingo and the curve is a flat line at the number of starters. That looks like a
	// working graph, so the client is given the means to say "no obligatoriske postlinjer"
	// instead of drawing it.
	StartedCount    int `json:"startedCount"`
	CheckgroupCount int `json:"checkgroupCount"`
}

// bingoTeam is one patrol's span on the curve.
type bingoTeam struct {
	TeamID     types.TeamID
	StartedUts int64
	// LostUts is when the card was forfeited, or 0 for a team that still holds it.
	LostUts int64
}

// bingoLoss is the first moment this team forfeited its card, or 0 if it has not.
//
// The earliest of the ways to lose, because losing is permanent: a team caught at 23:00
// that also misses a deadline at 01:00 left the curve at 23:00, and reporting the later
// moment would keep it on the graph for two hours it had no claim to.
//
// A missed line is forfeited *at its deadline*, not at the moment of the late scan and
// not at the moment of the query. That is what makes the curve a history rather than a
// projection of the present: a deadline that has not passed yet cannot have been missed,
// so a team still on its way to a post it has not reached is still on the curve — which
// is exactly the team an operator is watching.
func bingoLoss(teamID types.TeamID, cgIDs []types.CheckgroupID, timing *checkgroupTiming, caughtAtUts int64) int64 {
	loss := caughtAtUts
	consider := func(uts int64) {
		if uts <= 0 {
			return
		}
		if loss == 0 || uts < loss {
			loss = uts
		}
	}
	for _, cgID := range cgIDs {
		scan, seen := timing.Scans(cgID)[teamID]
		if seen && scan.OnTime {
			continue
		}
		// The line's own deadline for this team. Zero means there is none to have
		// missed — an unset window, or a relative line the team has no reference scan
		// for — and that is the same lenience scanWasOnTime applies: with no deadline
		// computed, nothing is late.
		if deadline := timing.Deadline(cgID, teamID); deadline > 0 {
			consider(deadline)
			continue
		}
		// A scan judged late with no deadline to name: a relative line with a zero
		// allowance, where arriving at all after the reference scan is too late. Rare,
		// but the team did forfeit, so the scan itself is the moment.
		if seen {
			consider(scan.FirstUts)
		}
	}
	return loss
}

// bingoSeries turns the per-team spans into the step function.
//
// Built from start/loss events rather than by sampling the night at a fixed interval:
// the curve only changes when a team starts or forfeits, so the events *are* the curve
// and there are a few hundred of them. Sampling would either miss changes between
// samples or emit thousands of identical points.
//
// endUts clamps the series to what has actually happened. Events before fromUts are
// folded into the opening point rather than dropped — a team that started before the
// first post opened is still racing — and events after endUts are left out, because a
// deadline in the future has not been missed yet.
func bingoSeries(teams []bingoTeam, fromUts, untilUts, nowUts int64) []BingoPoint {
	endUts := untilUts
	if nowUts < endUts {
		endUts = nowUts
	}
	if endUts < fromUts {
		endUts = fromUts
	}

	type delta struct {
		uts   int64
		count int
	}
	deltas := make([]delta, 0, len(teams)*2)
	for _, t := range teams {
		if t.StartedUts <= 0 {
			// A started patrol with no recorded start time cannot be placed on a time
			// axis. Left off rather than guessed at: the honest graph is one team short,
			// the guessed one is wrong in a way nobody can see.
			continue
		}
		deltas = append(deltas, delta{uts: t.StartedUts, count: 1})
		if t.LostUts > 0 {
			lost := t.LostUts
			// A loss recorded before the start — a deadline that passed while the team
			// was still waiting to be sent off — takes effect at the start, so the team
			// never appears on the curve rather than appearing with a negative span.
			if lost < t.StartedUts {
				lost = t.StartedUts
			}
			deltas = append(deltas, delta{uts: lost, count: -1})
		}
	}
	sort.Slice(deltas, func(i, j int) bool { return deltas[i].uts < deltas[j].uts })

	count := 0
	points := []BingoPoint{}
	i := 0
	// Everything up to and including the opening moment collapses into one point, so the
	// line starts at the left edge of the axis with the right value.
	for ; i < len(deltas) && deltas[i].uts <= fromUts; i++ {
		count += deltas[i].count
	}
	points = append(points, BingoPoint{Uts: fromUts, Count: count})

	for i < len(deltas) {
		uts := deltas[i].uts
		if uts > endUts {
			break
		}
		// All deltas at the same instant are one step: a team forfeiting at the same
		// second another starts must not draw a spike.
		for ; i < len(deltas) && deltas[i].uts == uts; i++ {
			count += deltas[i].count
		}
		points = append(points, BingoPoint{Uts: uts, Count: count})
	}

	// Carry the current count to the end of what is known, so the line does not stop
	// short at whenever the last thing happened to occur.
	if last := points[len(points)-1]; last.Uts < endUts {
		points = append(points, BingoPoint{Uts: endUts, Count: count})
	}
	return points
}

// mandatoryCheckgroups is the lines a bingo card has to be complete over.
//
// The obligatorisk flag, not every line: an optional postlinje is optional, and counting
// it would mean no team ever gets bingo. In the operator's own order — sortOrder, which is
// what GetAll returns and the order the postlinjer are listed everywhere else — so the
// columns of the bingo table read along the route rather than by opaque id.
func (app *application) mandatoryCheckgroups(ctx context.Context, year types.YearSlug) ([]checkgroup.Checkgroup, error) {
	groups, err := app.models.Checkgroup.GetAll(ctx, checkgroup.Filter{Year: year})
	if err != nil {
		return nil, err
	}
	lines := []checkgroup.Checkgroup{}
	for _, cg := range groups {
		if cg.Mandatory {
			lines = append(lines, cg)
		}
	}
	return lines, nil
}

// checkgroupIDs is the ids of the given lines, in the same order.
func checkgroupIDs(lines []checkgroup.Checkgroup) []types.CheckgroupID {
	ids := make([]types.CheckgroupID, 0, len(lines))
	for _, cg := range lines {
		ids = append(ids, cg.ID)
	}
	return ids
}

// raceWindow is the first post's opening hour and the last post's closing hour.
//
// Both zero when the year has no post with an opening hour recorded, which the caller
// treats as "no axis to draw" rather than as an error: a year being planned is in that
// state for months.
func (app *application) raceWindow(ctx context.Context, year types.YearSlug) (fromUts, untilUts int64, err error) {
	// Nullable because MIN/MAX over no rows is NULL, and because a checkpoint with no
	// hours set holds 0 — excluded rather than allowed to drag the axis back to 1970.
	var from, until *int64
	row := app.db.DB().QueryRowContext(ctx,
		`SELECT MIN(NULLIF(openFromUts, 0)), MAX(NULLIF(openUntilUts, 0)) FROM checkpoint WHERE LOWER(year) = LOWER(?)`,
		string(year))
	if err := row.Scan(&from, &until); err != nil {
		return 0, 0, err
	}
	if from != nil {
		fromUts = *from
	}
	if until != nil {
		untilUts = *until
	}
	return fromUts, untilUts, nil
}

// teamCatches is how often a patrol has been caught, and when it first happened.
//
// Both, because they answer different questions: the count is how the night went, the
// first is when the card was lost. A team caught four times lost it once.
type teamCatches struct {
	Count    int
	FirstUts int64
}

// banditCatches is every patrol's catches for the year.
//
// A bandit is a klan member out on the route, so a catch is a scan whose scanner is a
// senior — the same rule the patrol's trail uses to read a scan as "Fanget af" rather
// than "Scannet af" (see resolveScanner). Crew and gøgler scans at posts are not
// catches and must not cost a team its card.
//
// # Why the year comes from the patrol and not from the scan
//
// `scan.year` is NULL on every row: the projection's INSERT never sets it (see
// table/scan/consumer.go), and the column is nullable with no default. So a
// `WHERE scan.year = ?` matches nothing and returns silently empty — which is exactly what
// the first version of this did, reporting zero catches for everybody against a database
// holding thousands. Scoping through `patrulje`, which *is* year-stamped, is both correct
// and the only thing available. checkgroupTimings gets away with no year filter at all for
// the same underlying reason: it scopes through the checkgroup instead.
//
// The seniors are reduced to DISTINCT memberIds before the join, which is not tidiness:
// `senior` is keyed (year, memberId), so a senior who attends two years has two rows, and
// joining the scans against those directly would count every catch twice.
func (app *application) banditCatches(ctx context.Context, year types.YearSlug) (map[types.TeamID]teamCatches, error) {
	rows, err := app.db.DB().QueryContext(ctx,
		`SELECT s.teamId, COUNT(*), MIN(s.uts)
			FROM scan s
			JOIN patrulje p ON p.teamId = s.teamId AND LOWER(p.year) = LOWER(?)
			JOIN (SELECT DISTINCT memberId FROM senior WHERE LOWER(year) = LOWER(?)) sen
				ON sen.memberId = s.scannerId
			WHERE s.uts > 0
			GROUP BY s.teamId`,
		string(year), string(year))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	caught := map[types.TeamID]teamCatches{}
	for rows.Next() {
		var teamID types.TeamID
		var c teamCatches
		if err := rows.Scan(&teamID, &c.Count, &c.FirstUts); err != nil {
			return nil, err
		}
		caught[teamID] = c
	}
	return caught, rows.Err()
}

// bingoHandler serves the curve for the current year.
//
// Its own endpoint rather than a field on /api/home: the four figures there move when
// somebody signs up, while this moves on every scan during the race, and folding them
// together would make the whole dashboard recompute the night's history to learn that a
// gøgler paid.
func (app *application) bingoHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	year := app.YearSlug(r)

	fromUts, untilUts, err := app.raceWindow(ctx, year)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}
	lines, err := app.mandatoryCheckgroups(ctx, year)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}
	cgIDs := checkgroupIDs(lines)
	started, err := app.models.Patrulje.GetStartedTeams(ctx, patrulje.Filter{YearSlug: year})
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}

	series := BingoSeries{
		FromUts:         fromUts,
		UntilUts:        untilUts,
		NowUts:          time.Now().Unix(),
		Points:          []BingoPoint{},
		StartedCount:    len(started),
		CheckgroupCount: len(cgIDs),
	}
	// No axis is not an error — it is a year whose posts have no hours yet. An empty
	// series says so; inventing a window would draw a graph of nothing.
	if fromUts == 0 || untilUts <= fromUts {
		if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"bingo": series}, nil); err != nil {
			app.ServerErrorResponse(w, r, err)
		}
		return
	}

	timing, err := app.checkgroupTimings(ctx, cgIDs)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}
	caught, err := app.banditCatches(ctx, year)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}

	teams := make([]bingoTeam, 0, len(started))
	for _, t := range started {
		teams = append(teams, bingoTeam{
			TeamID:     t.TeamID,
			StartedUts: t.StartedUts,
			LostUts:    bingoLoss(t.TeamID, cgIDs, timing, caught[t.TeamID].FirstUts),
		})
	}
	series.Points = bingoSeries(teams, fromUts, untilUts, series.NowUts)

	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"bingo": series}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// BingoLine names one column of the bingo table.
type BingoLine struct {
	CheckgroupID types.CheckgroupID `json:"checkgroupId"`
	Name         string             `json:"name"`
}

// BingoCell is one team's standing at one obligatorisk postlinje: the tick or the cross,
// and the two times that explain it.
type BingoCell struct {
	// Status is the same four-token vocabulary as the post list (TeamAtCheckgroup*), not a
	// bare boolean. The table draws a tick for onTime and a cross for the rest, but *why*
	// it is a cross — late, udgået, or never seen — is what the hover says, and inventing
	// a second vocabulary for the same fact is how two screens come to disagree.
	Status string `json:"status"`
	// ScannedAtUts is when the team came through, 0 if never. Omitted rather than sent as
	// the epoch, so the client cannot render 1970 as a time.
	ScannedAtUts int64 `json:"scannedAtUts,omitempty"`
	// DeadlineUts is when it was due. Per team, because a relative line's clock starts
	// when that team left the line it measures from.
	DeadlineUts int64 `json:"deadlineUts,omitempty"`
}

// BingoTeamRow is one patrol's line in the table.
type BingoTeamRow struct {
	TeamID     types.TeamID `json:"teamId"`
	TeamNumber string       `json:"teamNumber"`
	Name       string       `json:"name"`
	Group      string       `json:"group"`
	// ActiveMemberCount is zero for a patrol that is udgået, which is why a row can be all
	// crosses without anybody having done anything wrong.
	ActiveMemberCount int `json:"activeMemberCount"`

	// Cells is parallel to the payload's `lines`, by index. Always as long as `lines`, so
	// the table needs no lookup and no guard for a missing cell.
	Cells []BingoCell `json:"cells"`

	CatchCount    int   `json:"catchCount"`
	FirstCatchUts int64 `json:"firstCatchUts,omitempty"`

	// Bingo is the verdict the green row is drawn from.
	//
	// Computed here and not in the client so the highlight cannot drift from the number on
	// the dashboard. Note it is deliberately *not* bingoLoss() == 0: that answers "is this
	// team still in the running at this moment", which keeps a team whose next post has not
	// closed yet, while this answers "does this row show a full card" — every cell a tick
	// and no catches, which is exactly what the operator is reading. A team mid-race is in
	// the running without being bingo, and the table must not claim otherwise.
	Bingo bool `json:"bingo"`
}

// bingoRow assembles one patrol's row.
//
// Bingo is all ticks and no catches. The `status` per cell comes from resolveTeamStatus, so
// a cross here and a team in the post list's late/missing/retired columns are the same
// judgement made once.
func bingoRow(p patrulje.Patrulje, lines []types.CheckgroupID, timing *checkgroupTiming, catches teamCatches) BingoTeamRow {
	row := BingoTeamRow{
		TeamID:            p.TeamID,
		TeamNumber:        p.TeamNumber,
		Name:              p.Name,
		Group:             p.Group,
		ActiveMemberCount: p.ActiveMemberCount,
		Cells:             make([]BingoCell, 0, len(lines)),
		CatchCount:        catches.Count,
		FirstCatchUts:     catches.FirstUts,
	}

	team := startedTeam{TeamID: p.TeamID, ActiveMemberCount: p.ActiveMemberCount}
	allOnTime := true
	for _, cgID := range lines {
		var scan *checkgroupScan
		if s, ok := timing.Scans(cgID)[p.TeamID]; ok {
			scan = &s
		}
		status, uts := resolveTeamStatus(team, scan)
		if status != TeamAtCheckgroupOnTime {
			allOnTime = false
		}
		row.Cells = append(row.Cells, BingoCell{
			Status:       status,
			ScannedAtUts: uts,
			DeadlineUts:  timing.Deadline(cgID, p.TeamID),
		})
	}
	// No obligatoriske postlinjer means no card, so nobody has a full one. Without this,
	// `allOnTime` over an empty list is vacuously true and every row would go green.
	row.Bingo = len(lines) > 0 && allOnTime && catches.Count == 0
	return row
}

// sortBingoRows orders the table: bingo first, then by team number.
//
// Bingo first because the list exists to answer "who has it", and a dozen green rows at the
// top is that answer without reading further. Within a group, by the number the patrols are
// called by — numerically, since string order puts 10 before 2.
func sortBingoRows(rows []BingoTeamRow) {
	sort.SliceStable(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		if a.Bingo != b.Bingo {
			return a.Bingo
		}
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
}

// bingoTeamsHandler serves the table behind the curve: every patrol, every obligatorisk
// postlinje, tick or cross.
//
// A separate endpoint from the curve for the reason the post list's dialog is separate from
// its numbers: this is seventeen rows by eight columns plus two aggregates, and it is wanted
// when somebody opens it — a few times a night — while the graph is recomputed on every
// scan.
//
// Started patrols only, like every other screen about the route. A patrol that never left
// has no cell that means anything, and listing seventy-six of them to find the seventeen
// that are racing is how a table stops being readable.
func (app *application) bingoTeamsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	year := app.YearSlug(r)

	lines, err := app.mandatoryCheckgroups(ctx, year)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}
	cgIDs := checkgroupIDs(lines)

	started, err := app.startedPatruljer(ctx, year)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}
	timing, err := app.checkgroupTimings(ctx, cgIDs)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}
	caught, err := app.banditCatches(ctx, year)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}

	columns := make([]BingoLine, 0, len(lines))
	for _, cg := range lines {
		columns = append(columns, BingoLine{CheckgroupID: cg.ID, Name: cg.Name})
	}

	rows := make([]BingoTeamRow, 0, len(started))
	bingoCount := 0
	for _, p := range started {
		row := bingoRow(p, cgIDs, timing, caught[p.TeamID])
		if row.Bingo {
			bingoCount++
		}
		rows = append(rows, row)
	}
	sortBingoRows(rows)

	envelope := jsonapi.Envelope{
		"lines":      columns,
		"teams":      rows,
		"bingoCount": bingoCount,
	}
	if err := app.WriteJSON(w, http.StatusOK, envelope, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}
