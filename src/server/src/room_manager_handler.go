package main

import (
	"database/sql"
	"time"

	"./msg"
	"github.com/lzhig/rapidgo/base"
)

func (obj *RoomManager) handleCreateRoomReq(p *ProtocolConnection) {
	req := p.p.CreateRoomReq
	rsp := &msg.Protocol{
		Msgid:         msg.MessageID_CreateRoom_Rsp,
		CreateRoomRsp: &msg.CreateRoomRsp{Ret: msg.ErrorID_Ok},
	}
	defer p.userconn.sendProtocol(rsp)

	if req == nil {
		rsp.CreateRoomRsp.Ret = msg.ErrorID_Invalid_Params
		return
	}

	// check name length
	nameLen := len([]rune(req.Name))
	if nameLen > gApp.config.Room.RoomNameLen || nameLen == 0 {
		rsp.CreateRoomRsp.Ret = msg.ErrorID_CreateRoom_Invalid_Room_Name
		return
	}

	// 最小和最大下注
	if req.MaxBet != req.MinBet*gApp.config.Room.MaxBetRate {
		rsp.CreateRoomRsp.Ret = msg.ErrorID_CreateRoom_Invalid_Min_Max_Bet
		return
	}

	// credit points
	bfound := false
	for _, v := range gApp.config.Room.CreditPoints {
		if v == req.CreditPoints {
			bfound = true
			break
		}
	}
	if !bfound {
		rsp.CreateRoomRsp.Ret = msg.ErrorID_CreateRoom_Invalid_Credit_Points
		return
	}

	// hands
	if req.Hands == 0 {
		rsp.CreateRoomRsp.Ret = msg.ErrorID_CreateRoom_Invalid_Hands
		return
	}

	// 创建的房间达到上限
	count, err := db.getRoomCreatedCount(p.userconn.user.uid)
	if err != nil {
		base.LogError("[RoomManager][createRoom] failed to query the count of rooms created. error:", err)
		rsp.CreateRoomRsp.Ret = msg.ErrorID_DB_Error
		return
	}
	if count >= gApp.config.Room.CountCreated {
		rsp.CreateRoomRsp.Ret = msg.ErrorID_CreateRoom_Exceed_Limitation_Rooms
		return
	}

	// 扣钻石
	if //req.Hands > 0 && // 目前不支持无限局
	!req.IsShare &&
		!userManager.consumeDiamonds(p.userconn.user.uid, gApp.config.Room.RoomRate*req.Hands, "create room") {
		rsp.CreateRoomRsp.Ret = msg.ErrorID_CreateRoom_Not_Enough_Diamonds
		return
	}

	number, err := roomNumberGenerator.get()
	if err != nil {
		rsp.CreateRoomRsp.Ret = msg.ErrorID_Internal_Error
		return
	}

	createTime := time.Now().Unix()
	room, err := obj.createRoom(number, req.Name, p.userconn.user.uid, req.Hands, req.MinBet, req.MaxBet, req.CreditPoints*req.MaxBet, req.IsShare, createTime)
	if err != nil {
		base.LogError("[RoomManager][createRoom] failed to create room. error:", err)
		rsp.CreateRoomRsp.Ret = msg.ErrorID_DB_Error
		return
	}
	obj.roomCountdown.Add(room.roomID, createTime)

	rsp.CreateRoomRsp.RoomId = room.roomID
	rsp.CreateRoomRsp.RoomNumber = number
}

