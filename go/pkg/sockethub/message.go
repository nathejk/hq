package sockethub

// Define our message object
type message struct {
	View string      `json:"view"`
	Key  string      `json:"key"`
	Body interface{} `json:"body"`
}

func NewMessage() *message {
	return &message{}
}

type filter struct {
}

type subscribeMessage struct {
	View    string
	Filters []filter
}
