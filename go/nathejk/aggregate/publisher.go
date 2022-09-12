package aggregate

import (
	"log"

	"nathejk.dk/nathejk/messages"
	"nathejk.dk/pkg/streaminterface"
	"nathejk.dk/pkg/streaminterface/caughtup"

	"github.com/cnf/structhash"
)

type Aggregate interface {
	ID() string
	IsValid() bool
}

type Publisher struct {
	publisher streaminterface.Publisher
	channel   string
	checksums map[string]string
	published map[string]bool
}

func NewPublisher(publisher streaminterface.Publisher, channel string) *Publisher {
	return &Publisher{
		publisher: publisher,
		channel:   channel,
		checksums: make(map[string]string),
		published: make(map[string]bool),
	}
}

func (p *Publisher) Flush(aggregates []Aggregate) {
	for _, aggregate := range aggregates {
		if aggregate.IsValid() {
			p.Publish(aggregate)
		}
	}
	p.SendCaughtUp()
}

func (p *Publisher) SendCaughtUp() {
	msg := caughtup.NewCaughtupMessage(p.channel)
	p.publisher.Publish(msg)
	/*
		if err := eventstream.PublishCatchupEvent(p.publisher, p.channel); err != nil {
			panic(fmt.Sprintf("Error publishing catchup event: %s", err.Error()))
		}*/
}

func (p *Publisher) Publish(aggregate Aggregate) bool {
	id := aggregate.ID()
	checksum, _ := structhash.Hash(aggregate, 1)
	if checksum == p.checksums[id] {
		// Unchanged
		return false
	}
	p.checksums[id] = checksum

	if aggregate.IsValid() {
		msg := p.publisher.MessageFunc()(streaminterface.SubjectFromStr(p.channel + ":updated"))
		//msg := eventstream.NewMessage()
		//msg.Msg().Type = "updated"
		msg.SetBody(&aggregate)
		msg.SetMeta(messages.MetaID{ID: id})
		if err := p.publisher.Publish(msg); err != nil {
			log.Printf("Error updating %s: %s", p.channel, err.Error())
		} else {
			p.published[id] = true
			return true
		}
	} else if p.published[id] {
		msg := p.publisher.MessageFunc()(streaminterface.SubjectFromStr(p.channel + ":removed"))
		//msg := eventstream.NewMessage()
		//msg.Msg().Type = "removed"
		msg.SetBody(&aggregate)
		msg.SetMeta(messages.MetaID{ID: id})
		if err := p.publisher.Publish(msg); err != nil {
			log.Printf("Error removing %s: %s", p.channel, err.Error())
		} else {
			p.published[id] = false
			return true
		}
	}
	return false
}
