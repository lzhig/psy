package main

import (
	"time"

	"../msg"
	"github.com/lzhig/rapidgo/base"
)

type protocolHandler struct {
	handler   map[msg.MessageID]func(*msg.Protocol)
	protoChan chan *msg.Protocol
	c         *client
}

func (obj *protocolHandler) init(c *client) {
	obj.c = c
	obj.handler = map[msg.MessageID]func(*msg.Protocol){
		msg.MessageID_Login_Rsp:        obj.handleLogin,
		msg.MessageID_CreateRoom_Rsp:   obj.handleCreateRoom,
		msg.MessageID_JoinRoom_Rsp:     obj.handleJoinRoom,
		msg.MessageID_SitDown_Rsp:      obj.handleSitDown,
		msg.MessageID_StandUp_Rsp:      obj.handleStandUp,
		msg.MessageID_LeaveRoom_Rsp:    obj.handleLeaveRoom,
		msg.MessageID_Bet_Rsp:          obj.handleBet,
		msg.MessageID_GameState_Notify: obj.handleGameStateNotify,
		msg.MessageID_ListRooms_Rsp:    obj.handleListRooms,
		msg.MessageID_JoinRoom_Notify:  obj.handleJoinRoomNotify,
		msg.MessageID_SitDown_Notify:   obj.handleSitDownNotify,
	}
	obj.protoChan = make(chan *msg.Protocol)

	go obj.loop()
}

func (obj *protocolHandler) getProtoChan() chan<- *msg.Protocol {
	return obj.protoChan
}

func (obj *protocolHandler) loop() {
	defer base.LogPanic()
	for {
		select {
		case p := <-obj.protoChan:
			obj.handle(p)
		}
	}
}

func (obj *protocolHandler) handle(p *msg.Protocol) {
	if f, ok := obj.handler[p.Msgid]; ok {
		log(obj.c, "received ", p)
		f(p)
	} else {
		log(obj.c, "cannot find handler for msg:", p)
	}
}

func (obj *protocolHandler) handleLogin(p *msg.Protocol) {
	if p.LoginRsp.Ret != msg.ErrorID_Ok {
		time.Sleep(time.Second)
		obj.c.sendLoginReq()
		return
	}
	obj.c.uid = p.LoginRsp.Uid
	obj.c.robot.DoAction(actionLogin)
	//obj.c.sendCreateRoom()
	//obj.c.sendJoinRoom()
}

func (obj *protocolHandler) handleCreateRoom(p *msg.Protocol) {
	if p.CreateRoomRsp.Ret == msg.ErrorID_Ok {
		c := make(chan struct{})
		roomManager.Send(roomManagerEventCreateRoom, []interface{}{p.CreateRoomRsp.RoomId, p.CreateRoomRsp.RoomNumber, c})
		<-c
		obj.c.robot.DoAction(actionListRooms)
	}
}

func (obj *protocolHandler) handleListRooms(p *msg.Protocol) {
	if p.ListRoomsRsp.Ret == msg.ErrorID_Ok {
		for _, room := range p.ListRoomsRsp.Rooms {
			c := make(chan struct{})
			roomManager.Send(roomManagerEventCreateRoom, []interface{}{room.RoomId, room.RoomNumber, c})
			<-c
		}

		obj.c.robot.DoAction(actionListRooms)
	}
}

func (obj *protocolHandler) handleJoinRoom(p *msg.Protocol) {
	if p.JoinRoomRsp.Ret == msg.ErrorID_Ok {
		obj.c.room = p.JoinRoomRsp.Room
		log(obj.c, "JoinRoom. roomid:", obj.c.room.RoomId)

		// 入座的玩家数
		num := uint32(0)
		for _, player := range p.JoinRoomRsp.Room.Players {
			if player.SeatId >= 0 {
				num++
			}
		}
		c := make(chan struct{})
		roomManager.Send(roomManagerEventJoinRoom, []interface{}{p.JoinRoomRsp.Room.RoomId, c})
		<-c
		obj.c.robot.DoAction(actionJoinRoom)
	} else if p.JoinRoomRsp.Ret == msg.ErrorID_JoinRoom_Full ||
		p.JoinRoomRsp.Ret == msg.ErrorID_JoinRoom_Already_In ||
		p.JoinRoomRsp.Ret == msg.ErrorID_JoinRoom_Released {
		obj.c.robot.DoAction(actionListRooms)
	} else {
		base.LogWarn(obj.c.uid, "No Action!!!!")
	}
}

