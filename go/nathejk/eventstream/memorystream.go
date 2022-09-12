package eventstream

import (
	"runtime"
	"sync"
)

type MemoryStream struct {
	events map[string][]Message
	mutex  *sync.Mutex
}

func NewMemoryStream() *MemoryStream {
	return &MemoryStream{
		events: make(map[string][]Message),
		mutex:  &sync.Mutex{},
	}
}

func (s *MemoryStream) LastSequence(channel string) int64 {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return int64(len(s.events[channel]))
}

func (s *MemoryStream) waitUntilSequence(channel string, sequence int64) {
	for {
		if sequence <= s.LastSequence(channel) {
			return
		}
		runtime.Gosched()
	}
}

func (s *MemoryStream) Subscribe(channel string, startAt int64, stopAt int64) (chan Message, error) {
	events := make(chan Message)
	go func(output chan Message) {
		current := startAt
		for {
			s.waitUntilSequence(channel, current)
			msg := s.events[channel][current-1]
			output <- msg
			if stopAt > 0 && msg.Sequence == stopAt {
				break
			}
			current++
		}
		close(output)
	}(events)
	return events, nil
}

func (s *MemoryStream) Publish(channel string, msg Message) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	msg.Sequence = int64(len(s.events[channel]) + 1)
	s.events[channel] = append(s.events[channel], msg)
	return nil
}
