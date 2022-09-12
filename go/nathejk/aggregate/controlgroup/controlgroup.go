package controlgroup

import (
	"log"
	"sync"
	"time"

	"nathejk.dk/nathejk/aggregate"
	"nathejk.dk/nathejk/messages"
	"nathejk.dk/nathejk/types"
	"nathejk.dk/pkg/streaminterface"
	"nathejk.dk/pkg/streamtable"
)

type ControlGroupAggregate struct {
	messages.NathejkControlGroupUpdated

	Scans     []*messages.NathejkQrScanned `json:"scans"`
	CreatedAt time.Time                    `json:"createdAt"`
	TestedAt  *time.Time                   `json:"testedAt"`
	deleted   bool
}

func (v *ControlGroupAggregate) ID() string {
	return string(v.ControlGroupID)
}

func (v *ControlGroupAggregate) IsValid() bool {
	return v.ControlGroupID != "" && !v.deleted
}

type controlGroupModel struct {
	state     streamtable.StateWriter
	Publisher streaminterface.Publisher

	sync.Mutex
	ap *aggregate.Publisher
	//validator     *validator.Validator
	controlGroups map[types.ControlGroupID]*ControlGroupAggregate
	scans         []messages.NathejkQrScanned
	live          bool
}

func NewControlGroupTable(state streamtable.StateWriter) *controlGroupModel {
	m := controlGroupModel{
		state:         state,
		controlGroups: map[types.ControlGroupID]*ControlGroupAggregate{},
		//validator:     &validator.Validator{IgnoreMissing: true},
	}
	err := state.Init(&ControlGroupAggregate{})
	if err != nil {
		log.Printf("Init error %q", err)
	}
	return &m
}
func NewControlGroupModel(publisher streaminterface.Publisher) *controlGroupModel {
	m := controlGroupModel{
		controlGroups: map[types.ControlGroupID]*ControlGroupAggregate{},
		//validator:     &validator.Validator{IgnoreMissing: true},
	}
	//vPublisher := &validator.Publisher{Publisher: publisher, Validator: m.validator}
	//mPublisher := model.Publisher(&m, vPublisher)
	m.ap = aggregate.NewPublisher(publisher, "controlgroup.aggregate")
	return &m
}

func (m *controlGroupModel) get(ID types.ControlGroupID) *ControlGroupAggregate {
	if m.controlGroups[ID] == nil {
		m.controlGroups[ID] = &ControlGroupAggregate{}
		m.controlGroups[ID].ControlGroupID = ID
	}
	return m.controlGroups[ID]
}

func (m *controlGroupModel) CaughtUp() {
	aggregates := aggregate.MapToAggregates(m.controlGroups)
	if m.ap != nil {
		m.ap.Flush(aggregates)
	}
	m.live = true
}

func (m *controlGroupModel) Consumes() []streaminterface.Subject {
	return []streaminterface.Subject{
		streaminterface.SubjectFromStr("nathejk:controlgroup.updated"),
		streaminterface.SubjectFromStr("nathejk:controlgroup.deleted"),
		streaminterface.SubjectFromStr("nathejk:qr.scanned"),
	}
}

func (m *controlGroupModel) Produces() []string {
	return []string{"controlgroup.aggregate:updated", "controlgroup.aggregate:removed", "controlgroup.aggregate:caughtup"}
}

func (m *controlGroupModel) HandleMessage(msg streaminterface.Message) error {
	/*
		msg, ok := i.(eventstream.Message)
		if !ok {
			return
		}
		/*if err := m.validator.Validate(msg); err != nil {
			log.Println(err)
			return
		}*/
	switch msg.Subject().Subject() {
	case "nathejk:controlgroup.updated":
		var body messages.NathejkControlGroupUpdated
		msg.Body(&body)
		cg := m.get(body.ControlGroupID)
		cg.deleted = false
		cg.CreatedAt = msg.Time()
		msg.Body(&cg)

		if m.state != nil {
			m.state.Write(cg)
		}
		if m.live && m.ap != nil {
			m.ap.Publish(cg)
		}
	case "nathejk:controlgroup.deleted":
		var body messages.NathejkControlGroupDeleted
		msg.Body(&body)
		cg := m.get(body.ControlGroupID)
		cg.deleted = true

		if m.live && m.ap != nil {
			m.ap.Publish(cg)
		}
	case "nathejk:qr.scanned":
		var body messages.NathejkQrScanned
		msg.Body(&body)
		/*
			for _, cg := range m.controlGroups {
				for _, c := range cg.Controls {
					for _, s := range c.Scanners {
						if s.UserID == body.ScannerPhone && s.DateRange.In(msg.Datetime) {
							cq.Scans = append(cg.Scans, &body)
						}
					}
				}
			}
		*/
	default:
		log.Printf("Unsupported message type %q", msg.Subject().Subject())
	}
	return nil
}