func (obj *protocolHandler) handleJoinRoomNotify(p *msg.Protocol) {
	player := &msg.Player{
		Uid:    p.JoinRoomNotify.Uid,
		Name:   p.JoinRoomNotify.Name,
		Avatar: p.JoinRoomNotify.Avatar,
		SeatId: -1,
	}
	obj.c.room.Players = append(obj.c.room.Players, player)
}

func (obj *protocolHandler) handleSitDown(p *msg.Protocol) {
	if p.SitDownRsp.Ret == msg.ErrorID_SitDown_CreditPoints_Out || p.SitDownRsp.Ret == msg.ErrorID_SitDown_Invalid_Seat_Id {
		obj.c.seatID = -1
		obj.c.sendLeaveRoom()
		return
	} else if p.SitDownRsp.Ret == msg.ErrorID_SitDown_Already_Exist_Player {
		obj.c.sitDown()
		return
	} else if p.SitDownRsp.Ret == msg.ErrorID_SitDown_Already_Sit {
		return
	} else if p.SitDownRsp.Ret == msg.ErrorID_Ok {
		//obj.c.sendStandUp()
		//obj.c.sendBet()
		log(obj.c, "SitDown. roomid:", obj.c.room.RoomId, ", seatID:", obj.c.seatID)
		c := make(chan struct{})
		roomManager.Send(roomManagerEventSitDown, []interface{}{obj.c.room.RoomId, c})
		<-c

		if obj.c.seatID == 0 {
			for _, player := range obj.c.room.Players {
				if player.Uid == obj.c.uid {
					player.SeatId = 0
					break
				}
			}
			obj.c.waittingPlayers = uint32(1 + obj.c.s.Int31n(3))
		}
	} else {
		obj.c.robot.DoAction(actionJoinRoom)
	}
}

func (obj *protocolHandler) handleSitDownNotify(p *msg.Protocol) {
	uid := p.SitDownNotify.Uid

	for _, player := range obj.c.room.Players {
		if player.Uid == uid {
			player.SeatId = int32(p.SitDownNotify.SeatId)
			break
		}
	}
	if obj.c.seatID == 0 {
		obj.c.robot.DoAction(actionSitDownNotify)
	}
}

func (obj *protocolHandler) handleStandUp(p *msg.Protocol) {
	if p.StandUpRsp.Ret == msg.ErrorID_Ok {
		//obj.c.sendLeaveRoom()
		c := make(chan struct{})
		roomManager.Send(roomManagerEventStandUp, []interface{}{obj.c.room.RoomId, c})
		<-c

		obj.c.robot.DoAction(actionStandUp)
	} else {
		base.LogWarn(obj.c.uid, "No Action!!!!")
	}
}

func (obj *protocolHandler) handleLeaveRoom(p *msg.Protocol) {
	if p.LeaveRoomRsp.Ret == msg.ErrorID_Ok {
		c := make(chan struct{})
		roomManager.Send(roomManagerEventLeaveRoom, []interface{}{obj.c.room.RoomId, c})
		<-c
		obj.c.room = nil
		obj.c.robot.DoAction(actionListRooms)
	} else {
		base.LogWarn(obj.c.uid, "No Action!!!!!")
	}
}

func (obj *protocolHandler) handleBet(p *msg.Protocol) {
}

func (obj *protocolHandler) handleGameStateNotify(p *msg.Protocol) {
	switch p.GameStateNotify.State {
	case msg.GameState_Ready:
		if obj.c.seatID == 0 {
			obj.c.sendStartGame()
		}
	case msg.GameState_Bet:
		obj.c.room.PlayedHands = p.GameStateNotify.PlayedHands
		if obj.c.seatID > 0 {
			time.Sleep(time.Duration(2+obj.c.s.Int31n(7)) * time.Second)
			obj.c.sendBet()
		}
	case msg.GameState_Deal:
		obj.c.cards = p.GameStateNotify.DealCards
	case msg.GameState_Combine:
		time.Sleep(time.Duration(10+obj.c.s.Int31n(30)) * time.Second)
		obj.c.sendCombine()

	case msg.GameState_Result:
		//base.LogInfo("uid:", obj.c.uid)
		if obj.c.room.PlayedHands+1 == obj.c.room.Hands {
			time.Sleep(2 * time.Second)
			if obj.c.seatID == 0 {
				c := make(chan struct{})
				roomManager.Send(roomManagerEventCloseRoom, []interface{}{obj.c.room.RoomId, c})
				<-c
			}

			//base.LogInfo("roomid:", obj.c.room.RoomId)

			obj.c.room = nil
			obj.c.seatID = -1

			obj.c.robot.DoAction(actionListRooms)
			return
		}
	}
}
