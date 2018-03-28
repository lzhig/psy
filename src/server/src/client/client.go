package main

import (
	"fmt"
	"time"

	"../msg"

	"github.com/golang/protobuf/proto"
	"github.com/lzhig/rapidgo/base"
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

	uid        uint32
	seatID     uint32
	cards      []uint32
	roomNumber uint32
}

func (obj *client) init(addr string, timeout uint32, fbID string, roomNumber uint32) {
	obj.serverAddr = addr
	obj.timeout = timeout
	obj.tcpClient = rapidnet.CreateTCPClient()
	obj.fbID = fbID
	obj.roomNumber = roomNumber

	obj.protoHandler.init(obj)
}

func (obj *client) start() {
	connectFunc := func() {
		for {
			var err error
			obj.conn, obj.eventChan, err = obj.tcpClient.Connect(obj.serverAddr, obj.timeout)
			if err != nil {
				log(obj, "connect error:", err)
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
				log(obj, event.Conn.RemoteAddr().String(), " connected")
				go obj.handleConnection(event.Conn)
				obj.sendLoginReq()
				//obj.sendCreateRoom()
			case rapidnet.EventDisconnected:
				log(obj, event.Conn.RemoteAddr().String(), " disconnected.", event.Err)
				connectFunc()
			}
		}
	}
}

func (obj *client) sendProtocol(p *msg.Protocol) {
	data, err := proto.Marshal(p)
	if err != nil {
		log(obj, "Failed to marshal. p:", p, "error:", err)
	}
	obj.conn.Send(data)
}

func (obj *client) sendLoginReq() {
	log(obj, "send login request")
	obj.sendProtocol(
		&msg.Protocol{
			Msgid: msg.MessageID_Login_Req,
			LoginReq: &msg.LoginReq{
				Type: msg.LoginType_Facebook,
				Fb: &msg.LoginFBReq{
					FbId:  obj.fbID,
					Token: "",
				},
			},
		})
}

func (obj *client) sendGetProfile() {
	obj.sendProtocol(
		&msg.Protocol{
			Msgid:         msg.MessageID_GetProfile_Req,
			GetProfileReq: &msg.GetProfileReq{},
		})
}

func (obj *client) sendSendDiamonds(uid, diamonds uint32) {
	obj.sendProtocol(
		&msg.Protocol{
			Msgid: msg.MessageID_SendDiamonds_Req,
			SendDiamondsReq: &msg.SendDiamondsReq{
				Uid:      uid,
				Diamonds: diamonds,
			},
		})
}

func (obj *client) sendDiamondsRecords() {
	tomorrow := time.Now().AddDate(0, 0, 1)
	end := tomorrow.Format("2006-1-2")
	begin := tomorrow.AddDate(0, 0, -30).Format("2006-1-2")
	obj.sendProtocol(
		&msg.Protocol{
			Msgid: msg.MessageID_DiamondsRecords_Req,
			DiamondsRecordsReq: &msg.DiamondsRecordsReq{
				BeginTime: begin,
				EndTime:   end,
			},
		})
}

func (obj *client) sendListRooms() {
	obj.sendProtocol(
		&msg.Protocol{
			Msgid:        msg.MessageID_ListRooms_Req,
			ListRoomsReq: &msg.ListRoomsReq{},
		})
}

func (obj *client) sendCreateRoom() {
	obj.sendProtocol(
		&msg.Protocol{
			Msgid: msg.MessageID_CreateRoom_Req,
			CreateRoomReq: &msg.CreateRoomReq{
				Name:         "fight",
				MinBet:       5,
				MaxBet:       100,
				Hands:        10,
				CreditPoints: 0,
				IsShare:      false,
			},
		})
}

func (obj *client) sendCloseRoom(roomID uint32) {
	obj.sendProtocol(
		&msg.Protocol{
			Msgid: msg.MessageID_CloseRoom_Req,
			CloseRoomReq: &msg.CloseRoomReq{
				RoomId: roomID,
			},
		})
}

func (obj *client) sendJoinRoom(number int) {
	obj.roomNumber = uint32(number)
	log(obj, "join room: ", obj.roomNumber)
	obj.sendProtocol(
		&msg.Protocol{
			Msgid: msg.MessageID_JoinRoom_Req,
			JoinRoomReq: &msg.JoinRoomReq{
				RoomNumber: fmt.Sprintf("%d", obj.roomNumber),
			},
		})
}

func (obj *client) sendLeaveRoom() {
	obj.sendProtocol(
		&msg.Protocol{
			Msgid:        msg.MessageID_LeaveRoom_Req,
			LeaveRoomReq: &msg.LeaveRoomReq{},
		})
}

func (obj *client) sendSitDown() {
	obj.sendProtocol(
		&msg.Protocol{
			Msgid:      msg.MessageID_SitDown_Req,
			SitDownReq: &msg.SitDownReq{SeatId: obj.seatID},
		})
}

func (obj *client) sendBet() {
	obj.sendProtocol(
		&msg.Protocol{
			Msgid:  msg.MessageID_Bet_Req,
			BetReq: &msg.BetReq{Chips: 50},
		})
}

func (obj *client) sendCombine() {
	obj.sendProtocol(
		&msg.Protocol{
			Msgid: msg.MessageID_Combine_Req,
			CombineReq: &msg.CombineReq{
				Autowin:    false,
				CardGroups: []*msg.CardGroup{&msg.CardGroup{Cards: []uint32{}}, &msg.CardGroup{Cards: []uint32{}}, &msg.CardGroup{Cards: []uint32{}}},
			},
		})
}

func (obj *client) sendStandUp() {
	obj.sendProtocol(
		&msg.Protocol{
			Msgid:      msg.MessageID_StandUp_Req,
			StandUpReq: &msg.StandUpReq{},
		})
}

func (obj *client) sendStartGame() {
	obj.sendProtocol(&msg.Protocol{
		Msgid:        msg.MessageID_StartGame_Req,
		StartGameReq: &msg.StartGameReq{},
	})
}

func (obj *client) sendGetScorebard() {
	obj.sendProtocol(&msg.Protocol{
		Msgid:            msg.MessageID_GetScoreboard_Req,
		GetScoreboardReq: &msg.GetScoreboardReq{Pos: 0},
	})
}

func (obj *client) sendGetRoundHistory(round uint32) {
	obj.sendProtocol(&msg.Protocol{
		Msgid:              msg.MessageID_GetRoundHistory_Req,
		GetRoundHistoryReq: &msg.GetRoundHistoryReq{Round: round},
	})
}

func (obj *client) handleConnection(conn *rapidnet.Connection) {
	defer base.LogPanic()
	defer func() {
		log(obj, "exit handleConnection.")
	}()

	for {
		select {
		case data := <-conn.ReceiveDataChan():
			if data == nil {
				return
			}

			//log(obj, "Recieve data. size:", len(data))

			var p msg.Protocol
			if err := proto.Unmarshal(data, &p); err != nil {
				log(obj, err)
				conn.Disconnect()
				return
			}

			obj.protoHandler.getProtoChan() <- &p
		}
	}
}

func log(c *client, args ...interface{}) {
	fmt.Println("fbID:", c.fbID, " ---- ", fmt.Sprint(args...))
}
