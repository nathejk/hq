package checkgroup

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/nathejk/shared-go/types"
	tables "nathejk.dk/nathejk/table"
)

type Queries interface {
	GetByID(context.Context, types.CheckgroupID) (*Checkgroup, error)
	GetAll(context.Context, Filter) ([]Checkgroup, error)
}

type querier struct {
	db *sql.DB
	r  *goqu.Database
}

func (q *querier) GetLatestTimeByTeam(ctx context.Context, cgID types.CheckgroupID, teamID types.TeamID) (*time.Time, error) {
	return nil, nil
}

type CheckgroupTeamTime map[types.CheckgroupID]map[types.TeamID]time.Time

func (q *querier) GetNewestCheckgroupTeamTime(ctx context.Context, filters Filter) (CheckgroupTeamTime, Metadata, error) {
	// Create a context with a 3-second timeout.
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	query := `SELECT cg.id, s.teamId, MAX(s.uts) FROM controlgroup cg
	JOIN controlpoint cp ON cg.id = cp.relativeControlGroupId AND (LOWER(cg.year) = LOWER(?) OR ? = '')
	JOIN controlgroup_user cgu ON cg.id = cgu.controlGroupId AND cp.controlIndex = cgu.controlIndex
	JOIN scan s ON cgu.userId = s.scannerId AND cgu.startUts <= s.uts AND cgu.endUts >= s.uts
	GROUP BY cg.id, s.teamId`
	args := []any{filters.Year, filters.Year}
	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, Metadata{}, err
	}
	defer rows.Close()

	totalRecords := 0
	ctt := CheckgroupTeamTime{}
	for rows.Next() {
		var r struct {
			CheckgroupID types.CheckgroupID
			TeamID       types.TeamID
			Uts          types.UnixtimeInteger
		}
		err := rows.Scan(&r.CheckgroupID, &r.TeamID, &r.Uts)
		if err != nil {
			return nil, Metadata{}, err
		}
		if _, found := ctt[r.CheckgroupID]; !found {
			ctt[r.CheckgroupID] = map[types.TeamID]time.Time{}
		}
		ctt[r.CheckgroupID][r.TeamID] = *r.Uts.Time()
	}

	// When the rows.Next() loop has finished, call rows.Err() to retrieve any error
	// that was encountered during the iteration.
	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}
	metadata := calculateMetadata(filters.Year, totalRecords, filters.Page, filters.PageSize)

	return ctt, metadata, nil
}

func (q *querier) GetByID(ctx context.Context, cgID types.CheckgroupID) (*Checkgroup, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	query := `SELECT id, year, name, showOnMap, mandatory, scheme, relativeCheckgroupId, sortOrder FROM checkgroup WHERE id = ?`

	var cg Checkgroup
	err := q.db.QueryRow(query, cgID).Scan(&cg.ID, &cg.YearSlug, &cg.Name, &cg.ShowOnMap, &cg.Mandatory, &cg.Scheme, &cg.RelativeCheckgroupID, &cg.SortOrder)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, tables.ErrRecordNotFound
		default:
			return nil, err
		}
	}
	return &cg, nil
}

func (q *querier) GetAll(ctx context.Context, f Filter) ([]Checkgroup /* Metadata,*/, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	columns := []interface{}{"id", "year", "name", "showOnMap", "mandatory", "scheme", "relativeCheckgroupId", "sortOrder"}
	where := goqu.Ex{}
	if f.Year != "" {
		where["year"] = string(f.Year)
	}

	var checkgroups []Checkgroup
	err := q.r.From("checkgroup").Select(columns...).Where(where).Order(goqu.I("sortOrder").Asc()).ScanStructs(&checkgroups)
	if err != nil {
		return nil, err
	}
	return checkgroups, nil

	//err := q.db.QueryRow(query, cgID).Scan(&cg.ID, &cg.YearSlug, &cg.Name, &cg.ShowOnMap, &cg.Mandatory, &cg.Scheme, &cg.RelativeCheckgroupID)
	/*
	   	query := `SELECT cg.id, cg.name, cp.controlIndex, cp.controlName, cp.scheme, cp.relativeControlGroupId, openFromUts, openUntilUts, plusMinutes, minusMinutes, userId, startUts, endUts
	   from controlgroup cg
	   left join controlpoint cp ON cg.id=cp.controlGroupId
	   left join controlgroup_user cgu on cg.id=cgu.controlGroupId AND cp.controlIndex=cgu.controlIndex
	   		   WHERE (LOWER(year) = LOWER(?) OR ? = '')
	   ORDER BY openFromUts, controlIndex`

	   	type row struct {
	   		CheckgroupID         types.CheckgroupID
	   		CheckgroupName       string
	   		CheckpointIndex      *int
	   		CheckpointName       *string
	   		Scheme               *types.CheckgroupScheme
	   		RelativeCheckgroupID *types.CheckgroupID
	   		OpenFromUts          *types.UnixtimeInteger
	   		OpenUntilUts         *types.UnixtimeInteger
	   		Plus                 *int
	   		Minus                *int
	   		UserID               *types.UserID
	   		StartUts             *types.UnixtimeInteger
	   		EndUts               *types.UnixtimeInteger
	   	}
	*/ /*
		args := []any{filters.Year, filters.Year}
		rows, err := q.db.QueryContext(ctx, query, args...)
		if err != nil {
			return nil, err
			//return nil, Metadata{}, err
		}
		defer rows.Close()

		cgs := map[types.CheckgroupID]*Checkgroup{}
		for rows.Next() {
			var r row
			err := rows.Scan(&r.CheckgroupID, &r.CheckgroupName, &r.CheckpointIndex, &r.CheckpointName, &r.Scheme, &r.RelativeCheckgroupID, &r.OpenFromUts, &r.OpenUntilUts, &r.Plus, &r.Minus, &r.UserID, &r.StartUts, &r.EndUts)
			if err != nil {
				return nil, err
				//	return nil, Metadata{}, err
			}
			if cgs[r.CheckgroupID] == nil {
				cgs[r.CheckgroupID] = &Checkgroup{
					ID:          r.CheckgroupID,
					Name:        r.CheckgroupName,
					Checkpoints: []*checkpoint.Checkpoint{},
				}
			}
			if r.CheckpointIndex == nil {
				continue
			}
			cp := cgs[r.CheckgroupID].Checkpoint(*r.CheckpointIndex)
			if cp == nil {
				cp = &checkpoint.Checkpoint{
					SortOrder:            *r.CheckpointIndex,
					Name:                 *r.CheckpointName,
					Scheme:               *r.Scheme,
					RelativeCheckgroupID: *r.RelativeCheckgroupID,
					OpenFrom:             *r.OpenFromUts.Time(),
					OpenUntil:            *r.OpenUntilUts.Time(),
					Plus:                 *r.Plus,
					Minus:                *r.Minus,
					Scanners:             []checkpersonnel.CheckPersonnel{},
				}
				if cp.Scheme == "" {
					cp.Scheme = types.CheckgroupSchemeFixed
				}
				cgs[r.CheckgroupID].Checkpoints = append(cgs[r.CheckgroupID].Checkpoints, cp)
			}
			if r.UserID == nil {
				continue
			}
			cp.Scanners = append(cp.Scanners, checkpersonnel.CheckPersonnel{
				UserID: *r.UserID,
				Start:  *r.StartUts.Time(),
				End:    *r.EndUts.Time(),
			})

		}

		checkgroups := []Checkgroup{}
		for _, cg := range cgs {
			checkgroups = append(checkgroups, *cg)
		}//*/
	//return checkgroups /*Metadata{},*/, nil
}
