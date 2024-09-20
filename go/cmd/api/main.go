/*
 * Genarate rsa keys.
 */

package main

import (
	"flag"
	"fmt"
	"os"

	"nathejk.dk/cmd/api/app"
	"nathejk.dk/internal/data"
	"nathejk.dk/internal/jsonlog"
	"nathejk.dk/internal/mailer"
	"nathejk.dk/internal/sms"
	"nathejk.dk/pkg/closers"
	"nathejk.dk/pkg/nats"
	"nathejk.dk/pkg/notification"
	websync "nathejk.dk/pkg/sockethub"
	"nathejk.dk/pkg/sqlstate"
	"nathejk.dk/pkg/streaminterface"

	"github.com/go-redis/redis"
)

// Define a config struct to hold all the configuration settings for our application.
type config struct {
	port    int
	webroot string

	db struct {
		dsn          string
		maxOpenConns int
		maxIdleConns int
		maxIdleTime  string
	}
	redis struct {
		addr string
	}
	jetstream struct {
		dsn string
	}
	stan struct {
		dsn string
	}
	sms struct {
		dsn string
	}
	smtp struct {
		host     string
		port     int
		username string
		password string
		sender   string
	}
}

type application struct {
	app.JsonApi

	config    config
	logger    *jsonlog.Logger
	models    data.Models
	commands  Commands
	mailer    mailer.Mailer
	sms       notification.SmsSender
	publisher streaminterface.Publisher
	jetstream streaminterface.Stream
	state     StateReader
}

func main() {
	fmt.Println("Starting API service")
	var cfg config

	flag.IntVar(&cfg.port, "port", 80, "API server port")
	flag.StringVar(&cfg.webroot, "webroot", getEnv("WEBROOT", "/www"), "Static web root")
	flag.StringVar(&cfg.sms.dsn, "sms-dsn", os.Getenv("SMS_DSN"), "SMS DSN")
	flag.StringVar(&cfg.stan.dsn, "stan-dsn", os.Getenv("STAN_DSN"), "NATS Streaming DSN")
	flag.StringVar(&cfg.jetstream.dsn, "jetstream-dsn", os.Getenv("JETSTREAM_DSN"), "NATS Streaming DSN")
	flag.StringVar(&cfg.redis.addr, "redis-addr", os.Getenv("REDIS_ADDR"), "Redis Address")

	flag.StringVar(&cfg.db.dsn, "db-dsn", os.Getenv("DB_DSN"), "Database DSN")
	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 25, "Database max open connections")
	flag.IntVar(&cfg.db.maxIdleConns, "db-max-idle-conns", 25, "Database max idle connections")
	flag.StringVar(&cfg.db.maxIdleTime, "db-max-idle-time", "15m", "Database max connection idle time")

	flag.StringVar(&cfg.smtp.host, "smtp-host", os.Getenv("SMTP_HOST"), "SMTP host")
	flag.IntVar(&cfg.smtp.port, "smtp-port", getEnvAsInt("SMTP_PORT", 25), "SMTP port")
	flag.StringVar(&cfg.smtp.username, "smtp-username", os.Getenv("SMTP_USERNAME"), "SMTP username")
	flag.StringVar(&cfg.smtp.password, "smtp-password", os.Getenv("SMTP_PASSWORD"), "SMTP password")
	flag.StringVar(&cfg.smtp.sender, "smtp-sender", "Nathejk <hej@nathejk.dk>", "SMTP sender")

	flag.Parse()

	//logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)
	logger := jsonlog.New(os.Stdout, jsonlog.LevelInfo)

	db := NewDatabase(cfg.db)
	if err := db.Open(); err != nil {
		logger.PrintFatal(err, nil)
	}
	defer db.Close()

	closer := closers.New().ExitOnSigInt()
	defer closer.Close()
	/*
		js, err := jetstream.New(cfg.jetstream.dsn)
		if err != nil {
			log.Printf("Error connecting %q", err)
		}

		msg := js.MessageFunc()(streaminterface.SubjectFromStr("NATHEJK:hello"))
		msg.SetBody("world")
		if err := js.Publish(msg); err != nil {
			log.Printf("Error publishing %q", err)
		}
		mux := xstream.NewMux(js)
		mux.AddConsumer(&y{})
		mux.Run(context.Background())
	*/
	natsstream := nats.NewNATSStreamUnique(cfg.stan.dsn, "hq-api")
	//defer natsstream.Close()
	//natsstream := nats.NatsStreamConnectUnique(os.Getenv("STAN_DSN"), "hq-api").Buffered(1000)
	closer.AddCloser(natsstream)

	redisclient := redis.NewClient(&redis.Options{Addr: cfg.redis.addr})
	closer.AddCloser(redisclient)

	hub := websync.NewHub(redisclient)
	state := sqlstate.New(cfg.db.dsn)

	dims := NewDims(natsstream, hub, state)
	dims.Subscribe()
	dims.WaitLive()

	smsclient, _ := sms.NewClient(cfg.sms.dsn)

	//server := NewServer(natsstream, hub, smsclient)

	models := data.NewModels(db.DB())
	app := &application{
		JsonApi: app.JsonApi{
			Logger: logger,
		},
		config:    cfg,
		logger:    logger,
		models:    models,
		commands:  NewCommands(natsstream, models),
		mailer:    mailer.New(cfg.smtp.host, cfg.smtp.port, cfg.smtp.username, cfg.smtp.password, cfg.smtp.sender),
		sms:       smsclient,
		publisher: natsstream,
		state:     hub,
	}

	// Start listening for incoming messages to websocket
	go handleMessages()

	logger.PrintFatal(app.Serve(fmt.Sprintf(":%d", cfg.port), app.routes()), nil)

	//fmt.Println("Running webserver")
	//log.Fatal(server.ListenAndServe(":80"))

}
