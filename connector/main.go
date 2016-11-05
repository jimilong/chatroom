package main

import (
	"chatroom/connector/connector"
	"sync"
)

func main() {
	addr := "127.0.0.1:9999"
	server := &connector.Container{
		Bind_to: addr,
		ConnMap: make(map[string]*connector.Connector),
		Servers: make(map[string]string),
		Lock:    new(sync.RWMutex),
	}

	server.ListenAndServe()
}
