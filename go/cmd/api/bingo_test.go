package main

import (
	"testing"

	"github.com/nathejk/shared-go/types"
	"nathejk.dk/nathejk/table/patrulje"
)

// fixedTiming builds a timing over fixed lines that close at the given moments, with the
// given per-team verdicts. Enough for bingoLoss, which only ever asks two questions of
// it: was this team on time here, and when was it due.
func fixedTiming(closesAt map[types.CheckgroupID]int64, scans map[types.CheckgroupID]map[types.TeamID]checkgroupScan) *checkgroupTiming {
	timing := &checkgroupTiming{
		windows:  map[types.CheckgroupID]checkgroupWindow{},
		limits:   map[types.CheckgroupID]checkgroupLimit{},
		lastScan: map[types.CheckgroupID]map[types.TeamID]int64{},
		scans:    scans,
	}
	for id, uts := range closesAt {
		timing.windows[id] = checkgroupWindow{Scheme: types.CheckgroupSchemeFixed}
		timing.limits[id] = checkgroupLimit{ClosesAtUts: uts}
	}
	return timing
}

func onTimeAt(uts int64) checkgroupScan {
	return checkgroupScan{OnTime: true, FirstUts: uts}
}

// The ordinary bingo: on time everywhere, never caught.
func TestBingoLossCleanCardHasNoLoss(t *testing.T) {
	timing := fixedTiming(
		map[types.CheckgroupID]int64{"a": 2000, "b": 4000},
		map[types.CheckgroupID]map[types.TeamID]checkgroupScan{
			"a": {"t1": onTimeAt(1500)},
			"b": {"t1": onTimeAt(3500)},
		})
	if loss := bingoLoss("t1", []types.CheckgroupID{"a", "b"}, timing, 0); loss != 0 {
		t.Errorf("loss = %d, want 0 for a team on time everywhere and never caught", loss)
	}
}

// The card is lost when the deadline passes, not when the late scan happens: a team that
// turned up an hour late left the curve at closing time, and drawing it as still in the
// running until it finally arrived would overstate the field.
func TestBingoLossLateScanForfeitsAtTheDeadline(t *testing.T) {
	timing := fixedTiming(
		map[types.CheckgroupID]int64{"a": 2000},
		map[types.CheckgroupID]map[types.TeamID]checkgroupScan{
			"a": {"t1": {OnTime: false, FirstUts: 5600}},
		})
	if loss := bingoLoss("t1", []types.CheckgroupID{"a"}, timing, 0); loss != 2000 {
		t.Errorf("loss = %d, want the deadline (2000), not the late scan", loss)
	}
}

// The point of separating the deadline from the query: a post that has not closed yet
// cannot have been missed, so a team still walking towards it keeps its card. This is the
// team an operator is watching, and it must be on the graph.
func TestBingoLossPendingDeadlineIsNotYetALoss(t *testing.T) {
	timing := fixedTiming(map[types.CheckgroupID]int64{"a": 9000}, nil)
	loss := bingoLoss("t1", []types.CheckgroupID{"a"}, timing, 0)
	if loss != 9000 {
		t.Errorf("loss = %d, want the line's deadline (9000)", loss)
	}
	// And the series is what clamps it away: with now at 5000 the team is still counted.
	points := bingoSeries([]bingoTeam{{TeamID: "t1", StartedUts: 1000, LostUts: loss}}, 1000, 10000, 5000)
	if last := points[len(points)-1]; last.Count != 1 {
		t.Errorf("count at now = %d, want 1: the deadline has not passed yet", last.Count)
	}
}

// Losing is a one-way door, so the first way out is the one that counts. Reporting the
// later of a catch and a missed deadline would leave the team on the curve for hours it
// had no claim to.
func TestBingoLossTakesTheEarliestWayOut(t *testing.T) {
	timing := fixedTiming(map[types.CheckgroupID]int64{"a": 8000}, nil)
	if loss := bingoLoss("t1", []types.CheckgroupID{"a"}, timing, 3000); loss != 3000 {
		t.Errorf("loss = %d, want the catch (3000), which came first", loss)
	}
}

// A catch by a bandit ends the card on its own, however well the team is doing on time.
func TestBingoLossCatchAloneForfeitsTheCard(t *testing.T) {
	timing := fixedTiming(
		map[types.CheckgroupID]int64{"a": 2000},
		map[types.CheckgroupID]map[types.TeamID]checkgroupScan{"a": {"t1": onTimeAt(1500)}})
	if loss := bingoLoss("t1", []types.CheckgroupID{"a"}, timing, 7000); loss != 7000 {
		t.Errorf("loss = %d, want the catch (7000)", loss)
	}
}

