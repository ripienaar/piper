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
}

func NewNotifier() *Notifier {
	return &Notifier{
		Name:        notifierName,
		Credentials: creds,
		Servers:     servers,
		Subject:     "piper." + notifierName,
		Message:     notifierMessage,
	}
}

func (n *Notifier) Notify(ctx context.Context) error {
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
		timeout, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()

		log.Debugf("Sending %d bytes of data to %s", len(n.Message), n.Subject)
		_, err = nc.RequestWithContext(timeout, n.Subject, []byte(n.Message))

		if err == nil {
			return nil
		}
	}
}
