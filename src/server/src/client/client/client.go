package client

import (
	"fmt"
	"math/rand"
	"time"

	"../../msg"
	"../robot"
	"../room"

	"github.com/golang/protobuf/proto"
	"github.com/lzhig/rapidgo/base"
	"github.com/lzhig/rapidgo/rapidnet"
)

const (
	actionLogin robot.ActionID = iota
	actionLeaveRoom
	actionListRooms
	actionJoinRoom
	actionSitDownNotify
	actionStandUp
)

const (
	clientEventNetworkMessage base.EventID = iota
)

// Client 客户端
type Client struct {
	base.EventSystem

	tcpClient  *rapidnet.TCPClient
	serverAddr string
	timeout    uint32
	conn       *rapidnet.Connection
	eventChan  <-chan *rapidnet.Event

	fbID string

	networkPacketHandler base.MessageHandlerImpl
	//protoHandler protocol.protocolHandler

	robot *robot.Robot

	room            *msg.Room
	waittingPlayers uint32

	uid        uint32
	seatID     int32
	cards      []uint32
	roomNumber uint32

	roomManager *room.RoomManager

	s *rand.Rand
}

// Init 初始化
func (obj *Client) Init(addr string, timeout uint32, fbID string, roomNumber uint32, roomManager *room.RoomManager) {
	obj.serverAddr = addr
	obj.timeout = timeout
	obj.tcpClient = rapidnet.CreateTCPClient()
	obj.fbID = fbID
	obj.roomNumber = roomNumber
	obj.seatID = -1
	obj.s = rand.New(rand.NewSource(time.Now().Unix()))
	obj.roomManager = roomManager

	obj.EventSystem.Init(1024, true)
	obj.SetEventHandler(clientEventNetworkMessage, func(args []interface{}) {
		p := args[0].(*msg.Protocol)
		if p == nil {
			base.LogError("args[0] isn't a msg.Protocol object.")
			return
		}

		if !obj.networkPacketHandler.Handle(p.Msgid, p) {
			base.LogError("cannot find handler for msgid:", msg.MessageID_name[int32(p.Msgid)])
			//obj.conn.Disconnect()
		}
	})

	//obj.protoHandler.init(obj)
	obj.networkPacketHandler.Init()
	obj.networkPacketHandler.SetMessageHandler(msg.MessageID_Login_Rsp, obj.handleLogin)
	obj.networkPacketHandler.SetMessageHandler(msg.MessageID_CreateRoom_Rsp, obj.handleCreateRoom)
	obj.networkPacketHandler.SetMessageHandler(msg.MessageID_JoinRoom_Rsp, obj.handleJoinRoom)
	obj.networkPacketHandler.SetMessageHandler(msg.MessageID_SitDown_Rsp, obj.handleSitDown)
	obj.networkPacketHandler.SetMessageHandler(msg.MessageID_StandUp_Rsp, obj.handleStandUp)
	obj.networkPacketHandler.SetMessageHandler(msg.MessageID_LeaveRoom_Rsp, obj.handleLeaveRoom)
	obj.networkPacketHandler.SetMessageHandler(msg.MessageID_Bet_Rsp, obj.handleBet)
	obj.networkPacketHandler.SetMessageHandler(msg.MessageID_GameState_Notify, obj.handleGameStateNotify)
	obj.networkPacketHandler.SetMessageHandler(msg.MessageID_ListRooms_Rsp, obj.handleListRooms)
	obj.networkPacketHandler.SetMessageHandler(msg.MessageID_JoinRoom_Notify, obj.handleJoinRoomNotify)
	obj.networkPacketHandler.SetMessageHandler(msg.MessageID_SitDown_Notify, obj.handleSitDownNotify)
	obj.networkPacketHandler.SetMessageHandler(msg.MessageID_ConsumeDiamonds_Notify, func(args interface{}) {})
	obj.networkPacketHandler.SetMessageHandler(msg.MessageID_Combine_Rsp, func(args interface{}) {})
	obj.networkPacketHandler.SetMessageHandler(msg.MessageID_StartGame_Rsp, func(args interface{}) {})
	obj.networkPacketHandler.SetMessageHandler(msg.MessageID_Bet_Notify, func(args interface{}) {})
	obj.networkPacketHandler.SetMessageHandler(msg.MessageID_Combine_Notify, func(args interface{}) {})
	obj.networkPacketHandler.SetMessageHandler(msg.MessageID_LeaveRoom_Notify, func(args interface{}) {})
	obj.networkPacketHandler.SetMessageHandler(msg.MessageID_StandUp_Notify, func(args interface{}) {})

	obj.initRobot()
}

