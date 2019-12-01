package main

import (
	"context"
	"fmt"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
	"github.com/sirupsen/logrus"
	"os"
)

var (
	piper *kingpin.Application

	creds   string
	servers string
	debug   bool

	notifier     *kingpin.CmdClause
	notifierName string

	listener    *kingpin.CmdClause
	listenName  string
	listenGroup bool
)

func main() {
	piper = kingpin.New("piper", "Network pipes")
	piper.Flag("creds", "NATS credentials").Envar("PIPER_CREDENTIALS").StringVar(&creds)
	piper.Flag("servers", "NATS servers").Envar("PIPER_SERVERS").StringVar(&servers)
	piper.Flag("debug", "Enable debug logging").BoolVar(&debug)

	listener = piper.Command("listen", "Listen for messages on the pipe")
	listener.Arg("name", "Pipe name to wait on for a message").Required().StringVar(&listenName)
	listener.Flag("group", "Listen on a group").BoolVar(&listenGroup)

	notifier = piper.Command("notify", "Notifies listeners")
	notifier.Arg("name", "Pipe name to publish a message to").Required().StringVar(&notifierName)

	command := kingpin.MustParse(piper.Parse(os.Args[1:]))
	var err error

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if debug {
		logrus.SetLevel(logrus.DebugLevel)
	}
	
	switch command {
	case "listen":
		err = NewListener().Listen(ctx)

	case "notify":
		err = NewNotifier().Notify(ctx)

	default:
		err = fmt.Errorf("Invalid command %s", command)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
		os.Exit(1)
	}
}
