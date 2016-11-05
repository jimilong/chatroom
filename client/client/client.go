package client

import (
	"bufio"
	"chat_distribute/protobuf/chat"
	"chat_distribute/protocol"
	"fmt"
	"github.com/golang/protobuf/proto"
	"net"
	"os"
)

const (
	LOGIN  = "Chat.Login"
	MSG    = "Chat.Msg"
	LOGOUT = "Chat.Logout"
)

type Client struct {
	Conn  *net.TCPConn
	Enter chan struct{} //登录成功
	Quit  chan struct{} //客户端退出
}

func (c *Client) Login() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("请输入昵称：")
loop:
	msg, _, _ := reader.ReadLine()

	if len(msg) == 0 {
		fmt.Println("昵称不能为空\n")
		goto loop
	}

	login := &chat.Login{Name: string(msg)}
	//编码
	result, err := proto.Marshal(login)
	if err != nil {
		fmt.Println(err)
		c.Quit <- struct{}{}
	}
	b, err := protocol.Pack(result, LOGIN, 0)
	if err != nil {
		fmt.Println(err)
		c.Quit <- struct{}{}
	}
	_, err = c.Conn.Write(b)
	if err != nil {
		fmt.Println(err)
		c.Quit <- struct{}{}
	}

	c.Enter <- struct{}{}
}

func (c *Client) Recv() {
	reader := bufio.NewReader(c.Conn)
	for {
		msg, _, err := protocol.Unpack(reader)
		if err != nil {
			c.Quit <- struct{}{}
			break
		}
		//解码
		j2 := &chat.ChatMsg{}
		proto.Unmarshal([]byte(msg), j2)

		fmt.Println(j2.Name+":", j2.Msg)
	}
}

func (c *Client) Send() {
	<-c.Enter
	reader := bufio.NewReader(os.Stdin)
	askId := uint32(0)
	fmt.Println("开始聊天吧！")
	for {
		msg, _, _ := reader.ReadLine()
		if string(msg) == "q" || string(msg) == "exit" {
			c.Quit <- struct{}{} //断开socket退出
			break
		}

		if len(msg) > 0 {
			aa := &chat.ChatMsg{
				Msg: string(msg),
			}
			//编码
			pb, _ := proto.Marshal(aa)

			askId++
			b, _ := protocol.Pack(pb, MSG, askId)
			c.Conn.Write(b)

			msg = msg[:0]
		}
	}
}
