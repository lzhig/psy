package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
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
				line, _, _ := reader.ReadLine()
				words := strings.Split(string(line), " ")
				count := len(words)
				if count == 0 {
					showHelp()
					continue
				}
				cmd := words[0]
				switch cmd {
				case "cr", "createroom":
					c.sendCreateRoom()
				case "jr", "joinroom":
					if count > 1 {

						if number, err := strconv.Atoi(words[1]); err != nil {
							continue
						} else {
							c.sendJoinRoom(number)
						}
					}
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

				case "sb", "scoreboard":
					c.sendGetScorebard()
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
	cr, createroom - create room
	jr, joinroom - join room
	sd, sitdown - sit down
	sg, startgame - start game
	b, bet - bet
	su, standup - stand up
	lr, leaveroom - leave room
	ab, autobanker - auto banker
	sb, scoreboard - scoreboard
	h, help - print help
	`
	fmt.Println(str)
}
