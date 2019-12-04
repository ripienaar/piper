package main

import (
	"bytes"
	"compress/gzip"
	"io/ioutil"

	"github.com/mitchellh/go-homedir"
	nats "github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"

	"os"
)

func connect(credentials string, servers string) (*nats.Conn, error) {
	errh := func(nc *nats.Conn, sub *nats.Subscription, err error) {
		if sub != nil {
			logrus.Errorf("async error for sub [%s]: %v", sub.Subject, err)
			os.Exit(1)
		} else {
			logrus.Errorf("async error: %v", err)
			os.Exit(1)
		}
	}

	closedh := func(nc *nats.Conn) {
		err := nc.LastError()
		if err != nil {
			logrus.Errorf("NATS connection closed: %v", err)
			os.Exit(1)
		}
	}

	connh := func(nc *nats.Conn) {
		err := nc.LastError()
		if err == nil {
			logrus.Debugf("Connected to %s", nc.ConnectedUrl())
		}
	}

	opts := []nats.Option{
		nats.MaxReconnects(100),
		nats.NoEcho(),
		nats.ErrorHandler(errh),
		nats.ClosedHandler(closedh),
		nats.ReconnectHandler(connh),
	}

	if credentials != "" {
		opts = append(opts, nats.UserCredentials(credentials))
	} else {
		pcreds, err := homedir.Expand("~/.piper.creds")
		if err == nil {
			if fileExist(pcreds) {
				log.Debugf("Using credentials in %s", pcreds)
				opts = append(opts, nats.UserCredentials(pcreds))
			}
		}
	}

	nc, err := nats.Connect(servers, opts...)
	if err != nil {
		return nil, err
	}

	log.Debugf("Connected to %s", nc.ConnectedUrl())

	return nc, err
}

func fileExist(f string) bool {
	_, err := os.Stat(f)
	return !os.IsNotExist(err)
}

func decompress(data []byte) (string, error) {
	b := bytes.NewBuffer(data)
	zr, err := gzip.NewReader(b)
	if err != nil {
		return "", err
	}

	d, err := ioutil.ReadAll(zr)
	if err != nil {
		return "", err
	}

	return string(d), nil
}

func compress(data string) ([]byte, error) {
	var b bytes.Buffer

	gz := gzip.NewWriter(&b)

	_, err := gz.Write([]byte(data))
	if err != nil {
		return []byte{}, err
	}

	err = gz.Flush()
	if err != nil {
		return []byte{}, err
	}

	err = gz.Close()
	if err != nil {
		return []byte{}, err
	}

	return b.Bytes(), nil
}