// A line with no opening hours has no deadline to miss, and that is the same lenience
// scanWasOnTime applies. Judging it as an instant loss would empty the graph on a route
// whose optional-timing lines are simply not yet filled in.
func TestBingoLossLineWithNoDeadlineIsNotALoss(t *testing.T) {
	timing := fixedTiming(map[types.CheckgroupID]int64{"a": 0}, nil)
	if loss := bingoLoss("t1", []types.CheckgroupID{"a"}, timing, 0); loss != 0 {
		t.Errorf("loss = %d, want 0: an unset window is nothing to be late for", loss)
	}
}

// With no line flagged obligatorisk there is nothing to fail. Deliberately *not* treated
// as "nobody has bingo" — the handler reports the count of lines so the client can say so
// rather than draw a flat line and call it a graph.
func TestBingoLossNoMandatoryLinesIsNoLoss(t *testing.T) {
	if loss := bingoLoss("t1", nil, fixedTiming(nil, nil), 0); loss != 0 {
		t.Errorf("loss = %d, want 0", loss)
	}
}

func TestBingoSeriesStartsAtTheOpeningHourWithTeamsAlreadyRacing(t *testing.T) {
	// Both started before the first post opened, which is the normal case: teams are sent
	// off ahead of the opening hour.
	teams := []bingoTeam{
		{TeamID: "t1", StartedUts: 500},
		{TeamID: "t2", StartedUts: 900},
	}
	points := bingoSeries(teams, 1000, 5000, 5000)
	if points[0].Uts != 1000 || points[0].Count != 2 {
		t.Errorf("first point = %+v, want {1000 2}: both teams are already on the route", points[0])
	}
}

func TestBingoSeriesFallsAsTeamsForfeit(t *testing.T) {
	teams := []bingoTeam{
		{TeamID: "t1", StartedUts: 1000},
		{TeamID: "t2", StartedUts: 1000, LostUts: 2000},
		{TeamID: "t3", StartedUts: 1000, LostUts: 3000},
	}
	points := bingoSeries(teams, 1000, 4000, 4000)
	want := []BingoPoint{{1000, 3}, {2000, 2}, {3000, 1}, {4000, 1}}
	if len(points) != len(want) {
		t.Fatalf("points = %+v, want %+v", points, want)
	}
	for i := range want {
		if points[i] != want[i] {
			t.Errorf("point %d = %+v, want %+v", i, points[i], want[i])
		}
	}
}

// Two teams changing state in the same second are one step. Emitted separately, a
// forfeit and a start at the same instant draw a spike that never happened.
func TestBingoSeriesCollapsesSimultaneousChanges(t *testing.T) {
	teams := []bingoTeam{
		{TeamID: "t1", StartedUts: 1000, LostUts: 2000},
		{TeamID: "t2", StartedUts: 2000},
	}
	points := bingoSeries(teams, 1000, 3000, 3000)
	for _, p := range points {
		if p.Count != 1 {
			t.Fatalf("points = %+v, want a flat 1: one team replaces the other at 2000", points)
		}
	}
}

// The line stops at now, not at the end of the night: a graph drawn to Sunday morning
// while it is Saturday evening asserts hours that have not happened.
func TestBingoSeriesStopsAtNowWhileTheRaceRuns(t *testing.T) {
	teams := []bingoTeam{{TeamID: "t1", StartedUts: 1000, LostUts: 9000}}
	points := bingoSeries(teams, 1000, 10000, 4000)
	last := points[len(points)-1]
	if last.Uts != 4000 {
		t.Errorf("series ends at %d, want now (4000)", last.Uts)
	}
	if last.Count != 1 {
		t.Errorf("count at now = %d, want 1: the forfeit is still in the future", last.Count)
	}
}

// A patrol projected before startedUts existed cannot be placed on a time axis. Left off
// rather than plotted at the epoch — one team short is visible, a wrong graph is not.
func TestBingoSeriesSkipsTeamsWithNoStartTime(t *testing.T) {
	points := bingoSeries([]bingoTeam{{TeamID: "t1"}}, 1000, 3000, 3000)
	for _, p := range points {
		if p.Count != 0 {
			t.Fatalf("points = %+v, want a flat 0", points)
		}
	}
}

// A deadline that passed before the team was even sent off must not draw a negative span.
func TestBingoSeriesClampsALossBeforeTheStart(t *testing.T) {
	points := bingoSeries([]bingoTeam{{TeamID: "t1", StartedUts: 2000, LostUts: 1500}}, 1000, 3000, 3000)
	for _, p := range points {
		if p.Count != 0 {
			t.Fatalf("points = %+v, want a flat 0: the card was gone before the start", points)
		}
	}
}

// Never empty: the client plots whatever comes back, and a zero-length series with a
// non-zero axis would have to be special-cased at every call site.
func TestBingoSeriesAlwaysHasAtLeastOnePoint(t *testing.T) {
	if points := bingoSeries(nil, 1000, 5000, 5000); len(points) < 1 {
		t.Error("series is empty; want at least the opening point")
	}
}

