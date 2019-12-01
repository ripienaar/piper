package main

import (
	"context"
	"fmt"
	"os"

	"github.com/nats-io/nats.go"
	log "github.com/sirupsen/logrus"
)

type Listener struct {
	Name         string
	Group        bool
	Credentials  string
	Servers      string
	DiscoverSubj string
	DataSubj     string

	nc    *nats.Conn
	donec chan struct{}
	errc  chan error
}

func NewListener() *Listener {
	var ds string
	if listenGroup {
		ds = nats.NewInbox()
	} else {
		ds = "piper." + listenName + ".data"
	}

	return &Listener{
		Name:         listenName,
		Group:        listenGroup,
		Credentials:  creds,
		Servers:      servers,
		DiscoverSubj: "piper." + listenName + ".discover",
		DataSubj:     ds,
		donec:        make(chan struct{}),
		errc:         make(chan error),
	}
}

func (l *Listener) Listen(ctx context.Context) error {
	var err error

	l.nc, err = connect(l.Credentials, l.Servers)
	if err != nil {
		return fmt.Errorf("could not connect to NATS: %s", err)
	}

	log.Debugf("Listening on %s", l.DataSubj)
	l.nc.Subscribe(l.DataSubj, l.ibHandler)

	log.Debugf("Listening on %s", l.DiscoverSubj)
	dc, _ := l.nc.QueueSubscribe(l.DiscoverSubj, "piper", l.discoverHandler)
	dc.AutoUnsubscribe(1)

	select {
	case <-ctx.Done():
	case <-l.donec:
	case err := <-l.errc:
		fmt.Fprintln(os.Stderr, err)
		l.close()
		os.Exit(1)
	}

	l.close()
	return nil
}

func (l *Listener) close() {
	l.nc.Flush()
	l.nc.Close()
}

func (l *Listener) ibHandler(m *nats.Msg) {
	err := m.Respond([]byte{})
	if err != nil {
		l.errc <- fmt.Errorf("Data response failed: %s", err)
		return
	}

	fmt.Println(string(m.Data))
	l.donec <- struct{}{}
}

func (l *Listener) discoverHandler(m *nats.Msg) {
	if m.Reply == "" {
		l.errc <- fmt.Errorf("no reply subject received on discovery channel")
		return
	}

	log.Debugf("Replying to discovery request")
	err := m.Respond([]byte(l.DataSubj))
	if err != nil {
		l.errc <- fmt.Errorf("Discovery response failed: %s", err)
	}
}
