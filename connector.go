package main

import (
	"fmt"
	"net"
	"time"
)

var counter uint64 = 1

// Connector struct represents an active client connection.
type connector struct {
	uid       uint64
	netConn   net.Conn
	next      *connector
	prev      *connector
	timestamp time.Time
	done      chan bool
}

// Duration since client connection was created.
func (cr *connector) duration() time.Duration {
	return time.Since(cr.timestamp)
}

// Read bytes from client connection.
func (cr *connector) read(data []byte) (int, error) {
	if cr.netConn == nil {
		return 0, nil
	}

	return cr.netConn.Read(data)
}

// Write all given bytes to client connection.
func (cr *connector) writeall(data []byte) (int, error) {
	if cr.netConn == nil {
		return 0, nil
	}

	nwrite := 0
	var err error
	for n := 0; nwrite < len(data); nwrite += n {
		n, err = cr.netConn.Write(data)
		if err != nil {
			return nwrite, err
		}
	}

	return nwrite, nil
}

// Finishes connection to client.
func (cr *connector) finish() error {
	var err error
	cr.next.prev, cr.prev.next = cr.prev, cr.next
	cr.prev, cr.next = nil, nil

	if cr.netConn != nil {
		err = cr.netConn.Close()
	}
	cr.done <- true // Ensure connection is closed

	return err
}

// Log to UDP port if configured.
func (cr *connector) log(line string, args ...interface{}) {
	if logger != nil {
		logger.Printf("[c%d] %s\n", cr.uid, fmt.Sprintf(line, args...))
	}
}