func patrol(number string, active int) patrulje.Patrulje {
	return patrulje.Patrulje{TeamID: types.TeamID("t" + number), TeamNumber: number, ActiveMemberCount: active}
}

// A moment after every deadline in these fixtures, so a cell with no scan reads as a real
// failure rather than as "not yet". The pending case has its own tests below.
const afterTheRace int64 = 1_000_000

// The green row: every cell a tick, nobody caught.
func TestBingoRowAllOnTimeAndUncaughtIsBingo(t *testing.T) {
	p := patrol("7", 4)
	timing := fixedTiming(
		map[types.CheckgroupID]int64{"a": 2000, "b": 4000},
		map[types.CheckgroupID]map[types.TeamID]checkgroupScan{
			"a": {p.TeamID: onTimeAt(1500)},
			"b": {p.TeamID: onTimeAt(3500)},
		})
	row := bingoRow(p, []types.CheckgroupID{"a", "b"}, timing, teamCatches{}, afterTheRace)
	if !row.Bingo {
		t.Error("row is not bingo; want bingo for a full card and no catches")
	}
	if len(row.Cells) != 2 {
		t.Fatalf("%d cells, want one per line", len(row.Cells))
	}
	for i, c := range row.Cells {
		if c.Status != TeamAtCheckgroupOnTime {
			t.Errorf("cell %d status = %q, want %q", i, c.Status, TeamAtCheckgroupOnTime)
		}
	}
	// The hover needs the time, so the cell has to carry it.
	if row.Cells[0].ScannedAtUts != 1500 {
		t.Errorf("scanned at %d, want 1500", row.Cells[0].ScannedAtUts)
	}
	if row.Cells[0].DeadlineUts != 2000 {
		t.Errorf("deadline %d, want the line's closing time (2000)", row.Cells[0].DeadlineUts)
	}
}

// A single catch costs the card however clean the row's ticks are.
func TestBingoRowACatchDeniesBingo(t *testing.T) {
	p := patrol("7", 4)
	timing := fixedTiming(
		map[types.CheckgroupID]int64{"a": 2000},
		map[types.CheckgroupID]map[types.TeamID]checkgroupScan{"a": {p.TeamID: onTimeAt(1500)}})
	row := bingoRow(p, []types.CheckgroupID{"a"}, timing, teamCatches{Count: 1, FirstUts: 6000}, afterTheRace)
	if row.Bingo {
		t.Error("row is bingo; want not bingo for a team that was caught")
	}
	if row.CatchCount != 1 || row.FirstCatchUts != 6000 {
		t.Errorf("catches = %d at %d, want 1 at 6000", row.CatchCount, row.FirstCatchUts)
	}
}

// One cross is enough. The cell keeps the late scan's time so the hover can say when.
func TestBingoRowOneMissedLineDeniesBingo(t *testing.T) {
	p := patrol("7", 4)
	timing := fixedTiming(
		map[types.CheckgroupID]int64{"a": 2000, "b": 4000},
		map[types.CheckgroupID]map[types.TeamID]checkgroupScan{
			"a": {p.TeamID: onTimeAt(1500)},
			"b": {p.TeamID: {OnTime: false, FirstUts: 4400}},
		})
	row := bingoRow(p, []types.CheckgroupID{"a", "b"}, timing, teamCatches{}, afterTheRace)
	if row.Bingo {
		t.Error("row is bingo; want not bingo with a late line")
	}
	if row.Cells[1].Status != TeamAtCheckgroupLate {
		t.Errorf("cell status = %q, want %q", row.Cells[1].Status, TeamAtCheckgroupLate)
	}
	if row.Cells[1].ScannedAtUts != 4400 {
		t.Errorf("scanned at %d, want 4400 so the hover can show it", row.Cells[1].ScannedAtUts)
	}
}

// A row of crosses because nobody is left racing reads as udgået, not as missing: the
// distinction is already made once, in resolveTeamStatus, and the table shows it.
func TestBingoRowWithdrawnTeamReadsAsRetired(t *testing.T) {
	p := patrol("7", 0)
	row := bingoRow(p, []types.CheckgroupID{"a"}, fixedTiming(map[types.CheckgroupID]int64{"a": 2000}, nil), teamCatches{}, afterTheRace)
	if row.Cells[0].Status != TeamAtCheckgroupRetired {
		t.Errorf("cell status = %q, want %q", row.Cells[0].Status, TeamAtCheckgroupRetired)
	}
	if row.Bingo {
		t.Error("row is bingo; want not bingo for a team that went home")
	}
}

