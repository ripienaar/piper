package main

import (
	"fmt"
	nats "github.com/nats-io/nats.go"
	"os"
)

func connect(credentials string, servers string) (*nats.Conn, error) {
	errh := func(nc *nats.Conn, sub *nats.Subscription, err error) {
		if sub != nil {
			fmt.Fprintf(os.Stderr, "async error for sub [%s]: %v", sub.Subject, err)
			os.Exit(1)
		} else {
			fmt.Fprintf(os.Stderr, "async error: %v", err)
			os.Exit(1)
		}
	}

	closedh := func(nc *nats.Conn) {
		err := nc.LastError()
		if err != nil {
			fmt.Fprintf(os.Stderr, "NATS connection closed: %v", err)
			os.Exit(1)
		}
	}

	opts := []nats.Option{
		nats.MaxReconnects(100),
		nats.NoEcho(),
		nats.ErrorHandler(errh),
		nats.ClosedHandler(closedh),
	}

	if credentials != "" {
		opts = append(opts, nats.UserCredentials(credentials))
	}

	return nats.Connect(servers, opts...)
}
