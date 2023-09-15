package main

import (
	"context"
	"database/sql"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"

	"nathejk.dk/nathejk/aggregate/controlgroup"
	"nathejk.dk/nathejk/aggregate/member"
	"nathejk.dk/nathejk/aggregate/sos"
	"nathejk.dk/nathejk/aggregate/team"
	"nathejk.dk/nathejk/aggregate/user"
	"nathejk.dk/nathejk/table"
	"nathejk.dk/pkg/memorystream"
	"nathejk.dk/pkg/sqlpersister"
	"nathejk.dk/pkg/stream"
	"nathejk.dk/pkg/streaminterface"

	//"nathejk.dk/pkg/model"
	//"nathejk.dk/pkg/multiplexed"
	websync "nathejk.dk/pkg/sockethub"
	"nathejk.dk/pkg/streamtable"
)

type dims struct {
	stream  streaminterface.Stream
	updater websync.Updater
	state   streamtable.StateWriter
	live    bool
}

func NewDims(stream streaminterface.Stream, updater websync.Updater, state streamtable.StateWriter) *dims {
	return &dims{
		stream:  stream,
		updater: updater,
		state:   state,
	}
}

func (d *dims) Subscribe() *dims {
	db, err := sql.Open("mysql", os.Getenv("DB_DSN"))
	if err != nil {
		log.Fatal(err)
	}
	sqlw := sqlpersister.New(db)

	memstream := memorystream.New()
	bufferedPublisher := memstream
	dstmux := stream.NewStreamMux(memstream)
	dstmux.Handles(d.stream, "monolith", "nathejk") //d.stream.Channels()...)
	dstswtch, err := stream.NewSwitch(dstmux, []streaminterface.Consumer{
		table.NewPatrulje(sqlw),
		table.NewPatruljeStatus(sqlw),
		table.NewPatruljeMerged(sqlw),
		table.NewSpejder(sqlw),
		table.NewSpejderStatus(sqlw),
		table.NewSos(sqlw),
		table.NewSosAssociation(sqlw),
		table.NewDepartment(sqlw, memstream),
		table.NewPersonnel(sqlw, memstream),
		table.NewControlPoint(sqlw, memstream),
		table.NewControlGroupUser(sqlw, memstream),
		table.NewControlGroup(sqlw, memstream),
		table.NewScan(sqlw, memstream),

		member.NewMemberModel(bufferedPublisher),
		team.NewSpejderModel(bufferedPublisher),
		team.NewTeamModel(bufferedPublisher),
		team.NewTeamMembersModel(bufferedPublisher),
		team.NewTeamTypeModel(bufferedPublisher, "klan"),
		team.NewTeamTypeModel(bufferedPublisher, "patrulje"),
		sos.NewSosModel(bufferedPublisher),

		user.NewUserModel(bufferedPublisher),
		controlgroup.NewControlGroupModel(bufferedPublisher),
		controlgroup.NewControlGroupTable(d.state),
		controlgroup.NewControlGroupScansModel(bufferedPublisher, d.state),

		websync.NewWebsyncModel(d.updater, map[string]string{
			"user.aggregate":              "personnel",
			"controlgroup.aggregate":      "cg",
			"controlgroupscans.aggregate": "controlgroup",
			"team-klan.aggregate":         "klan",
			"team-patrulje.aggregate":     "patrulje",
			"sos.aggregate":               "sos",
			"spejder.aggregate":           "spejder",
			// "teammembers.aggregate")
		}),
	})
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	live := make(chan struct{})
	go func() {
		err = dstswtch.Run(ctx, func() {
			//dstswtch.Close()
			//log.Printf("Closing")
			live <- struct{}{}
		})
		if err != nil {
			log.Fatal(err)
		}
	}()
	// Waiting for live
	select {
	case <-ctx.Done():
		log.Fatal(ctx.Err())
	case <-live:
	}
	return d
}

/*
	localstream := bufferstream.New(100000)
	natschannels := []string{"nathejk", "monolith"}
	subscriber := &multiplexed.Subscribers{
		Subscribers: multiplexed.ChannelSubscriberMap(d.stream, natschannels...),
		Default:     &eventstream.EventCatchupSubscriber{Subscriber: localstream},
	}

	bufferedPublisher := eventstream.NewBufferedPublisher(localstream, 100000)

	models := []model.Model{
		// Aggregates
		member.NewMemberModel(bufferedPublisher),
		team.NewSpejderModel(bufferedPublisher),
		team.NewTeamModel(bufferedPublisher),
		team.NewTeamMembersModel(bufferedPublisher),
		team.NewTeamTypeModel(bufferedPublisher, "klan"),
		team.NewTeamTypeModel(bufferedPublisher, "patrulje"),
		sos.NewSosModel(bufferedPublisher),

		user.NewUserModel(bufferedPublisher),
		controlgroup.NewControlGroupModel(bufferedPublisher),
		controlgroup.NewControlGroupTable(d.state),
		controlgroup.NewControlGroupScansModel(bufferedPublisher, d.state),

		/*
			// Views
			view.NewPincodeModel(bufferedPublisher),
			view.NewTeamModel(bufferedPublisher),

			// Persist to db
			persister.NewRedisPersister(redisdb, map[string]string{
				"pincode.view": "pincodes",
				"team.view":    "teams",
			}),
*/
/*
	NewRedisPersister(d.redisdb, map[string]string{
		"user.aggregate": "users",
	}),*/
/*		websync.NewWebsyncModel(d.updater, map[string]string{
			"user.aggregate":              "users",
			"controlgroup.aggregate":      "cg",
			"controlgroupscans.aggregate": "controlgroup",
			"team-klan.aggregate":         "klan",
			"team-patrulje.aggregate":     "patrulje",
			"sos.aggregate":               "sos",
			"spejder.aggregate":           "spejder",
			// "teammembers.aggregate")
		}),
	}
	warn, err := model.Sanitize(models, natschannels)
	if warn != nil {
		log.Print(warn.Error())
	}
	if err != nil {
		log.Fatal(err.Error())
	}

	logHandler := &eventstream.LogHandler{Prefix: "catchup", Mod: 10000}

	start := time.Now()
	log.Printf("Running models...")
	model.Run(subscriber, models, func() {
		logHandler.Prefix = "live"
		logHandler.Mod = 10000

		elapsed := time.Now().Sub(start)
		log.Printf("\n---------------------------------------------------\nAll caught up in: %s\n---------------------------------------------------\n", elapsed.String())
	}, logHandler)

	return d
}
*/
func (d *dims) WaitLive() {
	if d.live {
		return
	}
}

//	<-make(chan bool)
