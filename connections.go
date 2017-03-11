package main

import (
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

var connectionsMaxError = errors.New("Connections max reached")
var connectionsEmptyError = errors.New("Connections empty")
var connectionClosedError = errors.New("Connection closed")
var counter uint64 = 1

type request []byte

func (r request) isPop() bool {
	return (r[0] >> 7) == 1
}

func (r request) isPush() bool {
	return (r[0] >> 7) == 0
}

type connector struct {
	uid       uint64
	netConn   net.Conn
	next      *connector
	prev      *connector
	timestamp time.Time
	waiting   bool
	done      chan bool
}

func (ce *connector) duration() time.Duration {
	return time.Since(ce.timestamp)
}

func (ce *connector) disconnect() {
	ce.next.prev, ce.prev.next = ce.prev, ce.next
	ce.prev, ce.next = nil, nil
}

func (ce *connector) disconnected() bool {
	return ce.next == nil
}

func (ce *connector) read(data []byte) (int, error) {
	if ce.netConn == nil {
		return 0, nil
	}
	return ce.netConn.Read(data)
}

func (ce *connector) writeall(data []byte) (int, error) {
	if ce.netConn == nil {
		return 0, nil
	}

	nwrite := 0
	var err error
	for n := 0; nwrite < len(data); nwrite += n {
		n, err = ce.netConn.Write(data)
		if err != nil {
			return nwrite, err
		}
	}
	return nwrite, nil
}

func (ce *connector) finish() error {
	ce.disconnect()
	err := ce.netConn.Close()
	ce.done <- true // Ensure connection is closed
	return err
}

func (ce *connector) log(line string, args ...interface{}) {
	if logger != nil {
		output := fmt.Sprintf(line, args...)
		output = fmt.Sprintf("[c%d] %s", ce.uid, output)
		fmt.Fprintln(logger, "[StackServer]", output)
	}
}

type Connections struct {
	mu      *sync.Mutex
	head    *connector
	tail    *connector
	len     int
	max     int
	timeout time.Duration
}

func newConnections(max int, timeout time.Duration) *Connections {
	// create dummy head and tail to simplify adding and removing
	front, back := &connector{}, &connector{}
	front.next, back.prev = back, front

	return &Connections{
		mu:      &sync.Mutex{},
		head:    front,
		tail:    back,
		max:     max,
		timeout: timeout,
	}
}

func (cs *Connections) add(conn net.Conn) (*connector, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if cs.len == cs.max {
		oldest := cs.head.next
		if oldest.duration() > cs.timeout {
			oldest.finish()
			oldest.log("removed because oldest")
			cs.len--
		} else {
			return nil, connectionsMaxError
		}
	}

	newConn := &connector{
		uid:       counter,
		netConn:   conn,
		prev:      cs.tail.prev,
		next:      cs.tail,
		timestamp: time.Now(),
		done:      make(chan bool, 2),
	}
	counter++
	cs.tail.prev.next = newConn
	cs.tail.prev = newConn
	cs.len++

	return newConn, nil
}

func (cs *Connections) remove(c *connector) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if cs.len == 0 {
		// This should never happen
		return connectionsEmptyError
	}

	if c.prev != nil {
		c.finish()
		cs.len--
	}

	return nil
}

func (cs *Connections) scanClosed() {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	one := []byte{0x00}
	allClosed := []*connector{}
	for c := cs.head.next; c != cs.tail; c = c.next {
		if c.waiting {
			allClosed = append(allClosed, c)
		}
	}

	for _, c := range allClosed {
		c.netConn.SetReadDeadline(time.Now().Add(time.Millisecond))
		_, err := c.netConn.Read(one)
		if err == io.EOF {
			c.done <- true
			c.log("Client closed")
		}
	}
}