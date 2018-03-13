package main

import (
	"./msg"
)

// ProtoHandlerFunc type
type ProtoHandlerFunc func(*ProtocolConnection)

// IDispatchChan interface
type IDispatchChan interface {
	GetDispatchChan() chan<- *ProtocolConnection
}

type protocolHandler struct {
	dispatcher map[msg.MessageID]ProtoHandlerFunc
}

func (obj *protocolHandler) init() {
	obj.dispatcher = map[msg.MessageID]ProtoHandlerFunc{
		msg.MessageID_Login_Req:      obj.handleLogin,
		msg.MessageID_CreateRoom_Req: obj.handleRoom,
		msg.MessageID_JoinRoom_Req:   obj.handleRoom,
		msg.MessageID_LeaveRoom_Req:  obj.handleRoom,
		msg.MessageID_SitDown_Req:    obj.handleRoom,
		msg.MessageID_StandUp_Req:    obj.handleRoom,
		msg.MessageID_AutoBanker_Req: obj.handleRoom,
		msg.MessageID_StartGame_Req:  obj.handleRoom,
		msg.MessageID_Bet_Req:        obj.handleRoom,
		msg.MessageID_Combine_Req:    obj.handleRoom,
	}
}

func (obj *protocolHandler) handle(p *ProtocolConnection) {
	if f, ok := obj.dispatcher[p.p.Msgid]; ok {
		logInfo("received msgid:", msg.MessageID_name[int32(p.p.Msgid)], "uid:", p.userconn.uid)
		f(p)
	} else {
		logError("[protocolHandler][dispatch] cannot find dispatcher for msgid:", msg.MessageID_name[int32(p.p.Msgid)])
		p.userconn.Disconnect()
	}
}

func (obj *protocolHandler) handleLogin(p *ProtocolConnection) {
	loginService.GetDispatchChan() <- p
}

func (obj *protocolHandler) handleRoom(p *ProtocolConnection) {
	switch p.p.Msgid {
	case msg.MessageID_CreateRoom_Req,
		msg.MessageID_JoinRoom_Req,
		msg.MessageID_LeaveRoom_Req:
		roomManager.GetDispatchChan() <- p

	default:
		if p.userconn.room != nil {
			p.userconn.room.GetProtoChan() <- p
		} else {
			logError("[protocolHandler][handleRoom] cannot find room. proto:", p)
		}
	}
}
