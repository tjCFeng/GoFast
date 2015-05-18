package TCP

import (
	"net"
	"runtime"
	"strings"
	"sync"
	"time"
)

type onClientAccept func(Client *ClientTCP)

type ServerTCP struct {
	hServer    *net.TCPListener
	clientList map[string]*ClientTCP //[IPPort] *clientTCP
	blackList  map[string]time.Time  //[IP] DT
	gBuffer    []uint8

	chAccept chan string
	chRead   chan string
	chWrite  chan string
	chClose  chan string
	chStop   chan bool

	OverTime uint16
	OnClientAccept onClientAccept
	OnClientRead   onClientRead
	OnClientWrite  onClientWrite
	OnClientClose  onClientClose
}

func (this *ServerTCP) Listen(Port string) error {
	runtime.GOMAXPROCS(runtime.NumCPU())
	if this.OverTime == 0 { this.OverTime = 30 }

	addr, err := net.ResolveTCPAddr("tcp", ":"+Port)
	if err != nil { return err }

	this.chAccept = make(chan string)
	this.chRead = make(chan string)
	this.chWrite = make(chan string)
	this.chClose = make(chan string)
	this.chStop = make(chan bool)

	this.clientList = make(map[string]*ClientTCP)
	this.blackList = make(map[string]time.Time)
	this.gBuffer = make([]uint8, 1536)

	this.hServer, err = net.ListenTCP("tcp", addr)
	if err != nil { return err }

	go this.clientEvent()
	go this.clientAccept()

	return nil
}

func (this *ServerTCP) Stop() error {
	this.chStop <- true

	defer func() {
		for IPPort, Client := range this.clientList {
			this.CloseClient(Client)
			this.clearClient(IPPort)
		}
	}()

	return this.hServer.Close()
}

func (this *ServerTCP) clearClient(IPPort string) {
	delete(this.clientList, IPPort)
}

func (this *ServerTCP) CloseClient(Client *ClientTCP) {
	if Client == nil { return }
	if Client.hClient == nil { return }

	Client.hClient.Close()
}

func (this *ServerTCP) CloseClientByIPPort(IPPort string) {
	Client, ok := this.clientList[IPPort]
	if !ok { return }

	this.CloseClient(Client)
}

func (this *ServerTCP) AddBlackList(IP string) {
	this.blackList[IP] = time.Now()
}

func (this *ServerTCP) ClientCount() int {
	return len(this.clientList)
}

func (this *ServerTCP) clientAccept() {
	var IPPort string = ""

	for {
		c, err := this.hServer.AcceptTCP()
		if err != nil { continue }
		IPPort = c.RemoteAddr().String()

		if _, ok := this.blackList[strings.Split(IPPort, ":")[0]]; ok {
			//Log BlackList
			c.Close()
			continue
		}

		//c.SetDeadline(time.Now().Add(time.Duration(this.OverTime) * time.Second))
		Client := new(ClientTCP)
		Client.hClient = c
		Client.ipport = IPPort
		Client.dataBuf = make([]uint8, 0)
		Client.dataLen = 0
		Client.hClient.SetReadBuffer(1536)
		Client.mutex = &sync.Mutex{}
		this.clientList[IPPort] = Client

		this.chAccept <- Client.ipport
		go this.waitClient(Client)
	}
}

func (this *ServerTCP) waitClient(Client *ClientTCP) {
	for {
		count, err := Client.hClient.Read(this.gBuffer)
		if (err != nil) || (count == 0) {
			this.chClose <- Client.ipport
			runtime.Goexit()
			return
		}

		Client.mutex.Lock()
		Client.dataBuf = append(Client.dataBuf, this.gBuffer[0:count-1]...)
		Client.dataLen += count
		Client.mutex.Unlock()
		this.chRead <- Client.ipport
	}
}

func (this *ServerTCP) clientEvent() {
	var IPPort string = ""
	var Client *ClientTCP = nil

	for {
		select {
			case IPPort = <-this.chAccept:
				Client = this.clientList[IPPort]
				if Client == nil { continue }
				Client.datetime = time.Now()
				if this.OnClientAccept != nil { this.OnClientAccept(Client) }

			case IPPort = <-this.chRead:
				Client = this.clientList[IPPort]
				if Client == nil { continue }
				Client.datetime = time.Now()
				if this.OnClientRead != nil { this.OnClientRead(Client) }

			case IPPort = <-this.chWrite:
				//Log

			case IPPort = <-this.chClose:
				this.clearClient(IPPort)
				if this.OnClientClose != nil { this.OnClientClose(IPPort) }

			case <-this.chStop:
				return
		}
	}
}
