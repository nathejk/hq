package data

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/nathejk/shared-go/types"
)

type TeamModel struct {
	DB *sql.DB
}

type Patrulje struct {
	ID     types.TeamID `json:"id"`
	Number string       `json:"number"`
	// There is deliberately no `Status` field. One existed, was never selected by
	// GetPatrulje, and therefore serialized as `"status": ""` on every patrol — which the
	// SPA read to decide whether the pre-race member move was still allowed, and so always
	// got "not started". SignupStatus below is the status; a second, empty one is a trap.
	Name        string `json:"name"`
	Group       string `json:"group"`
	Korps       string `json:"korps"`
	Liga        string `json:"liga"`
	MemberCount int    `json:"memberCount"`

	// ActiveMemberCount is the team's strength on the route: members still racing.
	// Maintained by the spejderstatus projection (PRD 006). Zero on a team that
	// **started** means discontinued — zero on one that never started means nothing
	// at all, so the two must not be conflated by callers.
	ActiveMemberCount int `json:"activeMemberCount"`

	// SignupStatus is carried so a caller can tell those two cases apart without a
	// second request.
	SignupStatus types.SignupStatus `json:"signupStatus"`

	// StartedUts is when the patrol went on the route (unix seconds), or 0 if it has
	// not started. The patrol trail shows it as the first event above the scans.
	StartedUts int64 `json:"startedUts"`

	// Remark is HQ's operational note for banditter and postmandskab, RemarkSeverity how
	// loudly to show it — `inactive` meaning filed but not in force, and "" that no note
	// was ever written. See nathejk/table/patrulje/messages.go.
	Remark         string `json:"remark"`
	RemarkSeverity string `json:"remarkSeverity"`

	// Fototilladelse — see nathejk/table/patrulje.PhotoConsentSet.
	PhotoRefusedTeam      bool             `json:"photoRefusedTeam"`
	PhotoRefusedMemberIDs []types.MemberID `json:"photoRefusedMemberIds"`
}
type Klan struct {
	ID          types.TeamID       `json:"id"`
	Status      types.SignupStatus `json:"status"`
	Name        string             `json:"name"`
	Group       string             `json:"group"`
	Korps       string             `json:"korps"`
	MemberCount int                `json:"memberCount"`
}
type Contact struct {
	TeamID     types.TeamID       `json:"teamId"`
	Name       string             `json:"name"`
	Address    string             `json:"address"`
	PostalCode string             `json:"postal"`
	Email      types.EmailAddress `json:"email"`
	Phone      types.PhoneNumber  `json:"phone"`
	Role       string             `json:"role"`
}

func (m TeamModel) RequestedSeniorCount() int {
	query := `SELECT COUNT(memberId) FROM senior WHERE year=%d`
	var count int
	_ = m.DB.QueryRow(query, 2024).Scan(&count)
	return count
}

func (m TeamModel) GetPatrulje(teamID types.TeamID) (*Patrulje, error) {
	if len(teamID) == 0 {
		return nil, ErrRecordNotFound
	}

	query := `SELECT p.teamId, p.teamNumber, p.name, p.groupName, p.korps, p.liga, p.memberCount,
			p.activeMemberCount, p.signupStatus, p.startedUts, p.remark, p.remarkSeverity,
			p.photoRefusedTeam, p.photoRefusedMembers
		FROM patrulje p
		JOIN patruljestatus ps ON p.teamId = ps.teamID
		WHERE p.teamId = ?`
	var p Patrulje
	var photoRefusedMembers string
	err := m.DB.QueryRow(query, teamID).Scan(
		&p.ID,
		&p.Number,
		&p.Name,
		&p.Group,
		&p.Korps,
		&p.Liga,
		&p.MemberCount,
		&p.ActiveMemberCount,
		&p.SignupStatus,
		&p.StartedUts,
		&p.Remark,
		&p.RemarkSeverity,
		&p.PhotoRefusedTeam,
		&photoRefusedMembers,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	// Comma-separated in the column (see nathejk/table/patrulje.JoinMemberIDs); never nil,
	// so the client always gets an array.
	p.PhotoRefusedMemberIDs = []types.MemberID{}
	for _, id := range strings.Split(photoRefusedMembers, ",") {
		if id != "" {
			p.PhotoRefusedMemberIDs = append(p.PhotoRefusedMemberIDs, types.MemberID(id))
		}
	}
	return &p, nil
}

func (m TeamModel) GetKlan(teamID types.TeamID) (*Klan, error) {
	if len(teamID) == 0 {
		return nil, ErrRecordNotFound
	}

	query := `SELECT t.teamId, t.name, t.groupName, t.korps, t.memberCount, t.signupStatus
		FROM klan t
		JOIN patruljestatus ts ON t.teamId = ts.teamID
		WHERE t.teamId = ?`
	var t Klan
	err := m.DB.QueryRow(query, teamID).Scan(
		&t.ID,
		&t.Name,
		&t.Group,
		&t.Korps,
		&t.MemberCount,
		&t.Status,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	return &t, nil
}

func (m TeamModel) GetContact(teamID types.TeamID) (*Contact, error) {
	if len(teamID) == 0 {
		return nil, ErrRecordNotFound
	}

	query := `SELECT p.contactName, p.contactPhone, p.contactEmail, p.contactRole
		FROM patrulje p
		JOIN patruljestatus ps ON p.teamId = ps.teamID
		WHERE p.teamId = ?`
	c := Contact{TeamID: teamID}
	err := m.DB.QueryRow(query, teamID).Scan(
		&c.Name,
		&c.Phone,
		&c.Email,
		&c.Role,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	return &c, nil
}
