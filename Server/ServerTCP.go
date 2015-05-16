package Server

import (
	"net"
	"time"
	"sync"
	"strings"
	"runtime"
)

type ClientTCP struct {
	hClient		*net.TCPConn
	ipport		string
	datetime	time.Time
	dataBuf	[]byte
	dataLen	int
	mutex		*sync.Mutex
}

type onAccept func (Client *ClientTCP)
type onRead func (Client *ClientTCP)
type onWrite func (Client *ClientTCP)
type onClose func (IPPort string)

type ServerTCP struct {
	hServer			*net.TCPListener
	clientList		map[string]*ClientTCP //[IPPort] *clientTCP
	blackList		map[string] time.Time //[IP] DT
	
	chAccept		chan string
	chRead			chan string
	chWrite		chan string
	chClose		chan string
	chStop			chan bool
	
	OverTime		uint16
	OnAccept	onAccept
	OnRead		onRead
	OnWrite		onWrite
	OnClose		onClose
}

func (this *ServerTCP) Listen(Port string)  error {
	runtime.GOMAXPROCS(runtime.NumCPU())
	if this.OverTime == 0 { this.OverTime = 30}
	
	addr, err := net.ResolveTCPAddr("tcp", ":" + Port)
	if err != nil { return err }
	
	this.chAccept = make(chan string)
	this.chRead = make(chan string)
	this.chWrite = make(chan string)
	this.chClose = make(chan string)
	this.chStop = make(chan bool)
	
	this.clientList = make(map[string] *ClientTCP)
	this.blackList = make(map[string] time.Time)

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

		c.SetDeadline(time.Now().Add(time.Duration(this.OverTime) * time.Second))
		Client := new(ClientTCP)
		Client.hClient = c
		Client.ipport = IPPort
		Client.dataBuf = make([]uint8, 0)
		Client.dataLen = 0
		Client.mutex = &sync.Mutex{}
		this.clientList[IPPort] = Client
		
		this.chAccept <- Client.ipport
		go this.waitClient(Client)
	}
}

func (this *ServerTCP) waitClient(Client *ClientTCP) {
	buf := make([]byte, 1536)
	Client.hClient.SetReadBuffer(1536)
	
	for {
		count, err := Client.hClient.Read(buf)
		if err != nil {
			this.chClose <- Client.ipport
			runtime.Goexit()
		} else {
			if count == 0 {
				this.chClose <- Client.ipport
				runtime.Goexit()
				return
			}
			Client.mutex.Lock()
			Client.dataBuf = append(Client.dataBuf, buf[0:count-1]...)
			Client.dataLen += count
			Client.mutex.Unlock()
			this.chRead <- Client.ipport
		}
	}
}

func (this *ServerTCP) clientEvent() {
	var IPPort string = ""
	var Client *ClientTCP = nil

	for {
		select {
			case IPPort = <- this.chAccept:
				Client = this.clientList[IPPort]
				if Client== nil { continue }
				Client.datetime = time.Now()
				if this.OnAccept != nil { this.OnAccept(Client) }
				
			case IPPort = <- this.chRead:
				Client = this.clientList[IPPort]
				if Client == nil { continue }
				Client.datetime = time.Now()
				if this.OnRead != nil { this.OnRead(Client) }
				
			case IPPort = <- this.chWrite:
				//Log
				
			case IPPort = <- this.chClose:
				this.clearClient(IPPort)
				if this.OnClose != nil { this.OnClose(IPPort)}
				
			case <- this.chStop: return
		}
	}
}

/******************************************************************************/
func (this *ClientTCP) Send(Data []uint8) (int, error) {
	return this.hClient.Write(Data)
}

func (this *ClientTCP) IPPort() string {
	return this.ipport
}

func (this *ClientTCP) GetData() (Data []uint8, Len int) {
	return this.dataBuf, this.dataLen
}

func (this *ClientTCP) ClearData(Len int) {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	switch Len {
		case 0:
			this.dataBuf = this.dataBuf[0 : 0]
			this.dataLen = 0
		default:
			this.dataBuf = this.dataBuf[Len - 1 : ]
			this.dataLen -= Len
	}
}

func (this *ClientTCP) GetDateTime() time.Time {
	return this.datetime
}

func (this *ClientTCP) Close() {
	this.hClient.Close()
}
