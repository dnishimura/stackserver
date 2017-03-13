package main

import (
	"flag"
	"log"
	"time"
)

const (
	MaxConnections            = 100
	StackSize                 = 100
	MaxStackElementAgeSeconds = 10 * time.Second
)

func main() {
	hostFlag := flag.String("host",
		"127.0.0.1",
		"Hostname or IP address of interface to bind to. Specify empty string to listen on all interfaces.",
	)
	portFlag := flag.Uint("port",
		8080,
		"TCP port to listen to.",
	)
	debughostFlag := flag.String("debughost",
		"127.0.0.1",
		"Hostname or IP address of UDP listener to log to.",
	)
	debugportFlag := flag.Uint("debugport",
		0,
		"UDP debug port to log to. 0 to disable. (default 0)",
	)
	flag.Parse()

	server := New(MaxConnections, StackSize, MaxStackElementAgeSeconds)
	err := server.Start(*hostFlag, *debughostFlag, *portFlag, *debugportFlag)

	if err != nil {
		log.Println(err)
	}
}
