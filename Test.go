package main

import (
	"time"
	"fmt"
	"github.com/tjCFeng/GoFast/DDNS"
	"github.com/tjCFeng/GoFast/Server"
)

func ClientOnAccept(Client *Server.ClientTCP) {
	fmt.Println("Accept: ", Client.IPPort())
}

func ClientOnRead(Client *Server.ClientTCP) {
	fmt.Println("Read: " + Client.GetDateTime().String())
	Data, _ := Client.GetData()
	Client.Send([]uint8(Data))
	Client.Close()
}

func ClientOnClose(IPPort string) {
	fmt.Println("Close: ", IPPort)
}

func main() {
	ddns := DDNS.Inat123("username", "password", "domain", 3)
	ddns.Start()
	time.Sleep(time.Minute * 10)
	ddns.Stop()
	fmt.Println(ddns.UpdateResult())
	
	serverTCP := new(Server.ServerTCP)
	serverTCP.OnAccept = ClientOnAccept
	serverTCP.OnRead = ClientOnRead
	serverTCP.OnClose = ClientOnClose
	serverTCP.Listen("80")
	//server.AddBlackList("127.0.0.1")
	defer serverTCP.Stop()

	reader := bufio.NewReader(os.Stdin)
	key, _, _:= reader.ReadLine()
	if string(key) == "" {}
}
