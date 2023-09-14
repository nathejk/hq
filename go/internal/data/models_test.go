package data_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"nathejk.dk/internal/data"
	"nathejk.dk/nathejk/types"
)

func Time(rfc3339 string) time.Time {
	ts, _ := time.Parse(time.RFC3339, rfc3339)
	return ts
}

type CheckgroupMock struct {
	Checkgroups data.Checkgroups
	Metadata    data.Metadata
	Error       error
}

func (m CheckgroupMock) GetAll(filters data.Filters) (data.Checkgroups, data.Metadata, error) {
	return m.Checkgroups, m.Metadata, m.Error
}

type TeamMock struct {
	StartedTeamIDs      types.TeamIDs
	DiscontinuedTeamIDs types.TeamIDs
	Metadata            data.Metadata
	Error               error
}

func (m TeamMock) GetStartedTeamIDs(filters data.Filters) ([]types.TeamID, data.Metadata, error) {
	return m.StartedTeamIDs, m.Metadata, m.Error
}
func (m TeamMock) GetDiscontinuedTeamIDs(filters data.Filters) ([]types.TeamID, data.Metadata, error) {
	return m.DiscontinuedTeamIDs, m.Metadata, m.Error
}
func (m TeamMock) GetPatruljer(data.Filters) ([]*data.Patrulje, data.Metadata, error) {
	return nil, m.Metadata, m.Error
}
func (m TeamMock) GetPatrulje(types.TeamID) (*data.Patrulje, error) {
	return nil, m.Error
}

type ScanMock struct {
	CheckgroupsScans   []*data.CheckgroupScan
	CheckgroupTeamTime data.CheckgroupTeamTime
	Metadata           data.Metadata
	Error              error
}

func (m ScanMock) GetCheckgroupsScans(filters data.Filters) ([]*data.CheckgroupScan, data.Metadata, error) {
	return m.CheckgroupsScans, m.Metadata, m.Error
}
func (m ScanMock) GetNewestCheckgroupTeamTime(filters data.Filters) (data.CheckgroupTeamTime, data.Metadata, error) {
	return m.CheckgroupTeamTime, m.Metadata, m.Error
}

