package main

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/lithammer/shortuuid/v3"

	"nathejk.dk/nathejk/aggregate/member"
	"nathejk.dk/nathejk/messages"
	"nathejk.dk/nathejk/types"
	"nathejk.dk/pkg/notification"
	"nathejk.dk/pkg/streaminterface"
)

type sosCmd struct {
	publisher streaminterface.Publisher
	sms       notification.SmsSender
	state     StateReader
}

func NewSosCmd(publisher streaminterface.Publisher, state StateReader, sms notification.SmsSender) *sosCmd {
	return &sosCmd{
		publisher: publisher,
		sms:       sms,
		state:     state,
	}
}

func (cmd *sosCmd) Create(req SosRequest, user User) (SosResponse, error) {
	if req.SosID != "" || req.Headline == nil || req.Description == nil {
		e := errors.New("Invalid input")
		return SosResponse{Error: e.Error()}, e
	}
	ID := types.SosID("sos-" + uuid.New().String())
	body := messages.NathejkSosCreated{
		SosID:       ID,
		Headline:    *req.Headline,
		Description: *req.Description,
	}
	msg := cmd.publisher.MessageFunc()(streaminterface.SubjectFromStr("nathejk:sos.created"))
	//msg := eventstream.NewMessage()
	//msg.Msg().Type = "sos.created"
	msg.SetBody(body)
	meta := messages.Metadata{Producer: "hq-api"}
	//meta.RequestHeaders.Set(r.Header)
	msg.SetMeta(&meta)
	if err := cmd.publisher.Publish(msg); err != nil {
		return SosResponse{Error: err.Error()}, err
	}
	return SosResponse{SosID: ID, OK: true}, nil
}
func (cmd *sosCmd) UpdateHeadline(req SosRequest, user User) (SosResponse, error) {
	if req.SosID == "" || req.Headline == nil {
		e := errors.New("Invalid input")
		return SosResponse{Error: e.Error()}, e
	}
	msg := cmd.publisher.MessageFunc()(streaminterface.SubjectFromStr("nathejk:sos.headline.updated"))
	//msg := eventstream.NewMessage()
	//msg.Msg().Type = "sos.headline.updated"
	msg.SetBody(&messages.NathejkSosHeadlineUpdated{SosID: req.SosID, Headline: *req.Headline})
	msg.SetMeta(&messages.Metadata{Producer: "hq-api"})

	if err := cmd.publisher.Publish(msg); err != nil {
		return SosResponse{Error: err.Error()}, err
	}
	return SosResponse{SosID: req.SosID, OK: true}, nil
}
func (cmd *sosCmd) UpdateDescription(SosRequest, User) (SosResponse, error) {
	return SosResponse{}, nil
}

func (cmd *sosCmd) Close(req SosRequest, user User) (SosResponse, error) {
	body := messages.NathejkSosClosed{
		SosID: req.SosID,
	}
	msg := cmd.publisher.MessageFunc()(streaminterface.SubjectFromStr("nathejk:sos.closed"))
	//msg := eventstream.NewMessage()
	//msg.Msg().Type = "sos.closed"
	msg.SetBody(body)
	meta := messages.Metadata{Producer: "hq-api"}
	//meta.RequestHeaders.Set(r.Header)
	msg.SetMeta(&meta)
	if err := cmd.publisher.Publish(msg); err != nil {
		return SosResponse{Error: err.Error()}, err
	}
	return SosResponse{SosID: req.SosID, OK: true}, nil
}

func (cmd *sosCmd) Reopen(req SosRequest, user User) (SosResponse, error) {
	msg := cmd.publisher.MessageFunc()(streaminterface.SubjectFromStr("nathejk:sos.reopened"))
	//msg := eventstream.NewMessage()
	//msg.Msg().Type = "sos.reopened"
	msg.SetBody(&messages.NathejkSosReopened{SosID: req.SosID})
	msg.SetMeta(&messages.Metadata{Producer: "hq-api"})

	if err := cmd.publisher.Publish(msg); err != nil {
		return SosResponse{Error: err.Error()}, err
	}
	return SosResponse{SosID: req.SosID, OK: true}, nil
}

func (cmd *sosCmd) Delete(SosRequest, User) (SosResponse, error) { return SosResponse{}, nil }

