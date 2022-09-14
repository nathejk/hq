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

// nathejk:personnel.updated
type NathejkPersonnelUpdated struct {
	UserID     types.UserID      `json:"userId"`
	Name       string            `json:"name"`
	Phone      types.PhoneNumber `json:"phone"`
	Email      types.Email       `json:"email"`
	HqAccess   bool              `json:"hqAccess"`
	Department string            `json:"department"`
	MedlemNr   string            `json:"medlemnr"`
	Corps      types.CorpsSlug   `json:"corps"`
}

// nathejk:personnel.deleted
type NathejkPersonnelDeleted struct {
	UserID types.UserID `json:"userId"`
}