func (obj *Client) initRobot() {
	obj.robot = &robot.Robot{}
	obj.robot.Init()

	obj.initRobotNoAi()
	obj.initRobotNormalAi()

	if !obj.robot.SwitchDriver("normal") {
		base.LogError("Cannot find the driver")
	}
}

func (obj *Client) GetRobot() *robot.Robot {
	return obj.robot
}

func (obj *Client) initRobotNoAi() {
	obj.robot.Set("no ai", robot.NewRobotDriver())
}

func (obj *Client) initRobotNormalAi() {
	robotDriver := robot.NewRobotDriver()

	// Login
	strategy := &robot.RobotStrategy{}
	strategy.Set([]*robot.RobotAction{
		robot.NewRobotAction(1000, obj.SendListRooms),
	})
	robotDriver.Set(actionLogin, strategy)

	// list room
	strategy = &robot.RobotStrategy{}
	strategy.Set([]*robot.RobotAction{
		robot.NewRobotAction(10, obj.JoinAvaliableRoom),
	})
	robotDriver.Set(actionListRooms, strategy)
	robotDriver.Set(actionLeaveRoom, strategy)

	// join room
	strategy = &robot.RobotStrategy{}
	strategy.Set([]*robot.RobotAction{
		robot.NewRobotAction(10, obj.sitDown),
	})
	robotDriver.Set(actionJoinRoom, strategy)

	// sit down
	strategy = &robot.RobotStrategy{}
	strategy.Set([]*robot.RobotAction{
		robot.NewRobotAction(10, func() {
			log(obj, "players: ", obj.room.Players)
			tablePlayers := obj.getTablePlayers()
			if obj.room.State == msg.GameState_Ready && obj.seatID == 0 && tablePlayers > obj.waittingPlayers {
				obj.SendStartGame()
			} else {
				log(obj, "waitting more players.state=", obj.room.State, ", players=", obj.waittingPlayers, ", tablePlayers=", tablePlayers)
			}
		}),
	})
	robotDriver.Set(actionSitDownNotify, strategy)

	// stand up
	strategy = &robot.RobotStrategy{}
	strategy.Set([]*robot.RobotAction{
		//NewRobotAction(10, obj.sitDown),
	})
	robotDriver.Set(actionStandUp, strategy)

	obj.robot.Set("normal", robotDriver)
}

