package main

import (
	"net"
	"fmt"
	"time"
	"strings"
	"bufio"
)
//全局map
//创建一个记录全部已登录client map
var onlineClientMap map[client]bool
//创建用户名进行索引的map
var onlineClientNameMap map[string]client

//全局 channel
//登陆
var login = make(chan client)
//退出
var logout = make(chan client)
//广播消息
var message = make(chan string)

//client
type client struct {
	C chan string //接收容器
	Name string
}

func manager(){
	//初始化 两个全局map
	onlineClientMap = make(map[client]bool)
	onlineClientNameMap = make(map[string]client)

	for  {
		select{
		case cli := <-login: //登陆
			onlineClientMap[cli] = true
			onlineClientNameMap[cli.Name] = cli

		case cli:= <- logout: //退出
			delete(onlineClientMap,cli)
			delete(onlineClientNameMap,cli.Name)

		case msg := <- message://发送消息
			//截取用户名
			user := strings.Split(msg,":")[0]
			for key, value := range onlineClientMap {
				if value == true && user != key.Name {
					//给除了自己以外的在线用户的通道中发送消息
					key.C <- msg
				}
			}
		}
	}
}

func readfrommsg(conn net.Conn,client client){
	for value := range client.C {
		conn.Write([]byte(value+"\n"))
	}
}

func handlerConnection(conn net.Conn){
	defer conn.Close()

	/*基本登陆功能*/
	//用户超时设置
	myticker := time.NewTicker(100*time.Second)

	//获取用户端口号作为用户名
	myname := conn.RemoteAddr().String()
	myname = strings.Split(myname,":")[1]
	myclinent := client{make(chan string),myname}

	//时间更新
	time_out := make(chan interface{})
	user_out := make(chan interface{})

	//从用户channel中读取收到的信息
	go readfrommsg(conn,myclinent)
	login <- myclinent
	message <- "[" + myclinent.Name + "] 来到到聊天室"

	/*捕捉用户发送消息*/
	go func() {
		//创建一个扫描仪
		myscanner := bufio.NewScanner(conn)

		//如果有消息发送过来就进入循环
		for myscanner.Scan() == true{
			//获取接收到的消息
			tmp := myscanner.Text()

			if tmp == "who" {
				//显示当前在线人员和人数
				go who(conn)
			}else if len(tmp)>2 && tmp[:2] == "To" {
				//单独聊天
				go to(myname,tmp)
			}else {
				//将消息发送给消息管理机制
				message <- myname + ":" + tmp
			}

			//刷新当前用户的在线时长
			time_out <- struct{}{}
		}

		user_out <- struct{}{}
	}()

	/*用户退出管理机制*/
	for  {
		select{
		case <- myticker.C: //用户超时
			fmt.Fprintln(conn,"连接超时")
			logout <- myclinent
			message <- "[" + myname + "] 已断开连接"
			return
		case <- time_out: //用户在线刷新
			myticker.Stop()
			myticker = time.NewTicker(100*time.Second)
		case <- user_out: //用户自主退出
			logout <- myclinent
			message <- "[" + myname + "] 用户下线"
			return
		}
	}
}

//当前在线用户
func who (conn net.Conn){
	for key,_:= range onlineClientMap {
		fmt.Fprintln(conn,key.Name)
	}
	fmt.Fprintln(conn,"当前的在线人数为:",len(onlineClientMap))
}

//one to one format: To | client.name | msg
func to(myname string , tmp string){
	//截取发送者名字
	user := strings.Split(tmp,"|")[1]
	onlineClientNameMap[user].C <- "[" + myname + "] 对你说 : " + strings.Split(tmp,"|")[2]
}

func run(){
	ln,err := net.Listen("tcp","127.0.0.1:8081")
	if(err != nil){
		return
	}

	go manager() //消息管理机制

	for {
		conn,err := ln.Accept()
		if err != nil {
			fmt.Println("ln.accept error:",err)
			continue
		}

		go handlerConnection(conn)
	}
}

func main()  {
	run()
}
