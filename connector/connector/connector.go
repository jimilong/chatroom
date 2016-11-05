package connector

import (
	"bufio"
	"chat_distribute/protobuf/chat"
	"chat_distribute/protocol"
	//"fmt"
	"github.com/golang/protobuf/proto"
	"log"
	"net"
	"time"
)

type Connector struct {
	Container *Container
	Conn      *net.TCPConn
	IsServer  bool
	In        chan bool
	Timeout   chan struct{}
	Quit      chan struct{}
}

func (c *Connector) HandleTimeout() {
	tick := time.Tick(30 * time.Second)
	msgs := make([]bool, 0, 10)
	flag := false
	for {
		select {
		case msg := <-c.In:
			if len(msgs) < 1 {
				msgs = append(msgs, msg)
			}
		case <-tick:
			if len(msgs) == 0 {
				log.Println("time-out")
				close(c.Timeout)
				c.Conn.Close()
				flag = true
			} else {
				msgs = msgs[:0]
			}
		case <-c.Quit:
			flag = true
		}
		if flag {
			break
		}
	}
	log.Println("handleTimeout over 88")
}

func (c *Connector) Listen() {
	defer func() {
		ipStr := c.Conn.RemoteAddr().String()
		c.Conn.Close()
		delete(c.Container.ConnMap, ipStr)
		if c.IsServer {
			c.Container.Lock.Lock()
			for serv, ip := range c.Container.Servers {
				if ip == ipStr {
					delete(c.Container.Servers, serv)
				}
			}
			c.Container.Lock.Unlock()
		} else {
			go c.OnLogout()
		}

		log.Printf("disconnected :%s\n", ipStr)
	}()

	reader := bufio.NewReader(c.Conn)
	for {
		msg, header, err := protocol.Unpack(reader)
		if err != nil {
			close(c.Quit)
			log.Printf("conn error :%s\n", err)
			break
		}
		//通知超时处理方法
		c.In <- true

		switch header.Service {
		case LOGIN_CODE:
			go c.OnLogin(msg, header.AskId)
			log.Printf("dispatch server : %s\n", LOGIN)
		case MSG_CODE:
			go c.OnMsg(msg, header.AskId)
			log.Printf("dispatch server : %s\n", MSG)
		case KEEPLIVE_CODE:
			log.Printf("reciver server -- keeplive ....addr=%s\n", c.Conn.RemoteAddr().String())
		case REGISTER_CODE:
			go c.servReg(msg)
			log.Printf("dispatch server : %s\n", REGISTER)
		case RESPONSE_CODE:
			go c.Resp(msg, header.AskId)
			log.Printf("dispatch server : %s\n", RESPONSE)
		default:
			log.Printf("不匹配服务: %d\n", header.Service)
		}
	}
}

func (c *Connector) OnLogin(msg []byte, askId uint32) {
	//解码
	j2 := &chat.Login{}
	proto.Unmarshal([]byte(msg), j2)
	j2.Addr = c.Conn.RemoteAddr().String()
	pb, _ := proto.Marshal(j2)
	b, _ := protocol.Pack(pb, LOGIN, askId)

	c.Container.Lock.RLock()
	serIp := c.Container.Servers[LOGIN]
	conn := c.Container.ConnMap[serIp].Conn
	//c.Container.ConnMap[serIp].Conn.Write(b)
	c.Container.Lock.RUnlock()
	conn.Write(b)
}

func (c *Connector) OnMsg(msg []byte, askId uint32) {
	//解码
	j2 := &chat.ChatMsg{}
	proto.Unmarshal([]byte(msg), j2)
	j2.From = c.Conn.RemoteAddr().String()
	pb, _ := proto.Marshal(j2)
	b, _ := protocol.Pack(pb, MSG, askId)

	c.Container.Lock.RLock()
	serIp := c.Container.Servers[MSG]
	conn := c.Container.ConnMap[serIp].Conn
	//c.Container.ConnMap[serIp].Write(b)
	c.Container.Lock.RUnlock()
	conn.Write(b)
}

func (c *Connector) OnLogout() {
	ipStr := c.Conn.RemoteAddr().String()
	logout := &chat.Logout{Addr: ipStr}
	//编码
	pb, _ := proto.Marshal(logout)
	b, _ := protocol.Pack(pb, LOGOUT, 0)

	c.Container.Lock.RLock()
	serIp := c.Container.Servers[LOGOUT]
	conn := c.Container.ConnMap[serIp].Conn
	c.Container.Lock.RUnlock()

	conn.Write(b)
}

func (c *Connector) servReg(msg []byte) {
	c.IsServer = true
	ipStr := c.Conn.RemoteAddr().String()
	//解码
	j2 := &chat.Register{}
	proto.Unmarshal([]byte(msg), j2)

	for _, serv := range j2.Service {
		log.Printf("新服务:%s\n", serv)
		c.Container.Lock.Lock()
		c.Container.Servers[serv] = ipStr
		c.Container.Lock.Unlock()
	}
	go c.OnKeeplive()
}

func (c *Connector) OnKeeplive() {
	tick := time.Tick(3 * time.Second)
	flag := false
	for {
		select {
		case <-tick:
			b, _ := protocol.Pack([]byte(""), KEEPLIVE, 0)
			c.Conn.Write(b)

			log.Printf("send server -- keeplive ....addr=%s\n", c.Conn.RemoteAddr().String())
		case <-c.Quit:
			flag = true
		case <-c.Timeout:
			flag = true
		}

		if flag {
			break
		}
	}
}

func (c *Connector) Resp(msg []byte, askId uint32) {
	//解码
	j2 := &chat.ChatMsg{}
	proto.Unmarshal(msg, j2)
	b, _ := protocol.Pack(msg, RESPONSE, askId)

	for _, to := range j2.To {
		c.Container.Lock.RLock()
		conn := c.Container.ConnMap[to].Conn
		c.Container.Lock.RUnlock()
		conn.Write(b)
	}
}
