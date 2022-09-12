package eventstream_test

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"nathejk.dk/nathejk/eventstream"
)

func TestCanInvokeHandler(t *testing.T) {
	assert := assert.New(t)

	stream := eventstream.NewMemoryStream()
	stream.Publish("channel", eventstream.NewMessage())
	stream.Publish("channel", eventstream.NewMessage())
	stream.Publish("channel", eventstream.NewMessage())
	stream.Publish("channel", eventstream.NewMessage())
	stream.Publish("channel", eventstream.NewMessage())
	events, err := stream.Subscribe("channel", 1, 5)
	assert.Nil(err)

	mutex := &sync.Mutex{}
	result := ""

	handlers := []eventstream.EventHandler{
		&eventstream.DebugHandler{Mod: 1, Prefix: "chandler", Delay: time.Duration(1) * time.Millisecond},
		eventstream.HandlerFunc(func(msg eventstream.Message) {
			if msg.Sequence == 3 {
				time.Sleep(200 * time.Millisecond)
			}
			mutex.Lock()
			defer mutex.Unlock()
			result = result + "a"
		}),
		eventstream.HandlerFunc(func(msg eventstream.Message) {
			if msg.Sequence != 3 {
				time.Sleep(200 * time.Millisecond)
			}
			mutex.Lock()
			defer mutex.Unlock()
			result = result + "b"
		}),
	}
	<-eventstream.HandleEvents(events, handlers...)
	assert.Equal("ababababab", result, "Handlers not invoked serialized")
}

func TestCanInvokeConcurrentHandler(t *testing.T) {
	assert := assert.New(t)

	stream := eventstream.NewMemoryStream()
	stream.Publish("channel", eventstream.NewMessage())
	stream.Publish("channel", eventstream.NewMessage())
	stream.Publish("channel", eventstream.NewMessage())
	stream.Publish("channel", eventstream.NewMessage())
	stream.Publish("channel", eventstream.NewMessage())
	events, err := stream.Subscribe("channel", 1, 5)
	assert.Nil(err)

	mutex := &sync.Mutex{}
	result := ""

	handlers := []eventstream.EventHandler{
		&eventstream.DebugHandler{Mod: 1, Prefix: "chandler", Delay: time.Duration(1) * time.Millisecond},
		eventstream.HandlerFunc(func(msg eventstream.Message) {
			if msg.Sequence == 3 {
				time.Sleep(200 * time.Millisecond)
			}
			mutex.Lock()
			defer mutex.Unlock()
			result = result + "a"
		}),
		eventstream.HandlerFunc(func(msg eventstream.Message) {
			if msg.Sequence != 3 {
				time.Sleep(200 * time.Millisecond)
			}
			mutex.Lock()
			defer mutex.Unlock()
			result = result + "b"
		}),
	}
	<-eventstream.HandleEvents(events, eventstream.ConcurrentHandler(handlers...))
	assert.Equal("ababbaabab", result, "Handlers not invoked concurrently")
}
