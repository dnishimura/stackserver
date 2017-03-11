package main

import (
	"fmt"
	"log"
	"net"
	"sync/atomic"
	"time"
)

const (
	PushResponse   = 0x00
	BusyResponse   = 0xff
	MaxPayloadSize = 127
)

var logger *log.Logger

type Server struct {
	chans       chan bool
	stack       *stack
	connections *connections
	clientcount int64
}

func New(maxConnections, stackSize int, timeout time.Duration) *Server {
	return &Server{
		chans:       make(chan bool, stackSize),
		connections: newConnections(maxConnections, timeout),
		stack:       newStack(stackSize),
	}
}

func (server *Server) Start(host string, port, logport uint) error {
	var err error
	log.SetPrefix("[StackServer] ")
	log.SetFlags(log.Lmicroseconds)
	if port < 1 || port > 65535 {
		log.Fatal("Invalid port", port)
	}
	if logport > 65535 {
		log.Fatal("Invalid debug port", logport)
	} else if logport > 0 {
		syslogHost := fmt.Sprintf("%s:%d", host, logport)
		udp, err := net.Dial("udp", syslogHost)
		logger = log.New(udp, "[StackServer] ", log.Lmicroseconds)
		if err != nil {
			log.Println("Debug logging disabled.", err)
		} else {
			log.Println("Debug UDP port:", logport)
		}
	}

	tcpHost := fmt.Sprintf("%s:%d", host, port)
	log.Println("StackServer starting.")
	log.Println("Listening TCP:", tcpHost)
	listener, err := net.Listen("tcp", tcpHost)
	if err != nil {
		log.Fatal(err)
		return err
	}
	defer listener.Close()

	server.log("Started")
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
		server.log("Server busy.")
		server.log("Rejected: r:%v, l:%v", c.RemoteAddr(), c.LocalAddr())
		c.Write([]byte{BusyResponse})
		c.Close()
		return
	}

	connector.log("Accepted: r:%v, l:%v", c.RemoteAddr(), c.LocalAddr())
	buf := make([]byte, 128, 128)
	nread := 0
	for {
		n, err := c.Read(buf[nread:])
		if err != nil {
			connector.log("Read: %v", err)
			server.connections.remove(connector)
			return
		}
		nread += n

		if n <= 0 {
			return
		}

		if request(buf).isPop() {
			// only care about one byte for pop
			buf = buf[:1]
			break
		}

		if int(buf[0]) <= nread-1 {
			buf = buf[:int(buf[0])+1]
			if int(buf[0]) < nread-1 {
				connector.log("Ignoring %d extra bytes in payload", nread-1-int(buf[0]))
			}
			break
		}
	}

	connector.log("Read %d bytes, using %d bytes: %0x", nread, len(buf), buf)
	go func() {
		// Do a blocked read to watch connection. EOF means connection closed.
		atomic.AddInt64(&server.clientcount, 1)
		one := []byte{0x00}
		var err error
		for err == nil {
			_, err = connector.netConn.Read(one)
			if err != nil {
				connector.done <- true // unblocks push or pop
				connector.log("Closed: %v", err)
			}
		}
		atomic.AddInt64(&server.clientcount, -1)
		server.log("Num blocking clients: %d", server.clientcount)
	}()

	if request(buf).isPop() {
		data, err := server.pop(connector)

		if err == nil {
			n, err := connector.writeall(data)
			if err == nil {
				connector.log("pop() resp %d bytes: %x", n, data)
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
				connector.log("push(%x) resp %d bytes: %x", buf, n, resp)
			} else {
				connector.log("push(%x) write error: %v", buf, err)
			}
		} else {
			connector.log("push(%x) error: %v", buf, err)
		}
	}

	connector.log("Exiting")
	server.connections.remove(connector)
}

func (server *Server) log(line string, args ...interface{}) {
	if logger != nil {
		logger.Println(fmt.Sprintf(line, args...))
	}
}
