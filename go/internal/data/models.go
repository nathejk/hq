package data

import (
	"context"
	"database/sql"
	"errors"

	"github.com/nathejk/shared-go/tables/crewmember"
	"github.com/nathejk/shared-go/tables/klan"
	"github.com/nathejk/shared-go/tables/order"
	"github.com/nathejk/shared-go/tables/payment"
	"github.com/nathejk/shared-go/tables/section"
	"github.com/nathejk/shared-go/tables/senior"
	"github.com/nathejk/shared-go/tables/spejder"
	"github.com/nathejk/shared-go/tables/vehicle"
	"github.com/nathejk/shared-go/types"
	"nathejk.dk/nathejk/table/checkgroup"
	"nathejk.dk/nathejk/table/checkpersonnel"
	"nathejk.dk/nathejk/table/checkpoint"
	"nathejk.dk/nathejk/table/dispatch"
	"nathejk.dk/nathejk/table/kort"
	"nathejk.dk/nathejk/table/lok"
	"nathejk.dk/nathejk/table/patrulje"
	"nathejk.dk/nathejk/table/personnel"
	"nathejk.dk/nathejk/table/photo"
	"nathejk.dk/nathejk/table/photocover"
	"nathejk.dk/nathejk/table/scan"
	"nathejk.dk/nathejk/table/shelter"
	"nathejk.dk/nathejk/table/sos"
	"nathejk.dk/nathejk/table/spejdernote"
	"nathejk.dk/nathejk/table/spejderstatus"
	"nathejk.dk/nathejk/table/track"
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
	TeamPositions(context.Context, string) ([]scan.TeamPosition, error)
}
type LokInterface interface {
	GetAll(context.Context, lok.Filter) ([]*lok.Lok, lok.Metadata, error)
	GetByID(context.Context, types.LokID) (*lok.Lok, error)
}

// OrderInterface is what hq needs from the order entity: the read API, plus the
// per-member paid-line read that the seat-transfer preview shares with the transfer
// command itself (PRD 012).
//
// Composed here rather than widening order.Queries in shared-go, which is implemented
// by fakes in other repos. Sharing the read is the point: the dialog that tells an
// operator "seat 250 kr and t-shirt 175 kr will move" and the command that moves them
// must not be able to disagree about which lines those are.
type OrderInterface interface {
	order.Queries
	order.MemberLineReader
}

// PhotoInterface is what hq needs from the photograph read model: a team's
// pictures. The photo entity is foto's package copied verbatim, so this narrow
// interface is declared here rather than added there.
type PhotoInterface interface {
	ByTeam(ctx context.Context, year, teamID string) ([]photo.Photo, error)
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
		TeamIDsByType(context.Context, types.YearSlug, types.TeamType) (map[types.TeamID]bool, error)
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
	Vehicle        vehicle.Queries
	Order          OrderInterface
	Sos            sos.Queries
	Shelter        shelter.Queries
	Note           spejdernote.Queries
	Dispatch       dispatch.Queries
	// Kort is the printed sheets and their sets (PRD 010).
	Kort          kort.Queries
	SpejderStatus spejderstatus.Queries
	// Track is where people were: positions reported by the hej-app (PRD 011).
	Track track.Queries

	// Roster answers which team a member is currently on (PRD 012).
	//
	// A one-method read interface owned by the spejder entity rather than a method on
	// Members: Members is this package's own hand-rolled SQL over the roster, and a
	// second definition of "which team is this member on" is exactly the drift the
	// transfer command avoids by reading the roster itself. Handler and command
	// therefore agree by construction.
	Roster spejder.RosterReader

	// The photograph read models. Assigned by the caller after NewModels rather than
	// passed to it: that constructor is already twenty-odd positional arguments, and two
	// more would make a mis-ordered call a runtime surprise rather than a compile error.
	//
	// Photo is a team's photographs, projected from foto's events; PhotoCover is which
	// one an organizer picked to represent the team.
	Photo      PhotoInterface
	PhotoCover photocover.Queries
}

func NewModels(db *sql.DB, y year.Queries, klan KlanInterface, senior SeniorInterface, patrulje patrulje.Queries, personnel PersonnelInterface, payment payment.Queries, cg checkgroup.Queries, cp checkpoint.Queries, checkpersonnel checkpersonnel.Queries, scan ScanInterface, lok LokInterface, sec section.Queries, crew crewmember.Queries, veh vehicle.Queries, ord OrderInterface, sosq sos.Queries, memberq spejderstatus.Queries, shelterq shelter.Queries, noteq spejdernote.Queries, dispatchq dispatch.Queries, kortq kort.Queries, trackq track.Queries, roster spejder.RosterReader) Models {
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
		Vehicle:        veh,
		Order:          ord,
		Sos:            sosq,
		Shelter:        shelterq,
		Note:           noteq,
		Dispatch:       dispatchq,
		Kort:           kortq,
		SpejderStatus:  memberq,
		Track:          trackq,
		Roster:         roster,
	}
}
