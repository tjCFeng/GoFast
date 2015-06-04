package Socket

import (
	"net"
	"runtime"
	"strings"
	"sync"
	"time"
)

/*ClientTCP********************************************************************/
type onClientConnected func(Client *ClientTCP)
type onClientRead func(Client *ClientTCP)
type onClientWrite func(Client *ClientTCP, Data *[]uint8)
type onClientDisconnected func(IPPort string)

type ClientTCP struct {
	//ClientTCP、ServerTCP共用
	hClient  *net.TCPConn
	ipport   string
	datetime time.Time
	dataBuf  []uint8
	dataLen  int
	mutex    *sync.Mutex
	
	//ClientTCP单独使用
	chConnected chan bool
	chRead   chan bool
	chWrite  chan bool
	chClose  chan bool
	
	OnClientConnected onClientConnected
	OnClientRead   onClientRead
	OnClientWrite  onClientWrite
	OnClientDisconnected  onClientDisconnected

	Tag		interface{}
}

func (this *ClientTCP) Connect(RemoteIP, RemotePort string) error {
	addr, err := net.ResolveTCPAddr("tcp", RemoteIP + ":" + RemotePort)
	if err != nil { return err }
	
	this.chConnected = make(chan bool)
	this.chRead = make(chan bool)
	this.chWrite = make(chan bool)
	this.chClose= make(chan bool)

	this.hClient, err = net.DialTCP("tcp",  nil, addr)
	if err != nil { return err }

	go this.clientEvent()
	this.mutex = &sync.Mutex{}
	this.chConnected <- true

	return nil
}

func (this *ClientTCP) Close() {
	this.hClient.Close()
}

func (this *ClientTCP) Send(Data []uint8) (int, error) {
	Buf := &Data
	if this.OnClientWrite != nil { this.OnClientWrite(this, Buf) }
	return this.hClient.Write(*Buf)
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
			this.dataBuf = this.dataBuf[0:0]
			this.dataLen = 0
		default:
			this.dataBuf = this.dataBuf[Len - 1:]
			this.dataLen -= Len
	}
}

func (this *ClientTCP) GetDateTime() time.Time {
	return this.datetime
}

func (this *ClientTCP) read() {
	Buf := make([]uint8, 1536)

	for {
		count, err := this.hClient.Read(Buf)
		if (err != nil) || (count == 0) {
			this.chClose <- true
			runtime.Goexit()
			return
		}

		this.mutex.Lock()
		this.dataBuf = append(this.dataBuf, Buf[0:count]...)
		this.dataLen += count
		this.mutex.Unlock()
		this.chRead <- true
	}
}

func (this *ClientTCP) clientEvent() {
	for {
		select {
			case <- this.chConnected:
				go this.read()
				this.ipport = this.hClient.LocalAddr().String()
				this.datetime = time.Now()
				if this.OnClientConnected != nil { this.OnClientConnected(this) }

			case <- this.chRead:
				this.datetime = time.Now()
				if this.OnClientRead != nil { this.OnClientRead(this) }

			case <- this.chWrite:
				//Log

			case <- this.chClose:
				if this.OnClientDisconnected != nil { this.OnClientDisconnected(this.ipport) }
				return
		}
	}
}

/*ServerTCP********************************************************************/
type onClientAccept func(Client *ClientTCP)

type ServerTCP struct {
	hServer    *net.TCPListener
	clientList map[string]*ClientTCP //[IPPort] *ClientTCP
	blackList  map[string]time.Time  //[IP] DT
	gBuffer    []uint8

	chAccept chan *ClientTCP
	chRead   chan *ClientTCP
	chWrite  chan *ClientTCP
	chClose  chan *ClientTCP
	chStop   chan bool

	OverTime uint16
	OnClientAccept onClientAccept
	OnClientRead   onClientRead
	OnClientWrite  onClientWrite
	OnClientDisconnected  onClientDisconnected
}

func (this *ServerTCP) Listen(Port string) error {
	runtime.GOMAXPROCS(runtime.NumCPU())
	if this.OverTime == 0 { this.OverTime = 30 }

	addr, err := net.ResolveTCPAddr("tcp", ":" + Port)
	if err != nil { return err }

	this.chAccept = make(chan *ClientTCP, 100)
	this.chRead = make(chan *ClientTCP, 100)
	this.chWrite = make(chan *ClientTCP, 100)
	this.chClose = make(chan *ClientTCP, 100)
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

		c.SetDeadline(time.Now().Add(time.Duration(this.OverTime) * time.Second))
		Client := new(ClientTCP)
		Client.hClient = c
		Client.ipport = IPPort
		Client.dataBuf = make([]uint8, 0)
		Client.dataLen = 0
		Client.hClient.SetReadBuffer(1536)
		Client.mutex = &sync.Mutex{}
		this.clientList[IPPort] = Client

		this.chAccept <- Client
		go this.waitClient(Client)
	}
}

func (this *ServerTCP) waitClient(Client *ClientTCP) {
	for {
		count, err := Client.hClient.Read(this.gBuffer)
		if (err != nil) || (count == 0) {
			this.chClose <- Client
			runtime.Goexit()
			return
		}
		Client.hClient.SetDeadline(time.Now().Add(time.Duration(this.OverTime) * time.Second))

		Client.mutex.Lock()
		Client.dataBuf = append(Client.dataBuf, this.gBuffer[0:count]...)
		Client.dataLen += count
		Client.mutex.Unlock()
		this.chRead <- Client
	}
}

func (this *ServerTCP) clientEvent() {
	var IPPort string = ""
	var Client *ClientTCP = nil

	for {
		select {
			case Client = <-this.chAccept:
				if Client == nil { continue }
				Client.datetime = time.Now()
				if this.OnClientAccept != nil { this.OnClientAccept(Client) }

			case Client = <-this.chRead:
				if Client == nil { continue }
				Client.datetime = time.Now()
				if this.OnClientRead != nil { this.OnClientRead(Client) }

			case Client = <-this.chWrite:
				//Log

			case Client = <-this.chClose:
				if Client == nil { continue }
				IPPort = Client.ipport
				this.CloseClient(Client)
				this.clearClient(IPPort)
				if this.OnClientDisconnected != nil { this.OnClientDisconnected(IPPort) }

			case <-this.chStop: return
		}
	}
}
