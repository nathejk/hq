# Introduction

The aggregate package provides a "publisher". It's not a publisher in the eventstream sense, but a publisher that
publishes updated/removed events when necessary, and it is also capable of producing a caughtup event if necessary.

An aggregate publisher publishes "aggregates" which much adhere to the Aggregate interface.

```
type Aggregate interface {
	ID() string         // The ID of the aggregate
	IsValid() bool      // Whether or not the aggregate is valid. In turn produces either an updated or removed event.
}
```

The aggregate publisher is especially useful for postponing transmitting events until an entire stream
is caught up, thereby eliminating redundant transmissions.

The aggregate publisher also keeps a checksum for each aggregate, and only sends events on aggregates that have changed.

# Usage

```

stream := bufferstream.New(1000)

// Create an aggregate publisher instance that produces the events:
//  aggregate.name:updated
//  aggregate.name:removed
//  aggregate.name:caughtup
aggpub := aggregate.NewPublisher(stream, "aggregate.name")

type myaggregate struct {
    id string
    name string
}

func (a *myaggregate) ID() string {
    return a.id
}

func (a *myaggregate) IsValid() bool {
    return a.name != ""
}

var aggs map[string]*myaggregate
aggs["id1"] = &myaggregate{id: "id1", name: "name1"}
aggs["id2"] = &myaggregate{id: "id2", name: "name2"}

agg.Flush(MapToAggregates(aggs))
// Producues events
//  aggregate.name:updated{id:"id1", name:"name1"}
//  aggregate.name:updated{id:"id2", name:"name2"}
//  aggregate.name:caughtup{}


aggs["id2"].name = ""
agg.Publish(aggs["id2"])
// Producues events
//  aggregate.name:removed{id:"id2", name:""}
```
