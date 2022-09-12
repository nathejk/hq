package eventstream

import (
	"log"
	"time"
)

type DebugHandler struct {
	Mod    int64
	Len    int64
	Prefix string
	Delay  time.Duration
}

func (e *DebugHandler) Handle(msg Message) {
	if (e.Mod > 0 && msg.Sequence%e.Mod == 0) || (e.Len > 0 && e.Len-msg.Sequence <= e.Mod) {
		log.Printf("[%s] %d", e.Prefix, msg.Sequence)
	}
	if e.Delay > 0 {
		time.Sleep(e.Delay)
	}
}
