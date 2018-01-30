package main

import (
	"fmt"
	"time"

	"../msg"

	"github.com/golang/protobuf/proto"
	"github.com/lzhig/rapidgo/rapidnet"
)

type client struct {
	tcpClient  *rapidnet.TCPClient
	serverAddr string
	timeout    uint32
	conn       *rapidnet.Connection
	eventChan  <-chan *rapidnet.Event

	fbID string

	protoHandler protocolHandler
}

func (obj *client) init(addr string, timeout uint32, fbID string) {
	obj.serverAddr = addr
	obj.timeout = timeout
	obj.tcpClient = rapidnet.CreateTCPClient()
	obj.fbID = fbID

	obj.protoHandler.init(obj)
}

func (obj *client) start() {
	connectFunc := func() {
		for {
			var err error
			obj.conn, obj.eventChan, err = obj.tcpClient.Connect(obj.serverAddr, obj.timeout)
			if err != nil {
				fmt.Println("connect error:", err)
				time.Sleep(time.Second)
			} else {
				return
			}
		}
	}

	connectFunc()

	for {
		select {
		case event := <-obj.eventChan:
			switch event.Type {
			case rapidnet.EventConnected:
				fmt.Println(event.Conn.RemoteAddr().String(), "connected")
				go obj.handleConnection(event.Conn)
				obj.sendLoginReq()
				//obj.sendCreateRoom()
			case rapidnet.EventDisconnected:
				fmt.Println(event.Conn.RemoteAddr().String(), "disconnected.", event.Err)
				connectFunc()
			}
		}
	}
}

func (obj *client) sendProtocol(p *msg.Protocol) {
	data, err := proto.Marshal(p)
	if err != nil {
		fmt.Println("Failed to marshal. p:", p, "error:", err)
	}
	obj.conn.Send(data)
}

func (obj *client) sendLoginReq() {
	obj.sendProtocol(
		&msg.Protocol{
			Msgid: msg.MessageID_Login_Req,
			LoginReq: &msg.LoginReq{
				Type: msg.LoginType_Facebook,
				Fb: &msg.LoginFBReq{
					FbId:  obj.fbID,
					Token: "",
					Name:  fmt.Sprintf("name_%s", obj.fbID),
				},
			},
		},
	)
}

func (obj *client) sendCreateRoom() {
	obj.sendProtocol(
		&msg.Protocol{
			Msgid: msg.MessageID_CreateRoom_Req,
			CreateRoomReq: &msg.CreateRoomReq{
				Name:         "fight",
				MinBet:       50,
				MaxBet:       100,
				Hands:        10,
				CreditPoints: 0,
				IsShare:      false,
			},
		},
	)
}

func (obj *client) sendJoinRoom() {
	obj.sendProtocol(
		&msg.Protocol{
			Msgid: msg.MessageID_JoinRoom_Req,
			JoinRoomReq: &msg.JoinRoomReq{
				RoomNumber: "fight",
			},
		},
	)
}

func (obj *client) handleConnection(conn *rapidnet.Connection) {
	defer func() {
		fmt.Println("exit handleConnection.")
	}()

	for {
		select {
		case data := <-conn.ReceiveDataChan():
			if data == nil {
				return
			}

			fmt.Println("Recieve data. size:", len(data))

			var p msg.Protocol
			if err := proto.Unmarshal(data, &p); err != nil {
				fmt.Println(err)
				conn.Disconnect()
				return
			}

			obj.protoHandler.getProtoChan() <- &p
		}
	}
}
