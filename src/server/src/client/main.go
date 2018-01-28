package main

import (
	"flag"
	"fmt"
	"runtime"
)

func main() {
	var ip = flag.String("address", "192.168.2.50:8010", "help message for flagname")
	var num = flag.Int("num", 1, "connections")
	flag.Parse()
	runtime.GOMAXPROCS(4)
	fmt.Println("connecting - ", *ip)
	for i := 0; i < *num; i++ {
		go func() {
			c := &client{}
			c.init(*ip, 5000, fmt.Sprintf("fbid_%d", i))
			c.start()
		}()
	}
	select {}
}
