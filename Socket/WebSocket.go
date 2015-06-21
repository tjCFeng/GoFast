package Socket

import (
	"fmt"
	"bytes"
	"crypto/sha1"
	"encoding/base64"
)

type onWSClientConnected onClientConnected
type onWSClientRead func(Client *ClientTCP)
type onWSClientDisconnected onClientDisconnected

func WebSocketGetData(Client *ClientTCP) []uint8 {
	D := Client.Tag.([]uint8)
	return D
}

func WebSocketClearData(Client *ClientTCP, Len int) {
	//Client.mutex.Lock()
	//defer Client.mutex.Unlock()

	D := Client.Tag.([]uint8)
	switch Len {
		case 0: D = D[0:0]
		default: D = D[Len - 1:]
	}
	Client.Tag = D
}

const (
	CFIN        = 0x80 // 1000 0000
	CRSV1       = 0x40 // 0111 0000
	CRSV2       = 0x20 // 0011 0000
	CRSV3       = 0x10 // 0001 0000
	COPCODE     = 0x0F // 0000 1111
	CMASK       = 0x80 // 1000 0000
	CPAYLOADLEN = 0x7F // 0111 1111
	HAS_EXTEND_DATA = 126
	HAS_EXTEND_DATA_CONTINUE = 127
)

const (
	WS_CodeContinue = 0x00 //Continuation Frame
	WS_CodeText     = 0x01 //Text Frame
	WS_CodeBinary   = 0x02 //Binary Frame
	WS_CodeClose    = 0x08 //Close Frame
	WS_CodePing     = 0x09 //Ping Frame
	WS_CodePong     = 0x0A //Frame Frame
	
	WS_CloseNormal        = 1000 //Normal valid closure, connection purpose was fulfilled
	WS_CloseShutdown      = 1001 //Endpoint is going away (like server shutdown)
	WS_CloseErrorProtocol = 1002 //Protocol error
	WS_CloseErrorData     = 1003 //Unknown frame data type
	WS_CloseTooLargeFrame = 1004 //Too large frame received
	WS_CloseNoStatus      = 1005 //Close received by peer but without any close code
	WS_CloseErrorClose    = 1006 //Abnotmal connection shutdown close code
	WS_CloseErrorUTF8     = 1007 //Received text data are not valid UTF-8
)


type ServerWS struct {
	Server *ServerTCP

	Mask []uint8
	OnWSClientConnected onWSClientConnected
	OnWSClientRead   onWSClientRead
	OnWSClientDisconnected  onWSClientDisconnected
}

func (this *ServerWS) Listen(Port string) error {
	this.Server = new(ServerTCP)
	this.Server.OnClientRead = this.clientOnRead
	this.Server.OnClientDisconnected = this.clientOnClose
	return this.Server.Listen(Port)
}

func (this *ServerWS) Stop() error {
	return this.Server.Stop()
}

func (this *ServerWS) clientOnWrite(Client *ClientTCP, Buf *[]uint8) {
	Data := this.encodeData(*Buf, this.Mask)
	*Buf = Data
}

func (this *ServerWS) clientOnRead(Client *ClientTCP) {
	Data, _ := Client.GetData()
	Client.ClearData(0)
	
	if Client.Tag == nil {
		Client.Tag = make([]uint8, 0)
		Data, _ = this.shakeHand(Data)
		Client.Send(Data)
		if this.OnWSClientConnected != nil { this.OnWSClientConnected(Client) }
		Client.OnClientWrite = this.clientOnWrite
		return
	}
	
	//Client.mutex.Lock()
	//defer Client.mutex.Unlock()
	Data, code := this.decodeData(Data)
	if code == WS_CodeClose { Client.Close(); return }
	D := Client.Tag.([]uint8)
	D = append(D, Data...)
	Client.Tag = D
	if this.OnWSClientRead != nil { this.OnWSClientRead(Client) }
}

func (this *ServerWS) clientOnClose(IPPort string) {
	if this.OnWSClientDisconnected != nil { this.OnWSClientDisconnected(IPPort) }
}

