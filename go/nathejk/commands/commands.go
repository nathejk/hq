package commands

import (
	"github.com/nathejk/shared-go/messages"
	"github.com/nathejk/shared-go/types"
	"nathejk.dk/internal/data"
	"nathejk.dk/nathejk/table/checkgroup"
	"nathejk.dk/nathejk/table/checkpersonnel"
	"nathejk.dk/nathejk/table/checkpoint"
	"nathejk.dk/nathejk/table/year"
	"nathejk.dk/superfluids/streaminterface"
)

type Commands struct {
	Year           year.Commands
	Checkgroup     checkgroup.Commands
	Checkpoint     checkpoint.Commands
	Checkpersonnel checkpersonnel.Commands

	Team interface {
		Signup(types.TeamType, *messages.NathejkTeamSignedUp) error
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

func New(stream streaminterface.Publisher, models data.Models) Commands {
	return Commands{
		//Checkgroup: NewCheckgroup(stream, models.Checkgroup),
		//Checkpoint: models.Checkpoint,
		Team: NewTeam(stream, models.Teams),
		Lok:  NewLok(stream, models.Lok),
	}
}
