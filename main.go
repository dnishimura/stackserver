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
	hostnameFlag := flag.String("hostname", "", "Hostname or IP address to bind to. Leave blank to listen by port only.")
	portFlag := flag.Uint("port", 8080, "TCP port to listen on.")
	debugportFlag := flag.Uint("debugport", 0, "UDP debug port to listen on. Set to zero to disable.")
	flag.Parse()

	server := New(MaxConnections, StackSize, MaxStackElementAgeSeconds)
	err := server.Start(*hostnameFlag, *portFlag, *debugportFlag)

	if err != nil {
		log.Println(err)
	}
}
