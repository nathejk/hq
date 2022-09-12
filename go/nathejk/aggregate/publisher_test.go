package aggregate_test

/*
func TestAggregateFlush(t *testing.T) {
	assert := assert.New(t)

	publisher := make(test.Publisher, 100)

	aggregates := make(map[string]*aggregatetest)

	aggregates["id-1"] = &aggregatetest{UID: "id-1", Name: "initial1", Valid: true}
	aggregates["id-2"] = &aggregatetest{UID: "id-2", Name: "initial2", Valid: true}
	aggregates["id-3"] = &aggregatetest{UID: "id-3", Name: "initial3", Valid: false}

	slice := aggregate.MapToAggregates(aggregates)

	ap := aggregate.NewPublisher(&publisher, "channel1")

	ap.Flush(slice)

	assert.Len(publisher, 3)
	msg, ok := test.PopMessage(publisher)
	assert.True(ok)
	assert.Equal("updated", msg.Msg().Type)

	msg, ok = test.PopMessage(publisher)
	assert.True(ok)
	assert.Equal("updated", msg.Msg().Type)

	msg, ok = test.PopMessage(publisher)
	assert.True(ok)
	assert.Equal("caughtup", msg.Msg().Type)
}

func TestAggregatePublish(t *testing.T) {
	assert := assert.New(t)

	publisher := make(test.Publisher, 100)

	ag := &aggregatetest{UID: "id-1", Name: "initial1", Valid: true}

	ap := aggregate.NewPublisher(&publisher, "channel1")

	ap.Publish(ag)
	msg, ok := test.PopMessage(publisher)
	assert.True(ok)
	assert.Equal("updated", msg.Msg().Type)

	ag.Name = "updated1"
	ap.Publish(ag)
	msg, ok = test.PopMessage(publisher)
	assert.True(ok)
	assert.Equal("updated", msg.Msg().Type)

	ag.Name = "updated1"
	ap.Publish(ag)
	assert.Len(publisher, 0)

	ag.Valid = false
	ap.Publish(ag)
	msg, ok = test.PopMessage(publisher)
	assert.True(ok)
	assert.Equal("removed", msg.Msg().Type)

	ag.Name = "updated2"
	ap.Publish(ag)
	assert.Len(publisher, 0)
}
*/