func TestTeamOnlyCountedOnce(t *testing.T) {
	m := data.Models{
		Checkgroups: CheckgroupMock{Checkgroups: data.Checkgroups{&data.Checkgroup{
			ID: types.ControlGroupID("cg-id-1"),
			Checkpoints: []*data.Checkpoint{
				&data.Checkpoint{Scheme: types.CheckpointSchemeFixed, Index: 0, OpenFrom: Time("2020-12-31T21:00:00Z"), OpenUntil: Time("2020-12-31T23:00:00Z")},
				&data.Checkpoint{Scheme: types.CheckpointSchemeFixed, Index: 1, OpenFrom: Time("2020-12-31T21:00:00Z"), OpenUntil: Time("2020-12-31T23:00:00Z")},
				&data.Checkpoint{Scheme: types.CheckpointSchemeFixed, Index: 2, OpenFrom: Time("2020-12-31T21:00:00Z"), OpenUntil: Time("2020-12-31T23:00:00Z"), Minus: 1, Plus: 3},
				&data.Checkpoint{Scheme: types.CheckpointSchemeRelative, Index: 3, RelativeControlGroupID: "cg-id-2", Plus: 3},
			},
		}, &data.Checkgroup{}}},
		Teams: TeamMock{},
		Scans: ScanMock{
			CheckgroupsScans: []*data.CheckgroupScan{
				&data.CheckgroupScan{TeamID: types.TeamID("team-1"), ControlGroupID: types.ControlGroupID("cg-id-1"), ControlpointIndex: 1, Time: Time("2020-12-31T20:00:00Z")},
				&data.CheckgroupScan{TeamID: types.TeamID("team-1"), ControlGroupID: types.ControlGroupID("cg-id-1"), ControlpointIndex: 1, Time: Time("2020-12-31T22:00:00Z")},
				&data.CheckgroupScan{TeamID: types.TeamID("team-1"), ControlGroupID: types.ControlGroupID("cg-id-1"), ControlpointIndex: 1, Time: Time("2020-12-31T22:00:01Z")},
				&data.CheckgroupScan{TeamID: types.TeamID("team-1"), ControlGroupID: types.ControlGroupID("cg-id-1"), ControlpointIndex: 1, Time: Time("2020-12-31T23:30:00Z")},
				//&data.CheckgroupScan{TeamID: types.TeamID("team-8"), ControlGroupID: types.ControlGroupID("cg-id-1"), ControlpointIndex: 1, Time: Time("2020-12-31T23:30:01Z")},
				//&data.CheckgroupScan{TeamID: types.TeamID("team-8"), ControlGroupID: types.ControlGroupID("cg-id-1"), ControlpointIndex: 1, Time: Time("2020-12-31T23:30:02Z")},
				&data.CheckgroupScan{TeamID: types.TeamID("team-2"), ControlGroupID: types.ControlGroupID("cg-id-1"), ControlpointIndex: 2, Time: Time("2020-12-31T23:03:00Z")},
				&data.CheckgroupScan{TeamID: types.TeamID("team-3"), ControlGroupID: types.ControlGroupID("cg-id-1"), ControlpointIndex: 2, Time: Time("2020-12-31T20:59:00Z")},
				&data.CheckgroupScan{TeamID: types.TeamID("team-4"), ControlGroupID: types.ControlGroupID("cg-id-1"), ControlpointIndex: 2, Time: Time("2020-12-31T20:58:00Z")},
				&data.CheckgroupScan{TeamID: types.TeamID("team-5"), ControlGroupID: types.ControlGroupID("cg-id-1"), ControlpointIndex: 3, Time: Time("2020-12-31T19:02:00Z")},
				&data.CheckgroupScan{TeamID: types.TeamID("team-6"), ControlGroupID: types.ControlGroupID("cg-id-1"), ControlpointIndex: 3, Time: Time("2020-12-31T20:03:00Z")},
				&data.CheckgroupScan{TeamID: types.TeamID("team-7"), ControlGroupID: types.ControlGroupID("cg-id-1"), ControlpointIndex: 3, Time: Time("2020-12-31T20:04:00Z")},
			},
			CheckgroupTeamTime: data.CheckgroupTeamTime{
				"cg-id-2": map[types.TeamID]time.Time{
					"team-1": Time("2020-12-31T20:00:00Z"),
					"team-5": Time("2020-12-31T20:00:00Z"),
					"team-6": Time("2020-12-31T20:00:00Z"),
					"team-7": Time("2020-12-31T20:00:00Z"),
				},
			},
		},
	}
	status, _, _ := m.GetCheckgroupsStatus(data.Filters{})

	assert := assert.New(t)

	//spew.Dump(status)
	assert.Equal(2, len(status.Checkgroups), "There need to be 2 Checkgroups")
	assert.Equal(0, len(status.Checkgroups.Checkgroup("cg-id-1").Checkpoint(0).OnTime), "There exists no on time scans for Checkpoint 0")
	assert.Equal(0, len(status.Checkgroups.Checkgroup("cg-id-1").Checkpoint(0).OverTime), "There exists no over time scans for Checkpoint 0")
	assert.Equal(1, len(status.Checkgroups.Checkgroup("cg-id-1").Checkpoint(1).OnTime), "Each team should only be counted once, even if multiple scans exists")
	assert.Equal(0, len(status.Checkgroups.Checkgroup("cg-id-1").Checkpoint(1).OverTime), "When on time scan exists, over time scans is disregarded")
	assert.Equal(1, len(status.Checkgroups.Checkgroup("cg-id-1").Checkpoint(2).OverTime), "Only 1 outside of plus/minus time")
	assert.Equal(2, len(status.Checkgroups.Checkgroup("cg-id-1").Checkpoint(2).OnTime), "Plus/minus minutes should be considder on time")
	assert.Equal(1, len(status.Checkgroups.Checkgroup("cg-id-1").Checkpoint(3).OverTime), "Only 1 outside of plus/minus time")
	assert.Equal(2, len(status.Checkgroups.Checkgroup("cg-id-1").Checkpoint(3).OnTime), "There should be 2 within relative distance")

}

