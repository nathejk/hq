package data

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/nathejk/shared-go/types"
	"nathejk.dk/nathejk/table/checkgroup"
	"nathejk.dk/nathejk/table/checkpersonnel"
	"nathejk.dk/nathejk/table/checkpoint"
	"nathejk.dk/nathejk/table/klan"
	"nathejk.dk/nathejk/table/lok"
	"nathejk.dk/nathejk/table/patrulje"
	"nathejk.dk/nathejk/table/payment"
	"nathejk.dk/nathejk/table/personnel"
	"nathejk.dk/nathejk/table/scan"
	"nathejk.dk/nathejk/table/senior"
	"nathejk.dk/nathejk/table/spejder"
	"nathejk.dk/nathejk/table/year"
)

var (
	ErrRecordNotFound = errors.New("record not found")
	ErrEditConflict   = errors.New("edit conflict")
)

type KlanInterface interface {
	GetAll(context.Context, klan.Filter) ([]klan.Klan, error)
	GetByID(context.Context, types.TeamID) (*klan.Klan, error)
}
type SeniorInterface interface {
	GetAll(context.Context, senior.Filter) ([]*senior.Senior, error)
	GetByID(context.Context, types.MemberID) (*senior.Senior, error)
}
type PatruljeInterface interface {
	GetAll(context.Context, patrulje.Filter) ([]*patrulje.Patrulje, error)
	GetByID(context.Context, types.TeamID) (*patrulje.Patrulje, error)
}
type PersonnelInterface interface {
	GetAll(context.Context, personnel.Filter) ([]*personnel.Person, error)
	GetByID(context.Context, types.UserID) (*personnel.Person, error)
}
type PaymentInterface interface {
	GetAll(context.Context, types.TeamID) ([]*payment.Payment, error)
	GetByReference(context.Context, string) (*payment.Payment, error)
}
type SpejderInterface interface {
	GetAll(context.Context, spejder.Filter) ([]*spejder.Spejder, spejder.Metadata, error)
	GetByID(context.Context, types.MemberID) (*spejder.Spejder, error)
}
type CheckgroupInterface interface {
	GetAll(checkgroup.Filter) (checkgroup.Checkgroups, checkgroup.Metadata, error)
	GetByID(context.Context, types.CheckgroupID) (*checkgroup.Checkgroup, error)
	GetLatestTimeByTeam(ctx context.Context, cgID types.CheckgroupID, teamID types.TeamID) (*time.Time, error)
	GetNewestCheckgroupTeamTime(context.Context, checkgroup.Filter) (checkgroup.CheckgroupTeamTime, checkgroup.Metadata, error)
	//GetAllWithCounts(Filters, types.TeamIDs, types.TeamIDs) (Checkgroups, Metadata, error)
}
type ScanInterface interface {
	GetAll(context.Context, scan.Filter) ([]*scan.Scan, scan.Metadata, error)
	GetCheckgroupsScans(ctx context.Context, filters scan.Filter) ([]*scan.CheckgroupScan, scan.Metadata, error)
	//GetAllWithCounts(Filters, types.TeamIDs, types.TeamIDs) (Checkgroups, Metadata, error)
}
type LokInterface interface {
	GetAll(context.Context, lok.Filter) ([]*lok.Lok, lok.Metadata, error)
	GetByID(context.Context, types.LokID) (*lok.Lok, error)
}

