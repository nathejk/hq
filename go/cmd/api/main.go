/*
 * Genarate rsa keys.
 */

package main

import (
	"fmt"
	"log"
	"os"

	"nathejk.dk/pkg/closers"
	"nathejk.dk/pkg/cpsms"
	"nathejk.dk/pkg/nats"
	websync "nathejk.dk/pkg/sockethub"
	"nathejk.dk/pkg/sqlstate"

	"github.com/go-redis/redis"
)

func main() {
	fmt.Println("Starting API service")

	closer := closers.New().ExitOnSigInt()
	defer closer.Close()

	natsstream := nats.NewNATSStreamUnique(os.Getenv("STAN_DSN"), "hq-api")
	//defer natsstream.Close()
	//natsstream := nats.NatsStreamConnectUnique(os.Getenv("STAN_DSN"), "hq-api").Buffered(1000)
	closer.AddCloser(natsstream)

	redisclient := redis.NewClient(&redis.Options{Addr: os.Getenv("REDIS_ADDR")})
	closer.AddCloser(redisclient)

	hub := websync.NewHub(redisclient)

	state := sqlstate.New(os.Getenv("MYSQL_DSN"))

	dims := NewDims(natsstream, hub, state)
	dims.Subscribe()
	dims.WaitLive()

	sms := cpsms.New(os.Getenv("CPSMS_API_KEY"))

	server := NewServer(natsstream, hub, sms)
	fmt.Println("Running webserver")
	log.Fatal(server.ListenAndServe(":80"))

}
