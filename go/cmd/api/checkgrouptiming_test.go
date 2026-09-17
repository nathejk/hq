package main

import (
	"testing"

	"github.com/nathejk/shared-go/types"
)

// A relative line ("+N minutter efter Postlinje X") gives each team its own deadline.
// These tests pin the arithmetic that decides on time vs over time, which the SQL that
// preceded them could not express: it compared against the checkpoint's wall-clock
// columns, which are 0 on a relative line, so every scan there read as over time.

const minute = int64(60)

func relativeScan(cg types.CheckgroupID, team types.TeamID, uts int64, durationMinutes int64) rawScan {
	return rawScan{CheckgroupID: cg, TeamID: team, Uts: uts, OpenDurationMinutes: durationMinutes}
}

func TestScanWasOnTimeFixedWindow(t *testing.T) {
	window := checkgroupWindow{Scheme: types.CheckgroupSchemeFixed}
	s := rawScan{Uts: 1500, OpenFromUts: 1000, OpenUntilUts: 2000}
	if !scanWasOnTime(s, window, 0) {
		t.Error("a scan inside the wall-clock window is on time")
	}
	s.Uts = 2500
	if scanWasOnTime(s, window, 0) {
		t.Error("a scan after the window closed is over time")
	}
}

func TestScanWasOnTimeFixedWindowThatWasNeverSet(t *testing.T) {
	// Both columns 0 means nobody wrote opening hours. There is nothing to be late
	// for, and reading the epoch as a closing time would mark every team over time.
	s := rawScan{Uts: 1_700_000_000}
	if !scanWasOnTime(s, checkgroupWindow{Scheme: types.CheckgroupSchemeFixed}, 0) {
		t.Error("a line with no opening hours cannot make a team late")
	}
}

func TestScanWasOnTimeRelativeToAnotherLine(t *testing.T) {
	window := checkgroupWindow{Scheme: types.CheckgroupSchemeRelative, RelativeTo: "cg-1"}
	base := int64(1000)
	// 30 minutes allowed from the reference scan.
	s := relativeScan("cg-2", "t1", base+29*minute, 30)
	if !scanWasOnTime(s, window, base) {
		t.Error("inside the allowance, so on time")
	}
	s.Uts = base + 31*minute
	if scanWasOnTime(s, window, base) {
		t.Error("past the allowance, so over time")
	}
	s.Uts = base + 30*minute
	if !scanWasOnTime(s, window, base) {
		t.Error("exactly on the deadline counts as on time, as the fixed window's end does")
	}
}

func TestScanWasOnTimeRelativeWithoutAReferenceScan(t *testing.T) {
	window := checkgroupWindow{Scheme: types.CheckgroupSchemeRelative, RelativeTo: "cg-1"}
	if !scanWasOnTime(relativeScan("cg-2", "t1", 5000, 30), window, 0) {
		t.Error("no reference scan means no deadline: the team cannot have missed one")
	}
}

func TestScanWasOnTimeRelativeEarlierThanTheReference(t *testing.T) {
	window := checkgroupWindow{Scheme: types.CheckgroupSchemeRelative, RelativeTo: "cg-1"}
	if !scanWasOnTime(relativeScan("cg-2", "t1", 500, 30), window, 1000) {
		t.Error("arriving before the clock started is not being late")
	}
}

func TestScanWasOnTimeNoScheme(t *testing.T) {
	if !scanWasOnTime(rawScan{Uts: 5000}, checkgroupWindow{Scheme: types.CheckgroupSchemeNone}, 0) {
		t.Error("a line that keeps no time has no over time")
	}
}

