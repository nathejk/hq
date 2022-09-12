package eventstream

import (
	"log"
)

type Publisher interface {
	Publish(channel string, msg Message) error
}

type PublisherFuncWrapper struct {
	publisher func(channel string, msg Message) error
}

func PublisherFunc(publisher func(channel string, msg Message) error) Publisher {
	return &PublisherFuncWrapper{publisher: publisher}
}

func (h *PublisherFuncWrapper) Publish(channel string, msg Message) error {
	return h.publisher(channel, msg)
}

// Wrap a publisher in retry attempts
func FaultTolerantPublisher(publisher Publisher, retries int) Publisher {
	return PublisherFunc(func(channel string, msg Message) error {
		var err error
		for retry := 0; retry < retries; retry++ {
			err = publisher.Publish(channel, msg)
			if err == nil {
				break
			}
			log.Printf("Failed (%d): %#v", retry, err)
		}
		return err
	})
}
