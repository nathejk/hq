package main

import (
	"context"
	"expvar"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"time"

	// Embed the timezone database in the binary. The code calls
	// time.LoadLocation("Europe/Copenhagen") in several places, and the production
	// image has no system tzdata — without this those calls fail and every parsed
	// date silently becomes the zero time.
	_ "time/tzdata"

	"github.com/jrgensen/cqrs"
	"github.com/jrgensen/cqrs/sqlpersister"
	"github.com/jrgensen/stream"
	"github.com/jrgensen/stream/jetstream"
	"github.com/jrgensen/stream/metatagger"
	"github.com/jrgensen/stream/xstream"
	"github.com/nathejk/shared-go/tables/crewmember"
	"github.com/nathejk/shared-go/tables/klan"
	"github.com/nathejk/shared-go/tables/order"
	"github.com/nathejk/shared-go/tables/payment"
	"github.com/nathejk/shared-go/tables/product"
	"github.com/nathejk/shared-go/tables/section"
	"github.com/nathejk/shared-go/tables/senior"
	"github.com/nathejk/shared-go/tables/signup"
	"github.com/nathejk/shared-go/tables/spejder"
	"github.com/nathejk/shared-go/tables/vehicle"
	"github.com/nathejk/shared-go/types"
	"nathejk.dk/cmd/api/app"
	"nathejk.dk/internal/data"
	"nathejk.dk/internal/jsonlog"
	"nathejk.dk/internal/live"
	"nathejk.dk/internal/mailer"
	"nathejk.dk/internal/sms"
	"nathejk.dk/internal/vcs"
	"nathejk.dk/nathejk/commands"
	"nathejk.dk/nathejk/table"
	"nathejk.dk/nathejk/table/checkgroup"
	"nathejk.dk/nathejk/table/checkpersonnel"
	"nathejk.dk/nathejk/table/checkpoint"
	"nathejk.dk/nathejk/table/lok"
	"nathejk.dk/nathejk/table/patrulje"
	"nathejk.dk/nathejk/table/patruljemerged"
	"nathejk.dk/nathejk/table/patruljenumber"
	"nathejk.dk/nathejk/table/personnel"
	"nathejk.dk/nathejk/table/scan"
	"nathejk.dk/nathejk/table/sos"
	"nathejk.dk/nathejk/table/spejderstatus"
	"nathejk.dk/nathejk/table/year"
)

var (
	version = vcs.Version()
)

// Define a config struct to hold all the configuration settings for our application.
type config struct {
	port      int
	webroot   string
	baseurl   string
	countdown struct {
		time   string
		videos []string
	}
	db struct {
		dsn          string
		maxOpenConns int
		maxIdleConns int
		maxIdleTime  string
	}
	jetstream struct {
		dsn string
	}
	sms struct {
		dsn string
	}
	smtp mailer.Config
}

type application struct {
	app.JsonApi

	db        *database
	config    config
	models    data.Models
	publisher stream.Publisher
	commands  commands.Commands
	mailer    mailer.Mailer
	sms       sms.Sender
	logger    *jsonlog.Logger

	// live fans read-model changes out to connected browsers. Not configuration,
	// so it is a dependency rather than a config field.
	live *live.Hub

	// liveEntities is the set of entity tokens the stream can emit, advertised to
	// each client so a mistyped or invented dependency can be reported instead of
	// failing silently. Derived from the wired consumers; see task 040.
	liveEntities *live.EntitySet
}

