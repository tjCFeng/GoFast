package ICMP

import (
	"encoding/binary"
	"bytes"
	"os"
	"time"
	"net"
)

const (
	icmpV4EchoRequest = 8
	icmpV4EchoReply = 0
	icmpV6EchoRequest = 128
	icmpV6EchoReply = 129
)

type ICMP struct {
	icmpType	uint8
	icmpCode	uint8
	icmpCheck	uint16
	icmpID		uint16
	icmpSeq		uint16
}

func checkSum(data []byte) uint16 {
	var (
		sum uint32
		length int = len(data)
		index int
	)
	
	for length > 1 {
		sum += uint32((data[index] << 8) + data[index + 1])
		index += 2
		length -= 2
	}
	
	if length > 0 { sum += uint32(data[index]) }
	sum += (sum >> 16)
	
	return uint16(^sum)
}

func Ping(URL string, TimeoutS int) bool {
	addr, err := net.ResolveIPAddr("ip", URL)
	if err != nil { return false } 
	
	c, err := net.DialIP("ip4:icmp", nil, addr)
	if err != nil { return false }
	defer c.Close()
	c.SetDeadline(time.Now().Add(time.Duration(TimeoutS) * time.Second))
	
	icmp := new(ICMP)
	icmp.icmpType = icmpV4EchoRequest
	icmp.icmpCode = 0
	icmp.icmpID = uint16(os.Getpid() & 0xFFFF)
	icmp.icmpSeq = 1
	
	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, icmp)
	icmp.icmpCheck = checkSum(buf.Bytes())
	buf.Reset()
	binary.Write(&buf, binary.BigEndian, icmp)
	
	count, err := c.Write(buf.Bytes())
	if err != nil { return false }
	
	buffer := make([]uint8, count + 20)
	count, err = c.Read(buffer)
	if err != nil { return false }

	ID := (uint16(buffer[count - 4]) << 8) + uint16(buffer[count - 3])
	
	return icmp.icmpID == ID
}
