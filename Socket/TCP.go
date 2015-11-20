package Socket

import (
	"errors"
	"time"
	"bytes"
	"net"
	"sync"
	"bufio"
)

/*Client***********************************************************************/
	type onConnected func(Client *ClientTCP)
	type onDisconnected func(Client *ClientTCP)
	type onReceived func(Client *ClientTCP)
	type onSended func(Client *ClientTCP)

	type ClientTCP struct {
		hClient		net.Conn
		Server		*ServerTCP
		dataBuf		*bytes.Buffer
		dataLen		int
		mutex		*sync.Mutex
		ipport		string //Server用时保存Client地址，Client用时保存Server地址重连用
		
		OnConnected		onConnected
		OnDisconnected	onDisconnected
		OnReceived		onReceived
		OnSended		onSended
	}
	
func (this *ClientTCP) ReConnect() error {
	return this.Connect(this.ipport)
}
	
func (this *ClientTCP) Connect(RemoteIPPort string) error { //"IP:Port"
	this.ipport = RemoteIPPort
	
	addr, err := net.ResolveTCPAddr("tcp", this.ipport)
	if err != nil { return err }
	this.hClient, err = net.DialTCP("tcp",  nil, addr)
	if err != nil { return err }
	
	this.dataBuf = bytes.NewBuffer(nil)
	this.dataLen = 0
	this.mutex = &sync.Mutex{}
	
	
	go this.clientEvent()
	if this.OnConnected != nil { this.OnConnected(this) }
	
	return nil
}

func (this *ClientTCP) GetIPPort() string {
	return this.hClient.LocalAddr().String()
}
	
/*Server & Client 共用**********************************************************/
func (this *ClientTCP) Close() error {
	return this.hClient.Close()
}

func (this *ClientTCP) GetData(Len int) (Data []uint8, Count int) {
	if (Len == 0) || (Len > this.dataLen) {
		return this.dataBuf.Bytes(), this.dataLen
	} else {
		return this.dataBuf.Bytes()[:Len], Len
	}
}

func (this *ClientTCP) ClearData(Len int) {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	
	switch Len {
		case 0:
			this.dataBuf.Reset()
			this.dataLen = 0
		default:
			tmp := this.dataBuf.Bytes()[Len:]
			this.dataBuf.Reset()
			this.dataBuf.Write(tmp)
			this.dataLen -= Len
	}
}

func (this *ClientTCP) Send(Data []uint8) (int, error) {
	Len, err := this.hClient.Write(Data)
	if this.Server != nil { //Server用
		this.Server.chWrite <- this
	} else {
		if this.OnSended != nil { this.OnSended(this) } //Client用
	}
	
	return Len, err
}

func (this *ClientTCP) clientEvent() {
	if this.Server != nil {
		this.hClient.SetDeadline(time.Now().Add(time.Duration(this.Server.OverTime) * time.Second))
	}
	
	reader := bufio.NewReader(this.hClient)
	Buf := make([]uint8, 1536)
	var Len int = 0
	var err error = nil
	
	for {
		Len, err = reader.Read(Buf)
		if err != nil {
			this.hClient.Close()
			this.ClearData(0)
			if this.Server != nil { //Server用
				this.Server.chClose <- this
			} else { //Client用
				if this.OnDisconnected != nil {
					this.Close()
					this.OnDisconnected(this)
				}
			}
			return
		}
		
		if this.Server != nil {
			this.hClient.SetDeadline(time.Now().Add(time.Duration(this.Server.OverTime) * time.Second))
		}
		this.mutex.Lock()
		this.dataBuf.Write(Buf[:Len])
		this.dataLen += Len
		this.mutex.Unlock()
		
		if this.Server != nil { //Server用
			this.Server.chRead <- this
		} else { //Client用
			if this.OnReceived != nil { this.OnReceived(this) }
		}
	}
}

/*Server***********************************************************************/
	type onClientAccept func(Sender *ServerTCP, Client *ClientTCP)
	type onClientClosed func(Sender *ServerTCP, IPPort string)
	type onClientRead func(Sender *ServerTCP, Client *ClientTCP)
	type onClientWrite func(Sender *ServerTCP, Client *ClientTCP)
	
	type ServerTCP struct {
		hServer		net.Listener
		clients		map[string]*ClientTCP
		
		chAccept	chan *ClientTCP
		chClose		chan *ClientTCP
		chRead		chan *ClientTCP
		chWrite		chan *ClientTCP
		chStop		chan bool
		
		OverTime			uint16
		OnClientAccept		onClientAccept
		OnClientRead		onClientRead
		OnClientWrite		onClientWrite
		OnClientClosed		onClientClosed
	}

func (this *ServerTCP) Listen(Port string) error {
	var err error
	
	this.hServer, err = net.Listen("tcp", ":" + Port)
	if err != nil { return err}
	
	this.chAccept = make(chan *ClientTCP, 1000)
	this.chClose = make(chan *ClientTCP, 1000)
	this.chRead = make(chan *ClientTCP, 1000)
	this.chWrite = make(chan *ClientTCP, 1000)
	this.chStop = make(chan bool)
	
	this.clients = make(map[string]*ClientTCP)
	
	if this.OverTime == 0 { this.OverTime = 30 }
	
	go this.clientEvent()
	go this.waitAccept()
	
	return err
}

func (this *ServerTCP) Stop() {
	defer func() {
		for _, Client := range this.clients {
			Client.hClient.Close()
			this.chClose <- Client
		}
	}()
	
	this.chStop <- true
	this.hServer.Close()
}

func (this *ServerTCP) waitAccept() {
	for {
		c, err := this.hServer.Accept()
		if err != nil { return }
		
		Client := new(ClientTCP)
		//Client.OverTime = this.OverTime
		Client.hClient = c
		Client.Server = this
		Client.ipport = c.RemoteAddr().String()
		Client.dataBuf = bytes.NewBuffer(nil)
		Client.dataLen = 0
		Client.mutex = &sync.Mutex{}
		this.clients[Client.ipport] = Client
		
		go Client.clientEvent()
		this.chAccept <- Client
	}
}

func (this *ServerTCP) clientEvent() {
	for {
		select {
		case <- this.chStop:
			return
		case Client := <- this.chAccept:
			if this.OnClientAccept != nil { this.OnClientAccept(this, Client) }
		case Client := <- this.chClose:
			delete(this.clients, Client.ipport)
			if this.OnClientClosed!= nil { this.OnClientClosed(this, Client.ipport) }
		case Client := <- this.chRead:
			if this.OnClientRead != nil { this.OnClientRead(this, Client) }
		case Client := <- this.chWrite:
			if this.OnClientWrite != nil { this.OnClientWrite(this, Client) }
		}
	}
}

func (this *ServerTCP) Send(IPPort string, Data []uint8) (int, error) {
	Client, ok := this.clients[IPPort]
	if !ok { return 0, errors.New("unconnected") }

	Len, err := Client.Send(Data)
	this.chWrite <- Client
	
	return Len, err
}

func (this *ServerTCP) ClientCount() int {
	return len(this.clients)
}
