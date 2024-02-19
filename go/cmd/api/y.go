package main

import (
	"log"

	"github.com/davecgh/go-spew/spew"
	"nathejk.dk/superfluids/streaminterface"
)

type y struct {
}

func (yy *y) Consumes() []streaminterface.Subject {
	return []streaminterface.Subject{
		streaminterface.SubjectFromStr("NATHEJK:*"),
	}
}

func (yy *y) HandleMessage(msg streaminterface.Message) error {
	log.Printf("Message RECEIVED")
	spew.Dump(msg)
	return nil
}
