package main

import (
	"fmt"
	"bufio"
	"os"
	//"github.com/tjCFeng/GoFast/DDNS"
	//"github.com/tjCFeng/GoFast/Server"
)

/*DDNS************************************************************************/
func DDNSOnUpdated(UpdateResult string) {
	switch UpdateResult {
		case "good": fmt.Println("更新成功")
		case "nochg": fmt.Println("IP未变化")
		case "nohost": fmt.Println("域名不存在")
		case "badauth": fmt.Println("用户名或密码错误")
		case "abuse": fmt.Println("请求失败，连接频繁")
		case "servererror": fmt.Println("系统错误")
		default: fmt.Println(UpdateResult + ": 未知错误")
	}
}

/*ServerTCP********************************************************************/
func ClientOnAccept(Client *Server.ClientTCP) {
	//fmt.Println("Accept: ", Client.IPPort())
}

func ClientOnRead(Client *Server.ClientTCP) {
	//fmt.Println("Read: " + Client.GetDateTime().String())
	Data, _ := Client.GetData()
	Client.ClearData(0)
	Client.Send([]uint8(Data))
	//Client.Close()
}

func ClientOnClose(IPPort string) {
	//fmt.Println("Close: ", IPPort)
}

func main() {
	//DDNS
	ddns := DDNS.Inat123("Username", "Password", "Domain", 3)
	ddns.OnUpdated = DDNSOnUpdated
	go ddns.Start()
	defer ddns.Stop()

	//ServerTCP
	serverTCP := new(Server.ServerTCP)
	serverTCP.OnAccept = ClientOnAccept
	serverTCP.OnRead = ClientOnRead
	serverTCP.OnClose = ClientOnClose
	serverTCP.Listen("80")
	//serverTCP.AddBlackList("127.0.0.1")
	defer serverTCP.Stop()


	reader := bufio.NewReader(os.Stdin)
	for {
		key, _, _:= reader.ReadLine()
		switch string(key)  {
			
			case "client": fmt.Println(serverTCP.ClientCount())
			case "exit": return
			default: continue
		}
	}
}
