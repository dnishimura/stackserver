# Stack Server

## Prequisites

Tested with Go version 1.8

## Getting Started

Put all .go files in the directory `$GOPATH/src/stackserver/`

Build the executable in the stackserver directory by running `go build`

## Running the Server

Run the Stack Server with the default arguments: `./stackserver`

The Stack Server with listen on 127.0.0.1:8080 by default. Configurable settings are:

```
$ ./stackserver -h
Usage of ./stackserver:
  -debughost string
    	Hostname or IP address of UDP listener to log to. (default "127.0.0.1")
  -debugport uint
    	UDP debug port to log to. 0 to disable. (default 0)
  -host string
    	Hostname or IP address of interface to bind to. Specify empty string to listen on all interfaces. (default "127.0.0.1")
  -port uint
    	TCP port to listen to. (default 8080)
```

Ctrl-C to quit the server.

## Debugging

To view the debug log, set the debug UDP port with the flag `-debugport`. Use a tool, such as netcat, to listen on the UDP port. For example, if the debug UDP port is set to 8888, run netcat via:

```
$ nc -lu 8888
```

