package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/mitchellh/go-homedir"
	"github.com/nats-io/nats-server/v2/server"
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

func hasJS(nc *nats.Conn, timeout time.Duration) bool {
	res, err := nc.Request(server.JetStreamEnabled, nil, timeout)

	return err == nil && strings.HasPrefix(string(res.Data), server.OK)
}

func createMessageSet(timeout time.Duration, nc *nats.Conn) error {
	res, err := nc.Request(server.JetStreamMsgSetInfo, []byte("PIPER"), timeout)
	if err != nil {
		return err
	}

	if strings.HasPrefix(string(res.Data), "}") {
		return nil
	}

	cfg := server.MsgSetConfig{
		Name:      "PIPER",
		Subjects:  []string{"piper.ASYNC.>"},
		Retention: server.WorkQueuePolicy,
		MaxAge:    24 * 7 * time.Hour,
		Storage:   server.FileStorage,
	}

	j, err := json.Marshal(&cfg)
	if err != nil {
		return err
	}

	res, err = nc.Request(server.JetStreamCreateMsgSet, j, timeout)
	if err != nil {
		return err
	}

	if strings.HasPrefix(string(res.Data), server.OK) {
		return nil
	}

	return fmt.Errorf("%s", string(res.Data))
}

func createObservable(name string, timeout time.Duration, nc *nats.Conn) error {
	res, err := nc.Request(server.JetStreamObservables, []byte("PIPER"), timeout)
	if err != nil {
		return err
	}

	list := []string{}
	err = json.Unmarshal(res.Data, &list)
	if err != nil {
		return err
	}

	for _, o := range list {
		if o == name {
			log.Debugf("Observable %s already exist, no need to create", name)
			return nil
		}
	}

	cfg := server.CreateObservableRequest{
		MsgSet: "PIPER",
		Config: server.ObservableConfig{
			Delivery:      "",
			Durable:       name,
			DeliverAll:    true,
			AckPolicy:     server.AckExplicit,
			AckWait:       30 * time.Second,
			FilterSubject: asyncName(name),
		},
	}

	j, err := json.Marshal(&cfg)
	if err != nil {
		return err
	}

	res, err = nc.Request(server.JetStreamCreateObservable, j, timeout)
	if err != nil {
		return err
	}

	if strings.HasPrefix(string(res.Data), server.OK) {
		return nil
	}

	return fmt.Errorf("%s", string(res.Data))
}

func asyncName(s string) string {
	return "piper.ASYNC." + s
}

func syncName(s string) string {
	return "piper." + s
}
