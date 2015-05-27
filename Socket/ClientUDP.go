package Socket

import (
	"net"
	"time"
	"sync"
	"runtime"
)

type onClientReceived func(Client *ClientUDP)
type onClientClose func(Client *ClientUDP)

type ClientUDP struct {
	hClient  *net.UDPConn
	remoteAddr    *net.UDPAddr
	ipport   string
	datetime time.Time
	dataBuf  []byte
	dataLen  int
	mutex    *sync.Mutex
	
	chReceived		chan bool
	chClose  chan bool
	
	OnClientReceived		onClientReceived
	OnClientClose			onClientClose
}

func (this *ClientUDP) Connect(RemoteIP, RemotePort string) error {
	var err error
	this.remoteAddr, err = net.ResolveUDPAddr("udp", RemoteIP + ":" + RemotePort)
	if err != nil { return err }
	
	this.chReceived = make(chan bool)
	this.chClose = make(chan bool)

	this.hClient, err = net.DialUDP("udp",  nil, this.remoteAddr)
	if err != nil { return err }

	this.mutex = &sync.Mutex{}
	go this.clientEvent()
	go this.read()

	return nil
}

func (this *ClientUDP) Close() {
	this.hClient.Close()
}

func (this *ClientUDP) Send(Data []uint8) (int, error) {
	return this.hClient.Write(Data)
}

func (this *ClientUDP) SendUDP(Data []uint8) (int, error) {
	return this.hClient.WriteToUDP(Data, this.remoteAddr)
}

func (this *ClientUDP) IPPort() string {
	return this.ipport
}

func (this *ClientUDP) GetData() (Data []uint8, Len int) {
	return this.dataBuf, this.dataLen
}

func (this *ClientUDP) ClearData(Len int) {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	
	switch Len {
		case 0:
			this.dataBuf = this.dataBuf[0:0]
			this.dataLen = 0
		default:
			this.dataBuf = this.dataBuf[Len-1:]
			this.dataLen -= Len
	}
}

func (this *ClientUDP) GetDateTime() time.Time {
	return this.datetime
}

func (this *ClientUDP) read() {
	Buf := make([]uint8, 1536)

	for {
		count, _, err := this.hClient.ReadFromUDP(Buf)
		if (err != nil) || (count == 0) {
			this.chClose <- true
			runtime.Goexit()
			return
		}

		this.mutex.Lock()
		this.dataBuf = append(this.dataBuf, Buf[0:count]...)
		this.dataLen += count
		this.mutex.Unlock()
		this.chReceived <- true
	}
}

func (this *ClientUDP) clientEvent() {
	for {
		select {
			case <- this.chReceived:
				this.datetime = time.Now()
				if this.OnClientReceived != nil { this.OnClientReceived(this) }

			//case <- this.chWrite:
				//Log

			case <- this.chClose:
				if this.OnClientClose != nil { this.OnClientClose(this) }
				return
		}
	}
}
