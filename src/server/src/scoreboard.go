package main

import (
	"sort"

	"./msg"
	"github.com/lzhig/rapidgo/base"
)

// Scoreboard type
type Scoreboard struct {
	roomID uint32
	List   map[uint32]*msg.ScoreboardItem
	Uids   []uint32
}

// Init 初始化
func (obj *Scoreboard) Init(roomID uint32, playerNum uint32) {
	obj.roomID = roomID
	obj.List = make(map[uint32]*msg.ScoreboardItem)
	obj.Uids = make([]uint32, 0, playerNum)
}

// GetScore 获取用户积分
func (obj *Scoreboard) GetScore(uid uint32) int32 {
	if item, ok := obj.List[uid]; ok {
		return item.Score
	}
	return 0
}

// Update function 更新积分榜
func (obj *Scoreboard) Update(uid uint32, name, avatar string, score int32) {
	if _, ok := obj.List[uid]; ok {
		obj.List[uid].Score += score
		if err := db.UpdateScoreboardItem(obj.roomID, uid, score); err != nil {
			base.LogError("[Scoreboard][Update] Failed to update db. error:", err)
		}
	} else {
		obj.List[uid] = &msg.ScoreboardItem{
			Uid:    uid,
			Name:   name,
			Avatar: avatar,
			Score:  score,
		}
		obj.Uids = append(obj.Uids, uid)
		if err := db.addScoreboardItem(obj.roomID, uid, score); err != nil {
			base.LogError("[Scoreboard][Update] Failed to update db. error:", err)
		}
	}
	obj.sort()
}

// 根据积分排序
func (obj *Scoreboard) sort() {
	sort.Slice(obj.Uids, func(i, j int) bool {
		return obj.List[obj.Uids[i]].Score > obj.List[obj.Uids[j]].Score
	})
}

// Load 读取
func (obj *Scoreboard) Load() error {
	items, err := db.loadScoreboard(obj.roomID)
	if err != nil {
		return err
	}
	for _, item := range items {
		obj.List[item.Uid] = item
		obj.Uids = append(obj.Uids, item.Uid)
	}
	return nil
}

func (obj *Scoreboard) updateToDB() {

}
