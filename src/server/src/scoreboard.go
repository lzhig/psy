package main

import (
	"sort"
)

// Scoreboard type
type Scoreboard struct {
	roomID uint32
	List   map[uint32]int32
	Uids   []uint32
}

// Init 初始化
func (obj *Scoreboard) Init(roomID uint32, playerNum uint32) {
	obj.roomID = roomID
	obj.List = make(map[uint32]int32)
	obj.Uids = make([]uint32, 0, playerNum)
}

// Update function 更新积分榜
func (obj *Scoreboard) Update(uid uint32, score int32) {
	if _, ok := obj.List[uid]; ok {
		obj.List[uid] += score
	} else {
		obj.List[uid] = score
		obj.Uids = append(obj.Uids, uid)
	}
	obj.sort()
}

// 根据积分排序
func (obj *Scoreboard) sort() {
	sort.Slice(obj.Uids, func(i, j int) bool {
		return obj.List[obj.Uids[i]] > obj.List[obj.Uids[j]]
	})
}

// todo: 积分榜 每局的更新 加载房间时的读取
func (obj *Scoreboard) load() error {
	return nil
}

func (obj *Scoreboard) updateToDB() {

}
