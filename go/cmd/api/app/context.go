package app

import (
	"context"
	"net/http"
)

// It’s good practice to use your own custom type for the request context keys. This helps
// prevent naming collisions between this code and any third-party packages which are also
// using the request context to store information.
type contextKey string

const userContextKey = contextKey("user")

// The contextSetUser() method returns a new copy of the request with the provided
// User struct added to the context.
func (app *JsonApi) contextSetUser(r *http.Request, user User) *http.Request {
	ctx := context.WithValue(r.Context(), userContextKey, user)
	return r.WithContext(ctx)
}

// The contextGetUser() retrieves the User struct from the request context.
func (app *JsonApi) contextGetUser(r *http.Request) User {
	user, ok := r.Context().Value(userContextKey).(User)
	if !ok {
		panic("missing user value in request context")
	}
	return user
}
