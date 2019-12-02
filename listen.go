package main

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go"
	log "github.com/sirupsen/logrus"
)

type Listener struct {
	Name        string
	Group       bool
	Credentials string
	Servers     string
	DataSubj    string

	nc    *nats.Conn
	donec chan struct{}
	errc  chan error
}

func NewListener() *Listener {
	return &Listener{
		Name:        listenName,
		Group:       listenGroup,
		Credentials: creds,
		Servers:     servers,
		DataSubj:    "piper." + listenName,
		donec:       make(chan struct{}),
		errc:        make(chan error),
	}
}

func (l *Listener) Listen(ctx context.Context) error {
	var err error

	l.nc, err = connect(l.Credentials, l.Servers)
	if err != nil {
		return fmt.Errorf("could not connect to NATS: %s", err)
	}
	defer l.close()

	if l.Group {
		log.Debugf("Listening on %s in a group", l.DataSubj)
		l.nc.QueueSubscribe(l.DataSubj, "piper", l.ibHandler)
	} else {
		log.Debugf("Listening on %s", l.DataSubj)
		l.nc.Subscribe(l.DataSubj, l.ibHandler)
	}

	select {
	case <-ctx.Done():
	case <-l.donec:
	case err = <-l.errc:
	}

	return err
}

func (l *Listener) close() {
	l.nc.Flush()
	l.nc.Close()
}

func (l *Listener) ibHandler(m *nats.Msg) {
	m.Sub.Unsubscribe()

	err := m.Respond([]byte{})
	if err != nil {
		l.errc <- fmt.Errorf("Data response failed: %s", err)
		return
	}

	fmt.Println(string(m.Data))
	l.donec <- struct{}{}
}
