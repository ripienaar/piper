package main

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/jsm.go"
	"github.com/nats-io/nats.go"
	log "github.com/sirupsen/logrus"
)

type Listener struct {
	Name     string
	Group    bool
	Context  string
	DataSubj string

	nc   *nats.Conn
	errc chan error
}

func NewListener() *Listener {
	return &Listener{
		Name:     name,
		Group:    listenGroup,
		Context:  nctx,
		DataSubj: "piper." + name,
		errc:     make(chan error),
	}
}

func (l *Listener) Listen(ctx context.Context) error {
	var err error

	l.nc, err = connect(l.Context)
	if err != nil {
		return fmt.Errorf("could not connect to NATS: %s", err)
	}
	defer l.close()

	if async {
		if !hasJS(l.nc, 2*time.Second) {
			return fmt.Errorf("asynchronous operation required JetStream")
		}

		err = createConsumer(l.Name, 2*time.Second, l.nc)
		if err != nil {
			return fmt.Errorf("could not set up observable: %s", err)
		}
	}

	switch {
	case async:
		go func() {
			mgr, err := jsm.New(l.nc)
			if err != nil {
				l.errc <- err
				return
			}

			for {
				stop, err := func() (bool, error) {
					log.Debugf("Fetching 1 message from JetStream")
					rctx, cancel := context.WithTimeout(ctx, time.Minute)
					defer cancel()
					msg, err := mgr.NextMsgContext(rctx, "PIPER", l.Name)
					if err != nil {
						log.Errorf("Getting next message failed: %s", err)
						return false, nil
					}
					l.ibHandler(msg)
					msg.Ack()
					return true, nil
				}()
				if err != nil {
					l.errc <- err
					return
				}
				if stop {
					return
				}
			}
		}()

	case l.Group:
		log.Debugf("Listening on %s in a work group", l.DataSubj)
		_, err := l.nc.QueueSubscribe(l.DataSubj, "piper", l.ibHandler)
		if err != nil {
			l.errc <- err
		}

	default:
		log.Debugf("Listening on %s", l.DataSubj)
		_, err := l.nc.Subscribe(l.DataSubj, l.ibHandler)
		if err != nil {
			l.errc <- err
		}
	}

	select {
	case <-ctx.Done():
	case err = <-l.errc:
	}

	return err
}

func (l *Listener) close() {
	err := l.nc.Flush()
	if err != nil {
		log.Warnf("Could not flush NATS connection: %s", err)
	}
	l.nc.Close()
}

func (l *Listener) ibHandler(m *nats.Msg) {
	if !async {
		err := m.Sub.Unsubscribe()
		if err != nil {
			log.Warnf("Could not unsubscribe from data subject: %s", err)
		}
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
