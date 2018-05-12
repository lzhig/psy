package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"./client"
	"./room"
)

var roomManager = &room.RoomManager{}

func main() {
	var ip = flag.String("address", "127.0.0.1:8010", "help message for flagname")
	var num = flag.Int("num", 1, "connections")
	var id = flag.Int("id", 0, "id")
	var room = flag.Int("room", 0, "room number")
	flag.Parse()
	fmt.Println("id:", *id)
	runtime.GOMAXPROCS(4)

	roomManager.Init()

	if *num == 1 {
		fmt.Println("interactive mode")
		c := client.Client{}
		c.Init(*ip, 5000, fmt.Sprintf("fbid_%d", *id), uint32(*room), roomManager)
		c.GetRobot().SwitchDriver("no ai")
		go c.Start()

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
				case "gp", "getprofile":
					c.SendGetProfile()
				case "senddiamonds":
					if count > 2 {
						uid, err := strconv.Atoi(words[1])
						if err != nil {
							continue
						}
						diamonds, err := strconv.Atoi(words[2])
						if err != nil {
							continue
						}
						c.SendSendDiamonds(uint32(uid), uint32(diamonds))
					}
				case "dr":
					c.SendDiamondsRecords()
				case "lr", "listrooms":
					c.SendListRooms()
				case "cr", "createroom":
					c.SendCreateRoom()
				case "closeroom":
					if count > 1 {
						if roomID, err := strconv.Atoi(words[1]); err != nil {
							continue
						} else {
							c.SendCloseRoom(uint32(roomID))
						}
					}
				case "jr", "joinroom":
					if count > 1 {

						if number, err := strconv.Atoi(words[1]); err != nil {
							continue
						} else {
							c.SendJoinRoom(number)
						}
					}
				case "sd", "sitdown":
					if count > 1 {
						if number, err := strconv.Atoi(words[1]); err != nil {
							continue
						} else {
							c.SendSitDown(uint32(number))
						}
					}
				case "sg", "startgame":
					c.SendStartGame()
				case "b", "bet":
					c.SendBet()
				case "c", "combine":
					c.SendCombine()
				case "su", "standup":
					c.SendStandUp()
				case "leaveroom":
					c.SendLeaveRoom()
				case "ab", "autobanker":

				case "sb", "scoreboard":
					c.SendGetScorebard()
				case "rh", "roundhistory":
					if count > 1 {
						if round, err := strconv.Atoi(words[1]); err != nil {
							continue
						} else {
							c.SendGetRoundHistory(uint32(round))
						}
					}
				case "cd":
					if count > 1 {
						parse := func() ([]uint32, error) {
							days := make([]uint32, count-1)
							for i := 1; i < count; i++ {
								d, err := strconv.Atoi(words[i])
								if err != nil {
									return nil, err
								}
								days[i-1] = uint32(d)
							}
							return days, nil
						}
						if days, err := parse(); err != nil {
							continue
						} else {
							c.SendCareerWinLoseData(days)
						}
					}
				case "cr1":
					if count > 1 {
						if days, err := strconv.Atoi(words[1]); err != nil {
							continue
						} else {
							c.SendCareerRoomRecords(uint32(days))
						}
					}
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
		time.Sleep(time.Millisecond * 100)
		go func(i int) {
			c := &client.Client{}
			c.Init(*ip, 5000, fmt.Sprintf("fbid_%d", i), uint32(*room), roomManager)
			c.Start()
		}(i)
	}
	select {}
}

func showHelp() {
	str :=
		`commands list:
	gp, getprofile - get profile
	senddiamonds - send diamonds
	dr - diamonds records
	lr, listrooms - list rooms
	cr, createroom - create room
	closeroom - close room
	jr, joinroom - join room
	sd, sitdown - sit down
	sg, startgame - start game
	b, bet - bet
	c, combine - combine
	su, standup - stand up
	leaveroom - leave room
	ab, autobanker - auto banker
	sb, scoreboard - scoreboard
	rh, roundhistory - round history
	cd - career data
	cr1 - career record
	h, help - print help
	`
	fmt.Println(str)
}