type Models struct {
	Teams interface {
		GetStartedTeamIDs(Filters) ([]types.TeamID, Metadata, error)
		GetDiscontinuedTeamIDs(Filters) ([]types.TeamID, Metadata, error)
		GetPatruljer(Filters) ([]*Patrulje, Metadata, error)
		GetPatrulje(types.TeamID) (*Patrulje, error)
		GetKlan(types.TeamID) (*Klan, error)
		GetContact(types.TeamID) (*Contact, error)
		RequestedSeniorCount() int
	}
	Members interface {
		GetSpejdere(Filters) ([]*Spejder, Metadata, error)
		GetSeniore(Filters) ([]*Senior, Metadata, error)
		GetInactive(Filters) ([]*SpejderStatus, Metadata, error)
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
	Signup interface {
		GetByID(types.TeamID) (*Signup, error)
		ConfirmBySecret(string) (types.TeamID, error)
	}
	Year           year.Queries
	Klan           KlanInterface
	Senior         SeniorInterface
	Patrulje       patrulje.Queries
	Personnel      PersonnelInterface
	Payment        payment.Queries
	Spejder        SpejderInterface
	Checkgroup     checkgroup.Queries
	Checkpoint     checkpoint.Queries
	Checkpersonnel checkpersonnel.Queries
	Scan           ScanInterface
	Lok            LokInterface
}

func NewModels(db *sql.DB, y year.Queries, klan KlanInterface, senior SeniorInterface, patrulje patrulje.Queries, personnel PersonnelInterface, payment payment.Queries, spejder SpejderInterface, cg checkgroup.Queries, cp checkpoint.Queries, checkpersonnel checkpersonnel.Queries, scan ScanInterface, lok LokInterface) Models {
	return Models{
		Year:           y,
		Teams:          TeamModel{DB: db},
		Members:        MemberModel{DB: db},
		Permissions:    PermissionModel{DB: db},
		Tokens:         TokenModel{DB: db},
		Users:          UserModel{DB: db},
		Signup:         SignupModel{DB: db},
		Klan:           klan,
		Senior:         senior,
		Patrulje:       patrulje,
		Personnel:      personnel,
		Payment:        payment,
		Spejder:        spejder,
		Checkgroup:     cg,
		Checkpoint:     cp,
		Checkpersonnel: checkpersonnel,
		Scan:           scan,
		Lok:            lok,
	}
}

type CheckgroupsStatus struct {
	Checkgroups         []checkgroup.Checkgroup `json:"checkgroups"`
	StartedTeamIDs      types.TeamIDs           `json:"startedTeamIds"`
	DiscontinuedTeamIDs types.TeamIDs           `json:"discontinuedTeamIds"`
}

func (m *Models) GetCheckgroupsStatus(filters Filters) (CheckgroupsStatus, Metadata, error) {
	ctx := context.Background()
	var err error
	cs := CheckgroupsStatus{}
	if cs.Checkgroups, err = m.Checkgroup.GetAll(ctx, checkgroup.Filter{Year: "2025"}); err != nil {
		return CheckgroupsStatus{}, Metadata{}, err
	}
	if cs.StartedTeamIDs, _, err = m.Teams.GetStartedTeamIDs(filters); err != nil {
		return CheckgroupsStatus{}, Metadata{}, err
	}
	if cs.DiscontinuedTeamIDs, _, err = m.Teams.GetDiscontinuedTeamIDs(filters); err != nil {
		return CheckgroupsStatus{}, Metadata{}, err
	}
	/*
			ctt, _, err := m.Checkgroup.GetNewestCheckgroupTeamTime(ctx, checkgroup.Filters{Year: "2025"})
			if err != nil {
				return CheckgroupsStatus{}, Metadata{}, err
			}
		scans, _, err := m.Scan.GetCheckgroupsScans(ctx, scan.Filter{Year: "2025"})
		if err != nil {
			return CheckgroupsStatus{}, Metadata{}, err
		}

			for _, scan := range scans {
				cg := cs.Checkgroups.Checkgroup(scan.CheckgroupID)
				if cg == nil {
					continue
				}
				if cg.OnTime == nil {
					cg.OnTime = types.TeamIDs{}
					cg.OverTime = types.TeamIDs{}
					cg.NotArrived = types.TeamIDs{}
					cg.Discontinued = types.TeamIDs{}
				}
				cp := cg.Checkpoint(scan.CheckpointIndex)
				if cp == nil {
					continue
				}
				if cp.OnTime == nil {
					cp.OnTime = types.TeamIDs{}
					cp.OverTime = types.TeamIDs{}
				}
				onTime := false
				switch cp.Scheme {
				case types.CheckgroupSchemeFixed:
					openFrom := cp.OpenFrom.Add(-1 * time.Duration(cp.Minus) * time.Minute)
					openUntil := cp.OpenUntil.Add(time.Duration(cp.Plus) * time.Minute)
					if !scan.Time.Before(openFrom) && !scan.Time.After(openUntil) {
						onTime = true
					}
				case types.CheckgroupSchemeRelative:
					if tt, found := ctt[cp.RelativeCheckgroupID]; found {
						if ts, found := tt[scan.TeamID]; found {
							openUntil := ts.Add(time.Duration(cp.Plus) * time.Minute)
							if !scan.Time.After(openUntil) {
								onTime = true
							}
						}
					}
				case types.CheckgroupSchemeNone:
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
	*/
	// When the rows.Next() loop has finished, call rows.Err() to retrieve any error
	// that was encountered during the iteration.
	totalRecords := 0
	metadata := calculateMetadata(filters.Year, totalRecords, filters.Page, filters.PageSize)

	return cs, metadata, nil
}