/*
func TestMessageTypeChange(t *testing.T) {
	exp := mytype{Data: "Hello"}
	m := nats.NewMessage()
	m.SetBody(&exp)
	var got mytypetwo
	if err := m.Body(&got); err != nil {
		t.Fatal(err)
	}
	if got.Data != exp.Data {
		t.Fatalf("exp %s, got %s\n", exp.Data, got.Data)
	}
}

func BenchmarkMessageTypeAssert(b *testing.B) {
	exp := mytype{Data: "Hello"}

	for i := 0; i < b.N; i++ {
		m := nats.NewMessage()
		m.SetBody(&exp)

		var got mytype
		if err := m.Body(&got); err != nil {
			b.Fatal(err)
		}

		if got.Data != exp.Data {
			b.Fatalf("exp %s, got %s\n", exp.Data, got.Data)
		}
	}
}

func TestEventMessage(t *testing.T) {
	assert := assert.New(t)

	msg := nats.NewMessage()
	assert.NotEqual("", msg.EventID(), "Event id should not be empty")
	assert.Equal(msg.EventID(), msg.CausationID(), "Causation id should match event id")
	assert.Equal(msg.EventID(), msg.CorrelationID(), "Correlation id should match event id")

	var err error

	type TestBody struct {
		Hello string `json:"hello"`
	}
	body := TestBody{"world"}
	msg.SetBody(&body)

	type TestMeta struct {
		Goodbye string `json:"goodbye"`
	}
	meta := TestMeta{"universe"}
	err = msg.SetMeta(meta)
	assert.Nil(err)
	assert.Equal("{\"goodbye\":\"universe\"}", string(msg.RawMeta().(json.RawMessage)))

	err = msg.SetMeta("WORLD")
	assert.Nil(err)
	assert.Equal("\"WORLD\"", string(msg.RawMeta().(json.RawMessage)))
}

func TestMessageValidBodyAndMeta(t *testing.T) {
	assert := assert.New(t)

	type TestBody struct {
		Hello string `json:"hello"`
	}
	type TestMeta struct {
				Goodbye string `json:"goodbye"`
	}

	msg := nats.NewMessage()
	msg.SetBody(&TestBody{Hello: "world"})
	msg.SetMeta(&TestMeta{Goodbye: "universe"})

	var body TestBody
	if err := msg.Body(&body); err != nil {
		t.Fatal(err)
	}
	assert.Equal(TestBody{Hello: "world"}, body)

	var meta TestMeta
	msg.Meta(&meta)
	assert.Equal(TestMeta{Goodbye: "universe"}, meta)
}

func TestMessageInvalidBodyAndMeta(t *testing.T) {
	assert := assert.New(t)
	msg := nats.NewMessage()

	err := msg.SetMeta(make(chan int))
	assert.NotNil(err)
}

/*
func TestMessageEncoding(t *testing.T) {
	assert := assert.New(t)

	exp := `{
	"type": "",
	"eventId": "event-54a839b5-805c-4421-ac0c-9925f2dd5e78",
	"correlationId": "event-54a839b5-805c-4421-ac0c-9925f2dd5e78",
	"causationId": "event-54a839b5-805c-4421-ac0c-9925f2dd5e78",
	"datetime": "2019-10-31T07:26:18.6200067Z",
	"body": {
		"hello": "world"
	},
	"meta": {
		"goodbye": "universe"
	}
}`
	rm := nats.NewMessage()
	err := rm.DecodeData([]byte(exp))
	assert.Nil(err)

	got, err := json.Marshal(rm)
	assert.Nil(err)

	assert.JSONEq(exp, string(got))
}*/
