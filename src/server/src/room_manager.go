package main

import (
	"context"
	"sync"

	"./msg"
)

// RoomManager type
type RoomManager struct {
	rooms       sync.Map
	roomsNumber map[int]*Room
	protoChan   chan *ProtocolConnection
	handlers    map[msg.MessageID]func(*ProtocolConnection)
}

func (obj *RoomManager) init() {
	obj.roomsNumber = make(map[int]*Room)
	obj.protoChan = make(chan *ProtocolConnection, 128)
	obj.handlers = map[msg.MessageID]func(*ProtocolConnection){
		msg.MessageID_CreateRoom_Req: obj.handleCreateRoomReq,
		msg.MessageID_JoinRoom_Req:   obj.handleJoinRoomReq,
		msg.MessageID_LeaveRoom_Req:  obj.handleLeaveRoomReq,
	}
	ctx, _ := gApp.CreateCancelContext()
	gApp.GoRoutine(ctx, obj.loop)
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
				logError("[RoomManager][loop] cannot find handler for msgid:", msg.MessageID_name[int32(p.p.Msgid)])
				p.userconn.Disconnect()
			}
		}
	}
}

func (obj *RoomManager) createRoom(number, name string, uid, hands, minBet, maxBet, creditPoints uint32, isShare bool) (*Room, error) {
	num := roomNumberGenerator.encode(number)
	roomID, err := db.createRoom(num, name, uid, hands, minBet, maxBet, creditPoints, isShare)
	if err != nil {
		return nil, err
	}

	room := obj.createRoomBase(name, num, roomID, uid, hands, 0, minBet, maxBet, creditPoints, isShare)

	return room, nil
}

func (obj *RoomManager) createRoomBase(name string, num int, roomID, uid, hands, playedHands, minBet, maxBet, creditPoints uint32, isShare bool) *Room {
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
	room.init()
	return room
}
