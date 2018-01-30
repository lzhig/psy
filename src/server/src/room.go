/**
 * @author [Bruce]
 * @email [lzhig@outlook.com]
 * @create date 2018-01-18 03:42:03
 * @modify date 2018-01-18 03:42:20
 * @desc [description]
 */

package main

import (
	"context"
	"fmt"
)

const (
	roomEventJoin int = iota
)

type roomEvent struct {
	event int
	args  []interface{}
}

// Room type
type Room struct {
	name         string
	roomID       uint32
	number       int
	ownerUID     uint32
	hands        uint32
	playedHands  uint32
	isShare      bool
	minBet       uint32
	maxBet       uint32
	creditPoints uint32
	createTime   uint32
	closeTime    uint32
	closed       bool

	eventChan     chan *roomEvent
	eventHandlers map[int]func([]interface{})
}

func (obj *Room) init() {
	obj.eventChan = make(chan *roomEvent, 16)
	obj.eventHandlers = map[int]func([]interface{}){
		roomEventJoin: obj.handleJoinRoomEvent,
	}
	ctx, _ := gApp.CreateCancelContext()
	gApp.GoRoutine(ctx, obj.loop)
}

func (obj *Room) loop(ctx context.Context) {
	defer debug(fmt.Sprintf("exit Room %d goroutine", obj.roomID))
	for {
		select {
		case <-ctx.Done():
			return

		case event := <-obj.eventChan:
			if handler, ok := obj.eventHandlers[event.event]; ok {
				handler(event.args)
			} else {
				logError("[Room][loop] cannot find a handler for event:", event.event)
				gApp.Exit()
			}
		}
	}
}

func (obj *Room) getEventChan() chan<- *roomEvent {
	return obj.eventChan
}

func (obj *Room) handleJoinRoomEvent(args []interface{}) {

}
