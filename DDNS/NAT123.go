package DDNS

import (
	"runtime"
	"fmt"
	"time"
	"io/ioutil"
	"net/http"
	"strings"
)

type nat123_com struct {
	username	string
	password	string
	domain		string
	freqUpdate	uint8 //Minute
	
	lastUpdate	time.Time
	lastResult	string
	msg			chan uint8
}

var inat123_com *nat123_com = nil

func Inat123(Username, Password, Domain string, Freq uint8) *nat123_com {
	if inat123_com == nil {
		inat123_com = &nat123_com{}
		inat123_com.username = Username
		inat123_com.password = Password
		inat123_com.domain = Domain
		inat123_com.freqUpdate = Freq
		
		inat123_com.msg = make(chan uint8)
		inat123_com.lastUpdate = time.Now()
	}
	
	return inat123_com
}

func FreeInat123() {
	
}

/*****************************************************************************/
func (this *nat123_com) getIP() string {
	Resp, err := http.Get("http://ddns.nat123.com/")
	defer Resp.Body.Close()
	if err != nil {
		panic(err)
	}

	Body, err := ioutil.ReadAll(Resp.Body)
	if err != nil { return "" }
	
	IP := string(Body)
	I := strings.Index(IP, ":")
	IP = IP[I + 2:]
	
	return IP
}

func (this *nat123_com) update(IP string) {
	this.lastResult = "Unknow"
	this.lastUpdate = time.Now()
	
	URL := "http://%s:%s@ddns.nat123.com/update.jsp?hostname=%s&myip=%s"
	URL = fmt.Sprintf(URL, this.username, this.password, this.domain, IP)
	
	Resp, err := http.Get(URL)
	defer Resp.Body.Close()
	if err != nil { return }
	
	
	Body, err := ioutil.ReadAll(Resp.Body)
	if err != nil { return }

	this.lastResult = string(Body)
	
	/*switch this.lastResult {
		case "good": fmt.Println("更新成功")
		case "nochg": fmt.Println("IP未变化")
		case "nohost": fmt.Println("域名不存在")
		case "badauth": fmt.Println("用户名或密码错误")
		case "abuse": fmt.Println("请求失败，连接频繁")
		case "servererror": fmt.Println("系统错误")
	}*/
}

func (this *nat123_com) Start() {
	go func(){
		runtime.Gosched()
		for {
			select {
				case <- time.After(time.Minute * time.Duration(this.freqUpdate)): 
					this.update(this.getIP())
				case <- this.msg: return
			}
		}
	}()	
}

func (this *nat123_com) Stop() {
	this.msg <- 1
}

func (this *nat123_com) UpdateResult() string {
	return this.lastResult
}

func (this *nat123_com) UpdateTime() time.Time {
	return this.lastUpdate
}
