package main

import (
	"testing"
	"time"
)

func TestSimpleConnections(t *testing.T) {
	cs := newConnections(100, 2*time.Second)
	Assert(t, (cs.max == 100), "Expected max 100:", cs.max)

	c1, _ := cs.add(nil)
	c2, _ := cs.add(nil)
	c3, _ := cs.add(nil)
	c4, _ := cs.add(nil)
	c5, _ := cs.add(nil)
	Assert(t, cs.len == 5, "Expected len 5:", cs.len)

	Assert(t, !c1.disconnected(), "Expected not to be disconnected")
	cs.remove(c1)
	Assert(t, c1.disconnected(), "Expected to be disconnected")
	cs.remove(c2)
	cs.remove(c3)
	cs.remove(c4)
	Assert(t, cs.len == 1, "Expected len 1:", cs.len)

	cs.remove(c5)
	Assert(t, cs.len == 0, "Expected len 0:", cs.len)

	err := cs.remove(c5)
	Assert(t, err == connectionsEmptyError, "Connections should be empty")
}

func TestMaxConnections(t *testing.T) {
	cs := newConnections(100, 2*time.Second)

	for i := 0; i < 100; i++ {
		_, err := cs.add(nil)
		Assert(t, err == nil, "Expected to add 100 connections, stopped at", i+1)
	}

	_, err := cs.add(nil)
	Assert(t, err == connectionsMaxError, "Expected max connections error", err)
}

func TestTimeoutOldConnection(t *testing.T) {
	cs := newConnections(100, time.Second)

	cs.add(nil)
	for i := 1; i < 100; i++ {
		_, err := cs.add(nil)
		Assert(t, err == nil, "Expected to add 100 connections, stopped at", i+1)
	}

	time.Sleep(time.Millisecond * 1050)

	_, err := cs.add(nil)
	Assert(t, err != connectionsMaxError, "Expected old connection to be purged")
}
