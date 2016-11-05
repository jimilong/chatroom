package main

import (
	"chatroom/server/chat"
	"net"
	"sync"
)

func main() {
	var tcpAddr *net.TCPAddr
	tcpAddr, _ = net.ResolveTCPAddr("tcp", "127.0.0.1:9999")
	conn, _ := net.DialTCP("tcp", nil, tcpAddr)

	c := &chat.Chat{
		Conn:    conn,
		Lock:    new(sync.RWMutex),
		Clients: make(map[string]string),
		Online:  make(chan struct{}),
		Offline: make(chan struct{}),
	}

	go c.Register()
	go c.Recv()
	go c.ReportStatus()

	<-c.Offline
}
