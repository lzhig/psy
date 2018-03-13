package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"runtime"
)

func main() {
	var ip = flag.String("address", "127.0.0.1:8010", "help message for flagname")
	var num = flag.Int("num", 1, "connections")
	var id = flag.Int("id", 0, "id")
	var room = flag.Int("room", 0, "room number")
	flag.Parse()
	fmt.Println("id:", *id)
	runtime.GOMAXPROCS(4)
	if *num == 1 {
		fmt.Println("interactive mode")
		c := client{}
		c.init(*ip, 5000, fmt.Sprintf("fbid_%d", *id), uint32(*room))
		go c.start()

		go func() {
			reader := bufio.NewReader(os.Stdin)
			for {
				cmd, _, _ := reader.ReadLine()
				switch string(cmd) {
				case "jr", "joinroom":
					c.sendJoinRoom()
				case "sd", "sitdown":
					c.sendSitDown()
				case "sg", "startgame":
					c.sendStartGame()
				case "b", "bet":
					c.sendBet()
				case "su", "standup":
					c.sendStandUp()
				case "lr", "leaveroom":
					c.sendLeaveRoom()
				case "ab", "autobanker":

				case "h", "help":
					showHelp()
				default:
					fmt.Println("invalid command")
					showHelp()
				}
			}
		}()
		select {}
	}
	fmt.Println("connecting -", *ip)
	for i := 0; i < *num; i++ {
		go func(i int) {
			c := &client{}
			c.init(*ip, 5000, fmt.Sprintf("fbid_%d", i), uint32(*room))
			c.start()
		}(i)
	}
	select {}
}

func showHelp() {
	str :=
		`commands list:
	jr, joinroom - join room
	sd, sitdown - sit down
	sg, startgame - start game
	b, bet - bet
	su, standup - stand up
	lr, leaveroom - leave room
	ab, autobanker - auto banker
	h, help - print help
	`
	fmt.Println(str)
}
