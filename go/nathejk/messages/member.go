package messages

import "nathejk.dk/nathejk/types"

// nathejk:member.updated
type NathejkMemberUpdated struct {
	MemberID    types.MemberID    `json:"memberId"`
	TeamID      types.TeamID      `json:"teamId"`
	Name        string            `json:"name"`
	Address     string            `json:"address"`
	PostalCode  string            `json:"postalCode"`
	City        string            `json:"city"`
	Phone       types.PhoneNumber `json:"phone"`
	PhoneParent types.PhoneNumber `json:"phoneParent"`
	Email       types.Email       `json:"mail"`
	Birthday    string            `json:"birthday"`
	Returning   bool              `json:"returning"`
}

// nathejk:member.deleted
type NathejkMemberDeleted struct {
	MemberID types.MemberID `json:"memberId"`
	TeamID   types.TeamID   `json:"teamId"`
}

// nathejk:member.status.changed
type NathejkMemberStatusChanged struct {
	MemberID types.MemberID     `json:"memberId"`
	Status   types.MemberStatus `json:"status"`
}

// nathejk:member.positionsms.sent
// nathejk:member.positionsms.failed
type NathejkMemberPositionSmsSent struct {
	SMSID    string            `json:"smsId"`
	MemberID types.MemberID    `json:"memberId"`
	SosID    types.SosID       `json:"sosId,omitempty"`
	Phone    types.PhoneNumber `json:"phone"`
	Text     string            `json:"text"`
	Error    string            `json:"error,omitempty"`
}
