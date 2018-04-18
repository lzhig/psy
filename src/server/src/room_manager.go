package main

import (
	"sync"
	"time"

	"./msg"
	"github.com/lzhig/rapidgo/base"
)

const (
	roomManagerEventNetworkPacket base.EventID = iota
)

// RoomManager type
type RoomManager struct {
	base.EventSystem

	networkPacketHandler base.MessageHandlerImpl

	rooms         sync.Map
	roomsNumber   map[int]*Room
	roomlocker    RoomLocker
	roomCountdown RoomCountdown
}

func (obj *RoomManager) init() error {
	obj.EventSystem.Init(16)
	obj.SetEventHandler(roomManagerEventNetworkPacket, obj.handleEventNetworkPacket)

	obj.roomlocker.Init()
	if err := obj.roomCountdown.Init(); err != nil {
		return err
	}
	obj.roomsNumber = make(map[int]*Room)

	obj.networkPacketHandler.Init()
	obj.networkPacketHandler.SetMessageHandler(msg.MessageID_CreateRoom_Req, obj.handleCreateRoomReq)
	obj.networkPacketHandler.SetMessageHandler(msg.MessageID_JoinRoom_Req, obj.handleJoinRoomReq)
	obj.networkPacketHandler.SetMessageHandler(msg.MessageID_LeaveRoom_Req, obj.handleLeaveRoomReq)
	obj.networkPacketHandler.SetMessageHandler(msg.MessageID_ListRooms_Req, obj.handleListRoomsReq)
	obj.networkPacketHandler.SetMessageHandler(msg.MessageID_CloseRoom_Req, obj.handleCloseRoomReq)
	obj.networkPacketHandler.SetMessageHandler(msg.MessageID_GetPlayingRoom_Req, obj.handleGetPlayingRoomReq)
	return nil
}

func (obj *RoomManager) handleEventNetworkPacket(args []interface{}) {
	p := args[0].(*ProtocolConnection)
	if p == nil {
		base.LogError("args[0] isn't a ProtocolConnection object.")
		return
	}

	if !obj.networkPacketHandler.Handle(p.p.Msgid, p) {
		base.LogError("cannot find handler for msgid:", msg.MessageID_name[int32(p.p.Msgid)])
		p.userconn.Disconnect()
	}
}

func (obj *RoomManager) createRoom(number, name string, uid, hands, minBet, maxBet, creditPoints uint32, isShare bool, createTime int64) (*Room, error) {
	num := roomNumberGenerator.encode(number)
	roomID, err := db.createRoom(num, name, uid, hands, minBet, maxBet, creditPoints, isShare, createTime)
	if err != nil {
		return nil, err
	}

	room := obj.createRoomBase(name, num, roomID, uid, hands, 0, minBet, maxBet, creditPoints, isShare, false)

	return room, nil
}

func (obj *RoomManager) createRoomBase(name string, num int, roomID, uid, hands, playedHands, minBet, maxBet, creditPoints uint32, isShare bool, bLoadScoreboard bool) *Room {
	room := &Room{
		name:         name,
		roomID:       roomID,
		number:       num,
		ownerUID:     uid,
		hands:        hands,
		playedHands:  playedHands,
		isShare:      isShare,
		minBet:       minBet,
		maxBet:       maxBet,
		creditPoints: creditPoints,
		createTime:   0,
		closeTime:    0,
		closed:       false,
	}
	obj.rooms.Store(roomID, room)
	obj.roomsNumber[num] = room
	room.init(bLoadScoreboard)
	return room
}

func (obj *RoomManager) CloseRoom(roomid uint32) bool {
	base.LogInfo("CloseRoom, roomid:", roomid)
	obj.roomlocker.Lock(roomid)
	defer obj.roomlocker.Unlock(roomid)

	r, ok := obj.rooms.Load(roomid)
	if ok {
		room := r.(*Room)
		c := make(chan bool)
		//room.notifyCloseRoom(c)
		room.Send(roomEventClose, []interface{}{c})
		closed := <-c
		return closed
	}

	err := db.CloseRoom(roomid, time.Now().Unix())
	if err != nil {
		base.LogError("[RoomManager][CloseRoom] Failed to close room:", roomid, ". error:", err)
		return false
	}
	return true
}
