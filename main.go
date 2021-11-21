package main

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"time"

	log "github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	piper *kingpin.Application

	debug bool
	async bool
	nctx  string

	name string

	notifier        *kingpin.CmdClause
	notifierMessage string
	notifierTimeout time.Duration

	listener    *kingpin.CmdClause
	listenGroup bool
)

func main() {
	piper = kingpin.New("piper", "Network pipes")
	piper.Flag("context", "NATS context to use for connection").Envar("PIPER_CONTEXT").Default("piper").StringVar(&nctx)
	piper.Flag("async", "Operates asynchronously using JetStream work queues").Envar("PIPER_ASYNC").Short('a').BoolVar(&async)
	piper.Flag("timeout", "How long to wait for a listener to login before giving up").Envar("PIPER_TIMEOUT").DurationVar(&notifierTimeout)
	piper.Flag("debug", "Enable debug logging").BoolVar(&debug)

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
	nc, err := connect(nctx)
	if err != nil {
		return err
	}
	defer nc.Close()

	_, err = createStream(2*time.Second, nc)
	if err != nil {
		return err
	}

	log.Info("Created 'PIPER' Stream")

	return nil
}