// The trap in "every cell is a tick": over no cells that is vacuously true, and the whole
// table would go green on a route with nothing flagged obligatorisk.
func TestBingoRowNoMandatoryLinesIsNotBingo(t *testing.T) {
	row := bingoRow(patrol("7", 4), nil, fixedTiming(nil, nil), teamCatches{}, afterTheRace)
	if row.Bingo {
		t.Error("row is bingo over zero lines; want not bingo — there is no card to complete")
	}
}

// The case this exists for: at 23.00, a patrol that has not reached a post closing at 03.00 has
// done nothing wrong. A red cross there is a lie the operator has to undo on every row, and in
// the early hours that is most of the table.
func TestBingoRowUnclosedLineIsPendingNotMissing(t *testing.T) {
	p := patrol("7", 4)
	timing := fixedTiming(map[types.CheckgroupID]int64{"a": 9000}, nil)
	row := bingoRow(p, []types.CheckgroupID{"a"}, timing, teamCatches{}, 5000)
	if row.Cells[0].Status != TeamAtCheckgroupPending {
		t.Errorf("cell status = %q, want %q for a post that has not closed", row.Cells[0].Status, TeamAtCheckgroupPending)
	}
	// The deadline still travels, because "not yet — due at 03.00" is the useful hover.
	if row.Cells[0].DeadlineUts != 9000 {
		t.Errorf("deadline %d, want 9000", row.Cells[0].DeadlineUts)
	}
	// Pending is not a tick: the card is not full until the last post is behind them.
	if row.Bingo {
		t.Error("row is bingo; want not bingo while a post is still ahead of the team")
	}
}

// Once the post has shut, the same absence is a real miss.
func TestBingoRowClosedLineWithNoScanIsMissing(t *testing.T) {
	timing := fixedTiming(map[types.CheckgroupID]int64{"a": 4000}, nil)
	row := bingoRow(patrol("7", 4), []types.CheckgroupID{"a"}, timing, teamCatches{}, 5000)
	if row.Cells[0].Status != TeamAtCheckgroupMissing {
		t.Errorf("cell status = %q, want %q once the post has closed", row.Cells[0].Status, TeamAtCheckgroupMissing)
	}
}

// A line with no deadline reads as pending rather than missed: 0 means there is nothing to have
// missed, and printing a cross for a rule that was never stated is the worse error.
func TestBingoCellStatusNoDeadlineIsPending(t *testing.T) {
	if got := bingoCellStatus(TeamAtCheckgroupMissing, 0, 5000); got != TeamAtCheckgroupPending {
		t.Errorf("status = %q, want %q", got, TeamAtCheckgroupPending)
	}
}

// The clock only ever reconsiders `missing`. A scan is a fact about the past whatever the time
// is, and a patrol that went home will not arrive at a post that is still open — so calling
// that cell "not yet" would promise an arrival that cannot happen.
func TestBingoCellStatusLeavesEveryOtherVerdictAlone(t *testing.T) {
	for _, status := range []string{TeamAtCheckgroupOnTime, TeamAtCheckgroupLate, TeamAtCheckgroupRetired} {
		if got := bingoCellStatus(status, 9000, 5000); got != status {
			t.Errorf("status %q became %q; want it unchanged", status, got)
		}
	}
}

// The boundary: at the deadline the post has shut, so the cell is a miss rather than pending.
func TestBingoCellStatusAtTheDeadlineIsMissing(t *testing.T) {
	if got := bingoCellStatus(TeamAtCheckgroupMissing, 5000, 5000); got != TeamAtCheckgroupMissing {
		t.Errorf("status = %q, want %q at the deadline itself", got, TeamAtCheckgroupMissing)
	}
}

// Bingo first, because the list exists to answer "who has it"; then by the number the
// patrols are called by, numerically — string order would put 10 before 2.
func TestSortBingoRowsPutsBingoFirstThenTeamNumber(t *testing.T) {
	rows := []BingoTeamRow{
		{TeamNumber: "10"},
		{TeamNumber: "3", Bingo: true},
		{TeamNumber: "2"},
		{TeamNumber: "12", Bingo: true},
	}
	sortBingoRows(rows)
	got := []string{rows[0].TeamNumber, rows[1].TeamNumber, rows[2].TeamNumber, rows[3].TeamNumber}
	want := []string{"3", "12", "2", "10"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order = %v, want %v", got, want)
		}
	}
}

// An unnumbered patrol sorts last rather than first: a blank is not number zero, and a
// table that opens with a row that has no number reads as broken.
func TestSortBingoRowsPutsUnnumberedLast(t *testing.T) {
	rows := []BingoTeamRow{{TeamNumber: ""}, {TeamNumber: "9"}}
	sortBingoRows(rows)
	if rows[0].TeamNumber != "9" {
		t.Errorf("first row is %q, want the numbered one", rows[0].TeamNumber)
	}
}
