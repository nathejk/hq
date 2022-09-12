package aggregate_test

import (
	"nathejk.dk/nathejk/aggregate"

	"testing"

	"github.com/stretchr/testify/assert"
)

type aggregatetest struct {
	UID   string
	Name  string
	Valid bool
}

func (a *aggregatetest) ID() string {
	return a.UID
}

func (a *aggregatetest) IsValid() bool {
	return a.Valid
}

func TestMapToAggregates(t *testing.T) {
	assert := assert.New(t)

	//publisher := test.Publisher(make(chan eventstream.Message, 100))

	aggregates := make(map[string]*aggregatetest)

	aggregates["id-1"] = &aggregatetest{UID: "id-1", Name: "initial1", Valid: true}
	aggregates["id-2"] = &aggregatetest{UID: "id-2", Name: "initial2", Valid: true}
	aggregates["id-3"] = &aggregatetest{UID: "id-3", Name: "initial3", Valid: false}

	slice := aggregate.MapToAggregates(aggregates)

	assert.ElementsMatch([]aggregate.Aggregate{aggregates["id-1"], aggregates["id-2"], aggregates["id-3"]}, slice)
}
