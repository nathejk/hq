package main

import (
	"context"

	"github.com/nathejk/shared-go/tables/crewmember"
	"github.com/nathejk/shared-go/tables/klan"
	"github.com/nathejk/shared-go/tables/section"
	"github.com/nathejk/shared-go/tables/senior"
	"github.com/nathejk/shared-go/types"
	"nathejk.dk/nathejk/table/checkpersonnel"
	"nathejk.dk/nathejk/table/checkpoint"
	"nathejk.dk/nathejk/table/personnel"
	"nathejk.dk/nathejk/table/scan"
)

// The kinds of person a scan can be attributed to. The client turns these into the
// two verbs an operator reads: a bandit "Fanget af" (caught), everyone else
// "Scannet af" (scanned). Unknown is a scan by a userId we can no longer name —
// history is replayed from the log, and a scanner deleted since is not an error.
const (
	scannerBandit  = "bandit"
	scannerCrew    = "crew"
	scannerUnknown = "unknown"
)

// scanEvent is one scan on a patrol's trail, with the scanner resolved to a person.
//
// The phone number the scan carries is deliberately not here: it is what the raw row
// held and what the trail used to show, but a phone number is not something an
// operator can act on at a glance. Who the scanner was — and, for crew, where they
// stood — is.
type scanEvent struct {
	Uts       int64   `json:"uts"`
	Latitude  string  `json:"latitude"`
	Longitude string  `json:"longitude"`
	Scanner   scanner `json:"scanner"`
}

// scanner is the resolved identity behind a scan. Fields are populated per Kind:
// a bandit carries ArmNumber/Name/Klan, crew carries SectionLabel/Name and, when the
// scan fell inside one of their checkpoint shifts, CheckpointName. omitempty so the
// payload does not assert fields that do not apply to the kind.
type scanner struct {
	Kind           string `json:"kind"`
	Name           string `json:"name,omitempty"`
	ArmNumber      string `json:"armNumber,omitempty"`
	Klan           string `json:"klan,omitempty"`
	SectionLabel   string `json:"sectionLabel,omitempty"`
	CheckpointName string `json:"checkpointName,omitempty"`
	// Phone is the scan's raw scanner phone, carried only when the scanner could not
	// be named. A phone is hard to act on, which is why a resolved scanner drops it —
	// but for an unknown it is all we have, and better than nothing.
	Phone string `json:"phone,omitempty"`
}

// bandit is a klan member out catching patrols, reduced to what the trail shows. The
// klan name is resolved when the lookup is built so resolveScanner stays a pure map read.
type bandit struct {
	armNumber string
	name      string
	klan      string
}

// scannerLookup is everything needed to name a scanner, loaded once per request and
// shared across the handful of scans a patrol has. Kept as plain maps and a slice so
// resolveScanner is a pure function of its inputs and can be tested without a database.
//
// The three identity maps are the three populations that scan, keyed by the id the
// scan carries: seniors (the bandits), crew members, and personnel helpers
// (gøglere/venner). They are disjoint in practice, and resolveScanner checks them in
// that order.
type scannerLookup struct {
	senior         map[types.UserID]bandit
	crew           map[types.UserID]crewmember.CrewMember
	personnel      map[types.UserID]*personnel.Person
	sectionLabel   map[string]string
	checkpointName map[types.CheckpointID]string
	shifts         []checkpersonnel.Checkpersonnel
}

// resolveScanner names the person behind one scan.
//
// A bandit is a klan member (a senior), so it is caught ("Fanget af") and carries a
// klan; crew and personnel helpers staff the event, so they scanned ("Scannet af").
// The order is the priority: the populations barely overlap, but a person recorded in
// two should read as the catcher, the more specific event. A personnel helper has no
// section — only a name — but a name is still worth showing over "ukendt".
func resolveScanner(lk scannerLookup, id types.UserID, uts int64) scanner {
	if id == "" {
		return scanner{Kind: scannerUnknown}
	}
	if b, ok := lk.senior[id]; ok {
		return scanner{
			Kind:      scannerBandit,
			ArmNumber: b.armNumber,
			Name:      b.name,
			Klan:      b.klan,
		}
	}
	if c, ok := lk.crew[id]; ok {
		return scanner{
			Kind:           scannerCrew,
			Name:           c.Name,
			SectionLabel:   lk.sectionLabelFor(c.SectionSlug),
			CheckpointName: lk.checkpointAt(id, uts),
		}
	}
	if p, ok := lk.personnel[id]; ok {
		return scanner{
			Kind:           scannerCrew,
			Name:           p.Name,
			CheckpointName: lk.checkpointAt(id, uts),
		}
	}
	return scanner{Kind: scannerUnknown}
}

