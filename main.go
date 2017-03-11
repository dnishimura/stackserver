package main

import "time"

func main() {
	server := New(100, 100, 10*time.Second)
	server.Start("", 8080, 8888)
}
