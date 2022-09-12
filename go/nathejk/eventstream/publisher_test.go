package eventstream_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"nathejk.dk/nathejk/eventstream"
)

type TestPublisher struct {
	Published []eventstream.Message
}

func (t *TestPublisher) Publish(channel string, msg eventstream.Message) error {
	msg.Sequence = int64(len(t.Published) + 1)
	t.Published = append(t.Published, msg)
	return nil
}

func TestCanPublish(t *testing.T) {
	assert := assert.New(t)

	publisher := &TestPublisher{}
	for i := 0; i < 100; i++ {
		err := publisher.Publish("test", eventstream.NewMessage())
		assert.Nil(err)
	}
}

func TestCanPublishFaultTolerantly(t *testing.T) {
	assert := assert.New(t)

	publisher := &TestPublisher{}

	cnt := 0
	fpublisher := eventstream.PublisherFunc(func(channel string, msg eventstream.Message) error {
		cnt++
		if cnt == 1 {
			return errors.New("Could not publish event")
		}
		return publisher.Publish(channel, msg)
	})
	ftpublisher := eventstream.FaultTolerantPublisher(fpublisher, 3)

	err := ftpublisher.Publish("test", eventstream.NewMessage())
	assert.Nil(err)

}

func TestCanPublishFaultIntolerantly(t *testing.T) {
	assert := assert.New(t)

	ferr := errors.New("Could not publish event")
	fpublisher := eventstream.PublisherFunc(func(channel string, msg eventstream.Message) error {
		return ferr
	})
	ftpublisher := eventstream.FaultTolerantPublisher(fpublisher, 3)

	err := ftpublisher.Publish("test", eventstream.NewMessage())
	assert.Equal(ferr, err, "Wrong or no error produced")
}
