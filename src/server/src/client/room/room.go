package room

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// RoomManager Room管理器
type RoomManager struct {
	mu sync.RWMutex

	r *rand.Rand

	rooms map[uint32]*Room
}

// Init 初始化
func (obj *RoomManager) Init() {
	obj.rooms = make(map[uint32]*Room)
	obj.r = rand.New(rand.NewSource(time.Now().Unix()))
}

// AddRoom 添加房间
func (obj *RoomManager) AddRoom(room *Room) {
	obj.mu.Lock()
	if _, ok := obj.rooms[room.id]; !ok {
		obj.rooms[room.id] = room
	}
	obj.mu.Unlock()
}

// CloseRoom 关闭房间
func (obj *RoomManager) CloseRoom(id uint32) {
	obj.mu.Lock()
	if _, ok := obj.rooms[id]; ok {
		delete(obj.rooms, id)
	} else {
		//panic(fmt.Errorf("cannot find the room(%d)", id))
	}
	obj.mu.Unlock()
}

// JoiningRoom 正在加入房间
func (obj *RoomManager) JoiningRoom(rid uint32, joining bool) {
	obj.mu.Lock()
	if room, ok := obj.rooms[rid]; ok {
		room.PlayerJoining(joining)
	} else {
		panic(fmt.Errorf("cannot find the room(%d)", rid))
	}
	obj.mu.Unlock()
}

// SittingRoom 正在入座
func (obj *RoomManager) SittingRoom(rid uint32, sitting bool) {
	obj.mu.Lock()
	if room, ok := obj.rooms[rid]; ok {
		room.PlayerSitting(sitting)
	} else {
		panic(fmt.Errorf("cannot find the room(%d)", rid))
	}
	obj.mu.Unlock()
}

// JoinRoom 玩家加入房间
func (obj *RoomManager) JoinRoom(uid, id uint32) {
	obj.mu.Lock()
	if room, ok := obj.rooms[id]; ok {
		room.PlayerEnter(uid)
	} else {
		panic(fmt.Errorf("cannot find the room(%d)", id))
	}
	obj.mu.Unlock()
}

// LeaveRoom 玩家离开房间
func (obj *RoomManager) LeaveRoom(uid, id uint32) {
	obj.mu.Lock()
	if room, ok := obj.rooms[id]; ok {
		room.PlayerLeave(uid)
	} else {
		panic(fmt.Errorf("cannot find the room(%d)", id))
	}
	obj.mu.Unlock()
}

// SitDown 玩家坐下
func (obj *RoomManager) SitDown(uid, id uint32) {
	obj.mu.Lock()
	if room, ok := obj.rooms[id]; ok {
		room.PlayerSitDown(uid)
	} else {
		panic(fmt.Errorf("cannot find the room(%d)", id))
	}
	obj.mu.Unlock()
}

// StandUp 玩家坐下
func (obj *RoomManager) StandUp(uid, id uint32) {
	obj.mu.Lock()
	if room, ok := obj.rooms[id]; ok {
		room.PlayerStandUp(uid)
	} else {
		panic(fmt.Errorf("cannot find the room(%d)", id))
	}
	obj.mu.Unlock()
}

// GetRandomRoom 随机获取一个可以坐下的房间
func (obj *RoomManager) GetRandomRoom() *Room {
	obj.mu.RLock()
	defer obj.mu.RUnlock()
	l := len(obj.rooms)
	rooms := make([]*Room, 0, l)
	for _, room := range obj.rooms {
		if room.CanJoin() {
			rooms = append(rooms, room)
		}
	}

	l = len(rooms)
	if l == 0 {
		return nil
	}

	ndx := rand.Int31n(int32(l))
	return rooms[ndx]
}

// Room 桌子
type Room struct {
	id           uint32
	number       string
	players      map[uint32]uint32
	tablePlayers map[uint32]uint32

	playersJoining uint32
	playersSitting uint32
}

// GetID 获取id
func (obj *Room) GetID() uint32 {
	return obj.id
}

// GetNumber 获取房号
func (obj *Room) GetNumber() string {
	return obj.number
}

// NewRoom 创建一个房间
func NewRoom(id uint32, number string) *Room {
	return &Room{
		id:           id,
		number:       number,
		players:      make(map[uint32]uint32),
		tablePlayers: make(map[uint32]uint32),
	}
}

// PlayerJoining 有玩家正在加入房间
func (obj *Room) PlayerJoining(joining bool) {
	if joining {
		obj.playersJoining++
	} else {
		obj.playersJoining--
	}
}

// PlayerSitting 有玩家存在入座
func (obj *Room) PlayerSitting(sitting bool) {
	if sitting {
		obj.playersSitting++
	} else {
		obj.playersSitting--
	}
}

// PlayerEnter 进入
func (obj *Room) PlayerEnter(uid uint32) {
	obj.players[uid] = uid
}

// PlayerLeave 离开
func (obj *Room) PlayerLeave(uid uint32) {
	delete(obj.players, uid)
	delete(obj.tablePlayers, uid)
}

// PlayerSitDown 坐下
func (obj *Room) PlayerSitDown(uid uint32) {
	if _, ok := obj.players[uid]; ok {
		obj.tablePlayers[uid] = uid
	} else {
		panic(fmt.Errorf("there is not the player in the room"))
	}
}

// PlayerStandUp 站起
func (obj *Room) PlayerStandUp(uid uint32) {
	if _, ok := obj.tablePlayers[uid]; ok {
		delete(obj.tablePlayers, uid)
	} else {
		panic(fmt.Errorf("there is not the player in the seat"))
	}
}

// CanSitDown 是否可以坐下
func (obj *Room) CanSitDown() bool {
	return len(obj.tablePlayers) < 4
}

// CanJoin 是否可以加入
func (obj *Room) CanJoin() bool {
	return len(obj.players)+int(obj.playersJoining) < 4
}
