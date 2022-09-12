package eventstream_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"nathejk.dk/nathejk/eventstream"
)

func TestCatchup(t *testing.T) {
	assert := assert.New(t)

	stream := eventstream.NewMemoryStream()
	stream.Publish("channel", eventstream.NewMessage())
	stream.Publish("channel", eventstream.NewMessage())
	stream.Publish("channel", eventstream.NewMessage())
	stream.Publish("channel", eventstream.NewMessage())
	stream.Publish("channel", eventstream.NewMessage())
	events, lastSequence, err := eventstream.Catchup(stream, "channel", 1)
	assert.Nil(err)

	assert.Equal(lastSequence, stream.LastSequence("channel"), "last sequence doesn't match")
	stream.Publish("channel", eventstream.NewMessage())
	assert.Equal(lastSequence+1, stream.LastSequence("channel"), "last sequence doesn't match")

	var ok bool
	<-events
	<-events
	<-events
	<-events
	_, ok = <-events
	assert.True(ok, "No more events?")
	_, ok = <-events
	assert.False(ok, "Still more events?")
}

func TestCatchupEmptyChannel(t *testing.T) {
	assert := assert.New(t)

	stream := eventstream.NewMemoryStream()
	events, lastSequence, err := eventstream.Catchup(stream, "channel", 1)
	assert.Nil(err)
	assert.Zero(lastSequence)
	var ok bool
	_, ok = <-events
	assert.False(ok, "Still more events?")
}

func TestCatchupOutOfBounds(t *testing.T) {
	assert := assert.New(t)

	stream := eventstream.NewMemoryStream()
	stream.Publish("channel", eventstream.NewMessage())
	stream.Publish("channel", eventstream.NewMessage())
	events, lastSequence, err := eventstream.Catchup(stream, "channel", 100)
	assert.Nil(err)
	assert.Equal(int64(2), lastSequence)
	var ok bool
	_, ok = <-events
	assert.False(ok, "Still more events?")
}

func TestLive(t *testing.T) {
	assert := assert.New(t)

	stream := eventstream.NewMemoryStream()
	stream.Publish("channel", eventstream.NewMessage())
	stream.Publish("channel", eventstream.NewMessage())
	stream.Publish("channel", eventstream.NewMessage())
	stream.Publish("channel", eventstream.NewMessage())
	stream.Publish("channel", eventstream.NewMessage())
	events, err := eventstream.Live(stream, "channel", 5)
	assert.Nil(err)

	var ok bool

	_, ok = <-events
	assert.True(ok, "No more events?")
	stream.Publish("channel", eventstream.NewMessage())
	_, ok = <-events
	assert.True(ok, "No more events?")
}
