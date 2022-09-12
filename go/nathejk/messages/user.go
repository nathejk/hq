package messages

import "nathejk.dk/nathejk/types"

// nathejk:user.updated
type NathejkUserUpdated struct {
	UserID   types.UserID      `json:"userId"`
	Name     string            `json:"name"`
	Phone    types.PhoneNumber `json:"phone"`
	HqAccess bool              `json:"hqAccess"`
	Group    string            `json:"group"`
}

// nathejk:user.deleted
type NathejkUserDeleted struct {
	UserID types.UserID `json:"userId"`
}
