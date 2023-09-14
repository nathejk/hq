package data

import (
	"database/sql"
	"errors"
	"time"

	"nathejk.dk/nathejk/types"
)

var (
	ErrRecordNotFound = errors.New("record not found")
	ErrEditConflict   = errors.New("edit conflict")
)

type Models struct {
	Checkgroups interface {
		GetAll(Filters) (Checkgroups, Metadata, error)
		//GetAllWithCounts(Filters, types.TeamIDs, types.TeamIDs) (Checkgroups, Metadata, error)
	}
	Teams interface {
		GetStartedTeamIDs(Filters) ([]types.TeamID, Metadata, error)
		GetDiscontinuedTeamIDs(Filters) ([]types.TeamID, Metadata, error)
		GetPatruljer(Filters) ([]*Patrulje, Metadata, error)
		GetPatrulje(types.TeamID) (*Patrulje, error)
	}
	Members interface {
		GetSpejdere(Filters) ([]*Spejder, Metadata, error)
		GetInactive(Filters) ([]*SpejderStatus, Metadata, error)
	}
	Scans interface {
		GetCheckgroupsScans(Filters) ([]*CheckgroupScan, Metadata, error)
		GetNewestCheckgroupTeamTime(Filters) (CheckgroupTeamTime, Metadata, error)
	}
	Permissions interface {
		AddForUser(int64, ...string) error
		GetAllForUser(int64) (Permissions, error)
	}
	Tokens interface {
		New(userID int64, ttl time.Duration, scope string) (*Token, error)
		Insert(token *Token) error
		DeleteAllForUser(scope string, userID int64) error
	}
	Users interface {
		Insert(*User) error
		GetByEmail(string) (*User, error)
		Update(*User) error
		GetForToken(string, string) (*User, error)
	}
}

func NewModels(db *sql.DB) Models {
	return Models{
		Checkgroups: CheckgroupModel{DB: db},
		Teams:       TeamModel{DB: db},
		Members:     MemberModel{DB: db},
		Scans:       ScanModel{DB: db},
		Permissions: PermissionModel{DB: db},
		Tokens:      TokenModel{DB: db},
		Users:       UserModel{DB: db},
	}
}

type CheckgroupsStatus struct {
	Checkgroups         Checkgroups   `json:"checkgroups"`
	StartedTeamIDs      types.TeamIDs `json:"startedTeamIds"`
	DiscontinuedTeamIDs types.TeamIDs `json:"discontinuedTeamIds"`
}

func (m *Models) GetCheckgroupsStatus(filters Filters) (CheckgroupsStatus, Metadata, error) {
	var err error
	cs := CheckgroupsStatus{}
	if cs.Checkgroups, _, err = m.Checkgroups.GetAll(filters); err != nil {
		return CheckgroupsStatus{}, Metadata{}, err
	}
	if cs.StartedTeamIDs, _, err = m.Teams.GetStartedTeamIDs(filters); err != nil {
		return CheckgroupsStatus{}, Metadata{}, err
	}
	if cs.DiscontinuedTeamIDs, _, err = m.Teams.GetDiscontinuedTeamIDs(filters); err != nil {
		return CheckgroupsStatus{}, Metadata{}, err
	}

	ctt, _, err := m.Scans.GetNewestCheckgroupTeamTime(filters)
	if err != nil {
		return CheckgroupsStatus{}, Metadata{}, err
	}
	scans, _, err := m.Scans.GetCheckgroupsScans(filters)
	if err != nil {
		return CheckgroupsStatus{}, Metadata{}, err
	}
	for _, scan := range scans {
		cg := cs.Checkgroups.Checkgroup(scan.ControlGroupID)
		if cg == nil {
			continue
		}
		if cg.OnTime == nil {
			cg.OnTime = types.TeamIDs{}
			cg.OverTime = types.TeamIDs{}
			cg.NotArrived = types.TeamIDs{}
			cg.Discontinued = types.TeamIDs{}
		}
		cp := cg.Checkpoint(scan.ControlpointIndex)
		if cp == nil {
			continue
		}
		if cp.OnTime == nil {
			cp.OnTime = types.TeamIDs{}
			cp.OverTime = types.TeamIDs{}
		}
		onTime := false
		switch cp.Scheme {
		case types.CheckpointSchemeFixed:
			openFrom := cp.OpenFrom.Add(-1 * time.Duration(cp.Minus) * time.Minute)
			openUntil := cp.OpenUntil.Add(time.Duration(cp.Plus) * time.Minute)
			if !scan.Time.Before(openFrom) && !scan.Time.After(openUntil) {
				onTime = true
			}
		case types.CheckpointSchemeRelative:
			if tt, found := ctt[cp.RelativeControlGroupID]; found {
				if ts, found := tt[scan.TeamID]; found {
					openUntil := ts.Add(time.Duration(cp.Plus) * time.Minute)
					if !scan.Time.After(openUntil) {
						onTime = true
					}
				}
			}
		case types.CheckpointSchemeNone:
			onTime = true
		}

		if onTime {
			if !cp.OnTime.Exists(scan.TeamID) {
				cp.OnTime = append(cp.OnTime, scan.TeamID)
			}
			if !cg.OnTime.Exists(scan.TeamID) {
				cg.OnTime = append(cg.OnTime, scan.TeamID)
			}
			cp.OverTime = types.DiffTeamID(cp.OverTime, cp.OnTime)
			cg.OverTime = types.DiffTeamID(cg.OverTime, cg.OnTime)
		} else {
			if !cp.OnTime.Exists(scan.TeamID) && !cp.OverTime.Exists(scan.TeamID) {
				cp.OverTime = append(cp.OverTime, scan.TeamID)
			}
			if !cg.OnTime.Exists(scan.TeamID) && !cg.OverTime.Exists(scan.TeamID) {
				cg.OverTime = append(cg.OverTime, scan.TeamID)
			}
		}
	}

	for _, cg := range cs.Checkgroups {
		cg.Discontinued = types.DiffTeamID(cs.DiscontinuedTeamIDs, cg.OverTime, cg.OnTime)
		cg.NotArrived = types.DiffTeamID(cs.StartedTeamIDs, cs.DiscontinuedTeamIDs, cg.OverTime, cg.OnTime)
	}
	// When the rows.Next() loop has finished, call rows.Err() to retrieve any error
	// that was encountered during the iteration.
	totalRecords := 0
	metadata := calculateMetadata(filters.Year, totalRecords, filters.Page, filters.PageSize)

	return cs, metadata, nil
}
