package main

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"time"

	log "github.com/sirupsen/logrus"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	piper *kingpin.Application

	creds   string
	servers string
	ngs     bool
	debug   bool
	async   bool

	name string

	notifier        *kingpin.CmdClause
	notifierMessage string
	notifierTimeout time.Duration

	listener    *kingpin.CmdClause
	listenGroup bool
)

func main() {
	piper = kingpin.New("piper", "Network pipes")
	piper.Flag("creds", "NATS credentials").Envar("PIPER_CREDENTIALS").StringVar(&creds)
	piper.Flag("servers", "NATS servers").Envar("PIPER_SERVERS").StringVar(&servers)
	piper.Flag("ngs", "Use Synadia NGS").Envar("PIPER_NGS").BoolVar(&ngs)
	piper.Flag("debug", "Enable debug logging").BoolVar(&debug)
	piper.Flag("async", "Operates asynchronously using JetStream work queues").Envar("PIPER_ASYNC").Short('a').BoolVar(&async)
	piper.Flag("timeout", "How long to wait for a listener to login before giving up").Envar("PIPER_TIMEOUT").DurationVar(&notifierTimeout)

	listener = piper.Command("listen", "Listen for messages on the pipe")
	listener.Arg("name", "Pipe name to wait on for a message").Required().StringVar(&name)
	listener.Flag("group", "Listen on a group").BoolVar(&listenGroup)

	notifier = piper.Command("notify", "Notifies listeners").Default()
	notifier.Arg("name", "Pipe name to publish a message to").Required().StringVar(&name)
	notifier.Arg("message", "The message to sent, reads STDIN otherwise").StringVar(&notifierMessage)

	piper.Command("setup", "Creates JetStream configuration to enable asynchronous functionality")

	command := kingpin.MustParse(piper.Parse(os.Args[1:]))
	var err error

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.SetOutput(os.Stderr)
	if debug {
		log.SetLevel(log.DebugLevel)
	}

	if ngs && servers == "" {
		servers = "connect.ngs.global:4222"
	}

	switch command {
	case "listen":
		err = NewListener().Listen(ctx)

	case "notify":
		err = NewNotifier().Notify(ctx)

	case "setup":
		err = asyncSetup()

	default:
		err = fmt.Errorf("invalid command %s", command)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to run: %v\n", err)
		cancel()
		runtime.Goexit()
	}
}

func asyncSetup() error {
	nc, err := connect(creds, servers)
	if err != nil {
		return err
	}
	defer nc.Close()

	err = createMessageSet(2*time.Second, nc)
	if err != nil {
		return err
	}

	log.Info("Created 'PIPER' Message Set")

	return nil
}
