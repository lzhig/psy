package main

import (
	"context"
	"sync"
	"time"

	"./msg"
	"github.com/lzhig/rapidgo/base"
)

// RoomManager type
type RoomManager struct {
	rooms         sync.Map
	roomsNumber   map[int]*Room
	protoChan     chan *ProtocolConnection
	handlers      map[msg.MessageID]func(*ProtocolConnection)
	roomlocker    RoomLocker
	roomCountdown RoomCountdown
}

func (obj *RoomManager) init() error {
	obj.roomlocker.Init()
	if err := obj.roomCountdown.Init(); err != nil {
		return err
	}
	obj.roomsNumber = make(map[int]*Room)
	obj.protoChan = make(chan *ProtocolConnection, 128)
	obj.handlers = map[msg.MessageID]func(*ProtocolConnection){
		msg.MessageID_CreateRoom_Req: obj.handleCreateRoomReq,
		msg.MessageID_JoinRoom_Req:   obj.handleJoinRoomReq,
		msg.MessageID_LeaveRoom_Req:  obj.handleLeaveRoomReq,
		msg.MessageID_ListRooms_Req:  obj.handleListRoomsReq,
		msg.MessageID_CloseRoom_Req:  obj.handleCloseRoomReq,
	}
	ctx, _ := gApp.CreateCancelContext()
	gApp.GoRoutine(ctx, obj.loop)
	return nil
}

// GetDispatchChan function
func (obj *RoomManager) GetDispatchChan() chan<- *ProtocolConnection {
	return obj.protoChan
}

func (obj *RoomManager) loop(ctx context.Context) {
	defer debug("exit RoomManager goroutine")
	for {
		select {
		case <-ctx.Done():
			return

		case p := <-obj.protoChan:
			if handler, ok := obj.handlers[p.p.Msgid]; ok {
				handler(p)
			} else {
				base.LogError("[RoomManager][loop] cannot find handler for msgid:", msg.MessageID_name[int32(p.p.Msgid)])
				p.userconn.Disconnect()
			}
		}
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
		room.notifyCloseRoom(c)
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
