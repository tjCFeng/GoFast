package Socket

import (
	"net"
	"runtime"
	"sync"
	"time"
)

type onClientConnected func(Client *ClientTCP)
type onClientRead func(Client *ClientTCP)
type onClientWrite func(Client *ClientTCP)
type onClientDisconnected func(IPPort string)

type ClientTCP struct {
	//ClientTCP、ServerTCP共用
	hClient  *net.TCPConn
	ipport   string
	datetime time.Time
	dataBuf  []byte
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
