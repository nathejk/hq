package persister

import (
	"log"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/davecgh/go-spew/spew"
	"github.com/globalsign/mgo"

	"nathejk.dk/pkg/job"
	"nathejk.dk/pkg/streaminterface"
)

type mongoHandler struct {
	Collection *mgo.Collection
	Jobs       *job.Runner
	cnt        int64
}

// EventHandler.Handle()
// func (a *mongoHandler) Handle(msg eventstream.Message) {
func (a *mongoHandler) HandleMessage(msg streaminterface.Message) error {
	/*msg, ok := i.(aggregate.Message)
	if !ok {
		return
	}*/
	var meta struct {
		ID string `json:"id"`
	}
	msg.Meta(&meta)
	id := meta.ID
	if id == "" {
		spew.Dump(msg)
		log.Fatalf("NO ID!")
		return nil
	}

	channeltype := strings.Split(msg.Subject().Subject(), ":")
	switch channeltype[1] {

	case "updated":
		//channel := msg.Channel
		model := a
		a.Jobs.Add(id, func() {
			var doc interface{}
			msg.Body(&doc)
			model.Collection.UpsertId(id, doc)
			cnt := atomic.AddInt64(&model.cnt, 1)
			if cnt%1000 == 0 {
				log.Printf("[%s] written %d - %d skipped - %d remaining\n", channeltype[0], model.cnt, model.Jobs.Skipped(), model.Jobs.Remaining())
			}
		})

	case "removed":
		//channel := msg.Channel
		model := a
		a.Jobs.Add(id, func() {
			model.Collection.RemoveId(id)
			cnt := atomic.AddInt64(&model.cnt, 1)
			if cnt%1000 == 0 {
				log.Printf("[%s] written %d - %d skipped - %d remaining\n", channeltype[0], model.cnt, model.Jobs.Skipped(), model.Jobs.Remaining())
			}
		})

	}
	return nil
}

type mongo struct {
	//sync.Mutex
	//database    *mgo.Database
	collections map[string]string
	handlers    map[string]streaminterface.MessageHandler
	wg          *sync.WaitGroup
}

func NewMongoPersister(db *mgo.Database, collections map[string]string) *mongo {
	//func (a *Mongo) Initialize() {
	a := &mongo{
		collections: collections,
		handlers:    make(map[string]streaminterface.MessageHandler),
		wg:          &sync.WaitGroup{},
	}
	a.wg.Add(len(collections))
	for channel, collection := range collections {
		runner := job.NewRunner()
		go runner.Run()
		handler := &mongoHandler{Collection: db.C(collection), Jobs: runner}
		a.handlers[channel] = handler
		go func(channel string, m *mongoHandler) {
			m.Jobs.Run()
			//log.Printf("%s Jobs are done", channel)
			log.Printf("[%s] written %d - %d skipped\n", channel, atomic.LoadInt64(&m.cnt), m.Jobs.Skipped())
			a.wg.Done()
			m.Jobs.Run()
		}(channel, handler)
	}
	return a
}

func (a *mongo) CaughtUp() {
	for _, persister := range a.handlers {
		//log.Printf("%s finishing jobs", channel)
		persister.(*mongoHandler).Jobs.Done()
	}

	a.wg.Wait()
}

func (a *mongo) Consumes() (channels []streaminterface.Subject) {
	for channel := range a.collections {
		channels = append(channels, streaminterface.SubjectFromStr(channel+":updated"))
		channels = append(channels, streaminterface.SubjectFromStr(channel+":removed"))
	}
	return
}

func (a *mongo) HandleMessage(msg streaminterface.Message) {
	/*msg, ok := i.(aggregate.Message)
	if !ok {
		return
	}*/
	channeltype := strings.Split(msg.Subject().Subject(), ":")
	a.handlers[channeltype[0]].HandleMessage(msg)
}
