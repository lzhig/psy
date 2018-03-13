package main

import (
	"../msg"
)

type protocolHandler struct {
	handler   map[msg.MessageID]func(*msg.Protocol)
	protoChan chan *msg.Protocol
	c         *client
}

func (obj *protocolHandler) init(c *client) {
	obj.c = c
	obj.handler = map[msg.MessageID]func(*msg.Protocol){
		msg.MessageID_Login_Rsp:        obj.handleLogin,
		msg.MessageID_CreateRoom_Rsp:   obj.handleCreateRoom,
		msg.MessageID_JoinRoom_Rsp:     obj.handleJoinRoom,
		msg.MessageID_SitDown_Rsp:      obj.handleSitDown,
		msg.MessageID_StandUp_Rsp:      obj.handleStandUp,
		msg.MessageID_LeaveRoom_Rsp:    obj.handleLeaveRoom,
		msg.MessageID_Bet_Rsp:          obj.handleBet,
		msg.MessageID_GameState_Notify: obj.handleGameStateNotify,
	}
	obj.protoChan = make(chan *msg.Protocol)

	go obj.loop()
}

func (obj *protocolHandler) getProtoChan() chan<- *msg.Protocol {
	return obj.protoChan
}

func (obj *protocolHandler) loop() {
	for {
		select {
		case p := <-obj.protoChan:
			obj.handle(p)
		}
	}
}

func (obj *protocolHandler) handle(p *msg.Protocol) {
	if f, ok := obj.handler[p.Msgid]; ok {
		log(obj.c, "received ", p)
		f(p)
	} else {
		log(obj.c, "cannot find handler for msg:", p)
	}
}

func (obj *protocolHandler) handleLogin(p *msg.Protocol) {
	//obj.c.sendCreateRoom()
	obj.c.sendJoinRoom()
}

func (obj *protocolHandler) handleCreateRoom(p *msg.Protocol) {
}

func (obj *protocolHandler) handleJoinRoom(p *msg.Protocol) {
	obj.c.sendSitDown()
}

func (obj *protocolHandler) handleSitDown(p *msg.Protocol) {
	if p.SitDownRsp.Ret != msg.ErrorID_Ok && p.SitDownRsp.Ret != msg.ErrorID_SitDown_Already_Sit {
		obj.c.seatID++
		if obj.c.seatID >= 5 {
			obj.c.seatID = 0
		}
		obj.c.sendSitDown()
		return
	}
	//obj.c.sendStandUp()
	obj.c.sendBet()
}

func (obj *protocolHandler) handleStandUp(p *msg.Protocol) {
	obj.c.sendLeaveRoom()
}

func (obj *protocolHandler) handleLeaveRoom(p *msg.Protocol) {
}

func (obj *protocolHandler) handleBet(p *msg.Protocol) {
}

func (obj *protocolHandler) handleGameStateNotify(p *msg.Protocol) {
	switch p.GameStateNotify.State {
	case msg.GameState_Ready:
		if obj.c.seatID == 0 {
			obj.c.sendStartGame()
		}
	case msg.GameState_Bet:
		if obj.c.seatID != 0 {
			obj.c.sendBet()
		}
	case msg.GameState_Deal:
		obj.c.cards = p.GameStateNotify.DealCards
	case msg.GameState_Combine:
		obj.c.sendCombine()
	}
}