func (obj *RoomManager) handleJoinRoomReq(p *ProtocolConnection) {
	req := p.p.JoinRoomReq
	rsp := &msg.Protocol{
		Msgid:       msg.MessageID_JoinRoom_Rsp,
		JoinRoomRsp: &msg.JoinRoomRsp{Ret: msg.ErrorID_Ok},
	}

	reqRoomNum := roomNumberGenerator.encode(req.RoomNumber)
	room, ok := obj.roomsNumber[reqRoomNum]
	if !ok {
		// load room
		roomID, err := db.GetRoomID(uint32(reqRoomNum))
		switch {
		case err == sql.ErrNoRows:
			rsp.JoinRoomRsp.Ret = msg.ErrorID_JoinRoom_Wrong_Room_Number
			p.userconn.sendProtocol(rsp)
			return
		case err != nil:
			base.LogError("[RoomManager][joinRoom] failed to find room. number:", req.RoomNumber, ",error:", err)
			rsp.JoinRoomRsp.Ret = msg.ErrorID_DB_Error
			p.userconn.sendProtocol(rsp)
			return
		}
		obj.roomlocker.Lock(roomID)
		defer obj.roomlocker.Unlock(roomID)

		name, roomID, uid, hands, playedHands, minBet, maxBet, creditPoints, isShare, createTime, err := db.loadRoom(uint32(reqRoomNum))
		switch {
		case err == sql.ErrNoRows:
			rsp.JoinRoomRsp.Ret = msg.ErrorID_JoinRoom_Wrong_Room_Number
			p.userconn.sendProtocol(rsp)
			return
		case err != nil:
			base.LogError("[RoomManager][joinRoom] failed to find room. number:", req.RoomNumber, ",error:", err)
			rsp.JoinRoomRsp.Ret = msg.ErrorID_DB_Error
			p.userconn.sendProtocol(rsp)
			return
		default:
			room = obj.createRoomBase(name, reqRoomNum, roomID, uid, hands, playedHands, minBet, maxBet, creditPoints, isShare, true)
			obj.roomCountdown.Add(room.roomID, createTime)
		}
	}
	room.GetProtoChan() <- p
}

func (obj *RoomManager) handleLeaveRoomReq(p *ProtocolConnection) {
	rsp := &msg.Protocol{
		Msgid:        msg.MessageID_LeaveRoom_Rsp,
		LeaveRoomRsp: &msg.LeaveRoomRsp{Ret: msg.ErrorID_Ok},
	}

	room := userManager.getUserRoom(p.userconn.user.uid)
	if room == nil {
		rsp.LeaveRoomRsp.Ret = msg.ErrorID_LeaveRoom_Not_In
		p.userconn.sendProtocol(rsp)
		return
	}

	room.GetProtoChan() <- p
}

func (obj *RoomManager) handleListRoomsReq(p *ProtocolConnection) {
	rsp := &msg.Protocol{
		Msgid:        msg.MessageID_ListRooms_Rsp,
		ListRoomsRsp: &msg.ListRoomsRsp{Ret: msg.ErrorID_Ok},
	}
	defer p.userconn.sendProtocol(rsp)

	rooms, err := db.GetRoomsListJoined(p.userconn.user.uid)
	if err != nil {
		rsp.ListRoomsRsp.Ret = msg.ErrorID_DB_Error
		return
	}

	for _, r := range rooms {
		obj.roomlocker.Lock(r.RoomId)
		defer obj.roomlocker.Unlock(r.RoomId)

		v, ok := obj.rooms.Load(r.RoomId)
		if ok {
			room := v.(*Room)
			c := make(chan []*msg.ListRoomPlayerInfo)
			room.notifyGetSeatPlayers(c)
			r.Players = <-c
		}
	}

	rsp.ListRoomsRsp.Rooms = rooms
}

func (obj *RoomManager) handleCloseRoomReq(p *ProtocolConnection) {
	rsp := &msg.Protocol{
		Msgid:        msg.MessageID_CloseRoom_Rsp,
		CloseRoomRsp: &msg.CloseRoomRsp{Ret: msg.ErrorID_Ok},
	}
	defer p.userconn.sendProtocol(rsp)

	roomID := p.p.CloseRoomReq.RoomId
	rsp.CloseRoomRsp.RoomId = roomID

	if !obj.CloseRoom(roomID) {
		rsp.CloseRoomRsp.Ret = msg.ErrorID_CloseRoom_Cannot_Close
		return
	}
}

func (obj *RoomManager) handleGetPlayingRoomReq(p *ProtocolConnection) {
	rsp := &msg.Protocol{
		Msgid:             msg.MessageID_GetPlayingRoom_Rsp,
		GetPlayingRoomRsp: &msg.GetPlayingRoomRsp{Ret: msg.ErrorID_Ok},
	}
	defer p.userconn.sendProtocol(rsp)

	if user := userManager.GetUser(p.userconn.user.uid); user != nil && user.room != nil {
		rsp.GetPlayingRoomRsp.RoomNumber = roomNumberGenerator.decode(user.room.number)
	}
}
