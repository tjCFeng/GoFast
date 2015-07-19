package GNSS

import (
	"strconv"
	"fmt"
	"strings"
)

type NMEA0183 struct {
	AV		string //'A'|'V'
	DT		string //"2006-01-02 15:04:05"
	EWNS	string //'E''N'|'W''S'
	Lon		float32 //"xxx.xxxx"
	Lat		float32 //"xx.xxxx"
	Speed	float32 //"xxx.xx" * 1.852
	Dir		float32 //"xxx.x"
	High	float32 //""
	Starts	uint8 //0x0~0xF
	ANT		uint8 //0:正常|1:短路|2:开路
}

func lon(data string) float32 {
	S, err := strconv.ParseFloat(data[6:], 32)
	if err != nil { return 0.0 }
	F, err := strconv.ParseFloat(data[3:5], 32)
	if err != nil { return 0.0 }
	D, err := strconv.ParseFloat(data[:3], 32)
	if err != nil { return 0.0 }
	
	return float32(D + (F + (S / 100000)) / 60)
}

func lat(data string) float32 {
	S, err := strconv.ParseFloat(data[5:], 32)
	if err != nil { return 0.0 }
	F, err := strconv.ParseFloat(data[2:4], 32)
	if err != nil { return 0.0 }
	D, err := strconv.ParseFloat(data[:2], 32)
	if err != nil { return 0.0 }
	
	return float32(D + (F + (S / 100000)) / 60)
}

func speed(data string) float32 {
	s, err := strconv.ParseFloat(data, 32)
	if err != nil { return 0.0 }
	return float32(s * 1.852)
}

func direction(data string) float32 {
	d, err := strconv.ParseFloat(data, 32)
	if err != nil { return 0.0 }
	return float32(d)
}

func (this *NMEA0183) rmc(data string) {
	ss := strings.Split(data, ",")
	
	if ss[2] != "A" { return } else { this.AV = "A"}
	this.DT = fmt.Sprintf("20%s-%s-%s %s:%s:%s", ss[9][4:6], ss[9][2:4], ss[9][0:2], ss[1][0:2], ss[1][2:4], ss[1][4:6])
	this.Lon = lon(ss[5])
	this.Lat = lat(ss[3])
	this.EWNS = ss[6] + ss[4]
	this.Speed = speed(ss[7])
	this.Dir = direction(ss[8])
}

func (this *NMEA0183) gga(data string) {
	if this.AV != "A" { return }
	
	ss := strings.Split(data, ",")
	this.Lon = lon(ss[4])
	this.Lat = lat(ss[2])
	this.EWNS = ss[5] + ss[3]
}

func (this *NMEA0183) gsa(data string) {
	return
}

func (this *NMEA0183) gsv(data string) {
	if this.AV != "A" { return }

	ss := strings.Split(data, ",")
	s, err := strconv.Atoi(ss[3])
	if err != nil { return }
	this.Starts = uint8(s)
}

func (this *NMEA0183) txt(data string) {
	switch data[len(data) - 4] {
		case 'K': this.ANT = 0
		case 'H': this.ANT = 1
		case 'P': this.ANT = 2
		default: this.ANT = 255
	}
}


func (this *NMEA0183) Data(data string) {
	s := data[1 : 6]

	switch s {
		case "GNRMC": this.rmc(data)
		case "GNGGA": this.gga(data)
		case "GPGSA": this.gsa(data)
		case "GPGSV": this.gsv(data)
		case "GNTXT": this.txt(data)
		
		case "BDGSA": this.gsa(data)
		case "BDGSV": this.gsv(data)
		
		default: return
	}
}
