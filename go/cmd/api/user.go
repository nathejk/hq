package main

import (
	"net/http"

	"nathejk.dk/nathejk/types"
)

type User interface {
	ID() types.UserID
}

/********
 * Mock implementation
 */
type uo struct {
	userID types.UserID
}

func NewUserFromRequest(r *http.Request) *uo {

	return &uo{}
}

func (u *uo) ID() types.UserID {
	return u.userID
}