// sectionLabelFor is the human label for a crew member's section, falling back to the
// slug when no label is recorded rather than showing nothing — an unlabelled section
// is a data problem better made visible than hidden.
func (lk scannerLookup) sectionLabelFor(slug types.Slug) string {
	if label := lk.sectionLabel[string(slug)]; label != "" {
		return label
	}
	return string(slug)
}

// checkpointAt returns the checkpoint the scanner was staffing at the moment of the
// scan, or "" if none. A scan carries no checkpoint; the only link is that the
// scanner was on a registered shift there when they scanned — the same attribution
// the post list uses (scansByCheckgroup). The first matching shift wins; overlapping
// shifts for one person at one instant are not a case the rota produces.
func (lk scannerLookup) checkpointAt(id types.UserID, uts int64) string {
	for _, s := range lk.shifts {
		if s.UserID != id {
			continue
		}
		if uts < s.Start.Unix() || uts > s.End.Unix() {
			continue
		}
		return lk.checkpointName[s.CheckpointID]
	}
	return ""
}

// enrichScans turns raw scan rows into trail events with their scanner named.
//
// Everything is loaded once and reused: a patrol has a handful of scans by even
// fewer distinct scanners, so the cost is the reads below, not one per scan. Read
// failures degrade rather than fail the request — a scan whose scanner cannot be
// looked up still belongs on the trail, as an unknown.
func (app *application) enrichScans(ctx context.Context, year types.YearSlug, scans []*scan.Scan) []scanEvent {
	lk := scannerLookup{
		senior:         map[types.UserID]bandit{},
		crew:           map[types.UserID]crewmember.CrewMember{},
		personnel:      map[types.UserID]*personnel.Person{},
		sectionLabel:   map[string]string{},
		checkpointName: map[types.CheckpointID]string{},
	}

	// Klan names first: a bandit's klan is its senior row's teamId resolved here.
	klanName := map[types.TeamID]string{}
	if klaner, err := app.models.Klan.GetAll(ctx, klan.Filter{YearSlug: string(year)}); err == nil {
		for _, k := range klaner {
			klanName[k.ID] = k.Name
		}
	}
	if seniors, err := app.models.Senior.GetAll(ctx, senior.Filter{YearSlug: string(year)}); err == nil {
		for _, s := range seniors {
			lk.senior[types.UserID(s.MemberID)] = bandit{
				armNumber: s.ArmNumber,
				name:      s.Name,
				klan:      klanName[s.TeamID],
			}
		}
	}

	ids := uniqueScannerIDs(scans)
	if len(ids) > 0 {
		if people, err := app.models.Personnel.GetAll(ctx, personnel.Filter{YearSlug: year, UserIDs: ids}); err == nil {
			for _, p := range people {
				lk.personnel[p.ID] = p
			}
		}
	}
	if crew, err := app.models.CrewMember.GetAll(ctx, crewmember.Filter{YearSlug: year}); err == nil {
		for _, c := range crew {
			lk.crew[c.UserID] = c
		}
	}
	if secs, err := app.models.Section.GetAll(ctx, section.Filter{YearSlug: year}); err == nil {
		for _, s := range secs {
			lk.sectionLabel[string(s.Slug)] = s.Label
		}
	}
	if cps, err := app.models.Checkpoint.GetAll(ctx, checkpoint.Filter{YearSlug: string(year)}); err == nil {
		for _, cp := range cps {
			lk.checkpointName[cp.ID] = cp.Name
		}
	}
	if shifts, err := app.models.Checkpersonnel.GetAll(ctx, checkpersonnel.Filter{Year: year}); err == nil {
		lk.shifts = shifts
	}

	events := make([]scanEvent, 0, len(scans))
	for _, s := range scans {
		sc := resolveScanner(lk, types.UserID(s.ScannerID), int64(s.Uts))
		// A scanner we could not name still leaves a phone behind; show it rather than
		// nothing.
		if sc.Kind == scannerUnknown {
			sc.Phone = s.ScannerPhone
		}
		events = append(events, scanEvent{
			Uts:       int64(s.Uts),
			Latitude:  s.Latitude,
			Longitude: s.Longitude,
			Scanner:   sc,
		})
	}
	return events
}

func uniqueScannerIDs(scans []*scan.Scan) []types.UserID {
	seen := map[types.UserID]bool{}
	ids := []types.UserID{}
	for _, s := range scans {
		id := types.UserID(s.ScannerID)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	return ids
}
