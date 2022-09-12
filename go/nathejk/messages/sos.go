package messages

import (
	"nathejk.dk/nathejk/types"
)

type NathejkSosMetadata struct {
	Metadata

	SosID  types.SosID  `json:"sosId"`
	UserID types.UserID `json:"userId"`
}

// nathejk:sos.created
type NathejkSosCreated struct {
	SosID       types.SosID  `json:"sosId"`
	UserID      types.UserID `json:"userId"`
	Headline    string       `json:"headline"`
	Description string       `json:"description"`
}

// nathejk:sos.headline.updated
type NathejkSosHeadlineUpdated struct {
	SosID    types.SosID  `json:"sosId"`
	UserID   types.UserID `json:"userId"`
	Headline string       `json:"headline"`
}

// nathejk:sos.description.updated
type NathejkSosDescriptionUpdated struct {
	SosID       types.SosID  `json:"sosId"`
	UserID      types.UserID `json:"userId"`
	Description string       `json:"description"`
}

// nathejk:sos.commented
type NathejkSosCommented struct {
	SosID     types.SosID        `json:"sosId"`
	UserID    types.UserID       `json:"userId"`
	CommentID types.SosCommentID `json:"commentId"`
	Comment   string             `json:"comment"`
}

// nathejk:sos.comment.updated
type NathejkSosCommentUpdated struct {
	SosID     types.SosID        `json:"sosId"`
	UserID    types.UserID       `json:"userId"`
	CommentID types.SosCommentID `json:"commentId"`
	Comment   string             `json:"comment"`
}

// nathejk:sos.severity.specified
type NathejkSosSeveritySpecified struct {
	SosID    types.SosID  `json:"sosId"`
	UserID   types.UserID `json:"userId"`
	Severity types.Enum   `json:"severity"`
}

// nathejk:sos.assigned
type NathejkSosAssigned struct {
	SosID    types.SosID  `json:"sosId"`
	UserID   types.UserID `json:"userId"`
	Assignee types.Enum   `json:"assignee"`
}

// nathejk:sos.deleted
type NathejkSosDeleted struct {
	SosID  types.SosID  `json:"sosId"`
	UserID types.UserID `json:"userId"`
}

// nathejk:sos.closed
type NathejkSosClosed struct {
	SosID  types.SosID  `json:"sosId"`
	UserID types.UserID `json:"userId"`
}

// nathejk:sos.reopened
type NathejkSosReopened struct {
	SosID  types.SosID  `json:"sosId"`
	UserID types.UserID `json:"userId"`
}

// nathejk:sos.team.associated
type NathejkSosTeamAssociated struct {
	SosID  types.SosID  `json:"sosId"`
	UserID types.UserID `json:"userId"`
	TeamID types.TeamID `json:"teamId"`
}

// nathejk:sos.team.disassociated
type NathejkSosTeamDisassociated struct {
	SosID  types.SosID  `json:"sosId"`
	UserID types.UserID `json:"userId"`
	TeamID types.TeamID `json:"teamId"`
}
