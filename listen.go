package main

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	log "github.com/sirupsen/logrus"
)

type Listener struct {
	Name        string
	Group       bool
	Credentials string
	Servers     string
	DataSubj    string

	nc   *nats.Conn
	errc chan error
}

func NewListener() *Listener {
	return &Listener{
		Name:        name,
		Group:       listenGroup,
		Credentials: creds,
		Servers:     servers,
		DataSubj:    "piper." + name,
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

	if async {
		if !hasJS(l.nc, 2*time.Second) {
			return fmt.Errorf("asynchronous operation required JetStream")
		}

		err = createObservable(l.Name, 2*time.Second, l.nc)
		if err != nil {
			return fmt.Errorf("could not set up observable: %s", err)
		}
	}

	if async {
		go func() {
			res, err := l.nc.Request(server.JetStreamRequestNextPre+".PIPER."+l.Name, []byte("1"), 8760*time.Hour)

			if err != nil {
				l.errc <- fmt.Errorf("async request failed: %s", err)
			}
			l.ibHandler(res)
		}()
	} else if l.Group {
		log.Debugf("Listening on %s in a work group", l.DataSubj)
		l.nc.QueueSubscribe(l.DataSubj, "piper", l.ibHandler)
	} else {
		log.Debugf("Listening on %s", l.DataSubj)
		l.nc.Subscribe(l.DataSubj, l.ibHandler)
	}

	select {
	case <-ctx.Done():
	case err = <-l.errc:
	}

	return err
}

func (l *Listener) close() {
	l.nc.Flush()
	l.nc.Close()
}

func (l *Listener) ibHandler(m *nats.Msg) {
	if !async {
		m.Sub.Unsubscribe()
	}

	err := m.Respond([]byte{})
	if err != nil {
		l.errc <- fmt.Errorf("acknowledgement failed: %s", err)
		return
	}

	body, err := decompress(m.Data)
	if err != nil {
		l.errc <- fmt.Errorf("decompression failed: %s", err)
	}

	fmt.Print(body)

	l.errc <- nil
}
