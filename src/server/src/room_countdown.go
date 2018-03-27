package main

import (
	"context"
	"time"

	"github.com/HuKeping/rbtree"
)

type roomCreateTime struct {
	roomid     uint32
	createTime int64
}

func (obj *roomCreateTime) Less(than rbtree.Item) bool {
	return obj.createTime < than.(*roomCreateTime).createTime
}

// RoomCountdown 房间关闭倒计时
type RoomCountdown struct {
	rooms      *rbtree.Rbtree
	c          chan *roomCreateTime
	closeTimer *time.Timer
}

// Init function
func (obj *RoomCountdown) Init() error {
	obj.rooms = rbtree.New()
	obj.c = make(chan *roomCreateTime, 16)

	// 从数据库中加载未关闭的房间
	rooms, err := db.GetAllOpenRooms()
	if err != nil {
		return err
	}

	for _, room := range rooms {
		obj.rooms.Insert(&roomCreateTime{roomid: room.roomid, createTime: room.createTime})
	}

	ctx, _ := gApp.CreateCancelContext()
	gApp.GoRoutine(ctx, obj.loop)

	return nil
}

func (obj *RoomCountdown) loop(ctx context.Context) {
	t := time.NewTicker(time.Second)
	for {
		select {
		case <-ctx.Done():
			t.Stop()
			t = nil
			return

		case <-t.C:
			f := func() {
				for {
					select {
					case room := <-obj.c:
						obj.rooms.Insert(&roomCreateTime{roomid: room.roomid, createTime: room.createTime})
					default:
						return
					}
				}
			}
			f()
			obj.check()
		}
	}
}

// Add function
func (obj *RoomCountdown) Add(roomid uint32, createTime int64) {
	obj.c <- &roomCreateTime{roomid: roomid, createTime: createTime}
}

// Start function
func (obj *RoomCountdown) check() {
	timeout := time.Now().AddDate(0, 0, -1).Unix()
	for item := obj.rooms.Min(); item != nil; {
		rc := item.(*roomCreateTime)
		if rc.createTime <= timeout {
			roomManager.CloseRoom(rc.roomid)
		} else {
			return
		}
		obj.rooms.Delete(item)
		item = obj.rooms.Min()
	}
}
