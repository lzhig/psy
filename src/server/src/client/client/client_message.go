package client

import (
	"time"

	"../../msg"
	"../room"
	"github.com/lzhig/rapidgo/base"
)

func (obj *Client) handleLogin(args interface{}) {
	p := args.(*msg.Protocol)
	if p.LoginRsp.Ret != msg.ErrorID_Ok {
		time.Sleep(time.Second)
		obj.sendLoginReq()
		return
	}
	obj.uid = p.LoginRsp.Uid
	obj.robot.DoAction(actionLogin)
	//obj.sendCreateRoom()
	//obj.SendJoinRoom()
}

func (obj *Client) handleCreateRoom(args interface{}) {
	p := args.(*msg.Protocol)
	if p.CreateRoomRsp.Ret == msg.ErrorID_Ok {
		obj.roomManager.AddRoom(room.NewRoom(p.CreateRoomRsp.RoomId, p.CreateRoomRsp.RoomNumber))
		obj.robot.DoAction(actionListRooms)
	}
}

func (obj *Client) handleListRooms(args interface{}) {
	p := args.(*msg.Protocol)
	if p.ListRoomsRsp.Ret == msg.ErrorID_Ok {
		for _, r := range p.ListRoomsRsp.Rooms {
			obj.roomManager.AddRoom(room.NewRoom(r.RoomId, r.RoomNumber))
		}

		obj.robot.DoAction(actionListRooms)
	}
}

func (obj *Client) handleJoinRoom(args interface{}) {
	p := args.(*msg.Protocol)
	obj.roomManager.JoiningRoom(obj.joiningRoomID, false)
	if p.JoinRoomRsp.Ret == msg.ErrorID_Ok {
		obj.room = p.JoinRoomRsp.Room
		log(obj, "JoinRoom. roomid:", obj.room.RoomId)

		// 入座的玩家数
		// num := uint32(0)
		// for _, player := range p.JoinRoomRsp.Room.Players {
		// 	if player.SeatId >= 0 {
		// 		num++
		// 	}
		// }
		obj.roomManager.JoinRoom(obj.uid, p.JoinRoomRsp.Room.RoomId)
		obj.robot.DoAction(actionJoinRoom)
	} else if p.JoinRoomRsp.Ret == msg.ErrorID_JoinRoom_Full ||
		p.JoinRoomRsp.Ret == msg.ErrorID_JoinRoom_Already_In ||
		p.JoinRoomRsp.Ret == msg.ErrorID_JoinRoom_Released {
		obj.robot.DoAction(actionListRooms)
	} else {
		base.LogWarn(obj.uid, "No Action!!!!")
	}
}

func (obj *Client) handleJoinRoomNotify(args interface{}) {
	p := args.(*msg.Protocol)
	player := &msg.Player{
		Uid:    p.JoinRoomNotify.Uid,
		Name:   p.JoinRoomNotify.Name,
		Avatar: p.JoinRoomNotify.Avatar,
		SeatId: -1,
	}
	obj.room.Players = append(obj.room.Players, player)
}

func (obj *Client) handleSitDown(args interface{}) {
	p := args.(*msg.Protocol)
	if p.SitDownRsp.Ret == msg.ErrorID_SitDown_CreditPoints_Out || p.SitDownRsp.Ret == msg.ErrorID_SitDown_Invalid_Seat_Id {
		obj.seatID = -1
		obj.SendLeaveRoom()
		return
	} else if p.SitDownRsp.Ret == msg.ErrorID_SitDown_Already_Exist_Player {
		obj.sitDown()
		return
	} else if p.SitDownRsp.Ret == msg.ErrorID_SitDown_Already_Sit {
		return
	} else if p.SitDownRsp.Ret == msg.ErrorID_Ok {
		//obj.SendStandUp()
		//obj.SendBet()
		log(obj, "SitDown. roomid:", obj.room.RoomId, ", seatID:", obj.seatID)
		obj.roomManager.SitDown(obj.uid, obj.room.RoomId)

		if obj.seatID == 0 {
			for _, player := range obj.room.Players {
				if player.Uid == obj.uid {
					player.SeatId = 0
					break
				}
			}
			obj.waittingPlayers = uint32(1 + obj.s.Int31n(3))
		}
	} else {
		obj.robot.DoAction(actionJoinRoom)
	}
}

func (obj *Client) handleSitDownNotify(args interface{}) {
	p := args.(*msg.Protocol)
	uid := p.SitDownNotify.Uid

	for _, player := range obj.room.Players {
		if player.Uid == uid {
			player.SeatId = int32(p.SitDownNotify.SeatId)
			break
		}
	}
	if obj.seatID == 0 {
		obj.robot.DoAction(actionSitDownNotify)
	}
}

func (obj *Client) handleStandUp(args interface{}) {
	p := args.(*msg.Protocol)
	if p.StandUpRsp.Ret == msg.ErrorID_Ok {
		//obj.SendLeaveRoom()
		obj.roomManager.StandUp(obj.uid, obj.room.RoomId)
		obj.robot.DoAction(actionStandUp)
	} else {
		base.LogWarn(obj.uid, "No Action!!!!")
	}
}

func (obj *Client) handleLeaveRoom(args interface{}) {
	p := args.(*msg.Protocol)
	if p.LeaveRoomRsp.Ret == msg.ErrorID_Ok {
		obj.roomManager.LeaveRoom(obj.uid, obj.room.RoomId)
		obj.room = nil
		obj.robot.DoAction(actionListRooms)
	} else {
		base.LogWarn(obj.uid, "No Action!!!!!")
	}
}

func (obj *Client) handleBet(args interface{}) {
	//p := args.(*msg.Protocol)
}

func (obj *Client) handleGameStateNotify(args interface{}) {
	p := args.(*msg.Protocol)
	switch p.GameStateNotify.State {
	case msg.GameState_Ready:
		if obj.seatID == 0 {
			obj.SendStartGame()
		}
	case msg.GameState_Bet:
		obj.room.PlayedHands = p.GameStateNotify.PlayedHands
		if obj.seatID > 0 {
			time.Sleep(time.Duration(2+obj.s.Int31n(7)) * time.Second)
			obj.SendBet()
		}
	case msg.GameState_Deal:
		obj.cards = p.GameStateNotify.DealCards
	case msg.GameState_Combine:
		time.Sleep(time.Duration(10+obj.s.Int31n(30)) * time.Second)
		obj.SendCombine()

	case msg.GameState_Result:
		//base.LogInfo("uid:", obj.uid)
		if obj.room.PlayedHands+1 == obj.room.Hands {
			time.Sleep(2 * time.Second)
			if obj.seatID == 0 {
				obj.roomManager.CloseRoom(obj.room.RoomId)
			}

			//base.LogInfo("roomid:", obj.room.RoomId)

			obj.room = nil
			obj.seatID = -1

			obj.robot.DoAction(actionListRooms)
			return
		}
	}
}
