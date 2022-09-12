package main

import (
	"github.com/google/uuid"

	"nathejk.dk/nathejk/messages"
	"nathejk.dk/nathejk/types"
	"nathejk.dk/pkg/streaminterface"
)

type commander struct {
	publisher streaminterface.Publisher
}

func NewCommander(publisher streaminterface.Publisher) *commander {
	return &commander{
		publisher: publisher,
	}
}

func (c *commander) SaveUser(r PostUserRequest) error {
	body := messages.NathejkUserUpdated{
		UserID:   r.UserID,
		Name:     r.Name,
		Phone:    r.Phone,
		HqAccess: r.HqAccess,
		Group:    r.Group,
	}
	if body.UserID == "" {
		body.UserID = types.UserID("user-" + uuid.New().String())
	}
	msg := c.publisher.MessageFunc()(streaminterface.SubjectFromStr("nathejk:user.updated"))
	//msg := eventstream.NewMessage()
	//msg.Msg().Type = "user.updated"
	msg.SetBody(body)
	meta := messages.Metadata{Producer: "hq-api"}
	//meta.RequestHeaders.Set(r.Header)
	msg.SetMeta(&meta)
	return c.publisher.Publish(msg)
}

func (c *commander) DeleteUser(r DeleteUserRequest) error {
	body := messages.NathejkUserDeleted{
		UserID: r.UserID,
	}
	msg := c.publisher.MessageFunc()(streaminterface.SubjectFromStr("nathejk:user.deleted"))
	//msg := eventstream.NewMessage()
	//msg.Msg().Type = "user.deleted"
	msg.SetBody(body)
	meta := messages.Metadata{Producer: "hq-api"}
	//meta.RequestHeaders.Set(r.Header)
	msg.SetMeta(&meta)
	return c.publisher.Publish(msg)

}
