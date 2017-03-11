package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"time"
)

const (
	PushResponse   = 0x00
	BusyResponse   = 0xff
	MaxPayloadSize = 127
)

var logger net.Conn

type Server struct {
	chans       chan bool
	stack       *stack
	connections *Connections
}

func New(maxConnections, stackSize int, timeout time.Duration) *Server {
	return &Server{
		chans:       make(chan bool, stackSize),
		connections: newConnections(maxConnections, timeout),
		stack:       newStack(stackSize),
	}
}

func (server *Server) Start(host string, port, logPort uint16) error {
	var err error
	syslogHost := fmt.Sprintf("%s:%d", host, logPort)
	logger, err = net.Dial("udp", syslogHost)
	fmt.Println("Start logger", syslogHost)
	if err != nil {
		fmt.Println("Error opening syslog", err)
	}
	defer server.logClose()

	tcpHost := fmt.Sprintf("%s:%d", host, port)
	fmt.Println("Start StackServer", tcpHost)
	listener, err := net.Listen("tcp", tcpHost)
	if err != nil {
		log.Fatal(err)
		return err
	}
	defer listener.Close()

	server.log("Started")
	/*
		ticker := time.NewTicker(time.Millisecond)
		defer ticker.Stop()
		go func() {
			for range ticker.C {
				server.connections.scanClosed()
			}
		}()
	*/

	for {
		conn, err := listener.Accept()
		if err == nil && conn != nil {
			go server.accept(conn)
		} else {
			server.log("Accept error %v", err)
		}
	}

	return nil
}

func (server *Server) log(line string, args ...interface{}) {
	if logger != nil {
		output := fmt.Sprintf(line, args...)
		fmt.Fprintln(logger, "[StackServer]", output)
	}
}

func (server *Server) logClose() {
	if logger != nil {
		logger.Close()
	}
}

func (server *Server) push(c *connector, data []byte) error {
	select {
	case server.chans <- true:
		return server.stack.push(data)
	case <-c.done:
	}

	return connectionClosedError
}

func (server *Server) pop(c *connector) ([]byte, error) {
	select {
	case <-server.chans:
		return server.stack.pop()
	case <-c.done:
	}
	return nil, connectionClosedError
}

func (server *Server) accept(c net.Conn) {
	connector, err := server.connections.add(c)
	if err == connectionsMaxError {
		server.log("Server busy")
		c.Write([]byte{BusyResponse})
		c.Close()

		return
	}

	connector.log("Connection accepted: r:%v, s:%v", c.RemoteAddr(), c.LocalAddr())
	buf := make([]byte, 128, 128)
	nread := 0
	for {
		n, err := c.Read(buf[nread:])
		if err != nil {
			connector.log("Connection quit: %v", err)
			server.connections.remove(connector)
			return
		}
		nread += n

		if n <= 0 {
			return
		}

		if buf[0] != 0x80 && int(buf[0]) > MaxPayloadSize {
			connector.log("Invalid payload header [%x]: %v", buf[0])
			server.connections.remove(connector)
			return
		}

		if request(buf).isPop() {
			// only care about one byte for pop
			buf = buf[:1]
			break
		}

		if int(buf[0]) == nread-1 {
			buf = buf[:int(buf[0])+1]
			break
		} else if int(buf[0]) < nread-1 {
			connector.log("Dropping, payload too large")
			server.connections.remove(connector)
			return
		}
	}

	connector.log("Read %d bytes: %x", nread, buf)
	connector.waiting = true
	go func() {
		one := []byte{0x00}
		var err error
		for err == nil {
			_, err = connector.netConn.Read(one)
			if err == io.EOF {
				connector.done <- true
				connector.log("EOF")
				return
			}
		}
	}()

	if request(buf).isPop() {
		data, err := server.pop(connector)

		if err == nil {
			n, err := connector.writeall(data)
			if err == nil {
				connector.log("pop() %d bytes: %x", n, data)
			} else {
				connector.log("pop() write error: %v", err)
			}
		} else {
			connector.log("pop() error: %v", err)
		}
	} else if request(buf).isPush() {
		err := server.push(connector, buf)

		if err == nil {
			resp := []byte{PushResponse}
			n, err := connector.writeall([]byte{PushResponse})
			if err == nil {
				connector.log("push() %d bytes: %x", n, resp)
			} else {
				connector.log("push() write error: %v", err)
			}
		} else {
			connector.log("push() error: %v", err)
		}
	}

	connector.log("Exiting")
	server.connections.remove(connector)
}
