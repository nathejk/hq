package aggregate

import (
	"fmt"
	"reflect"
)

func MapToAggregates(aggregateMap interface{}) (aggregates []Aggregate) {
	val := reflect.ValueOf(aggregateMap)
	if val.Kind() != reflect.Map {
		panic(fmt.Sprintf("Not a map: %#v", aggregateMap))
	}

	for _, e := range val.MapKeys() {
		v := val.MapIndex(e).Interface()

		if a, ok := v.(Aggregate); ok {
			aggregates = append(aggregates, a)
		} else {
			panic(fmt.Sprintf("Not an aggregate: %#v", v))
		}
	}
	return
}
