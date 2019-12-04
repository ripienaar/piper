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
	if n.Timeout == 0 {
		n.Timeout = 1 * time.Hour
	}

	log.Debugf("Publishing to %s with a timeout of %v", n.Name, n.Timeout)

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

	compressed, err := compress(n.Message)
	if err != nil {
		return fmt.Errorf("compression failed: %s", err)
	}

	for {
		attemptTimeout, attemptCancel := context.WithTimeout(timeout, 2*time.Second)
		defer attemptCancel()

		log.Debugf("Sending %d bytes of data compressed to %d on subject %s", len(n.Message), len(compressed), n.Subject)
		_, err = nc.RequestWithContext(attemptTimeout, n.Subject, compressed)
		if err == nil {
			return nil
		}

		if err != context.Canceled && err != context.DeadlineExceeded {
			log.Errorf("notification failed, will retry in a second: %s", err)
			time.Sleep(time.Second)
			continue
		}

		err = timeout.Err()
		if err != nil {
			return fmt.Errorf("timeout after %v", n.Timeout)
		}
	}
}
