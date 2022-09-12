package main

import "nathejk.dk/nathejk/types"

type SosCommander interface {
	Create(SosRequest, User) (SosResponse, error)
	UpdateHeadline(SosRequest, User) (SosResponse, error)
	UpdateDescription(SosRequest, User) (SosResponse, error)
	Close(SosRequest, User) (SosResponse, error)
	Reopen(SosRequest, User) (SosResponse, error)
	Delete(SosRequest, User) (SosResponse, error)

	AddComment(SosCommentRequest, User) (SosResponse, error)
	UpdateComment(SosCommentRequest, User) (SosResponse, error)

	AssociateTeam(SosTeamRequest, User) (SosResponse, error)
	DisassociateTeam(SosTeamRequest, User) (SosResponse, error)
	MergeTeams(SosTeamRequest, User) (SosResponse, error)
	SplitTeam(SosTeamRequest, User) (SosResponse, error)
	MemberStatusChange(SosMemberRequest, User) (SosResponse, error)
	SendPositionSms(SosMemberRequest, User) (SosResponse, error)

	SetSeverity(SosSeverityRequest, User) (SosResponse, error)

	Assign(SosAssignRequest, User) (SosResponse, error)
}

type SosResponse struct {
	OK        bool               `json:"ok"`
	SosID     types.SosID        `json:"sosId,omitempty"`
	CommentID types.SosCommentID `json:"commentId,omitempty"`
	TeamID    types.TeamID       `json:"teamId,omitempty"`
	MemberID  types.MemberID     `json:"memberId,omitempty"`
	Error     string             `json:"error,omitempty"`
}

type SosRequest struct {
	SosID       types.SosID `json:"sosId"`
	Headline    *string     `json:"headline"`
	Description *string     `json:"description"`
}

type SosCommentRequest struct {
	SosID     types.SosID        `json:"sosId"`
	CommentID types.SosCommentID `json:"commentId"`
	Comment   string             `json:"comment"`
}

type SosTeamRequest struct {
	SosID        types.SosID  `json:"sosId"`
	TeamID       types.TeamID `json:"teamId"`
	ParentTeamID types.TeamID `json:"parentTeamId,omitempty"`
}

type SosMemberRequest struct {
	SosID    types.SosID        `json:"sosId"`
	MemberID types.MemberID     `json:"memberId"`
	Status   types.MemberStatus `json:"status"`
}

type SosSeverityRequest struct {
	SosID    types.SosID `json:"sosId"`
	Severity types.Enum  `json:"severity"`
}

type SosAssignRequest struct {
	SosID    types.SosID `json:"sosId"`
	Assignee types.Enum  `json:"assignee"`
}
