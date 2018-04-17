package main

import (
	"./msg"
	"github.com/lzhig/rapidgo/base"
)

// IDispatchChan interface
type IDispatchChan interface {
	GetDispatchChan() chan<- *ProtocolConnection
}

type protocolHandler struct {
	dispatcher map[msg.MessageID]base.MessageHandlerFunc
}

func (obj *protocolHandler) init() {
	obj.dispatcher = map[msg.MessageID]base.MessageHandlerFunc{
		msg.MessageID_Login_Req:             obj.handleLogin,
		msg.MessageID_GetProfile_Req:        obj.handleLogin,
		msg.MessageID_GetPlayingRoom_Req:    roomManager.Handle,
		msg.MessageID_CreateRoom_Req:        roomManager.Handle,
		msg.MessageID_JoinRoom_Req:          roomManager.Handle,
		msg.MessageID_LeaveRoom_Req:         roomManager.Handle,
		msg.MessageID_ListRooms_Req:         roomManager.Handle,
		msg.MessageID_CloseRoom_Req:         roomManager.Handle,
		msg.MessageID_SitDown_Req:           obj.handleRoom,
		msg.MessageID_StandUp_Req:           obj.handleRoom,
		msg.MessageID_AutoBanker_Req:        obj.handleRoom,
		msg.MessageID_StartGame_Req:         obj.handleRoom,
		msg.MessageID_Bet_Req:               obj.handleRoom,
		msg.MessageID_Combine_Req:           obj.handleRoom,
		msg.MessageID_GetScoreboard_Req:     obj.handleRoom,
		msg.MessageID_GetRoundHistory_Req:   obj.handleRoom,
		msg.MessageID_SendDiamonds_Req:      diamondsCenter.Handle,
		msg.MessageID_DiamondsRecords_Req:   diamondsCenter.Handle,
		msg.MessageID_CareerWinLoseData_Req: careerCenter.Handle,
		msg.MessageID_CareerRoomRecords_Req: careerCenter.Handle,
	}
}

func (obj *protocolHandler) handle(p *ProtocolConnection) {
	if f, ok := obj.dispatcher[p.p.Msgid]; ok {
		base.LogInfo("received msg:", p.p, ", user:", p.userconn.user)
		f(p)
	} else {
		base.LogError("Cannot find dispatcher for msgid:", msg.MessageID_name[int32(p.p.Msgid)])
		p.userconn.Disconnect()
	}
}

func (obj *protocolHandler) handleLogin(arg interface{}) {
	p := arg.(*ProtocolConnection)
	loginService.GetDispatchChan() <- p
}

func (obj *protocolHandler) handleRoom(arg interface{}) {
	p := arg.(*ProtocolConnection)

	if p.userconn.user.room != nil {
		p.userconn.user.room.GetProtoChan() <- p
	} else {
		base.LogError("Cannot find room. proto:", p)
	}
}
