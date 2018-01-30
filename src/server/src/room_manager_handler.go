package main

import (
	"database/sql"

	"./msg"
)

func (obj *RoomManager) handleCreateRoomReq(conn *userConnection, p *msg.Protocol) {
	req := p.CreateRoomReq
	rsp := &msg.Protocol{
		Msgid:         msg.MessageID_CreateRoom_Rsp,
		CreateRoomRsp: &msg.CreateRoomRsp{},
	}
	defer conn.sendProtocol(rsp)

	// check name length
	nameLen := len([]rune(req.Name))
	if nameLen > gApp.config.Room.RoomNameLen || nameLen == 0 {
		rsp.CreateRoomRsp.Ret = msg.ErrorID_CreateRoom_Invalid_Room_Name
		return
	}

	// 最小和最大下注
	if req.MinBet > req.MaxBet {
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

	// 创建的房间达到上限
	count, err := db.getRoomCreatedCount(conn.uid)
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
	if req.Hands > 0 &&
		!req.IsShare &&
		!userManager.consumeDiamonds(conn.uid, gApp.config.Room.RoomRate*req.Hands, "create room") {
		rsp.CreateRoomRsp.Ret = msg.ErrorID_CreateRoom_Not_Enough_Diamonds
		return
	}

	number, err := roomNumberGenerator.get()
	if err != nil {
		rsp.CreateRoomRsp.Ret = msg.ErrorID_Internal_Error
		return
	}

	room, err := obj.createRoom(number, req.Name, conn.uid, req.Hands, req.MinBet, req.MaxBet, req.CreditPoints, req.IsShare)
	if err != nil {
		logError("[RoomManager][createRoom] failed to create room. error:", err)
		rsp.CreateRoomRsp.Ret = msg.ErrorID_DB_Error
		return
	}

	rsp.CreateRoomRsp.RoomId = room.roomID
	rsp.CreateRoomRsp.RoomNumber = number
}

func (obj *RoomManager) handleJoinRoomReq(conn *userConnection, p *msg.Protocol) {
	req := p.JoinRoomReq
	rsp := &msg.Protocol{
		Msgid:       msg.MessageID_JoinRoom_Rsp,
		JoinRoomRsp: &msg.JoinRoomRsp{},
	}
	defer conn.sendProtocol(rsp)

	reqRoomNum := roomNumberGenerator.encode(req.RoomNumber)
	room, ok := obj.roomsNumber[reqRoomNum]
	if !ok {
		// load room
		var err error
		name, roomID, uid, hands, playedHands, minBet, maxBet, creditPoints, isShare, err := db.loadRoom(uint32(reqRoomNum))
		switch {
		case err == sql.ErrNoRows:
			rsp.JoinRoomRsp.Ret = msg.ErrorID_JoinRoom_Wrong_Room_Number
			return
		case err != nil:
			logError("[RoomManager][joinRoom] failed to find room. number:", req.RoomNumber, ",error:", err)
			rsp.JoinRoomRsp.Ret = msg.ErrorID_DB_Error
			return
		default:
			room = obj.createRoomBase(name, reqRoomNum, roomID, uid, hands, playedHands, minBet, maxBet, creditPoints, isShare)
		}

		room.getEventChan() <- &roomEvent{event: roomEventJoin, args: []interface{}{}}
	}
}