func (this *ServerWS)shakeHand(HeaderWS []uint8) ([]uint8, bool) {
	var S, Upgrade, WSVersion, WSKey []uint8

	Header := bytes.Split(HeaderWS, []uint8("\r\n"))
	for I := 0; I < len(Header); I++ {
		if len(Header[I]) == 0 { continue }

		S = Header[I]
		switch S[0] {
			case 'G': //URL = S[4 : bytes.Index(S, []uint8("HTTP"))] //GET
			case 'U':
				switch S[1] {
					case 'p': Upgrade = S[9 : ] //Upgrade
					case 's': //UserAgent = S[12 : ] //User-Agent
				}
			case 'C':
				switch S[1] {
					case 'o': //Connection = S[12 : ] //Connection
					case 'a': //CacheControl = S[15 : ] //Cache-Control
				}
			case 'H': //Host = S[6 : ] //Host
			case 'O': //Origin = S[8 : ] //Origin
			case 'S':
				switch S[14] {
					case 'O': //WSOrigin = S[22 : ]//Sec-WebSocket-Origin
					case 'E': //WSExtensions = S[27 : ] //Sec-WebSocket-Extensions
					case 'V': WSVersion = S[23 : ] //Sec-WebSocket-Version
					case 'K': WSKey = S[19 : ] //Sec-WebSocket-Key
				}
		}
	}

	switch string(WSVersion) {
		case "13":
			w := sha1.New()
			_, err := w.Write([]uint8(string(WSKey) + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
			if err != nil { return nil, false }
			Key := base64.StdEncoding.EncodeToString(w.Sum(nil))
			
			Result := []uint8(fmt.Sprintf(
				"HTTP/1.1 101 Switching Protocols\r\n" + 
				"Upgrade: %s\r\n" +
				"Connection: Upgrade\r\n" +
				"Sec-WebSocket-Accept: %s\r\n\r\n", Upgrade, Key))

			return Result, true
			
		default: return nil, false
	}
}

/*****************************************************************************/


func (this *ServerWS)decodeData(Data []uint8) ([]uint8, uint8) {
	MaskChar := make([]uint8, 4)

	/*FIN := (Data[0] & CFIN) == CFIN
	RSV1 := (Data[0] & CRSV1) == CRSV1
	RSV2 := (Data[0] & CRSV2) == CRSV2
	RSV3 := (Data[0] & CRSV3) == CRSV3*/
	OPCODE := (Data[0] & COPCODE)

	Mask := (Data[1] & CMASK) == CMASK
	Size := (Data[1] & CPAYLOADLEN)

	Index := 2
	switch Size {
		case 126:
			Size = Data[Index] << 8; Index++
			Size += Data[Index]; Index++
		case 127:
			Size = Data[Index] << 56; Index++
			Size += Data[Index] << 48; Index++
			Size += Data[Index] << 40; Index++
			Size += Data[Index] << 32; Index++
			Size += Data[Index] << 24; Index++
			Size += Data[Index] << 16; Index++
			Size += Data[Index] << 8; Index++
			Size += Data[Index]; Index++
		default: ;
	}

	if Mask {
		MaskChar = Data[Index : Index + 4]
		Index += 4
	}

	I := 0
	Len := len(Data)
	Buf := make([]uint8, Len - Index + 1)
	for {
		if Index >= Len { break }
		Buf[I] = Data[Index] ^ MaskChar[(I % 4)]
		Index++
		I++
	}

	return Buf, OPCODE
}

func (this *ServerWS)encodeData(Data []uint8, Mask []uint8) []uint8 {
	var I int = 0

	B1 := uint8(CFIN)
	B1 += WS_CodeText

	B2 := uint8(0)
	if Mask != nil {
		B2 = uint8(CMASK)
	}

	Len := len(Data)
	if (Len <= 125) {
		B2 += uint8(Len)
		I = 2
	} else if (Len >= 126) && (Len <= 65535) {
		B2 += 126
		I = 6
	} else {
		B2 += 127
		I = 18
	}

	Result := make([]uint8, 2)
	Result[0] = B1
	Result[1] = B2
	if (Len <= 125) {

	} else if (Len >= 126) && (Len <= 65535) {
		Result = append(Result, uint8(Len >> 8), uint8(Len & 0xFF))
		I += 2
	} else {
		Result = append(Result, 
			uint8((Len >> 56) & 0xFF),
			uint8((Len >> 48) & 0xFF),
			uint8((Len >> 40) & 0xFF),
			uint8((Len >> 32) & 0xFF),
			uint8((Len >> 24) & 0xFF),
			uint8((Len >> 16) & 0xFF),
			uint8((Len >> 8) & 0xFF),
			uint8(Len & 0xFF))
		I += 8
	}

	if Mask != nil {
		Result = append(Result, Mask...)
		I += 4
		J := 0
		for {
			if J == Len { break }
			Data[J] = Data[J] ^ Mask[(I % 4)]
			J++
			I++
		}
	}
		
	Result = append(Result, Data...)
	return Result
}
