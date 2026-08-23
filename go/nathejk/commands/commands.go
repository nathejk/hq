package commands

import (
	"github.com/jrgensen/stream"
	"github.com/nathejk/shared-go/tables/crewmember"
	"github.com/nathejk/shared-go/tables/section"
	"github.com/nathejk/shared-go/tables/vehicle"
	"github.com/nathejk/shared-go/types"
	"nathejk.dk/internal/data"
	"nathejk.dk/nathejk/table/checkgroup"
	"nathejk.dk/nathejk/table/checkpersonnel"
	"nathejk.dk/nathejk/table/checkpoint"
	"nathejk.dk/nathejk/table/shelter"
	"nathejk.dk/nathejk/table/sos"
	"nathejk.dk/nathejk/table/spejderstatus"
	"nathejk.dk/nathejk/table/year"
)

type Commands struct {
	Year           year.Commands
	Checkgroup     checkgroup.Commands
	Checkpoint     checkpoint.Commands
	Checkpersonnel checkpersonnel.Commands
	Section        section.Commands
	CrewMember     crewmember.Commands
	Vehicle        vehicle.Commands
	Sos            sos.Commands
	Member         spejderstatus.Commands

	// Shelter is the placering's write side, separate from Member because a command belongs
	// with the read model it dirty-checks against — and because spejderstatus cannot import
	// the shelter table (PRD 007; see shelter/commands.go).
	Shelter shelter.Commands

	Team interface {
		UpdatePatrulje(types.TeamID, Patrulje, Contact, []Spejder) error
		StartPatrulje(types.TeamID, []StartPatruljeMember) error
		UpdateKlan(types.TeamID, Klan, []Senior) error
		AssignToLok(types.TeamID, string) error
	}
	Lok interface {
		UpdateLok(types.LokID, string, int, []types.UserID, []types.TeamID) error
		DeleteLok(types.LokID) error
		UpdateUser(types.UserID, string) error
		UpdateMember(types.MemberID, string) error
	}
}

func New(stream stream.Publisher, models data.Models) Commands {
	return Commands{
		Team: NewTeam(stream, models.Teams),
		Lok:  NewLok(stream, models.Lok),
	}
}
