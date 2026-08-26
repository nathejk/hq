package main

import (
	"testing"

	"github.com/nathejk/shared-go/types"
)

func racing(id types.TeamID, active int) startedTeam {
	return startedTeam{TeamID: id, ActiveMemberCount: active}
}

func TestResolveTeamStatusScanInsideTheWindowIsOnTime(t *testing.T) {
	status, uts := resolveTeamStatus(racing("t1", 4), &checkgroupScan{OnTime: true, FirstUts: 1000})
	if status != TeamAtCheckgroupOnTime {
		t.Errorf("status = %q, want %q", status, TeamAtCheckgroupOnTime)
	}
	if uts != 1000 {
		t.Errorf("scanned at %d, want the first scan (1000)", uts)
	}
}

func TestResolveTeamStatusScanOutsideTheWindowIsLate(t *testing.T) {
	status, _ := resolveTeamStatus(racing("t1", 4), &checkgroupScan{OnTime: false, FirstUts: 2000})
	if status != TeamAtCheckgroupLate {
		t.Errorf("status = %q, want %q", status, TeamAtCheckgroupLate)
	}
}

// The case the whole feature exists for: with a line about to close, the missing list
// is who to chase, and it must not send anybody after a patrol that went home.
func TestResolveTeamStatusWithdrawnAndUnseenIsRetiredNotMissing(t *testing.T) {
	status, uts := resolveTeamStatus(racing("t1", 0), nil)
	if status != TeamAtCheckgroupRetired {
		t.Errorf("status = %q, want %q", status, TeamAtCheckgroupRetired)
	}
	if uts != 0 {
		t.Errorf("scanned at %d, want 0 for a team never seen here", uts)
	}
}

// A team that came through and *later* withdrew passed this post. Calling it retired
// would erase what the post staff recorded, and make the screen disagree with the
// paper on the table.
func TestResolveTeamStatusScanWinsOverWithdrawal(t *testing.T) {
	status, _ := resolveTeamStatus(racing("t1", 0), &checkgroupScan{OnTime: true, FirstUts: 500})
	if status != TeamAtCheckgroupOnTime {
		t.Errorf("status = %q, want %q: the team was seen here before it withdrew", status, TeamAtCheckgroupOnTime)
	}
}

func TestResolveTeamStatusStillRacingAndUnseenIsMissing(t *testing.T) {
	status, _ := resolveTeamStatus(racing("t1", 3), nil)
	if status != TeamAtCheckgroupMissing {
		t.Errorf("status = %q, want %q", status, TeamAtCheckgroupMissing)
	}
}

// The bug this replaces: `missing` started at the number of started teams and was
// decremented per scanning team, so the four numbers only added up by luck — and
// nothing ever landed in the retired bucket, because nothing computed it.
func TestCheckgroupCountsPartitionEveryStartedTeam(t *testing.T) {
	teams := []startedTeam{
		racing("on-time", 4),
		racing("late", 4),
		racing("retired", 0),
		racing("missing-1", 3),
		racing("missing-2", 2),
	}
	scans := map[types.TeamID]checkgroupScan{
		"on-time": {OnTime: true, FirstUts: 100},
		"late":    {OnTime: false, FirstUts: 200},
	}

	stats := countStatuses("cg-1", resolveTeamStatuses(teams, scans))

	if stats.OnTime != 1 || stats.Late != 1 || stats.Retired != 1 || stats.Missing != 2 {
		t.Errorf("got onTime=%d late=%d retired=%d missing=%d; want 1/1/1/2",
			stats.OnTime, stats.Late, stats.Retired, stats.Missing)
	}
	if total := stats.OnTime + stats.Late + stats.Retired + stats.Missing; total != len(teams) {
		t.Errorf("the four numbers sum to %d, want every started team (%d)", total, len(teams))
	}
}

// A scan from a team that never started — or one since deleted — used to decrement
// `missing` and push the total below the number of started teams. Driving the
// computation from the team list makes that impossible.
func TestCheckgroupCountsIgnoreScansFromTeamsThatDidNotStart(t *testing.T) {
	teams := []startedTeam{racing("started", 4)}
	scans := map[types.TeamID]checkgroupScan{
		"started": {OnTime: true, FirstUts: 100},
		"ghost":   {OnTime: true, FirstUts: 100},
	}

	stats := countStatuses("cg-1", resolveTeamStatuses(teams, scans))

	if stats.OnTime != 1 {
		t.Errorf("onTime = %d, want 1", stats.OnTime)
	}
	if total := stats.OnTime + stats.Late + stats.Retired + stats.Missing; total != 1 {
		t.Errorf("total = %d, want 1: a stranger's scan must not change the count", total)
	}
}

// Before any scanner is on duty, every racing team is missing and every withdrawn one
// retired. This is the ordinary state at the start of the night, not an edge case.
func TestCheckgroupCountsWithNoScansAtAll(t *testing.T) {
	teams := []startedTeam{racing("a", 4), racing("b", 0)}

	stats := countStatuses("cg-1", resolveTeamStatuses(teams, map[types.TeamID]checkgroupScan{}))

	if stats.Missing != 1 || stats.Retired != 1 {
		t.Errorf("got missing=%d retired=%d, want 1/1", stats.Missing, stats.Retired)
	}
}

func TestCheckgroupCountsWithNoStartedTeams(t *testing.T) {
	stats := countStatuses("cg-1", resolveTeamStatuses(nil, map[types.TeamID]checkgroupScan{}))
	if stats.OnTime+stats.Late+stats.Retired+stats.Missing != 0 {
		t.Errorf("counted something for an empty race: %+v", stats)
	}
}

// Team numbers are read out and written down, so they sort as numbers: string order
// would file 10 between 1 and 2.
func TestTeamNumberValue(t *testing.T) {
	cases := []struct {
		in    string
		value int
		ok    bool
	}{
		{"1", 1, true},
		{"10", 10, true},
		{"007", 7, true},
		{"", 0, false},
		{"12b", 0, false},
		{"x", 0, false},
	}
	for _, c := range cases {
		value, ok := teamNumberValue(c.in)
		if value != c.value || ok != c.ok {
			t.Errorf("teamNumberValue(%q) = (%d, %t), want (%d, %t)", c.in, value, ok, c.value, c.ok)
		}
	}
}