func TestAggregateScansRelativeLineUsesTheTeamsOwnReference(t *testing.T) {
	windows := map[types.CheckgroupID]checkgroupWindow{
		"cg-1": {Scheme: types.CheckgroupSchemeFixed},
		"cg-2": {Scheme: types.CheckgroupSchemeRelative, RelativeTo: "cg-1"},
	}
	// Two teams through cg-1 an hour apart, both reaching cg-2 twenty minutes later.
	// Judged against a shared wall clock one of them would be late; against their own
	// reference scan both are on time, which is the point of the relative scheme.
	scans := []rawScan{
		{CheckgroupID: "cg-1", TeamID: "early", Uts: 1000, OpenFromUts: 0, OpenUntilUts: 0},
		{CheckgroupID: "cg-1", TeamID: "late", Uts: 1000 + 60*minute},
		relativeScan("cg-2", "early", 1000+20*minute, 30),
		relativeScan("cg-2", "late", 1000+80*minute, 30),
	}

	got := aggregateScans(scans, windows)

	for _, team := range []types.TeamID{"early", "late"} {
		s, ok := got["cg-2"][team]
		if !ok {
			t.Fatalf("team %q missing from cg-2", team)
		}
		if !s.OnTime {
			t.Errorf("team %q at cg-2: over time, want on time", team)
		}
	}
}

func TestAggregateScansRelativeLineCatchesARealOverrun(t *testing.T) {
	windows := map[types.CheckgroupID]checkgroupWindow{
		"cg-1": {Scheme: types.CheckgroupSchemeNone},
		"cg-2": {Scheme: types.CheckgroupSchemeRelative, RelativeTo: "cg-1"},
	}
	scans := []rawScan{
		{CheckgroupID: "cg-1", TeamID: "t1", Uts: 1000},
		relativeScan("cg-2", "t1", 1000+45*minute, 30),
	}

	if aggregateScans(scans, windows)["cg-2"]["t1"].OnTime {
		t.Error("45 minutes into a 30 minute allowance is over time")
	}
}

func TestAggregateScansRelativeReferenceIsTheLatestScanAtTheBaseLine(t *testing.T) {
	// A line has several posts. The clock starts when the team leaves the line, so the
	// reference is its last scan there, not its first.
	windows := map[types.CheckgroupID]checkgroupWindow{
		"cg-1": {Scheme: types.CheckgroupSchemeNone},
		"cg-2": {Scheme: types.CheckgroupSchemeRelative, RelativeTo: "cg-1"},
	}
	scans := []rawScan{
		{CheckgroupID: "cg-1", TeamID: "t1", Uts: 1000},
		{CheckgroupID: "cg-1", TeamID: "t1", Uts: 1000 + 40*minute},
		relativeScan("cg-2", "t1", 1000+60*minute, 30),
	}

	if !aggregateScans(scans, windows)["cg-2"]["t1"].OnTime {
		t.Error("measured from the last scan at cg-1 the team is 20 minutes in, so on time")
	}
}

func TestAggregateScansReportsTheEarliestScanAsArrival(t *testing.T) {
	windows := map[types.CheckgroupID]checkgroupWindow{"cg-1": {Scheme: types.CheckgroupSchemeNone}}
	scans := []rawScan{
		{CheckgroupID: "cg-1", TeamID: "t1", Uts: 3000},
		{CheckgroupID: "cg-1", TeamID: "t1", Uts: 2000},
	}
	if got := aggregateScans(scans, windows)["cg-1"]["t1"].FirstUts; got != 2000 {
		t.Errorf("arrival = %d, want the earliest scan (2000)", got)
	}
}

func TestAggregateScansOneOnTimeScanIsEnough(t *testing.T) {
	windows := map[types.CheckgroupID]checkgroupWindow{"cg-1": {Scheme: types.CheckgroupSchemeFixed}}
	scans := []rawScan{
		{CheckgroupID: "cg-1", TeamID: "t1", Uts: 5000, OpenFromUts: 1000, OpenUntilUts: 2000},
		{CheckgroupID: "cg-1", TeamID: "t1", Uts: 1500, OpenFromUts: 1000, OpenUntilUts: 2000},
	}
	if !aggregateScans(scans, windows)["cg-1"]["t1"].OnTime {
		t.Error("the team was seen inside the window; a later scan must not undo that")
	}
}

// The deadline the dialog shows per row. Same arithmetic as the on-time verdict, so a row
// cannot say "til tiden" next to a deadline it appears to have missed.

func TestTeamDeadlineFixedLineIsTheLineClosing(t *testing.T) {
	got := teamDeadline(
		checkgroupWindow{Scheme: types.CheckgroupSchemeFixed},
		checkgroupLimit{ClosesAtUts: 2000},
		0,
	)
	if got != 2000 {
		t.Errorf("deadline = %d, want the line's closing time (2000)", got)
	}
}

