package main

import (
	"eims/client"
	"flag"
	"fmt"
)

var serverIp string
var serverPort int

func init() {
	flag.StringVar(&serverIp, "ip", "127.0.0.1", "设置服务器的IP地址(默认为127.0.0.1)")
	flag.IntVar(&serverPort, "port", 8888, "设置服务器的Port(默认为8888)")
}

func main() {
	// 命令行解析
	flag.Parse()

	c := client.NewClient(serverIp, serverPort)
	if c == nil {
		fmt.Println(">>>连接服务器失败<<<")
		return
	}
	fmt.Println(">>>连接服务器成功<<<")
	c.Run()
}
