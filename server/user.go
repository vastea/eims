package server

import (
	"fmt"
	"net"
	"strings"
)

type User struct {
	Name string
	Addr string
	C    chan string
	conn net.Conn
	// 用户的在线状态
	IsOnline bool

	// 当前user属于的server，方便user获取到server的信息
	s *Server
}

// NewUser 创建一个user
func NewUser(conn net.Conn, s *Server) *User {
	// 当前客户端连接的地址
	userAddr := conn.RemoteAddr().String()
	user := &User{
		Name:     userAddr,
		Addr:     userAddr,
		C:        make(chan string),
		conn:     conn,
		s:        s,
		IsOnline: false,
	}

	// 创建user时，就应该开启消息监听
	go user.listenMessage(conn)

	return user
}

// Online 用户上线的逻辑处理
func (this *User) Online() {
	// 存储用户的上线状态
	this.s.OnlineMapLock.Lock()
	this.IsOnline = true
	this.s.OnlineMap[this.Name] = this
	this.s.OnlineMapLock.Unlock()

	// 广播该user上线的消息
	msg := "已上线"
	this.s.PutMessage(this, msg)
}

// Offline 用户下线的逻辑处理
func (this *User) Offline() {
	// 将用户从onlineMap中删除
	this.s.OnlineMapLock.Lock()
	this.IsOnline = false
	delete(this.s.OnlineMap, this.Name)
	this.s.OnlineMapLock.Unlock()

	msg := "已下线"
	this.s.PutMessage(this, msg)

	// 发送完最后一条消息，将channel关闭
	close(this.C)
}

// sendMsg 将msg发送给当前user
func (this *User) sendMsg(msg string) {
	this.C <- msg
}

// DoMessage 用户处理消息的业务
func (this *User) DoMessage(msg string) {
	if msg == "who" {
		// 定义"who"为查询当前所有的在线用户的指令
		this.s.OnlineMapLock.RLock()
		for _, onlineUser := range this.s.OnlineMap {
			onlineUserMsg := "[" + onlineUser.Addr + "]" + onlineUser.Name + " : 在线..."
			this.sendMsg(onlineUserMsg)
		}
		this.s.OnlineMapLock.RUnlock()
	} else if len(msg) > 7 && msg[:7] == "rename|" {
		// 用于user修改其用户名，例如rename|张三
		newName := strings.Split(msg, "|")[1]

		// 判断newName是否存在
		if _, ok := this.s.OnlineMap[newName]; ok {
			this.sendMsg("当前用户名已被使用，请修改！")
		} else {
			this.s.OnlineMapLock.Lock()
			delete(this.s.OnlineMap, this.Name)
			this.Name = newName
			this.s.OnlineMap[newName] = this
			this.s.OnlineMapLock.Unlock()
			this.sendMsg("您已修改用户名为：" + newName)
		}
	} else if len(msg) > 4 && msg[:3] == "to|" {
		// 处理私聊功能, 形如to|zhangsan|xxx
		msgArray := strings.Split(msg, "|")
		if len(msgArray) < 2 {
			this.sendMsg("消息格式不正确，请使用\"to|username|msg\"的格式进行发送")
		}
		// 获取用户名
		targetName := strings.Split(msg, "|")[1]
		if targetName == "" {
			this.sendMsg("消息格式不正确，请使用\"to|username|msg\"的格式进行发送")
			return
		}
		// 根据用户名获取对方的user用于发送
		targetUser, ok := this.s.OnlineMap[targetName]
		if !ok {
			this.sendMsg(targetName + "用户不在线，无法向其发送信息")
			return
		}
		// 获取消息内容，发送到对方的channel
		content := strings.Split(msg, "|")[2]
		targetUser.sendMsg(this.Name + "对您说: " + content)
	} else {
		this.s.PutMessage(this, msg)
	}
}

// listenMessage 监听当前user的channel，当收到消息时发送给客户端
func (this *User) listenMessage(conn net.Conn) {
	for {
		if msg, ok := <-this.C; ok {
			_, err := this.conn.Write([]byte(msg + "\n"))
			if err != nil {
				fmt.Println("给Name为", this.Name, "的客户端发送消息失败, error信息为：", err)
			}
		} else {
			// channel被销毁代表客户端已离线，此时可以关闭连接
			fmt.Println(this.Name, ": listenMessage退出")
			conn.Close()
			return
		}
	}
}