func TestTeamDeadlineRelativeLineIsPerTeam(t *testing.T) {
	window := checkgroupWindow{Scheme: types.CheckgroupSchemeRelative, RelativeTo: "cg-1"}
	limit := checkgroupLimit{AllowanceSeconds: 30 * minute}
	if got := teamDeadline(window, limit, 1000); got != 1000+30*minute {
		t.Errorf("deadline = %d, want reference + allowance (%d)", got, 1000+30*minute)
	}
	if got := teamDeadline(window, limit, 1000+60*minute); got != 1000+90*minute {
		t.Error("a team an hour behind has a deadline an hour later; that is the whole scheme")
	}
}

func TestTeamDeadlineRelativeWithoutAReferenceScan(t *testing.T) {
	window := checkgroupWindow{Scheme: types.CheckgroupSchemeRelative, RelativeTo: "cg-1"}
	if got := teamDeadline(window, checkgroupLimit{AllowanceSeconds: 30 * minute}, 0); got != 0 {
		t.Errorf("deadline = %d, want none: the team's clock has not started", got)
	}
}

func TestTeamDeadlineNoScheme(t *testing.T) {
	// Even with hours on the posts, a line that keeps no time states no deadline.
	limit := checkgroupLimit{ClosesAtUts: 2000, AllowanceSeconds: 30 * minute}
	if got := teamDeadline(checkgroupWindow{Scheme: types.CheckgroupSchemeNone}, limit, 1000); got != 0 {
		t.Errorf("deadline = %d, want none", got)
	}
}

func TestTimingDeadlineUsesTheReferencedLinesScan(t *testing.T) {
	timing := &checkgroupTiming{
		windows: map[types.CheckgroupID]checkgroupWindow{
			"cg-2": {Scheme: types.CheckgroupSchemeRelative, RelativeTo: "cg-1"},
		},
		limits: map[types.CheckgroupID]checkgroupLimit{
			"cg-2": {AllowanceSeconds: 30 * minute},
		},
		lastScan: map[types.CheckgroupID]map[types.TeamID]int64{
			"cg-1": {"early": 1000, "late": 1000 + 60*minute},
		},
		scans: map[types.CheckgroupID]map[types.TeamID]checkgroupScan{},
	}

	if got := timing.Deadline("cg-2", "early"); got != 1000+30*minute {
		t.Errorf("early team's deadline = %d, want %d", got, 1000+30*minute)
	}
	if got := timing.Deadline("cg-2", "late"); got != 1000+90*minute {
		t.Errorf("late team's deadline = %d, want %d", got, 1000+90*minute)
	}
	if got := timing.Deadline("cg-2", "never-seen"); got != 0 {
		t.Errorf("unseen team's deadline = %d, want none", got)
	}
}

func TestTimingClosesAtIsSilentOnARelativeLine(t *testing.T) {
	// A relative line has no single closing time. Reporting one — the largest allowance read
	// as a clock time — would put a moment on screen that is nobody's deadline.
	timing := &checkgroupTiming{
		windows: map[types.CheckgroupID]checkgroupWindow{
			"cg-1": {Scheme: types.CheckgroupSchemeFixed},
			"cg-2": {Scheme: types.CheckgroupSchemeRelative, RelativeTo: "cg-1"},
		},
		limits: map[types.CheckgroupID]checkgroupLimit{
			"cg-1": {ClosesAtUts: 2000},
			"cg-2": {AllowanceSeconds: 30 * minute},
		},
	}
	if got := timing.ClosesAt("cg-1"); got != 2000 {
		t.Errorf("fixed line closes at %d, want 2000", got)
	}
	if got := timing.ClosesAt("cg-2"); got != 0 {
		t.Errorf("relative line closes at %d, want 0", got)
	}
}

func TestTimingScansIsNeverNil(t *testing.T) {
	timing := &checkgroupTiming{scans: map[types.CheckgroupID]map[types.TeamID]checkgroupScan{}}
	if timing.Scans("unknown") == nil {
		t.Error("Scans must return an empty map, so callers need no nil guard")
	}
}
