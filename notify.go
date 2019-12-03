package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
)

type Notifier struct {
	Name        string
	Credentials string
	Servers     string
	Subject     string
	Message     string
	Timeout     time.Duration
}

func NewNotifier() *Notifier {
	return &Notifier{
		Name:        name,
		Credentials: creds,
		Servers:     servers,
		Subject:     "piper." + name,
		Message:     notifierMessage,
		Timeout:     notifierTimeout,
	}
}

func (n *Notifier) Notify(ctx context.Context) error {
	timeout, cancel := context.WithTimeout(ctx, n.Timeout)
	defer cancel()

	nc, err := connect(n.Credentials, n.Servers)
	if err != nil {
		return fmt.Errorf("could not connect to NATS: %s", err)
	}

	if n.Message == "" {
		reader := bufio.NewReader(os.Stdin)
		text := make([]byte, reader.Size())
		_, err = reader.Read(text)
		if err != nil {
			return fmt.Errorf("could not read STDIN: %s", err)
		}
		n.Message = string(text)
	}

	for {
		attemptTimeout, attemptCancel := context.WithTimeout(timeout, 2*time.Second)
		defer attemptCancel()

		log.Debugf("Sending %d bytes of data to %s", len(n.Message), n.Subject)
		_, err = nc.RequestWithContext(attemptTimeout, n.Subject, []byte(n.Message))
		if err == context.Canceled || err == context.DeadlineExceeded {
			return fmt.Errorf("notify interrupted: %w", err)
		}

		if err == nil {
			return nil
		}

		log.Debugf("Sending failed, will retry: %s", err)
	}
}