func (cmd *sosCmd) AddComment(req SosCommentRequest, user User) (SosResponse, error) {
	if req.SosID == "" || req.Comment == "" {
		e := errors.New("Invalid input")
		return SosResponse{Error: e.Error()}, e
	}
	ID := types.SosCommentID("comment-" + uuid.New().String())
	body := messages.NathejkSosCommented{
		SosID:     req.SosID,
		CommentID: ID,
		Comment:   req.Comment,
	}
	msg := cmd.publisher.MessageFunc()(streaminterface.SubjectFromStr("nathejk:sos.commented"))
	//msg := eventstream.NewMessage()
	//msg.Msg().Type = "sos.commented"
	msg.SetBody(body)
	meta := messages.Metadata{Producer: "hq-api"}
	//meta.RequestHeaders.Set(r.Header)
	msg.SetMeta(&meta)
	if err := cmd.publisher.Publish(msg); err != nil {
		return SosResponse{Error: err.Error()}, err
	}
	return SosResponse{SosID: body.SosID, CommentID: ID, OK: true}, nil
	return SosResponse{}, nil
}
func (cmd *sosCmd) UpdateComment(SosCommentRequest, User) (SosResponse, error) {
	return SosResponse{}, nil
}

func (cmd *sosCmd) DisassociateTeam(req SosTeamRequest, user User) (SosResponse, error) {
	msg := cmd.publisher.MessageFunc()(streaminterface.SubjectFromStr("nathejk:sos.team.disassociated"))
	//msg := eventstream.NewMessage()
	//msg.Msg().Type = "sos.team.disassociated"
	msg.SetBody(&messages.NathejkSosTeamDisassociated{SosID: req.SosID, TeamID: req.TeamID})
	msg.SetMeta(&messages.Metadata{Producer: "hq-api"})

	if err := cmd.publisher.Publish(msg); err != nil {
		return SosResponse{Error: err.Error()}, err
	}
	return SosResponse{SosID: req.SosID, TeamID: req.TeamID, OK: true}, nil
}

func (cmd *sosCmd) AssociateTeam(req SosTeamRequest, user User) (SosResponse, error) {
	msg := cmd.publisher.MessageFunc()(streaminterface.SubjectFromStr("nathejk:sos.team.associated"))
	//msg := eventstream.NewMessage()
	//msg.Msg().Type = "sos.team.associated"
	msg.SetBody(&messages.NathejkSosTeamAssociated{SosID: req.SosID, TeamID: req.TeamID})
	msg.SetMeta(&messages.Metadata{Producer: "hq-api"})

	if err := cmd.publisher.Publish(msg); err != nil {
		return SosResponse{Error: err.Error()}, err
	}
	return SosResponse{SosID: req.SosID, TeamID: req.TeamID, OK: true}, nil
}

func (cmd *sosCmd) MergeTeams(req SosTeamRequest, user User) (SosResponse, error) {
	msg := cmd.publisher.MessageFunc()(streaminterface.SubjectFromStr("nathejk:team.merged"))
	//msg := eventstream.NewMessage()
	//msg.Msg().Type = "team.merged"
	msg.SetBody(&messages.NathejkTeamMerged{SosID: req.SosID, TeamID: req.TeamID, ParentTeamID: req.ParentTeamID})
	msg.SetMeta(&messages.Metadata{Producer: "hq-api"})

	if err := cmd.publisher.Publish(msg); err != nil {
		return SosResponse{Error: err.Error()}, err
	}
	return SosResponse{SosID: req.SosID, TeamID: req.TeamID, OK: true}, nil
}

func (cmd *sosCmd) SplitTeam(req SosTeamRequest, user User) (SosResponse, error) {
	msg := cmd.publisher.MessageFunc()(streaminterface.SubjectFromStr("nathejk:team.splited"))
	//msg := eventstream.NewMessage()
	//msg.Msg().Type = "team.splited"
	msg.SetBody(&messages.NathejkTeamSplited{SosID: req.SosID, TeamID: req.TeamID})
	msg.SetMeta(&messages.Metadata{Producer: "hq-api"})

	if err := cmd.publisher.Publish(msg); err != nil {
		return SosResponse{Error: err.Error()}, err
	}
	return SosResponse{SosID: req.SosID, TeamID: req.TeamID, OK: true}, nil
}

func (cmd *sosCmd) MemberStatusChange(req SosMemberRequest, user User) (SosResponse, error) {
	if !req.Status.Valid() {
		return SosResponse{}, errors.New(fmt.Sprintf("Invalid status %q", req.Status))
	}
	msg := cmd.publisher.MessageFunc()(streaminterface.SubjectFromStr("nathejk:member.status.changed"))
	//msg := eventstream.NewMessage()
	//msg.Msg().Type = "member.status.changed"
	msg.SetBody(&messages.NathejkMemberStatusChanged{MemberID: req.MemberID, Status: req.Status})
	msg.SetMeta(&messages.NathejkSosMetadata{Metadata: messages.Metadata{Producer: "hq-api"}, SosID: req.SosID})

	if err := cmd.publisher.Publish(msg); err != nil {
		return SosResponse{Error: err.Error()}, err
	}
	return SosResponse{SosID: req.SosID, MemberID: req.MemberID, OK: true}, nil
}

