package main

import (
	"testing"
	"time"

	"github.com/nathejk/shared-go/tables/crewmember"
	"github.com/nathejk/shared-go/types"
	"nathejk.dk/nathejk/table/checkpersonnel"
	"nathejk.dk/nathejk/table/personnel"
)

func TestResolveScannerBanditIsCaughtByNumberNameAndKlan(t *testing.T) {
	lk := scannerLookup{
		senior: map[types.UserID]bandit{
			"u-bandit": {armNumber: "42", name: "Freja", klan: "Ulvene"},
		},
	}

	got := resolveScanner(lk, "u-bandit", 1000)

	if got.Kind != scannerBandit {
		t.Fatalf("kind = %q, want %q", got.Kind, scannerBandit)
	}
	if got.ArmNumber != "42" || got.Name != "Freja" || got.Klan != "Ulvene" {
		t.Errorf("bandit = %+v, want number 42, name Freja, klan Ulvene", got)
	}
}

// A bandit with no arm number assigned yet is still a bandit: the trail shows the
// name and klan, which is what an operator recognises.
func TestResolveScannerBanditWithoutArmNumberStillReadsAsBandit(t *testing.T) {
	lk := scannerLookup{
		senior: map[types.UserID]bandit{
			"u": {name: "Freja", klan: "Ulvene"},
		},
	}

	got := resolveScanner(lk, "u", 1000)
	if got.Kind != scannerBandit || got.Name != "Freja" || got.Klan != "Ulvene" {
		t.Errorf("bandit = %+v, want a bandit named Freja from Ulvene", got)
	}
}

func TestResolveScannerCrewCarriesSectionLabel(t *testing.T) {
	lk := scannerLookup{
		crew: map[types.UserID]crewmember.CrewMember{
			"u-crew": {UserID: "u-crew", Name: "Bo", SectionSlug: "hq"},
		},
		sectionLabel: map[string]string{"hq": "HQ"},
	}

	got := resolveScanner(lk, "u-crew", 1000)

	if got.Kind != scannerCrew {
		t.Fatalf("kind = %q, want %q", got.Kind, scannerCrew)
	}
	if got.Name != "Bo" || got.SectionLabel != "HQ" {
		t.Errorf("crew = %+v, want name Bo, section HQ", got)
	}
	if got.CheckpointName != "" {
		t.Errorf("checkpoint = %q, want empty when the scanner staffed no post at that time", got.CheckpointName)
	}
}

// The section label falls back to the slug rather than showing nothing when a section
// carries no label — a missing label is a data problem worth seeing.
func TestResolveScannerCrewFallsBackToSectionSlug(t *testing.T) {
	lk := scannerLookup{
		crew: map[types.UserID]crewmember.CrewMember{
			"u": {UserID: "u", Name: "Bo", SectionSlug: "postmandskab"},
		},
	}

	if got := resolveScanner(lk, "u", 1000); got.SectionLabel != "postmandskab" {
		t.Errorf("section = %q, want the slug as a fallback", got.SectionLabel)
	}
}

// A personnel helper (gøgler/ven) scanning is named rather than left unknown, even
// though it has no section.
func TestResolveScannerPersonnelHelperIsNamed(t *testing.T) {
	lk := scannerLookup{
		personnel: map[types.UserID]*personnel.Person{
			"u": {ID: "u", Name: "Klaus"},
		},
	}

	got := resolveScanner(lk, "u", 1000)
	if got.Kind != scannerCrew {
		t.Fatalf("kind = %q, want %q", got.Kind, scannerCrew)
	}
	if got.Name != "Klaus" || got.SectionLabel != "" {
		t.Errorf("helper = %+v, want name Klaus with no section", got)
	}
}

// A crew member scanning while on a checkpoint shift gets the checkpoint's name; the
// same person scanning outside every shift gets none.
func TestResolveScannerCrewOnShiftGetsTheCheckpointName(t *testing.T) {
	lk := scannerLookup{
		crew: map[types.UserID]crewmember.CrewMember{
			"u": {UserID: "u", Name: "Bo", SectionSlug: "postmandskab"},
		},
		sectionLabel:   map[string]string{"postmandskab": "Postmandskab"},
		checkpointName: map[types.CheckpointID]string{"cp-1": "Troldeskoven"},
		shifts: []checkpersonnel.Checkpersonnel{
			{UserID: "u", CheckpointID: "cp-1", Start: time.Unix(500, 0), End: time.Unix(1500, 0)},
		},
	}

	if got := resolveScanner(lk, "u", 1000); got.CheckpointName != "Troldeskoven" {
		t.Errorf("checkpoint = %q, want Troldeskoven for a scan inside the shift", got.CheckpointName)
	}
	if got := resolveScanner(lk, "u", 2000); got.CheckpointName != "" {
		t.Errorf("checkpoint = %q, want empty for a scan after the shift ended", got.CheckpointName)
	}
}

// A shift belonging to someone else at the same time must not be attributed.
func TestResolveScannerIgnoresAnotherPersonsShift(t *testing.T) {
	lk := scannerLookup{
		crew: map[types.UserID]crewmember.CrewMember{
			"u": {UserID: "u", Name: "Bo"},
		},
		checkpointName: map[types.CheckpointID]string{"cp-1": "Troldeskoven"},
		shifts: []checkpersonnel.Checkpersonnel{
			{UserID: "someone-else", CheckpointID: "cp-1", Start: time.Unix(0, 0), End: time.Unix(9999, 0)},
		},
	}

	if got := resolveScanner(lk, "u", 1000); got.CheckpointName != "" {
		t.Errorf("checkpoint = %q, want empty; the shift is another person's", got.CheckpointName)
	}
}

// A senior is checked before crew: a person recorded as both reads as the catcher.
func TestResolveScannerPrefersBanditOverCrew(t *testing.T) {
	lk := scannerLookup{
		senior: map[types.UserID]bandit{"u": {name: "Freja", klan: "Ulvene"}},
		crew:   map[types.UserID]crewmember.CrewMember{"u": {UserID: "u", Name: "Freja"}},
	}
	if got := resolveScanner(lk, "u", 1000); got.Kind != scannerBandit {
		t.Errorf("kind = %q, want %q; a senior is a bandit even when also crew", got.Kind, scannerBandit)
	}
}

func TestResolveScannerUnknownWhenNotFoundOrEmpty(t *testing.T) {
	lk := scannerLookup{}
	if got := resolveScanner(lk, "", 1000); got.Kind != scannerUnknown {
		t.Errorf("kind = %q, want %q for an empty scanner id", got.Kind, scannerUnknown)
	}
	if got := resolveScanner(lk, "nobody", 1000); got.Kind != scannerUnknown {
		t.Errorf("kind = %q, want %q for an unknown scanner", got.Kind, scannerUnknown)
	}
}