func main() {
	var cfg config

	flag.IntVar(&cfg.port, "port", 80, "API server port")
	flag.StringVar(&cfg.webroot, "webroot", getEnv("WEBROOT", "/www"), "Static web root")
	flag.StringVar(&cfg.baseurl, "baseurl", getEnv("BASEURL", "https://tilmelding.nathejk.dk"), "Base url of website")

	flag.StringVar(&cfg.sms.dsn, "sms-dsn", os.Getenv("SMS_DSN"), "SMS DSN")
	flag.StringVar(&cfg.jetstream.dsn, "jetstream-dsn", os.Getenv("JETSTREAM_DSN"), "NATS Streaming DSN")

	flag.StringVar(&cfg.db.dsn, "db-dsn", os.Getenv("DB_DSN"), "Database DSN")
	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 25, "Database max open connections")
	flag.IntVar(&cfg.db.maxIdleConns, "db-max-idle-conns", 25, "Database max idle connections")
	flag.StringVar(&cfg.db.maxIdleTime, "db-max-idle-time", "15m", "Database max connection idle time")

	flag.StringVar(&cfg.smtp.Host, "smtp-host", os.Getenv("SMTP_HOST"), "SMTP host")
	flag.IntVar(&cfg.smtp.Port, "smtp-port", getEnvAsInt("SMTP_PORT", 25), "SMTP port")
	flag.StringVar(&cfg.smtp.Username, "smtp-username", os.Getenv("SMTP_USERNAME"), "SMTP username")
	flag.StringVar(&cfg.smtp.Password, "smtp-password", os.Getenv("SMTP_PASSWORD"), "SMTP password")
	flag.StringVar(&cfg.smtp.Sender, "smtp-sender", "Nathejk <kontakt@nathejk.dk>", "SMTP sender")

	flag.StringVar(&cfg.countdown.time, "countdown", getEnv("COUNTDOWN", ""), "Time for countdown")
	cfg.countdown.videos = getEnvAsSlice("COUNTDOWN_VIDEOS", []string{}, "\n")

	flag.Parse()

	//logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)
	logger := jsonlog.New(os.Stdout, jsonlog.LevelInfo)
	logger.PrintInfo("Starting API...", nil)

	js, err := jetstream.New(cfg.jetstream.dsn)
	if err != nil {
		log.Printf("Error connecting %q", err)
		return
	}
	logger.PrintInfo("Jetstream connected", nil)
	// jrgensen/stream's jetstream.New takes only a URL; the default publish
	// metadata that superfluids' jetstream applied via SetDefaultMeta is now
	// provided by the metatagger publisher decorator (per-message meta wins).
	publisher, err := metatagger.New(js, map[string]any{"producer": "hq-api", "version": "1234"})
	if err != nil {
		logger.PrintFatal(err, nil)
	}
	/*msg, err := js.LastMessage(subject.FromStr("NATHEJK.2024.>"))
	if err != nil {
		log.Fatalf("Last message: %q", err)
	}
	log.Printf("Last message (%d) %v", msg.Sequence(), msg)
	*/

	db := NewDatabase(cfg.db)
	if err := db.Open(); err != nil {
		logger.PrintFatal(err, nil)
	}
	defer db.Close()
	logger.PrintInfo("Database connected", nil)

	reader := db.DB()
	writer := sqlpersister.New(db.DB())
	currentYear := types.YearSlug(fmt.Sprintf("%d", time.Now().Year()))

	year := year.New(publisher, writer, reader)
	signuptable := signup.New(publisher, writer, db.DB())
	klantable := klan.New(publisher, writer, reader)
	seniortable := senior.New(writer, db.DB())
	patruljetable := patrulje.New(writer, db.DB())
	patruljemergedtable := patruljemerged.New(writer, db.DB())
	personneltable := personnel.New(writer, db.DB())
	paymenttable := payment.New(publisher, writer, db.DB(), currentYear)
	spejdertable := spejder.New(writer, db.DB())
	spejderstatustable := spejderstatus.New(writer, db.DB())
	checkgroup := checkgroup.New(publisher, writer, reader)
	checkpoint := checkpoint.New(publisher, writer, reader)
	checkpersonnel := checkpersonnel.New(publisher, writer, reader)
	scantable := scan.New(writer, db.DB())
	loktable := lok.New(writer, db.DB())
	sectiontable := section.New(publisher, writer, db.DB())
	crewmembertable := crewmember.New(publisher, writer, db.DB())
	vehicletable := vehicle.New(publisher, writer, db.DB())
	sostable := sos.New(publisher, writer, db.DB())
	producttable := product.New(writer, db.DB())
	if err := producttable.Seed(product.Seeds2026()); err != nil {
		logger.PrintFatal(err, nil)
	}
	ordertable := order.New(publisher, writer, db.DB(), currentYear, producttable)
	// order.NewSaga is deliberately NOT wired here. The Pay saga publishes
	// order.paid in reaction to payment.received, and tilmelding already runs it.
	//
	// Projectors are safe to mount in several services — each writes only its own
	// read model. A saga is not: it writes to the shared event log. Subscriptions
	// here are ephemeral ordered consumers with no queue group, so every process
	// receives every message rather than sharing them out, and the saga's
	// "transition only if still open" check is a read-then-publish with no
	// compare-and-swap. Two instances would therefore both read `open` and both
	// publish, and nothing sets Nats-Msg-Id, so JetStream's duplicate window would
	// not collapse them either.
	//
	// Today that would still converge, because the only subscriber to order.paid
	// is the order projector and its UPDATE is conditional on status='open'. But
	// it would impose that same idempotency on every future subscriber — the
	// patrulje-number assignment in PRD 003 being the next one — for no benefit
	// beyond redundancy nobody asked for.
	//
	// tilmelding owns it because it owns the payment lifecycle: it creates the
	// payments and takes the provider callback. hq wires no payment provider and
	// only reads orders and payments, so it has no business transitioning them.

	// The patrulje number saga is the mirror image: hq *is* its sole owner (PRD
	// 003). Numbering is an organizer concern, and hq holds the patrulje read
	// model the eligibility check and the seeding read from. The single-owner rule
	// binds harder here than it does for the Pay saga — the patrulje projector's
	// `UPDATE patrulje SET teamNumber=?` is unconditional, so two mounts would not
	// converge on one number, they would overwrite each other. It must not be added
	// to tilmelding, and hq must not run two replicas.
	//
	// It goes in the `projections` slice rather than straight onto the mux because
	// that slice is what live.NotifyAll wraps, and the wrapper is what forwards
	// CaughtUp. Outside the slice the saga would never learn it was live and would
	// silently publish nothing at all.
	patruljenumbers := patruljenumber.New(publisher, ordertable, producttable, patruljetable, currentYear, 0)

	mux := xstream.NewMux(js)

	// Live updates: every projection is wrapped so that applying an event also
	// tells connected browsers to refetch. One decorator makes the whole SPA live —
	// betalinger, patruljer, klaner, poster — with no per-page backend code.
	//
	// The wrapping happens after HandleMessage succeeds, so a signal can never
	// precede the write it announces — and a projection that fails announces
	// nothing. See internal/live/notify.go for what would change if a deadletter
	// Writer were introduced.
	livehub := live.NewHub()
	defer livehub.Close()

	// Named rather than inlined into AddConsumer: this list is the read model, and
	// it is worth being able to see it.
	//
	// Everything here is wrapped, including consumers that are arguably not
	// projections (order's saga behaviour). A consumer that writes no row a client
	// would refetch produces a signal nothing depends on — clients declare the
	// entities they care about, and coalescing collapses duplicates from the
	// several projections that handle one event — so curating the list would buy
	// nothing and would rot the moment a consumer changed shape.
	projections := []cqrs.Consumer{
		signuptable,
		klantable,
		seniortable,
		patruljetable,
		patruljenumbers,
		table.NewPatruljeStatus(writer),
		spejderstatustable,
		personneltable,
		paymenttable,
		spejdertable,
		checkgroup,
		checkpoint,
		checkpersonnel,
		scantable,
		patruljemergedtable,
		loktable,
		year,
		sectiontable,
		crewmembertable,
		vehicletable,
		ordertable,
		sostable,
	}
	for _, consumer := range live.NotifyAll(livehub, projections...) {
		mux.AddConsumer(consumer)
	}

	// Which entity tokens the stream can possibly emit — derived from the same slice,
	// so the advertisement cannot describe a different stream from the one served.
	// The SPA uses it to warn (in dev) about a dependency nothing can ever satisfy.
	liveEntities := live.EntitiesFrom(projections...)
	logger.PrintInfo("Live entities advertised", map[string]string{
		"entities":   strings.Join(liveEntities.Entities, ","),
		"exhaustive": fmt.Sprintf("%t", liveEntities.Exhaustive),
	})

	if err := mux.Run(context.Background()); err != nil {
		logger.PrintFatal(err, nil)
	}

	models := data.NewModels(db.DB(), year, klantable, seniortable, patruljetable, personneltable, paymenttable, checkgroup, checkpoint, checkpersonnel, scantable, loktable, sectiontable, crewmembertable, vehicletable, ordertable, sostable)
	cmds := commands.New(publisher, models)
	cmds.Year = year
	cmds.Checkpoint = checkpoint
	cmds.Checkgroup = checkgroup
	cmds.Checkpersonnel = checkpersonnel
	cmds.Section = sectiontable
	cmds.CrewMember = crewmembertable
	cmds.Vehicle = vehicletable
	cmds.Sos = sostable

	expvar.NewString("version").Set(version)
	expvar.NewInt("timestamp").Set(time.Now().Unix())
	expvar.NewInt("goroutines").Set(int64(runtime.NumGoroutine()))

	smsclient, err := sms.NewClient(cfg.sms.dsn)
	if err != nil {
		logger.PrintFatal(err, nil)
	}

	app := &application{
		JsonApi: app.JsonApi{
			Logger: logger,
		},
		db:           db,
		config:       cfg,
		models:       models,
		publisher:    publisher,
		commands:     cmds,
		mailer:       mailer.NewFromConfig(cfg.smtp),
		sms:          smsclient,
		logger:       logger,
		live:         livehub,
		liveEntities: &liveEntities,
	}
	logger.PrintInfo("Application initialized", nil)

	logger.PrintFatal(app.Serve(fmt.Sprintf(":%d", cfg.port), app.routes()), nil)
}