func (cmd *sosCmd) SendPositionSms(req SosMemberRequest, user User) (SosResponse, error) {
	var member member.MemberAggregate
	if err := cmd.state.Read("spejder", string(req.MemberID), &member); err != nil {
		return SosResponse{Error: err.Error()}, err
	}
	smsID := shortuuid.New()
	text := "Følg link for at dele din position https://hej.nathejk.dk/hej/" + smsID
	//phone := types.PhoneNumber("12345678")

	msgType := "member.positionsms.sent"
	body := &messages.NathejkMemberPositionSmsSent{SMSID: smsID, MemberID: req.MemberID, SosID: req.SosID, Phone: member.Phone, Text: text}
	if err := cmd.sms.Send(string(member.Phone), text); err != nil {
		msgType = "member.positionsms.failed"
		body.Error = err.Error()
	}

	msg := cmd.publisher.MessageFunc()(streaminterface.SubjectFromStr("nathejk:" + msgType))
	//msg := eventstream.NewMessage()
	//msg.Msg().Type = msgType
	msg.SetBody(body)
	msg.SetMeta(&messages.Metadata{Producer: "hq-api"})

	if err := cmd.publisher.Publish(msg); err != nil {
		return SosResponse{Error: err.Error()}, err
	}
	return SosResponse{SosID: req.SosID, MemberID: req.MemberID, OK: true}, nil
}

func (cmd *sosCmd) SetSeverity(req SosSeverityRequest, user User) (SosResponse, error) {
	msg := cmd.publisher.MessageFunc()(streaminterface.SubjectFromStr("nathejk:sos.severity.specified"))
	//msg := eventstream.NewMessage()
	//msg.Msg().Type = "sos.severity.specified"
	msg.SetBody(&messages.NathejkSosSeveritySpecified{SosID: req.SosID, Severity: req.Severity})
	msg.SetMeta(&messages.Metadata{Producer: "hq-api"})

	if err := cmd.publisher.Publish(msg); err != nil {
		return SosResponse{Error: err.Error()}, err
	}
	return SosResponse{SosID: req.SosID, OK: true}, nil
}

func (cmd *sosCmd) Assign(req SosAssignRequest, user User) (SosResponse, error) {
	msg := cmd.publisher.MessageFunc()(streaminterface.SubjectFromStr("nathejk:sos.assigned"))
	//msg := eventstream.NewMessage()
	//msg.Msg().Type = "sos.assigned"
	msg.SetBody(&messages.NathejkSosAssigned{SosID: req.SosID, Assignee: req.Assignee})
	msg.SetMeta(&messages.Metadata{Producer: "hq-api"})

	if err := cmd.publisher.Publish(msg); err != nil {
		return SosResponse{Error: err.Error()}, err
	}
	return SosResponse{SosID: req.SosID, OK: true}, nil
	return SosResponse{}, nil
}

/*
func (cmd sosCmd) Read(req interface{}) (interface{}, error) {
	return nil, nil
}
func (cmd sosCmd) Update(req interface{}) (interface{}, error) {
	r := req.(*SosUpdateRequest)
	if r.ID == "" {
		return nil, errors.New("Can't update controlgroup, no ID specified")
	}
	body := messages.NathejkSosCreated{
		SosID:       r.ID,
		Headline:    r.Name,
		Description: "",
	}
	msg := eventstream.NewMessage()
	msg.Msg().Type = "controlgroup.updated"
	msg.SetBody(body)
	meta := messages.Metadata{Producer: cmd.publisher.ClientID()}
	//meta.RequestHeaders.Set(r.Header)
	msg.SetMeta(&meta)
	return "ok", cmd.publisher.Publish("nathejk", msg)
}

func (cmd sosCmd) Delete(req interface{}) (interface{}, error) {
	r := req.(*DeleteRequest)
	body := messages.NathejkControlGroupDeleted{
		ControlGroupID: r.ID,
	}
	msg := eventstream.NewMessage()
	msg.Msg().Type = "controlgroup.deleted"
	msg.SetBody(body)
	meta := messages.Metadata{Producer: cmd.publisher.ClientID()}
	//meta.RequestHeaders.Set(r.Header)
	msg.SetMeta(&meta)
	return "ok", cmd.publisher.Publish("nathejk", msg)
}

type SosCreateRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}
type SosReadRequest struct {
}
type SosUpdateRequest struct {
	ID types.SosID `json:"sosId"`

	SosCreateRequest
}

type SosDeleteRequest struct {
	ID types.SosID `json:"sosId"`
}
*/
