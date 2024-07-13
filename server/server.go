package server

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

type Server struct {
	// Server端的基本信息，ip和端口号
	Ip   string
	Port int

	// 在线用户的列表
	OnlineMap map[string]*User
	// 对于在线用户的读写锁
	OnlineMapLock sync.RWMutex

	// 广播消息的channel
	Message chan string
}

// NewServer 创建一个server对象，并返回这个对象的指针
func NewServer(ip string, port int) *Server {
	server := &Server{
		Ip:        ip,
		Port:      port,
		OnlineMap: make(map[string]*User),
		Message:   make(chan string),
	}
	return server
}

// Start 启动server端监听
func (this *Server) Start() {
	// socket listen
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", this.Ip, this.Port))
	if err != nil {
		fmt.Printf("net.Listen error, network is %s, ip is %s, port is %d, error is %v\n",
			"tcp", this.Ip, this.Port, err)
		return
	}

	// close listen socket
	defer func(listener net.Listener) {
		err := listener.Close()
		if err != nil {
			fmt.Println("listener close error!!!")
		}
	}(listener)

	// 在各个user连接之前先开启监听，否则可能会遗漏user的消息
	go this.Broadcast()

	// 监听各个user的连接情况
	for {
		// accept
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("listener accept error:", err)
			continue
		}

		// do handler
		go this.Handler(conn)
	}
}

// Handler 处理当前连接的业务
func (this *Server) Handler(conn net.Conn) {
	fmt.Println("连接建立成功")

	// 客户端连接之后，在服务端建立一个user对象，用于处理客户端的信息
	onlineUser := NewUser(conn, this)
	onlineUser.Online()

	// 监听用户是否活跃的channel
	isLive := make(chan bool)

	// 接受客户端发送的消息
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := conn.Read(buf)
			if n == 0 {
				if onlineUser.IsOnline {
					onlineUser.Offline()
				}
				return
			}
			if err != nil && err != io.EOF {
				fmt.Println("Conn Read error:", err)
				return
			}

			// 提取用户的消息，去除'\n'
			msg := string(buf[:n-1])
			onlineUser.DoMessage(msg)

			// user的任意消息都会向isLive中存放true，代表当前user活跃
			isLive <- true
		}
	}()

	for {
		select {
		case <-isLive:
			// 当前用户是活跃的，会执行进入下一次for循环，并重新创建定时器，原来的定时器会被回收
		case <-time.After(time.Second * 30):
			// 如果定时器到时间了，但是客户端已经离线了，就直接退出此user的handler了
			if !onlineUser.IsOnline {
				return
			}
			// 超时强踢：此处设置10s不发送消息就会被强行下线
			// 已经超时，将当前的user强制关闭
			onlineUser.sendMsg("超时未响应，您已下线")
			// 用户下线
			onlineUser.Offline()
			// 退出handler
			return
		}
	}

}

// Broadcast 将需要广播的信息广播至各个客户端
func (this *Server) Broadcast() {
	for {
		msg := <-this.Message
		this.OnlineMapLock.RLock()
		for _, onlineUser := range this.OnlineMap {
			onlineUser.C <- msg
		}
		this.OnlineMapLock.RUnlock()
	}
}

// PutMessage 将需要广播的消息放入消息管道中
func (this *Server) PutMessage(c *User, msg string) {
	sendMsg := "[" + c.Addr + "]" + c.Name + ":" + msg
	this.Message <- sendMsg
}
