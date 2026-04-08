package requestctx

import (
	"context"

	"github.com/nathejk/shared-go/types"
)

type userKeyType struct{} // unexported

var userKey = userKeyType{}

type User struct {
	ID   types.UserID
	Name string
}

func WithUser(ctx context.Context, u *User) context.Context {
	return context.WithValue(ctx, userKey, u)
}

func UserFrom(ctx context.Context) (*User, bool) {
	u, ok := ctx.Value(userKey).(*User)
	return u, ok
}
