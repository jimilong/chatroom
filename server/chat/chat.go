package chat

import (
	"bufio"
	"chat_distribute/protobuf/chat"
	"chat_distribute/protocol"
	"fmt"
	"github.com/golang/protobuf/proto"
	"hash/crc32"
	"io"
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
	KEEPLIVE_CODE = crc32.ChecksumIEEE([]byte("Keeplive"))
)

type Chat struct {
	Conn    *net.TCPConn
	Lock    *sync.RWMutex
	Clients map[string]string
	Online  chan struct{}
	Offline chan struct{}
}

func (c *Chat) Register() {
	services := []string{LOGIN, LOGOUT, MSG}
	reg := &chat.Register{Service: services}
	//编码
	pb, err := proto.Marshal(reg)
	if err != nil {
		fmt.Println(err)
		c.Offline <- struct{}{}
		return
	}
	b, err := protocol.Pack(pb, REGISTER, 0)
	if err != nil {
		fmt.Println(err)
		c.Offline <- struct{}{}
		return
	}
	_, err = c.Conn.Write(b)
	if err != nil {
		fmt.Println(err)
		c.Offline <- struct{}{}
		return
	}
	c.Online <- struct{}{}
}

func (c *Chat) Recv() {
	defer func() {
		c.Conn.Close()
		c.Offline <- struct{}{}
	}()

	<-c.Online
	reader := bufio.NewReader(c.Conn)
	for {
		msg, header, err := protocol.Unpack(reader)
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println(err)
			continue
		}

		switch header.Service {
		case LOGIN_CODE:
			go c.OnLogin(msg)
			log.Printf("dispatch server : %s\n", LOGIN)
		case LOGOUT_CODE:
			go c.OnLogout(msg)
			log.Printf("dispatch server : %s\n", LOGOUT)
		case MSG_CODE:
			go c.OnMsg(msg, header.AskId)
			log.Printf("dispatch server : %s\n", MSG)
		case KEEPLIVE_CODE:
			go c.OnKeeplive()
		default:
			log.Printf("不匹配服务：%s\n", header.Service)
		}
	}
}

func (c *Chat) OnLogin(msg []byte) {
	//解码
	j2 := &chat.Login{}
	err := proto.Unmarshal([]byte(msg), j2)
	if err != nil {
		fmt.Println(err)
		return
	}
	c.Lock.Lock()
	c.Clients[j2.Addr] = j2.Name
	c.Lock.Unlock()
}

func (c *Chat) OnLogout(msg []byte) {
	//解码
	j2 := &chat.Logout{}
	err := proto.Unmarshal([]byte(msg), j2)
	if err != nil {
		fmt.Println(err)
		return
	}
	c.Lock.Lock()
	delete(c.Clients, j2.Addr)
	c.Lock.Unlock()
}

func (c *Chat) OnMsg(msg []byte, askId uint32) {
	//解码
	j2 := &chat.ChatMsg{}
	err := proto.Unmarshal([]byte(msg), j2)
	if err != nil {
		fmt.Println(err)
		return
	}
	c.Lock.RLock()
	j2.Name = c.Clients[j2.From]
	c.Lock.RUnlock()
	//接收者
	rNum := len(c.Clients)
	clients := make([]string, rNum)
	i := 0
	for client := range c.Clients {
		clients[i] = client
		i++
	}
	j2.To = clients

	//编码
	pb, err := proto.Marshal(j2)
	if err != nil {
		fmt.Println(err)
		return
	}
	b, err := protocol.Pack(pb, RESPONSE, askId)
	if err != nil {
		fmt.Println(err)
		return
	}

	c.Conn.Write(b)
}

func (c *Chat) OnKeeplive() {
	addr := c.Conn.RemoteAddr().String()
	log.Printf("receive container keeplive ...addr＝%s\n", addr)
	b, _ := protocol.Pack([]byte(""), KEEPLIVE, 0)
	c.Conn.Write(b)
	log.Printf("send container keeplive ...addr＝%s\n", addr)
}

func (c *Chat) ReportStatus() {
	for {
		time.Sleep(3 * time.Second)
		log.Printf("Status: clients num -- %d\n", len(c.Clients))
	}
}
