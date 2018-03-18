package main

import (
	"sort"
)

// Scoreboard type
type Scoreboard struct {
	List map[uint32]int32
	Uids []uint32
}

// Init 初始化
func (obj *Scoreboard) Init(playerNum uint32) {
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
