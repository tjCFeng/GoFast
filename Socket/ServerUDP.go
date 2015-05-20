package Socket

import (
	"fmt"
	"net"
	"time"
	"sync"
	"runtime"
)

type ServerUDP struct {
	hServer		*net.UDPConn	
	clientList map[string]*ClientUDP //[IPPort] *ClientUDP
	blackList  map[string]time.Time  //[IP] DT
	gBuffer    []uint8
	
	chReceived		chan *ClientUDP
	chStop				chan bool
	
	OnClientReceived		onClientReceived
}

func (this *ServerUDP) Listen(Port string) error {
	runtime.GOMAXPROCS(runtime.NumCPU())
	
	addr, err := net.ResolveUDPAddr("udp", ":" + Port)
	if err != nil { return err }
	
	this.chReceived = make(chan *ClientUDP, 100)
	this.chStop = make(chan bool)
	
	this.clientList = make(map[string]*ClientUDP)
	this.blackList = make(map[string]time.Time)
	this.gBuffer = make([]uint8, 1536)
	
	this.hServer, err = net.ListenUDP("udp", addr)
	if err != nil { return err }
	
	go this.clientEvent()
	go this.waitClient()
	
	return nil
}

func (this *ServerUDP) Stop() error {
	this.chStop <- true
	
	defer func() {
		for IPPort, _ := range this.clientList {
			this.clearClient(IPPort)
		}
	}()
	
	return this.hServer.Close()
}

func (this *ServerUDP) clearClient(IPPort string) {
	delete(this.clientList, IPPort)
}

func (this *ServerUDP) waitClient() {
	var IPPort string
	var Client  *ClientUDP
	
	for {
		count, addr, err := this.hServer.ReadFromUDP(this.gBuffer)
		if err != nil { continue }

		IPPort = fmt.Sprintf("%s:%d", addr.IP, addr.Port)
		Client = this.clientList[IPPort]
		if Client == nil {
			Client = &ClientUDP{}
			Client.ipport = IPPort
			Client.datetime = time.Now()
			Client.dataBuf = make([]uint8, 0)
			Client.dataLen = 0
			Client.mutex = &sync.Mutex{}
			this.clientList[IPPort] = Client
		} else {
			Client.datetime = time.Now()
		}
		
		Client.mutex.Lock()
		Client.dataBuf = append(Client.dataBuf, this.gBuffer[0:count]...)
		Client.dataLen += count
		Client.mutex.Unlock()
		this.chReceived <- Client
	}
}

func (this *ServerUDP) clientEvent() {
	var Client *ClientUDP

	for {
		select {
			case Client = <- this.chReceived:
				if this.OnClientReceived != nil { this.OnClientReceived(Client) }
				
			case <- this.chStop: return
		}
	}
}
