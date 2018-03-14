package main

import (
	"database/sql"

	"./msg"
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
	count, err := db.getRoomCreatedCount(p.userconn.uid)
	if err != nil {
		logError("[RoomManager][createRoom] failed to query the count of rooms created. error:", err)
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
		!userManager.consumeDiamonds(p.userconn.uid, gApp.config.Room.RoomRate*req.Hands, "create room") {
		rsp.CreateRoomRsp.Ret = msg.ErrorID_CreateRoom_Not_Enough_Diamonds
		return
	}

	number, err := roomNumberGenerator.get()
	if err != nil {
		rsp.CreateRoomRsp.Ret = msg.ErrorID_Internal_Error
		return
	}

	room, err := obj.createRoom(number, req.Name, p.userconn.uid, req.Hands, req.MinBet, req.MaxBet, req.CreditPoints, req.IsShare)
	if err != nil {
		logError("[RoomManager][createRoom] failed to create room. error:", err)
		rsp.CreateRoomRsp.Ret = msg.ErrorID_DB_Error
		return
	}

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
		var err error
		name, roomID, uid, hands, playedHands, minBet, maxBet, creditPoints, isShare, err := db.loadRoom(uint32(reqRoomNum))
		switch {
		case err == sql.ErrNoRows:
			rsp.JoinRoomRsp.Ret = msg.ErrorID_JoinRoom_Wrong_Room_Number
			p.userconn.sendProtocol(rsp)
			return
		case err != nil:
			logError("[RoomManager][joinRoom] failed to find room. number:", req.RoomNumber, ",error:", err)
			rsp.JoinRoomRsp.Ret = msg.ErrorID_DB_Error
			p.userconn.sendProtocol(rsp)
			return
		default:
			room = obj.createRoomBase(name, reqRoomNum, roomID, uid, hands, playedHands, minBet, maxBet, creditPoints, isShare)
		}

	}
	room.GetProtoChan() <- p
}

func (obj *RoomManager) handleLeaveRoomReq(p *ProtocolConnection) {
	rsp := &msg.Protocol{
		Msgid:        msg.MessageID_LeaveRoom_Rsp,
		LeaveRoomRsp: &msg.LeaveRoomRsp{Ret: msg.ErrorID_Ok},
	}

	room := userManager.getUserRoom(p.userconn.uid)
	if room == nil {
		rsp.LeaveRoomRsp.Ret = msg.ErrorID_LeaveRoom_Not_In
		p.userconn.sendProtocol(rsp)
		return
	}

	room.GetProtoChan() <- p
}
