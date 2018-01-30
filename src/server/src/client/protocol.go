package main

import (
	"fmt"

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
		msg.MessageID_Login_Rsp:      obj.handleLogin,
		msg.MessageID_CreateRoom_Rsp: obj.handleCreateRoom,
		msg.MessageID_JoinRoom_Rsp:   obj.handleJoinRoom,
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
		fmt.Println("received msgid:", msg.MessageID_name[int32(p.Msgid)])
		f(p)
	} else {
		fmt.Println("cannot find handler for msgid:", msg.MessageID_name[int32(p.Msgid)])
	}
}

func (obj *protocolHandler) handleLogin(p *msg.Protocol) {
	fmt.Println(p)
	//obj.c.sendCreateRoom()
	obj.c.sendJoinRoom()
}

func (obj *protocolHandler) handleCreateRoom(p *msg.Protocol) {
	fmt.Println(p)
}

func (obj *protocolHandler) handleJoinRoom(p *msg.Protocol) {
	fmt.Println(p)
}
