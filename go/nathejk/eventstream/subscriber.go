package eventstream

const STOP_NEVER = 0

type Subscriber interface {
	Subscribe(channel string, startAt int64, stopAt int64) (chan Message, error)
	LastSequence(channel string) int64
}

func Catchup(subscriber Subscriber, channel string, startAt int64) (chan Message, int64, error) {
	lastSequence := subscriber.LastSequence(channel)
	if startAt > lastSequence {
		nonevents := make(chan Message)
		close(nonevents)
		return nonevents, lastSequence, nil
	}
	events, err := subscriber.Subscribe(channel, startAt, lastSequence)
	return events, lastSequence, err
}

func Live(subscriber Subscriber, channel string, startAt int64) (chan Message, error) {
	return subscriber.Subscribe(channel, startAt, STOP_NEVER)
}

func MustCatchup(subscriber Subscriber, channel string, startAt int64) (chan Message, int64) {
	events, lastSequence, err := Catchup(subscriber, channel, startAt)
	if err != nil {
		panic(err)
	}
	return events, lastSequence
}

func MustLive(subscriber Subscriber, channel string, startAt int64) chan Message {
	events, err := Live(subscriber, channel, startAt)
	if err != nil {
		panic(err)
	}
	return events
}
