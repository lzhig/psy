package main

import (
	"sync"
	"time"

	"./msg"
	"github.com/lzhig/rapidgo/base"
)

const (
	roomManagerEventNetworkPacket base.EventID = iota
	roomManagerEventCloseRoom
	roomManagerEventReleaseRoom
	roomManagerEventStopServer
)

// RoomManager type
type RoomManager struct {
	base.EventSystem

	networkPacketHandler base.MessageHandlerImpl

	rooms         sync.Map
	roomsNumber   map[int]*Room
	roomlocker    RoomLocker
	roomCountdown RoomCountdown

	enable bool // 是否停服
}

func (obj *RoomManager) init() error {
	obj.EventSystem.Init(1024, false)
	obj.SetEventHandler(roomManagerEventNetworkPacket, obj.handleEventNetworkPacket)
	obj.SetEventHandler(roomManagerEventCloseRoom, obj.handleEventCloseRoom)
	obj.SetEventHandler(roomManagerEventReleaseRoom, obj.handleEventReleaseRoom)
	obj.SetEventHandler(roomManagerEventStopServer, obj.handleEventStopServer)

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

	obj.Enable(true)

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

func (obj *RoomManager) handleEventCloseRoom(args []interface{}) {
	room := args[0].(*Room)
	obj.CloseRoom(room.roomID)
}

func (obj *RoomManager) handleEventReleaseRoom(args []interface{}) {
	room := args[0].(*Room)
	base.LogInfo("RoomManager.handleEventReleaseRoom, room_id:", room.roomID)
	c := make(chan bool)
	room.Send(roomEventRelease, []interface{}{c})
	released := <-c
	if released {
		base.LogInfo("room_id:", room.roomID, " released")
		obj.rooms.Delete(room.roomID)
		delete(obj.roomsNumber, room.number)
		onlineStatistic.RoomsChange(false)
	}
}

// Enable 设置是否停服
func (obj *RoomManager) Enable(enable bool) {
	c := make(chan struct{})
	obj.Send(roomManagerEventStopServer, []interface{}{c, enable})
	<-c
}

func (obj *RoomManager) handleEventStopServer(args []interface{}) {
	c := args[0].(chan struct{})
	obj.enable = args[1].(bool)
	var wg sync.WaitGroup
	wg.Add(len(obj.roomsNumber))
	for _, room := range obj.roomsNumber {
		go func() {
			room.Enable(false)
			wg.Done()
		}()
	}
	wg.Wait()
	c <- struct{}{}
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
	onlineStatistic.RoomsChange(true)
	return room
}

// CloseRoom 关闭房间
func (obj *RoomManager) CloseRoom(roomid uint32) bool {
	base.LogInfo("CloseRoom, room_id:", roomid, ", left:", len(obj.roomsNumber))
	for _, room := range obj.roomsNumber {
		base.LogInfo("room_id:", room.roomID, ", status:", room.round.state, ", players:", room.players)
		break
	}
	obj.roomlocker.Lock(roomid)
	defer obj.roomlocker.Unlock(roomid)

	r, ok := obj.rooms.Load(roomid)
	if ok {
		room := r.(*Room)
		c := make(chan bool)
		//room.notifyCloseRoom(c)
		room.Send(roomEventClose, []interface{}{c})
		closed := <-c

		if closed {
			base.LogInfo("room_id:", roomid, ", closed")
			obj.rooms.Delete(roomid)
			delete(obj.roomsNumber, room.number)
			onlineStatistic.RoomsChange(false)
		}

		return closed
	}

	err := db.CloseRoom(roomid, time.Now().Unix())
	if err != nil {
		base.LogError("[RoomManager][CloseRoom] Failed to close room:", roomid, ". error:", err)
		return false
	}
	return true
}
