package data

import (
	"context"
	"database/sql"
	"errors"

	"github.com/nathejk/shared-go/tables/crewmember"
	"github.com/nathejk/shared-go/tables/klan"
	"github.com/nathejk/shared-go/tables/section"
	"github.com/nathejk/shared-go/tables/senior"
	"github.com/nathejk/shared-go/types"
	"nathejk.dk/nathejk/table/checkgroup"
	"nathejk.dk/nathejk/table/checkpersonnel"
	"nathejk.dk/nathejk/table/checkpoint"
	"nathejk.dk/nathejk/table/lok"
	"nathejk.dk/nathejk/table/order"
	"nathejk.dk/nathejk/table/patrulje"
	"nathejk.dk/nathejk/table/payment"
	"nathejk.dk/nathejk/table/personnel"
	"nathejk.dk/nathejk/table/scan"
	"nathejk.dk/nathejk/table/sos"
	"nathejk.dk/nathejk/table/year"
)

var (
	ErrRecordNotFound = errors.New("record not found")
)

type KlanInterface interface {
	GetAll(context.Context, klan.Filter) ([]klan.Klan, error)
	GetByID(context.Context, types.TeamID) (*klan.Klan, error)
}
type SeniorInterface interface {
	GetAll(context.Context, senior.Filter) ([]*senior.Senior, error)
	GetByID(context.Context, types.MemberID) (*senior.Senior, error)
}
type PersonnelInterface interface {
	GetAll(context.Context, personnel.Filter) ([]*personnel.Person, error)
	GetByID(context.Context, types.UserID) (*personnel.Person, error)
}
type ScanInterface interface {
	GetAll(context.Context, scan.Filter) ([]*scan.Scan, scan.Metadata, error)
	GetCheckgroupsScans(ctx context.Context, filters scan.Filter) ([]*scan.CheckgroupScan, scan.Metadata, error)
}
type LokInterface interface {
	GetAll(context.Context, lok.Filter) ([]*lok.Lok, lok.Metadata, error)
	GetByID(context.Context, types.LokID) (*lok.Lok, error)
}

type Models struct {
	Teams interface {
		GetPatrulje(types.TeamID) (*Patrulje, error)
		GetKlan(types.TeamID) (*Klan, error)
		GetContact(types.TeamID) (*Contact, error)
		RequestedSeniorCount() int
	}
	Members interface {
		GetSpejdere(Filters) ([]*Spejder, Metadata, error)
		GetSeniore(Filters) ([]*Senior, Metadata, error)
	}
	Signup interface {
		GetByID(types.TeamID) (*Signup, error)
	}
	Year           year.Queries
	Klan           KlanInterface
	Senior         SeniorInterface
	Patrulje       patrulje.Queries
	Personnel      PersonnelInterface
	Payment        payment.Queries
	Checkgroup     checkgroup.Queries
	Checkpoint     checkpoint.Queries
	Checkpersonnel checkpersonnel.Queries
	Scan           ScanInterface
	Lok            LokInterface
	Section        section.Queries
	CrewMember     crewmember.Queries
	Order          order.Queries
	Sos            sos.Queries
}

func NewModels(db *sql.DB, y year.Queries, klan KlanInterface, senior SeniorInterface, patrulje patrulje.Queries, personnel PersonnelInterface, payment payment.Queries, cg checkgroup.Queries, cp checkpoint.Queries, checkpersonnel checkpersonnel.Queries, scan ScanInterface, lok LokInterface, sec section.Queries, crew crewmember.Queries, ord order.Queries, sosq sos.Queries) Models {
	return Models{
		Year:           y,
		Teams:          TeamModel{DB: db},
		Members:        MemberModel{DB: db},
		Signup:         SignupModel{DB: db},
		Klan:           klan,
		Senior:         senior,
		Patrulje:       patrulje,
		Personnel:      personnel,
		Payment:        payment,
		Checkgroup:     cg,
		Checkpoint:     cp,
		Checkpersonnel: checkpersonnel,
		Scan:           scan,
		Lok:            lok,
		Section:        sec,
		CrewMember:     crew,
		Order:          ord,
		Sos:            sosq,
	}
}
