package client

import (
	"fmt"
	"io"
	"net"
	"os"
)

type Client struct {
	ServerIp   string
	ServerPort int
	Name       string
	conn       net.Conn
	flag       int
}

func NewClient(serverIp string, serverPort int) *Client {
	// 创建客户端接口
	client := &Client{
		ServerIp:   serverIp,
		ServerPort: serverPort,
		flag:       999,
	}
	// 连接server
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", serverIp, serverPort))
	if err != nil {
		fmt.Println("net.Dial error:", err)
		return nil
	}
	client.conn = conn

	// 创建client时，就应该开启消息监听
	go client.DealResponse()

	// 返回对象
	return client
}

func (this *Client) menu() bool {
	var flag int
	fmt.Println("1-公聊模式")
	fmt.Println("2-私聊模式")
	fmt.Println("3-更新用户名")
	fmt.Println("0-退出")

	fmt.Scanln(&flag)
	if flag >= 0 && flag <= 3 {
		// 输入合法
		this.flag = flag
		return true
	} else {
		fmt.Println("菜单选择不合法")
		return false
	}

}

func (this *Client) Run() {
	// 0的意思是退出
	for this.flag != 0 {
		// 不合法就重新输入，合法再往下判断
		for !this.menu() {
		}
		switch this.flag {
		case 1:
			this.PublicChat()
		case 2:
			this.PrivateChat()
		case 3:
			this.UpdateName()
		}
	}
}

// PublicChat case1 公聊模式
func (this *Client) PublicChat() {
	// 提示用户发送消息
	var chatMsg string
	fmt.Println("请输入聊天内容，输入exit退出该模式...")
	fmt.Scanln(&chatMsg)

	// 将消息发送给服务器处理
	for chatMsg != "exit" {
		if len(chatMsg) != 0 {
			sendMsg := chatMsg + "\n"
			_, err := this.conn.Write([]byte(sendMsg))
			if err != nil {
				fmt.Println("conn Write error:", err)
				break
			}
		}

		// 如果不置空，当本次输入空格或者回车，会重复发送之前的消息
		chatMsg = ""
		fmt.Println("请输入聊天内容，输入exit退出该模式...")
		fmt.Scanln(&chatMsg)
	}
}

// PublicChat case2 私聊模式
func (this *Client) PrivateChat() {
	var remoteName string

	// 私聊主流程，类似于公聊模式
	// 提示用户选择一个用户进入私聊
	fmt.Println("请输入聊天对象的用户名，输入exit退出")
	// 查询当前都有哪些用户在线
	this.SelectOnlineUser()
	fmt.Scanln(&remoteName)
	for remoteName != "exit" {
		var chatMsg string
		fmt.Println("请输入消息内容，输入exit退出")
		fmt.Scanln(&chatMsg)
		for chatMsg != "exit" {
			if len(chatMsg) != 0 {
				sendMsg := "to|" + remoteName + "|" + chatMsg + "\n"
				_, err := this.conn.Write([]byte(sendMsg))
				if err != nil {
					fmt.Println("conn Write error:", err)
					break
				}
			}
			chatMsg = ""
			fmt.Println("请输入消息内容，输入exit退出")
			fmt.Scanln(&chatMsg)
		}
		remoteName = ""
		fmt.Println("请输入聊天对象的用户名，输入exit退出")
		// 查询当前都有哪些用户在线
		this.SelectOnlineUser()
		fmt.Scanln(&remoteName)
	}

}

// SelectOnlineUser 查询在线用户
func (this *Client) SelectOnlineUser() {
	msg := "who\n"
	_, err := this.conn.Write([]byte(msg))
	if err != nil {
		fmt.Println("conn.Write error:", err)
	}
}

// UpdateName case3 更新用户名
func (this *Client) UpdateName() bool {
	fmt.Println("请输入用户名：")
	fmt.Scanln(&this.Name)

	msg := "rename|" + this.Name + "\n"

	// 把消息写在客户端，服务端会对rename|格式的消息进行处理
	_, err := this.conn.Write([]byte(msg))
	if err != nil {
		fmt.Println("conn.Write error:", err)
		return false
	}
	return true
}

// DealResponse 用于处理服务端发送过来的消息
func (this *Client) DealResponse() {
	// 阻塞，从conn读取消息到标准输出
	io.Copy(os.Stdout, this.conn)

	// 以上等价于下面代码
	//for{
	//	buf := make([]byte, 4096)
	//	n, err := this.conn.Read(buf)
	//	if n == 0 {
	//		return
	//	}
	//	if err != nil && err != io.EOF {
	//		fmt.Println("Conn Read error:", err)
	//		return
	//	}
	//	msg := string(buf[:n-1])
	//	fmt.Println(msg)
	//}
}
