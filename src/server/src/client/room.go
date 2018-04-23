package main

import (
	"sort"

	"github.com/lzhig/rapidgo/base"
)

const (
	roomManagerEventCreateRoom base.EventID = iota
	roomManagerEventGetAvaliableRoom
	roomManagerEventUpdateRoomPlayersNumber
	roomManagerEventJoinRoom
	roomManagerEventLeaveRoom
	roomManagerEventSitDown
	roomManagerEventStandUp
	roomManagerEventCloseRoom
)

// RoomManager Room管理器
type RoomManager struct {
	base.EventSystem

	rooms map[uint32]*Room

	avaliableRoomsID []uint32
}

// Init 初始化
func (obj *RoomManager) Init() {
	obj.EventSystem.Init(1014, true)
	obj.SetEventHandler(roomManagerEventCreateRoom, obj.handleEventCreateRoom)
	obj.SetEventHandler(roomManagerEventGetAvaliableRoom, obj.handleEventGetAvaliableRoom)
	obj.SetEventHandler(roomManagerEventUpdateRoomPlayersNumber, obj.handleEventUpdateRoomPlayersNumber)
	obj.SetEventHandler(roomManagerEventJoinRoom, obj.handleEventJoinRoom)
	obj.SetEventHandler(roomManagerEventLeaveRoom, obj.handleEventLeaveRoom)
	obj.SetEventHandler(roomManagerEventSitDown, obj.handleEventSitDown)
	obj.SetEventHandler(roomManagerEventStandUp, obj.handleEventStandUp)
	obj.SetEventHandler(roomManagerEventCloseRoom, obj.handleEventCloseRoom)

	obj.rooms = make(map[uint32]*Room)
	obj.avaliableRoomsID = make([]uint32, 0, 1024)
}

func (obj *RoomManager) addRoomID(id uint32) {
	obj.avaliableRoomsID = append(obj.avaliableRoomsID, id)
	obj.sortRoomdID()
}

func (obj *RoomManager) removeRoomID(id uint32) {
	for ndx, roomID := range obj.avaliableRoomsID {
		if roomID == id {
			if ndx == len(obj.avaliableRoomsID)-1 {
				obj.avaliableRoomsID = obj.avaliableRoomsID[:ndx]
			} else if ndx == 0 {
				obj.avaliableRoomsID = obj.avaliableRoomsID[1:]
			} else {
				copy(obj.avaliableRoomsID[ndx:], obj.avaliableRoomsID[ndx+1:])
				obj.avaliableRoomsID = obj.avaliableRoomsID[:len(obj.avaliableRoomsID)-1]
			}
			return
		}
	}
}

func (obj *RoomManager) sortRoomdID() {
	sort.Slice(obj.avaliableRoomsID, func(i, j int) bool {
		return obj.rooms[obj.avaliableRoomsID[i]].tablePlayers > obj.rooms[obj.avaliableRoomsID[j]].tablePlayers
	})
}

func (obj *RoomManager) handleEventCreateRoom(args []interface{}) {
	id := args[0].(uint32)
	number := args[1].(string)
	if _, ok := obj.rooms[id]; !ok {
		room := &Room{
			id:     id,
			number: number,
		}
		obj.rooms[id] = room
		obj.addRoomID(id)
	}

	c := args[2].(chan struct{})
	c <- struct{}{}
}

func (obj *RoomManager) handleEventGetAvaliableRoom(args []interface{}) {
	c := args[0].(chan *Room)
	if len(obj.avaliableRoomsID) == 0 {
		c <- nil
	} else {
		room := obj.rooms[obj.avaliableRoomsID[0]]
		room.given++
		if room.given >= 4 {
			obj.removeRoomID(room.id)
		}
		c <- room
	}
}

func (obj *RoomManager) handleEventJoinRoom(args []interface{}) {
	id := args[0].(uint32)
	if room, ok := obj.rooms[id]; ok {
		room.players++
	} else {
		base.LogError("Cannot find the room, id:", id)
	}

	c := args[1].(chan struct{})
	c <- struct{}{}
}

func (obj *RoomManager) handleEventLeaveRoom(args []interface{}) {
	id := args[0].(uint32)
	if room, ok := obj.rooms[id]; ok {
		if room.players == 0 {
			base.LogError("Invalid players number")
		}
		room.players--
	} else {
		base.LogError("Cannot find the room, id:", id)
	}

	c := args[1].(chan struct{})
	c <- struct{}{}
}

func (obj *RoomManager) handleEventSitDown(args []interface{}) {
	id := args[0].(uint32)
	if room, ok := obj.rooms[id]; ok {
		room.tablePlayers++
		if room.tablePlayers > 4 {
			base.LogError("Invalid table players number. roomid:", id)
		} else if room.tablePlayers == 4 {
			obj.removeRoomID(id)
		} else {
			obj.sortRoomdID()
		}
	} else {
		base.LogError("Cannot find the room, id:", id)
	}

	c := args[1].(chan struct{})
	c <- struct{}{}
}

func (obj *RoomManager) handleEventStandUp(args []interface{}) {
	id := args[0].(uint32)
	if room, ok := obj.rooms[id]; ok {
		if room.tablePlayers == 0 {
			base.LogError("Invalid table players number")
		}
		room.tablePlayers--
		if room.tablePlayers == 3 {
			obj.addRoomID(id)
		} else {
			obj.sortRoomdID()
		}
	} else {
		base.LogError("Cannot find the room, id:", id)
	}

	c := args[1].(chan struct{})
	c <- struct{}{}
}

func (obj *RoomManager) handleEventCloseRoom(args []interface{}) {
	id := args[0].(uint32)
	if _, ok := obj.rooms[id]; ok {
		delete(obj.rooms, id)
		obj.removeRoomID(id)
	} else {
		base.LogError("Cannot find the room, id:", id)
	}

	c := args[1].(chan struct{})
	c <- struct{}{}
}

func (obj *RoomManager) handleEventUpdateRoomPlayersNumber(args []interface{}) {
	id := args[0].(uint32)
	players := args[1].(uint32)
	tablePlayers := args[2].(uint32)

	if room, ok := obj.rooms[id]; ok {
		room.players = players
		room.tablePlayers = tablePlayers
	} else {
		base.LogError("Cannot find the room, id:", id)
	}

	c := args[3].(chan struct{})
	c <- struct{}{}
}

// Room 桌子
type Room struct {
	id           uint32
	number       string
	players      uint32
	tablePlayers uint32
	given        uint32
}

// PlayerEnter 进入
func (obj *Room) PlayerEnter() {
	obj.players++
}

// PlayerLeave 离开
func (obj *Room) PlayerLeave() {
	obj.players--
}

// PlayerSitDown 坐下
func (obj *Room) PlayerSitDown() {
	obj.tablePlayers++
}

// PlayerStandUp 站起
func (obj *Room) PlayerStandUp() {
	obj.tablePlayers--
}
