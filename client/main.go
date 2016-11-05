package main

import (
	"chatroom/client/client"
	"net"
)

func main() {
	var tcpAddr *net.TCPAddr
	tcpAddr, _ = net.ResolveTCPAddr("tcp", "127.0.0.1:9999")
	conn, _ := net.DialTCP("tcp", nil, tcpAddr)

	c := &client.Client{
		Conn:  conn,
		Enter: make(chan struct{}),
		Quit:  make(chan struct{}),
	}

	go c.Login()
	go c.Recv()
	go c.Send()

	<-c.Quit
}
