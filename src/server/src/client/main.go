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
)

var roomManager = &RoomManager{}

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
		c := client{}
		c.init(*ip, 5000, fmt.Sprintf("fbid_%d", *id), uint32(*room))
		c.robot.SwitchDriver("no ai")
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
				case "gp", "getprofile":
					c.sendGetProfile()
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
						c.sendSendDiamonds(uint32(uid), uint32(diamonds))
					}
				case "dr":
					c.sendDiamondsRecords()
				case "lr", "listrooms":
					c.sendListRooms()
				case "cr", "createroom":
					c.sendCreateRoom()
				case "closeroom":
					if count > 1 {
						if roomID, err := strconv.Atoi(words[1]); err != nil {
							continue
						} else {
							c.sendCloseRoom(uint32(roomID))
						}
					}
				case "jr", "joinroom":
					if count > 1 {

						if number, err := strconv.Atoi(words[1]); err != nil {
							continue
						} else {
							c.sendJoinRoom(number)
						}
					}
				case "sd", "sitdown":
					seatID := c.getEmptySeatID()
					if seatID >= 0 {
						c.sendSitDown(uint32(seatID))
					}
				case "sg", "startgame":
					c.sendStartGame()
				case "b", "bet":
					c.sendBet()
				case "c", "combine":
					c.sendCombine()
				case "su", "standup":
					c.sendStandUp()
				case "leaveroom":
					c.sendLeaveRoom()
				case "ab", "autobanker":

				case "sb", "scoreboard":
					c.sendGetScorebard()
				case "rh", "roundhistory":
					if count > 1 {
						if round, err := strconv.Atoi(words[1]); err != nil {
							continue
						} else {
							c.sendGetRoundHistory(uint32(round))
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
							c.sendCareerWinLoseData(days)
						}
					}
				case "cr1":
					if count > 1 {
						if days, err := strconv.Atoi(words[1]); err != nil {
							continue
						} else {
							c.sendCareerRoomRecords(uint32(days))
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
