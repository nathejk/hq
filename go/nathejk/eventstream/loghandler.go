package eventstream

import (
	"log"
)

type LogHandler struct {
	Prefix string
	Mod    int64
}

func (h *LogHandler) Handle(msg Message) {
	if msg.Sequence%h.Mod == 0 {
		log.Printf("%s:[%s] %d", h.Prefix, msg.Channel, msg.Sequence)
	}
}
