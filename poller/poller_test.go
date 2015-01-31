package poller

import (
	"strconv"
	"testing"
	"time"
)

func TestPollerManager(t *testing.T) {

	mgr := NewPollerShutdownManager()

	for i := 1; i < 10; i++ {
		p := TestPoller{strconv.FormatInt(int64(i), 10), make(chan bool, 1), make(chan bool, 1)}
		mgr.Register(p.name, p.stop, p.stopAck)
		mgr.Deregister(p.name)
	}

	for i := 1; i < 10; i++ {
		p := TestPoller{strconv.FormatInt(int64(i), 10), make(chan bool, 1), make(chan bool, 1)}
		go p.eventLoop()
		mgr.Register(p.name, p.stop, p.stopAck)
	}

	shutdown := make(chan struct{})
	go func() {
		mgr.StopPollers()
		shutdown <- struct{}{}
	}()

	select {
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting on shtutdown")
	case <-shutdown:

	}

}

type TestPoller struct {
	name    string
	stop    chan bool
	stopAck chan bool
}

func (t *TestPoller) eventLoop() {
	<-t.stop
	t.stopAck <- true
}