// Start 开始
func (obj *Client) Start() {
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

func (obj *Client) sendProtocol(p *msg.Protocol) {
	//log(obj, "send:", p)
	data, err := proto.Marshal(p)
	if err != nil {
		log(obj, "Failed to marshal. p:", p, "error:", err)
	}
	obj.conn.Send(data)
}

func (obj *Client) sendLoginReq() {
	//log(obj, "send login request")
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

// SendGetProfile 发送GetProfile请求
func (obj *Client) SendGetProfile() {
	obj.sendProtocol(
		&msg.Protocol{
			Msgid:         msg.MessageID_GetProfile_Req,
			GetProfileReq: &msg.GetProfileReq{},
		})
}

func (obj *Client) SendSendDiamonds(uid, diamonds uint32) {
	obj.sendProtocol(
		&msg.Protocol{
			Msgid: msg.MessageID_SendDiamonds_Req,
			SendDiamondsReq: &msg.SendDiamondsReq{
				Uid:      uid,
				Diamonds: diamonds,
			},
		})
}

func (obj *Client) SendDiamondsRecords() {
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

func (obj *Client) SendListRooms() {
	obj.sendProtocol(
		&msg.Protocol{
			Msgid:        msg.MessageID_ListRooms_Req,
			ListRoomsReq: &msg.ListRoomsReq{},
		})
}

func (obj *Client) SendCreateRoom() {
	obj.sendProtocol(
		&msg.Protocol{
			Msgid: msg.MessageID_CreateRoom_Req,
			CreateRoomReq: &msg.CreateRoomReq{
				Name:         "fight",
				MinBet:       5,
				MaxBet:       100,
				Hands:        20,
				CreditPoints: 0,
				IsShare:      false,
			},
		})
}

func (obj *Client) SendCloseRoom(roomID uint32) {
	obj.sendProtocol(
		&msg.Protocol{
			Msgid: msg.MessageID_CloseRoom_Req,
			CloseRoomReq: &msg.CloseRoomReq{
				RoomId: roomID,
			},
		})
}

func (obj *Client) SendJoinRoom(number int) {
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

func (obj *Client) JoinAvaliableRoom() {

	room := obj.roomManager.GetRandomRoom()
	if room == nil {
		obj.SendCreateRoom()
	} else {
		obj.sendProtocol(
			&msg.Protocol{
				Msgid: msg.MessageID_JoinRoom_Req,
				JoinRoomReq: &msg.JoinRoomReq{
					RoomNumber: room.GetNumber(),
				},
			})
	}
}

func (obj *Client) SendLeaveRoom() {
	obj.sendProtocol(
		&msg.Protocol{
			Msgid:        msg.MessageID_LeaveRoom_Req,
			LeaveRoomReq: &msg.LeaveRoomReq{},
		})
}

func (obj *Client) sitDown() {
	obj.seatID = obj.getEmptySeatID()
	log(obj, "seatID = ", obj.seatID)
	if obj.seatID >= 0 {
		obj.SendSitDown(uint32(obj.seatID))
	} else {
		obj.SendLeaveRoom()
	}
}

func (obj *Client) SendSitDown(seatID uint32) {
	if obj.room == nil {
		base.LogError("cannot sit down because not in a room")
		return
	}

	obj.sendProtocol(
		&msg.Protocol{
			Msgid:      msg.MessageID_SitDown_Req,
			SitDownReq: &msg.SitDownReq{SeatId: seatID},
		})
}

func (obj *Client) getEmptySeatID() int32 {
	for seatID := uint32(0); seatID < 4; seatID++ {
		found := false
		for _, player := range obj.room.Players {
			if player.SeatId >= 0 && uint32(player.SeatId) == seatID {
				found = true
				break
			}
		}

		if !found {
			return int32(seatID)
		}
	}
	return -1
}

func (obj *Client) getTablePlayers() uint32 {
	num := uint32(0)
	for _, player := range obj.room.Players {
		if player.SeatId >= 0 {
			num++
		}
	}

	return num
}

func (obj *Client) SendBet() {
	obj.sendProtocol(
		&msg.Protocol{
			Msgid:  msg.MessageID_Bet_Req,
			BetReq: &msg.BetReq{Chips: 50},
		})
}

func (obj *Client) SendCombine() {
	obj.sendProtocol(
		&msg.Protocol{
			Msgid: msg.MessageID_Combine_Req,
			CombineReq: &msg.CombineReq{
				Autowin:    false,
				CardGroups: []*msg.CardGroup{&msg.CardGroup{Cards: []uint32{}}, &msg.CardGroup{Cards: []uint32{}}, &msg.CardGroup{Cards: []uint32{}}},
			},
		})
}

func (obj *Client) SendStandUp() {
	obj.sendProtocol(
		&msg.Protocol{
			Msgid:      msg.MessageID_StandUp_Req,
			StandUpReq: &msg.StandUpReq{},
		})
}

func (obj *Client) SendStartGame() {
	obj.sendProtocol(&msg.Protocol{
		Msgid:        msg.MessageID_StartGame_Req,
		StartGameReq: &msg.StartGameReq{},
	})
}

func (obj *Client) SendGetScorebard() {
	obj.sendProtocol(&msg.Protocol{
		Msgid:            msg.MessageID_GetScoreboard_Req,
		GetScoreboardReq: &msg.GetScoreboardReq{Pos: 0},
	})
}

func (obj *Client) SendGetRoundHistory(round uint32) {
	obj.sendProtocol(&msg.Protocol{
		Msgid:              msg.MessageID_GetRoundHistory_Req,
		GetRoundHistoryReq: &msg.GetRoundHistoryReq{Round: round},
	})
}

func (obj *Client) SendCareerWinLoseData(days []uint32) {
	obj.sendProtocol(&msg.Protocol{
		Msgid:                msg.MessageID_CareerWinLoseData_Req,
		CareerWinLoseDataReq: &msg.CareerWinLoseDataReq{Days: days},
	})
}

func (obj *Client) SendCareerRoomRecords(days uint32) {
	obj.sendProtocol(&msg.Protocol{
		Msgid:                msg.MessageID_CareerRoomRecords_Req,
		CareerRoomRecordsReq: &msg.CareerRoomRecordsReq{Days: days},
	})
}

func (obj *Client) handleConnection(conn *rapidnet.Connection) {
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

			obj.Send(clientEventNetworkMessage, []interface{}{&p})
		}
	}
}

func log(c *Client, args ...interface{}) {
	base.LogInfo("fbID:", c.fbID, " ---- ", fmt.Sprint(args...))
}
