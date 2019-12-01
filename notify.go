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
	Name         string
	Credentials  string
	Servers      string
	DiscoverSubj string
}

func NewNotifier() *Notifier {
	return &Notifier{
		Name:         notifierName,
		Credentials:  creds,
		Servers:      servers,
		DiscoverSubj: "piper." + notifierName + ".discover",
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

	discover := func() (string, error) {
		timeout, cancel := context.WithTimeout(ctx, 1*time.Second)
		defer cancel()

		log.Debugf("discovering on %s", n.DiscoverSubj)
		reply, err := nc.RequestWithContext(timeout, n.DiscoverSubj, []byte{})
		if err != nil {
			return "", err
		}

		return string(reply.Data), nil
	}

	send := func(target string, text string) error {
		timeout, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()

		log.Debugf("Sending data to %s", target)
		_, err = nc.RequestWithContext(timeout, target, []byte(text))

		return err
	}

	checkAndNotify := func() bool {
		var err error
		var replyTo string

		for {
			replyTo, err = discover()
			if err != nil {
				log.Debugf("discover failed: %s", err)
			}

			if replyTo != "" {
				break
			}

			log.Debugf("Retrying discovery")
		}

		for {
			err = send(replyTo, text)
			if err == nil {
				return true
			}

			log.Debugf("Retrying send: %s", err)
		}
	}

	checkAndNotify()

	return nil
}
