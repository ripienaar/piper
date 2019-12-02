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
}

func NewNotifier() *Notifier {
	return &Notifier{
		Name:        notifierName,
		Credentials: creds,
		Servers:     servers,
		Subject:     "piper." + notifierName,
	}
}

func (n *Notifier) Notify(ctx context.Context) error {
	nc, err := connect(n.Credentials, n.Servers)
	if err != nil {
		return fmt.Errorf("could not connect to NATS: %s", err)
	}

	reader := bufio.NewReader(os.Stdin)
	text, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("could not read STDIN: %s", err)
	}

	for {
		timeout, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()

		log.Debugf("Sending %d bytes of data to %s", len(text), n.Subject)
		_, err = nc.RequestWithContext(timeout, n.Subject, []byte(text))

		if err == nil {
			return nil
		}
	}
}
