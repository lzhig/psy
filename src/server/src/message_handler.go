package main

import (
	"context"

	"./msg"
	"github.com/lzhig/rapidgo/base"
)

// MessageHandlerFunc type
type MessageHandlerFunc func(*ProtocolConnection)

type MessageHandler interface {
	Init()
	AddMessageHandler(msg.MessageID, MessageHandlerFunc)
	AddBusyHandler(msg.MessageID, MessageHandlerFunc)
	Handle(p *ProtocolConnection)
}

type MessageHandlerImpl struct {
	protoChan         chan *ProtocolConnection
	messageDispatcher map[msg.MessageID]MessageHandlerFunc
	busyDispatcher    map[msg.MessageID]MessageHandlerFunc
}

func (obj *MessageHandlerImpl) Init() {
	obj.protoChan = make(chan *ProtocolConnection, 16)
	obj.messageDispatcher = make(map[msg.MessageID]MessageHandlerFunc)
	obj.busyDispatcher = make(map[msg.MessageID]MessageHandlerFunc)
	ctx, _ := gApp.CreateCancelContext()
	gApp.GoRoutine(ctx, obj.loop)
}

func (obj *MessageHandlerImpl) loop(ctx context.Context) {
	defer debug("exit MessageHandlerImpl goroutine")
	for {
		select {
		case <-ctx.Done():
			return

		case p := <-obj.protoChan:
			if f, ok := obj.messageDispatcher[p.p.Msgid]; ok {
				f(p)
			} else {
				base.LogError("Cannot find message dispatcher for msg:", p)
				p.userconn.Disconnect()
			}
		}
	}
}

func (obj *MessageHandlerImpl) AddMessageHandler(msgid msg.MessageID, f MessageHandlerFunc) {
	obj.messageDispatcher[msgid] = f
}

func (obj *MessageHandlerImpl) AddBusyHandler(msgid msg.MessageID, f MessageHandlerFunc) {
	obj.busyDispatcher[msgid] = f
}

func (obj *MessageHandlerImpl) Handle(p *ProtocolConnection) {
	select {
	case obj.protoChan <- p:
	default:
		obj.handleBusy(p)
	}
}

func (obj *MessageHandlerImpl) handleBusy(p *ProtocolConnection) {
	if f, ok := obj.busyDispatcher[p.p.Msgid]; ok {
		f(p)
	} else {
		base.LogError("Cannot find busy dispatcher for msg:", p)
		p.userconn.Disconnect()
	}
}
