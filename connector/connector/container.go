package connector

import (
	//"bufio"
	//"chat_distribute/protobuf/chat"
	//"chat_distribute/protocol"
	"fmt"
	//"github.com/golang/protobuf/proto"
	"hash/crc32"
	"log"
	"net"
	"sync"
	"time"
)

const (
	LOGIN    = "Chat.Login"
	MSG      = "Chat.Msg"
	RESPONSE = "Chat.Response"
	LOGOUT   = "Chat.Logout"
	KEEPLIVE = "Keeplive"
	REGISTER = "Service.Register"
)

var (
	LOGIN_CODE    = crc32.ChecksumIEEE([]byte("Chat.Login"))
	LOGOUT_CODE   = crc32.ChecksumIEEE([]byte("Chat.Logout"))
	MSG_CODE      = crc32.ChecksumIEEE([]byte("Chat.Msg"))
	RESPONSE_CODE = crc32.ChecksumIEEE([]byte("Chat.Response"))
	KEEPLIVE_CODE = crc32.ChecksumIEEE([]byte("Keeplive"))
	REGISTER_CODE = crc32.ChecksumIEEE([]byte("Service.Register"))
)

type Container struct {
	Bind_to string
	ConnMap map[string]*Connector
	Servers map[string]string
	Lock    *sync.RWMutex
}

func (c *Container) ListenAndServe() {
	tcpAddr, _ := net.ResolveTCPAddr("tcp", c.Bind_to)
	listener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	go c.ReportStatus()
	// Main loop
	for {
		tcpConn, err := listener.AcceptTCP()
		if err != nil {
			log.Printf("Accept error: %s\n", err.Error())
			time.Sleep(time.Second)
			continue
		}

		fmt.Println("A client connected : " + tcpConn.RemoteAddr().String())
		conn := &Connector{
			Container: c,
			Conn:      tcpConn,
			IsServer:  false,
			In:        make(chan bool),
			Timeout:   make(chan struct{}),
			Quit:      make(chan struct{}),
		}
		c.Lock.Lock()
		c.ConnMap[tcpConn.RemoteAddr().String()] = conn
		c.Lock.Unlock()

		go conn.Listen()
		go conn.HandleTimeout()
	}
}

func (c *Container) ReportStatus() {
	tick := time.Tick(3 * time.Second)
	for {
		<-tick
		c.Lock.RLock()
		log.Printf("status: conn - %d | server - %d", len(c.ConnMap), len(c.Servers))
		c.Lock.RUnlock()
	}
}
