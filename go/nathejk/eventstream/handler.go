package eventstream

type EventHandler interface {
	Handle(msg Message)
}

type HandlerFunc func(msg Message)

func (h HandlerFunc) Handle(msg Message) {
	h(msg)
}

func ConcurrentHandler(handlers ...EventHandler) EventHandler {
	return HandlerFunc(func(msg Message) {
		handlersDone := make([]chan bool, len(handlers))
		for i, eventHandler := range handlers {
			handlersDone[i] = make(chan bool)
			go func(u int, h EventHandler, m Message) {
				h.Handle(m)
				close(handlersDone[u])
			}(i, eventHandler, msg)
		}
		for i := 0; i < len(handlers); i++ {
			<-handlersDone[i]
		}
	})
}

func HandleEvents(events chan Message, handlers ...EventHandler) chan bool {
	done := make(chan bool)
	go func() {
		for msg := range events {
			for _, eventHandler := range handlers {
				eventHandler.Handle(msg)
			}
		}
		close(done)
	}()
	return done
}
