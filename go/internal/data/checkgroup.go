package data

import (
	"context"
	"database/sql"
	"time"

	"nathejk.dk/internal/validator"
	"nathejk.dk/nathejk/types"
)

type CheckpointScan struct {
	ScanID types.ScanID `json:"scanId"`
	TeamID types.TeamID `json:"teamId"`
	Time   time.Time    `json:"time"`
}
type CheckpointScanner struct {
	UserID types.UserID     `json:"userId"`
	Start  time.Time        `json:"start"`
	End    time.Time        `json:"end"`
	Scans  []CheckpointScan `json:"scans"`
}
type Checkpoint struct {
	Name                   string                 `json:"name"`
	Index                  int                    `json:"index"`
	Scheme                 types.CheckpointScheme `json:"scheme"`
	RelativeControlGroupID types.ControlGroupID   `json:"relativeControlGroupId"`
	OpenFrom               time.Time              `json:"openFrom"`
	OpenUntil              time.Time              `json:"openUntil"`
	Plus                   int                    `json:"plus"`
	Minus                  int                    `json:"minus"`

	OnTime   types.TeamIDs       `json:"onTimeTeamIds"`
	OverTime types.TeamIDs       `json:"overTimeTeamIds"`
	Scanners []CheckpointScanner `json:"scanners"`
}

type Checkgroup struct {
	ID        types.ControlGroupID `json:"id"`
	CreatedAt time.Time            `json:"created_at"`
	Version   uint64               `json:"version"`

	Name         string        `json:"name"`
	Checkpoints  []*Checkpoint `json:"checkpoints"`
	OnTime       types.TeamIDs `json:"onTimeTeamIds"`
	OverTime     types.TeamIDs `json:"overTimeTeamIds"`
	NotArrived   types.TeamIDs `json:"notArrivedTeamIds"`
	Discontinued types.TeamIDs `json:"discontinuedTeamIds"`
}
type Checkgroups []*Checkgroup

func (cgs *Checkgroups) Checkgroup(ID types.ControlGroupID) *Checkgroup {
	for _, cg := range *cgs {
		if cg.ID == ID {
			return cg
		}
	}
	return nil
}

func (cg *Checkgroup) Checkpoint(index int) *Checkpoint {
	for _, cp := range cg.Checkpoints {
		if cp.Index == index {
			return cp
		}
	}
	return nil
}

func (p *Checkgroup) Validate(v validator.Validator) {
	//v.Check(p.Timestamp.IsZero(), "timestamp", "must be provided")
}

type CheckgroupModel struct {
	DB *sql.DB
}

func (m CheckgroupModel) GetAll(filters Filters) (Checkgroups, Metadata, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `SELECT cg.id, cg.name, cp.controlIndex, cp.controlName, cp.scheme, cp.relativeControlGroupId, openFromUts, openUntilUts, plusMinutes, minusMinutes, userId, startUts, endUts
from controlgroup cg
left join controlpoint cp ON cg.id=cp.controlGroupId
left join controlgroup_user cgu on cg.id=cgu.controlGroupId AND cp.controlIndex=cgu.controlIndex
		   WHERE (LOWER(year) = LOWER(?) OR ? = '') 
ORDER BY openFromUts, controlIndex`

	type row struct {
		ControlGroupID       types.ControlGroupID
		ControlGroupName     string
		CheckpointIndex      *int
		CheckpointName       *string
		Scheme               *types.CheckpointScheme
		RelativeCheckGroupID *types.ControlGroupID
		OpenFromUts          *types.UnixtimeInteger
		OpenUntilUts         *types.UnixtimeInteger
		Plus                 *int
		Minus                *int
		UserID               *types.UserID
		StartUts             *types.UnixtimeInteger
		EndUts               *types.UnixtimeInteger
	}

	args := []any{filters.Year, filters.Year}
	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, Metadata{}, err
	}
	defer rows.Close()

	cgs := map[types.ControlGroupID]*Checkgroup{}
	for rows.Next() {
		var r row
		err := rows.Scan(&r.ControlGroupID, &r.ControlGroupName, &r.CheckpointIndex, &r.CheckpointName, &r.Scheme, &r.RelativeCheckGroupID, &r.OpenFromUts, &r.OpenUntilUts, &r.Plus, &r.Minus, &r.UserID, &r.StartUts, &r.EndUts)
		if err != nil {
			return nil, Metadata{}, err
		}
		if cgs[r.ControlGroupID] == nil {
			cgs[r.ControlGroupID] = &Checkgroup{
				ID:          r.ControlGroupID,
				Name:        r.ControlGroupName,
				Checkpoints: []*Checkpoint{},
			}
		}
		if r.CheckpointIndex == nil {
			continue
		}
		cp := cgs[r.ControlGroupID].Checkpoint(*r.CheckpointIndex)
		if cp == nil {
			cp = &Checkpoint{
				Index:                  *r.CheckpointIndex,
				Name:                   *r.CheckpointName,
				Scheme:                 *r.Scheme,
				RelativeControlGroupID: *r.RelativeCheckGroupID,
				OpenFrom:               *r.OpenFromUts.Time(),
				OpenUntil:              *r.OpenUntilUts.Time(),
				Plus:                   *r.Plus,
				Minus:                  *r.Minus,
				Scanners:               []CheckpointScanner{},
			}
			if cp.Scheme == "" {
				cp.Scheme = types.CheckpointSchemeFixed
			}
			cgs[r.ControlGroupID].Checkpoints = append(cgs[r.ControlGroupID].Checkpoints, cp)
		}
		if r.UserID == nil {
			continue
		}
		cp.Scanners = append(cp.Scanners, CheckpointScanner{
			UserID: *r.UserID,
			Start:  *r.StartUts.Time(),
			End:    *r.EndUts.Time(),
		})

	}

	checkgroups := []*Checkgroup{}
	for _, cg := range cgs {
		checkgroups = append(checkgroups, cg)
	}
	return checkgroups, Metadata{}, nil
}
