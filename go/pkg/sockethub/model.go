package sockethub

import (
	"log"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"nathejk.dk/pkg/streaminterface"
)

type Updater interface {
	Update(string, string, interface{})
}

type websyncModel struct {
	updater     Updater
	collections map[string]string
}

func NewWebsyncModel(updater Updater, collections map[string]string) *websyncModel {
	m := websyncModel{
		updater:     updater,
		collections: collections,
	}
	return &m
}

func (m *websyncModel) Consumes() (channels []streaminterface.Subject) {
	for channel := range m.collections {
		channels = append(channels, streaminterface.SubjectFromStr(channel+":updated"))
		channels = append(channels, streaminterface.SubjectFromStr(channel+":removed"))
	}
	return
}

func (m *websyncModel) HandleMessage(msg streaminterface.Message) error {
	/*msg, ok := i.(eventstream.Message)
	if !ok {
		log.Printf("NOT OK")
		return
	}*/
	var meta struct {
		ID string `json:"id"`
	}
	msg.Meta(&meta)
	if meta.ID == "" {
		spew.Dump(msg)
		log.Fatalf("NO ID!")
		return nil
	}

	channeltype := strings.Split(msg.Subject().Subject(), ":")
	view := m.collections[channeltype[0]]
	switch channeltype[1] {

	case "updated":
		var body interface{}
		msg.Body(&body)
		m.updater.Update(view, meta.ID, body)
	case "removed":
		m.updater.Update(view, meta.ID, nil)

	}
	return nil
}
