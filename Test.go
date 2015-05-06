package main

import (
	"time"
	"fmt"
	"github.com/tjCFeng/GoServer/DDNS"
)

func main() {
	ddns := DDNS.Inat123("username", "password", "domain", 3)
	ddns.Start()
	time.Sleep(time.Minute * 10)
	ddns.Stop()
	fmt.Println(ddns.UpdateResult())
}
