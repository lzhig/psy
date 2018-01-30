package main

import (
	"context"

	"./msg"
)

// IDispatchChan interface
type IDispatchChan interface {
	GetDispatchChan() chan<- *ProtocolConnection
}

type protocolHandler struct {
	protoChan  chan *ProtocolConnection
	dispatcher map[msg.MessageID]IDispatchChan
}

func (obj *protocolHandler) init() {
	obj.dispatcher = map[msg.MessageID]IDispatchChan{
		msg.MessageID_Login_Req:      loginService,
		msg.MessageID_CreateRoom_Req: roomManager,
	}
	obj.protoChan = make(chan *ProtocolConnection)

	ctx, _ := gApp.CreateCancelContext()
	gApp.GoRoutine(ctx, obj.loop)
}

func (obj *protocolHandler) getProtoChan() chan<- *ProtocolConnection {
	return obj.protoChan
}

func (obj *protocolHandler) loop(ctx context.Context) {
	defer debug("exit protocolHandler goroutine")
	for {
		select {
		case <-ctx.Done():
			return

		case p := <-obj.protoChan:
			obj.dispatch(p)
		}
	}
}

func (obj *protocolHandler) dispatch(p *ProtocolConnection) {
	if d, ok := obj.dispatcher[p.p.Msgid]; ok {
		debug("received msgid:", msg.MessageID_name[int32(p.p.Msgid)])
		select {
		case d.GetDispatchChan() <- p:
		default:
			logWarn("[protocolHandler][dispatch] too much requests to dispatch")
			p.userconn.Disconnect()
		}

	} else {
		logError("[protocolHandler][dispatch] cannot find dispatcher for msgid:", msg.MessageID_name[int32(p.p.Msgid)])
		p.userconn.Disconnect()
	}
}
