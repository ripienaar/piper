package main

import (
	"bytes"
	"compress/gzip"
	"io/ioutil"
	"os"
	"time"

	"github.com/nats-io/jsm.go"
	"github.com/nats-io/jsm.go/natscontext"
	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

func connect(nctx string) (*nats.Conn, error) {
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

	nc, err := natscontext.Connect(nctx, nats.MaxReconnects(100),
		nats.NoEcho(),
		nats.ErrorHandler(errh),
		nats.ClosedHandler(closedh),
		nats.ReconnectHandler(connh),
		nats.UseOldRequestStyle(),
	)

	log.Debugf("Connected to %s", nc.ConnectedUrl())

	return nc, err
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
	mgr, err := jsm.New(nc, jsm.WithTimeout(timeout))
	if err != nil {
		return false
	}

	_, err = mgr.JetStreamAccountInfo()
	return err == nil
}

func createStream(timeout time.Duration, nc *nats.Conn) (*jsm.Stream, error) {
	mgr, err := jsm.New(nc, jsm.WithTimeout(timeout))
	if err != nil {
		return nil, err
	}

	return mgr.LoadOrNewStream("PIPER",
		jsm.Subjects("piper.ASYNC.>"),
		jsm.WorkQueueRetention(),
		jsm.MaxAge(24*time.Hour),
		jsm.FileStorage())
}

func createConsumer(name string, timeout time.Duration, nc *nats.Conn) error {
	stream, err := createStream(timeout, nc)
	if err != nil {
		return err
	}

	_, err = stream.NewConsumer(
		jsm.DurableName(name),
		jsm.DeliverAllAvailable(),
		jsm.AcknowledgeExplicit(),
		jsm.AckWait(30*time.Second),
		jsm.FilterStreamBySubject(asyncName(name)),
	)

	return err
}

func asyncName(s string) string {
	return "piper.ASYNC." + s
}

func syncName(s string) string {
	return "piper." + s
}
