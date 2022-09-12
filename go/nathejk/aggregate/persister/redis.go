package persister

import (
	"encoding/json"
	"log"
	"strings"
	"sync/atomic"

	"nathejk.dk/pkg/job"
	"nathejk.dk/pkg/streaminterface"

	"github.com/davecgh/go-spew/spew"
	"github.com/go-redis/redis"
)

type redisPersister struct {
	client      *redis.Client
	collections map[string]string
	jobs        *job.Runner
	cnt         int64
}

func NewRedisPersister(client *redis.Client, collections map[string]string) *redisPersister {
	m := redisPersister{
		client:      client,
		collections: collections,
		jobs:        job.NewRunner(),
	}
	go m.jobs.Run()
	/*
		handler := &mongoHandler{Collection: db.C(collection), Jobs: runner}
		a.handlers[channel] = handler
		go func(channel string, m *mongoHandler) {
			m.jobs.Run()
			//log.Printf("%s Jobs are done", channel)
			log.Printf("[%s] written %d - %d skipped\n", channel, atomic.LoadInt64(&m.cnt), m.Jobs.Skipped())
			m.Jobs.Run()
		}(channel, handler)
	*/
	return &m
}

func (m *redisPersister) Consumes() (channels []streaminterface.Subject) {
	for channel := range m.collections {
		channels = append(channels, streaminterface.SubjectFromStr(channel+":updated"))
		channels = append(channels, streaminterface.SubjectFromStr(channel+":removed"))
	}
	return
}

func (m *redisPersister) HandleMessage(msg streaminterface.Message) error {
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
	key := m.collections[channeltype[0]]
	switch channeltype[1] {

	case "updated":
		//channel := msg.Channel
		//model := m
		log.Printf("Job added %q", meta.ID)
		m.jobs.Add(meta.ID, func() {
			var doc interface{}
			msg.Body(&doc)
			value, _ := json.Marshal(doc)
			_, err := m.client.HSet(key, meta.ID, value).Result()
			if err != nil && err != redis.Nil {
				panic(err)
			}

			cnt := atomic.AddInt64(&m.cnt, 1)
			if cnt%1000 == 0 {
				log.Printf("[%s] written %d - %d skipped - %d remaining\n", key, m.cnt, m.jobs.Skipped(), m.jobs.Remaining())
			}
		})

	case "removed":
		//channel := msg.Channel
		//model := a
		m.jobs.Add(meta.ID, func() {
			m.client.HDel(key, meta.ID).Result()

			cnt := atomic.AddInt64(&m.cnt, 1)
			if cnt%1000 == 0 {
				log.Printf("[%s] written %d - %d skipped - %d remaining\n", key, m.cnt, m.jobs.Skipped(), m.jobs.Remaining())
			}
		})

	}
	return nil
}

/*
func (a *mongo) Handle(i interface{}) {
	msg, ok := i.(aggregate.Message)
	if !ok {
		return
	}
	channeltype := strings.Split(msg.ChannelType(), ":")
	a.handlers[channeltype[0]].Handle(msg)
}

func (s *redisPersister) Add(key, hash, value string) {
	_, err := s.client.HSet(key, hash, value).Result()
	if err != nil && err != redis.Nil {
		panic(err)
	}
	/*
		if err != nil {
			log.Printf("Success: %b [%s, %s, %s] %s", success, key, hash, value, err.Error())
		} else {
			log.Printf("Success: %b [%s, %s, %s]", success, key, hash, value)
		}*
}
func (s *redisPersister) GetOne(key, hash string) string {
	one, err := s.client.HGet(key, hash).Result()
	if err != nil && err != redis.Nil {
		panic(err)
	}
	return one
}

func (s *redisPersister) Remove(key, hash string) {
	s.client.HDel(key, hash).Result()
}

func (s *redisPersister) GetAll(key string) map[string]string {
	all, err := s.client.HGetAll(key).Result()
	if err != nil {
		panic(err)
	}
	return all
}

func (s *redisPersister) Set(key, value string) error {
	duration := time.Duration(1000 * time.Hour)
	_, err := s.client.Set(key, value, duration).Result()
	return err
}
func (s *redisPersister) Get(key string) (string, error) {
	return s.client.Get(key).Result()
}
*/
