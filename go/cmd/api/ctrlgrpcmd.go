package main

import (
	"errors"
	"log"
	"time"

	"github.com/google/uuid"

	"nathejk.dk/nathejk/messages"
	"nathejk.dk/nathejk/types"
	"nathejk.dk/pkg/streaminterface"
)

type ctrlgrpCmd struct {
	publisher streaminterface.Publisher
}

func NewControlGroupCmd(publisher streaminterface.Publisher) *ctrlgrpCmd {
	return &ctrlgrpCmd{
		publisher: publisher,
	}
}

func (cmd ctrlgrpCmd) Create(req interface{}) (interface{}, error) {
	return cmd.Update(&UpdateRequest{
		ID:            types.ControlGroupID("ctrlgrp-" + uuid.New().String()),
		CreateRequest: *req.(*CreateRequest),
	})
}
func (cmd ctrlgrpCmd) Read(req interface{}) (interface{}, error) {
	return nil, nil
}
func (cmd ctrlgrpCmd) Update(req interface{}) (interface{}, error) {
	r := req.(*UpdateRequest)
	if r.ID == "" {
		return nil, errors.New("Can't update controlgroup, no ID specified")
	}
	body := messages.NathejkControlGroupUpdated{
		ControlGroupID: r.ID,
		Name:           r.Name,
		Controls:       []messages.NathejkControlGroup_Control{},
	}
	for _, rc := range r.Controls {
		control := messages.NathejkControlGroup_Control{
			Name:                 rc.Name,
			Scheme:               rc.Scheme,
			RelativeCheckgroupID: rc.RelativeCheckgroupID,
			DateRange: messages.NathejkControlGroup_DateRange{
				StartDate: rc.DateRange.StartDate,
				EndDate:   rc.DateRange.EndDate,
			},
			Minus:    rc.Minus,
			Plus:     rc.Plus,
			Scanners: []messages.NathejkControlGroup_Scanner{},
		}
		for _, rs := range rc.Scanners {
			control.Scanners = append(control.Scanners, messages.NathejkControlGroup_Scanner{
				UserID: rs.UserID,
				DateRange: messages.NathejkControlGroup_DateRange{
					StartDate: rs.DateRange.StartDate,
					EndDate:   rs.DateRange.EndDate,
				},
			})
		}
		body.Controls = append(body.Controls, control)
	}
	msg := cmd.publisher.MessageFunc()(streaminterface.SubjectFromStr("nathejk:controlgroup.updated"))
	//msg := eventstream.NewMessage()
	//msg.Msg().Type = "controlgroup.updated"
	msg.SetBody(body)
	meta := messages.Metadata{Producer: "hq-api"}
	//meta.RequestHeaders.Set(r.Header)
	msg.SetMeta(&meta)
	return "ok", cmd.publisher.Publish(msg)
}
func (cmd ctrlgrpCmd) Delete(req interface{}) (interface{}, error) {
	r := req.(*DeleteRequest)
	body := messages.NathejkControlGroupDeleted{
		ControlGroupID: r.ID,
	}
	msg := cmd.publisher.MessageFunc()(streaminterface.SubjectFromStr("nathejk:controlgroup.deleted"))
	//msg := eventstream.NewMessage()
	//msg.Msg().Type = "controlgroup.deleted"
	msg.SetBody(body)
	meta := messages.Metadata{Producer: "hq-api"}
	//meta.RequestHeaders.Set(r.Header)
	msg.SetMeta(&meta)
	return "ok", cmd.publisher.Publish(msg)
}

type Date string

func (d Date) ToTime() time.Time {
	loc, err := time.LoadLocation("Europe/Copenhagen")
	if err != nil {
		log.Printf("Recoverable error %q", err)
		return time.Time{}
	}
	t, err := time.ParseInLocation("2006-01-02 15:04", string(d), loc)
	if err != nil {
		return time.Time{}
	}
	return t
}

type CreateRequest struct {
	Name     string `json:"name"`
	Controls []struct {
		Name                 string               `json:"name"`
		Scheme               string               `json:"scheme"`
		RelativeCheckgroupID types.ControlGroupID `json:"relativeControlGroupId"`
		DateRange            struct {
			StartDate time.Time `json:"startDate"`
			EndDate   time.Time `json:"endDate"`
		} `json:"dateRange"`
		Minus    int `json:"minus"`
		Plus     int `json:"plus"`
		Scanners []struct {
			DateRange struct {
				StartDate time.Time `json:"startDate"`
				EndDate   time.Time `json:"endDate"`
			} `json:"dateRange"`
			UserID types.UserID `json:"userId"`
		} `json:"scanners"`
	} `json:"controls"`
}
type ReadRequest struct {
}
type UpdateRequest struct {
	ID types.ControlGroupID `json:"controlGroupId"`

	CreateRequest
}

type DeleteRequest struct {
	ID types.ControlGroupID `json:"controlGroupId"`
}
