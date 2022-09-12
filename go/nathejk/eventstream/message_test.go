package eventstream_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"nathejk.dk/nathejk/eventstream"
)

func TestCanCreateNewMessageWithBodyAndMeta(t *testing.T) {
	assert := assert.New(t)

	msg := eventstream.NewMessage()
	assert.NotEqual("", msg.EventId, "Event id should not be empty")
	assert.Equal(msg.EventId, msg.CausationId, "Causation id should match event id")
	assert.Equal(msg.EventId, msg.CorrelationId, "Correlation id should match event id")

	var err error

	type TestBody struct {
		Hello string `json:"hello"`
	}
	body := TestBody{"world"}
	err = msg.SetBody(body)
	assert.Nil(err)
	assert.Equal("{\"hello\":\"world\"}", string(msg.Body))

	type TestMeta struct {
		Goodbye string `json:"goodbye"`
	}
	meta := TestMeta{"universe"}
	err = msg.SetMeta(meta)
	assert.Nil(err)
	assert.Equal("{\"goodbye\":\"universe\"}", string(msg.Meta))

	err = msg.SetBody("HELLO")
	assert.Nil(err)
	assert.Equal("\"HELLO\"", string(msg.Body))

	err = msg.SetMeta("WORLD")
	assert.Nil(err)
	assert.Equal("\"WORLD\"", string(msg.Meta))
}

func TestInvalidBodyAndMeta(t *testing.T) {
	assert := assert.New(t)

	msg := eventstream.NewMessage()

	var err error

	err = msg.SetBody(make(chan int))
	assert.NotNil(err)

	err = msg.SetMeta(make(chan int))
	assert.NotNil(err)
}
