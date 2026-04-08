package requestctx

import (
	"context"

	"github.com/nathejk/shared-go/types"
)

type valueKeyType struct{} // unexported

var valueKey = valueKeyType{}

type Value struct {
	Service       string
	Version       string
	Year          types.YearSlug
	RequestID     string
	CausationID   string
	CorrelationID string
}

func WithValue(ctx context.Context, v *Value) context.Context {
	return context.WithValue(ctx, valueKey, v)
}

func ValueFrom(ctx context.Context) (*Value, bool) {
	v, ok := ctx.Value(valueKey).(*Value)
	return v, ok
}
