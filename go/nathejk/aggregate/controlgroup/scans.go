package controlgroup

import (
	"log"
	"sync"

	"nathejk.dk/nathejk/aggregate"
	"nathejk.dk/nathejk/aggregate/user"
	"nathejk.dk/nathejk/types"
	"nathejk.dk/pkg/streaminterface"
	"nathejk.dk/pkg/streamtable"
)

type ControlGroupScansAggregate struct {
	ControlGroupAggregate
	Users map[types.UserID]user.UserAggregate `json:"users"`
	//	Scans []scan.ScanAggregate

	userIDs map[types.UserID]bool
	deleted bool
}

func (v *ControlGroupScansAggregate) ID() string {
	return string(v.ControlGroupID)
}

func (v *ControlGroupScansAggregate) IsValid() bool {
	return v.ControlGroupID != "" && !v.deleted
}

type controlGroupScansModel struct {
	Publisher streaminterface.Publisher
	state     streamtable.StateWriter

	sync.Mutex
	ap *aggregate.Publisher
	//validator     *validator.Validator
	controlGroups map[types.ControlGroupID]*ControlGroupScansAggregate
	users         map[types.UserID]*user.UserAggregate
	live          bool
}

func NewControlGroupScansModel(publisher streaminterface.Publisher, state streamtable.StateWriter) *controlGroupScansModel {
	m := controlGroupScansModel{
		state:         state,
		controlGroups: map[types.ControlGroupID]*ControlGroupScansAggregate{},
		users:         map[types.UserID]*user.UserAggregate{},
		//validator:     &validator.Validator{IgnoreMissing: true},
	}
	//vPublisher := &validator.Publisher{Publisher: publisher, Validator: m.validator}
	//mPublisher := model.Publisher(&m, vPublisher)
	m.ap = aggregate.NewPublisher(publisher, "controlgroupscans.aggregate")

	if err := state.Query("DROP TABLE If EXISTS scans"); err != nil {
		log.Printf("Error creating table %q", err)
	}
	err := state.Query("CREATE TABLE scans (id int not null primary key) ")
	if err != nil {
		log.Printf("Error creating table %q", err)
	}

	return &m
}

func (m *controlGroupScansModel) get(ID types.ControlGroupID) *ControlGroupScansAggregate {
	if m.controlGroups[ID] == nil {
		m.controlGroups[ID] = &ControlGroupScansAggregate{Users: map[types.UserID]user.UserAggregate{}}
		m.controlGroups[ID].ControlGroupID = ID
	}
	return m.controlGroups[ID]
}

func (m *controlGroupScansModel) CaughtUp() {
	aggregates := aggregate.MapToAggregates(m.controlGroups)
	m.ap.Flush(aggregates)
	m.live = true
}

func (m *controlGroupScansModel) Consumes() []streaminterface.Subject {
	return []streaminterface.Subject{
		streaminterface.SubjectFromStr("controlgroup.aggregate:updated"),
		streaminterface.SubjectFromStr("controlgroup.aggregate:removed"),
		streaminterface.SubjectFromStr("user.aggregate:updated"),
		streaminterface.SubjectFromStr("user.aggregate:removed"),
		//"userscans.aggregate:updated",
		//"userscans.aggregate:deleted",
	}
}

func (m *controlGroupScansModel) Produces() []string {
	return []string{"controlgroupscans.aggregate:updated", "controlgroupscans.aggregate:removed", "controlgroupscans.aggregate:caughtup"}
}

func (m *controlGroupScansModel) HandleMessage(msg streaminterface.Message) error {
	/*msg, ok := i.(eventstream.Message)
	if !ok {
		return
	}
	/*if err := m.validator.Validate(msg); err != nil {
		log.Println(err)
		return
	}*/
	switch msg.Subject().Subject() {
	case "controlgroup.aggregate:updated":
		var body ControlGroupAggregate
		msg.Body(&body)
		cg := m.get(body.ControlGroupID)
		cg.deleted = false
		cg.ControlGroupAggregate = body
		//msg.DecodeBody(&cg)
		cg.userIDs = map[types.UserID]bool{}
		cg.Users = map[types.UserID]user.UserAggregate{}
		for _, control := range cg.Controls {
			for _, scanner := range control.Scanners {
				cg.userIDs[scanner.UserID] = true
				if u, found := m.users[scanner.UserID]; found {
					cg.Users[scanner.UserID] = *u
				}
			}
		}

		if m.live {
			m.ap.Publish(cg)
		}
	case "controlgroup.aggregate:removed":
		var body ControlGroupAggregate
		msg.Body(&body)
		cg := m.get(body.ControlGroupID)
		cg.deleted = true

		if m.live {
			m.ap.Publish(cg)
		}
	case "user.aggregate:updated":
		var body user.UserAggregate
		msg.Body(&body)
		m.users[body.UserID] = &body
		for _, cg := range m.controlGroups {
			if cg.userIDs[body.UserID] {
				cg.Users[body.UserID] = body
			}
			if m.live {
				m.ap.Publish(cg)
			}
		}

	default:
		log.Printf("Unsupported message type %q", msg.Subject().Subject())
	}
	return nil
}
