package main

import (
	"fmt"
	"bufio"
	"os"
	"time"
	"github.com/tjCFeng/GoFast/DDNS"
	"github.com/tjCFeng/GoFast/Socket"
	"github.com/tjCFeng/GoFast/WebMap"
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
func ClientOnAccept(Client *Socket.ClientTCP) {
	fmt.Println("Accept: ", Client.IPPort())
}

func ClientOnRead(Client *Socket.ClientTCP) {
	Data, _ := Client.GetData()
	Client.ClearData(0)
	fmt.Println("Read: " + string(Data))
	Client.Send(Data)
}

func ClientOnClose(IPPort string) {
	fmt.Println("CloseClient: ", IPPort)
}

/*ClientTCP********************************************************************/
func OnConnectedClient(Client *Socket.ClientTCP) {
	fmt.Println("Connected: ", Client.IPPort())
}

func OnReadClient(Client *Socket.ClientTCP) {
	Data, _ := Client.GetData()
	fmt.Println("Read: ", string(Data))
	Client.ClearData(0)
}

func OnCloseClient(IPPort string) {
	fmt.Println("Close: ", IPPort)
}

/*ServerUDP********************************************************************/
func ServerUDPReceived(Client *Socket.ClientUDP) {
	Data, _ := Client.GetData()
	Client.SendUDP(Data) //必须用SendUDP
	Client.ClearData(0)
	Client.Close()
}

/*ClientUDP********************************************************************/
func ClientUDPReceived(Client *Socket.ClientUDP) {
	Data, _ := Client.GetData()
	Client.ClearData(0)
	fmt.Println(string(Data))
}

func ClientUDPClose (Client *Socket.ClientUDP) {
	fmt.Println("UDP Client: ", Client)
}

/*WebSocket********************************************************************/
func WSOnAccept(Client *Socket.ClientTCP) {
	fmt.Println("WSAccept: ", Client.IPPort())
}

func WSOnRead(Client *Socket.ClientTCP) {
	Data := Socket.WebSocketGetData(Client)
	Socket.WebSocketClearData(Client, 0)
	fmt.Println("WSRead: " + string(Data))
	Client.Send(Data)
}

func WSOnClose(IPPort string) {
	fmt.Println("WSCloseClient: ", IPPort)
}

func main() {
	//DDNS
	ddns := DDNS.Inat123("Username", "Password", "Domain", 3)
	ddns.OnUpdated = DDNSOnUpdated
	go ddns.Start()
	defer ddns.Stop()

	//ServerTCP
	serverTCP := new(Socket.ServerTCP)
	serverTCP.OnClientAccept = ClientOnAccept
	serverTCP.OnClientRead = ClientOnRead
	serverTCP.OnClientDisconnected = ClientOnClose
	serverTCP.Listen("80")
	//serverTCP.AddBlackList("127.0.0.1")
	defer serverTCP.Stop()
	
	//ClientTCP
	clientTCP := new(Socket.ClientTCP)
	clientTCP.OnClientConnected = OnConnectedClient
	clientTCP.OnClientRead = OnReadClient
	clientTCP.OnClientDisconnected = OnCloseClient
	clientTCP.Connect("127.0.0.1", "80")
	time.Sleep(time.Second * 3)
	clientTCP.Send([]uint8("Client Send TCP Data!"))
	defer clientTCP.Close()
	
	//ServerUDP
	serverUDP := new(Socket.ServerUDP)
	serverUDP.OnClientReceived = ServerUDPReceived
	serverUDP.Listen("5000")
	defer serverUDP.Stop()
	
	//ClientUDP
	time.Sleep(time.Second * 1)
	clientUDP := new(Socket.ClientUDP)
	clientUDP.OnClientReceived = ClientUDPReceived
	clientUDP.OnClientClose = ClientUDPClose
	clientUDP.Connect("127.0.0.1", "5000")
	defer clientUDP.Close()
	time.Sleep(time.Second * 1)
	clientUDP.Send([]uint8("Client Send UDP Data!")) //必须用Send
	
	serverWS := new(Socket.ServerWS)
	serverWS.OnWSClientConnected = WSOnAccept
	serverWS.OnWSClientRead = WSOnRead
	serverWS.OnWSClientDisconnected = WSOnClose
	serverWS.Listen("8080")
	defer serverWS.Stop()

	mapBD := &WebMap.MapBD {Token: "Baidu Map Token"}
	X, Y, err := mapBD.ConvertLL(117.2585, 39.0940)
	fmt.Println(X, "_", Y, "_", err)

	s, err := mapBD.LocationLL(X, Y)
	fmt.Println(s, err)


	reader := bufio.NewReader(os.Stdin)
	for {
		key, _, _:= reader.ReadLine()
		switch string(key)  {
			
			case "tcp": fmt.Println(serverTCP.ClientCount())
			case "udp": fmt.Println(serverUDP.ClientCount())
			case "exit": return
			default: continue
		}
	}
}
