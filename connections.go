package main

import (
	"errors"
	"net"
	"sync"
	"time"
)

var connectionsMaxError = errors.New("Connections max reached")
var connectionsEmptyError = errors.New("Connections empty")
var connectionClosedError = errors.New("Connection closed")

// Connections keep a list of active client connections as connector instances.
// Oldest is at the head of the list.
type connections struct {
	mu      *sync.Mutex
	head    *connector
	tail    *connector
	len     int
	max     int
	timeout time.Duration
}

// Create a new connections list with max size and timeout duration
// of before an connector can be removed if queue is full and there
// is an incoming connection.
func newConnections(max int, timeout time.Duration) *connections {
	// create dummy head and tail to simplify adding and removing
	front, back := &connector{}, &connector{}
	front.next, back.prev = back, front

	return &connections{
		mu:      &sync.Mutex{},
		head:    front,
		tail:    back,
		max:     max,
		timeout: timeout,
	}
}

// Creates and adds a connector to the list from an active client connection.
func (cs *connections) add(conn net.Conn) (*connector, error) {
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

	cr := &connector{
		uid:       counter,
		netConn:   conn,
		prev:      cs.tail.prev,
		next:      cs.tail,
		timestamp: time.Now(),
		done:      make(chan bool, 2),
	}
	counter++
	cs.tail.prev.next = cr
	cs.tail.prev = cr
	cs.len++

	return cr, nil
}

// Removes connector from list when client connection is done.
func (cs *connections) remove(cr *connector) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if cs.len == 0 {
		// This should never happen
		return connectionsEmptyError
	}

	if cr.prev != nil {
		cr.finish()
		cs.len--
	}

	return nil
}
