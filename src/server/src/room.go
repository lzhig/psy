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
	"math"
	"time"

	"github.com/golang/protobuf/proto"

	"./msg"
	"github.com/lzhig/rapidgo/base"
)

const (
	roomEventUserDisconnect int = iota
	roomEventUserReconnect
	roomEventGameStateTimeout
	roomEventClose
	roomEventGetSeatPlayers
)

type roomEvent struct {
	event int
	args  []interface{}
}

// RoomPlayer type
type RoomPlayer struct {
	uid        uint32
	name       string
	avatar     string
	conn       *userConnection
	seatID     int32 // -1 没有入座
	handsNoBet uint32
}

func (obj *RoomPlayer) sendProtocol(p *msg.Protocol) {
	if obj.conn != nil {
		sendProtocol(obj.conn.conn, p)
	}
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

	autoBanker bool

	eventChan     chan *roomEvent
	eventHandlers map[int]func([]interface{})

	protoChan     chan *ProtocolConnection
	protoHandlers map[msg.MessageID]func(*ProtocolConnection)

	players      map[uint32]*RoomPlayer // 房间中的玩家, key为uid
	tablePlayers []*RoomPlayer          // 入座的玩家

	round      Round
	scoreboard Scoreboard // 积分榜
}

func (obj *Room) init(bLoadScoreboard bool) {
	obj.eventChan = make(chan *roomEvent, 16)
	obj.eventHandlers = map[int]func([]interface{}){
		roomEventUserDisconnect:   obj.handleEventUserDisconnect,
		roomEventUserReconnect:    obj.handleEventUserReconnect,
		roomEventGameStateTimeout: obj.handleEventGameStateTimeout,
		roomEventClose:            obj.handleEventClose,
		roomEventGetSeatPlayers:   obj.handleEventGetSeatPlayers,
	}

	obj.protoChan = make(chan *ProtocolConnection, 128)
	obj.protoHandlers = map[msg.MessageID]func(*ProtocolConnection){
		msg.MessageID_JoinRoom_Req:        obj.handleJoinRoomReq,
		msg.MessageID_LeaveRoom_Req:       obj.handleLeaveRoomReq,
		msg.MessageID_SitDown_Req:         obj.handleSitDownReq,
		msg.MessageID_StandUp_Req:         obj.handleStandUpReq,
		msg.MessageID_AutoBanker_Req:      obj.handleAutoBankerReq,
		msg.MessageID_StartGame_Req:       obj.handleStartGameReq,
		msg.MessageID_Bet_Req:             obj.handleBetReq,
		msg.MessageID_Combine_Req:         obj.handleCombineReq,
		msg.MessageID_GetScoreboard_Req:   obj.handleGetScoreboardReq,
		msg.MessageID_GetRoundHistory_Req: obj.handleGetRoundHistoryReq,
	}

	obj.players = make(map[uint32]*RoomPlayer)
	obj.tablePlayers = make([]*RoomPlayer, gApp.config.Room.MaxTablePlayers, gApp.config.Room.MaxTablePlayers)

	obj.round.Init(obj)
	obj.round.Begin()

	obj.scoreboard.Init(obj.roomID, gApp.config.Room.MaxTablePlayers)
	if bLoadScoreboard {
		if err := obj.scoreboard.Load(); err != nil {
			base.LogError("[Room][Init] Failed to load scoreboard. error:", err)
		}
	}

	ctx, _ := gApp.CreateCancelContext()
	gApp.GoRoutine(ctx, obj.loop)
}

func (obj *Room) nextRound() {
	obj.playedHands++
}

// GetProtoChan function
func (obj *Room) GetProtoChan() chan<- *ProtocolConnection {
	return obj.protoChan
}

func (obj *Room) loop(ctx context.Context) {
	defer debug(fmt.Sprintf("exit Room %d goroutine", obj.roomID))
	for {
		select {
		case <-ctx.Done():
			return

		case p := <-obj.protoChan:
			if handler, ok := obj.protoHandlers[p.p.Msgid]; ok {
				handler(p)
			} else {
				base.LogError("[Room][loop] cannot find handler for msgid:", msg.MessageID_name[int32(p.p.Msgid)])
				p.userconn.Disconnect()
			}

		case event := <-obj.eventChan:
			if handler, ok := obj.eventHandlers[event.event]; ok {
				handler(event.args)
			} else {
				base.LogError("[Room][loop] cannot find a handler for event:", event.event)
				gApp.Exit()
			}
		}
	}
}

func (obj *Room) getEventChan() chan<- *roomEvent {
	return obj.eventChan
}

func (obj *Room) notifyUserDisconnect(uid uint32) {
	obj.eventChan <- &roomEvent{event: roomEventUserDisconnect, args: []interface{}{uid}}
}

func (obj *Room) handleEventUserDisconnect(args []interface{}) {
	if args == nil || len(args) == 0 {
		base.LogError("[Room][handleEventUserDisconnect] invalid args")
		return
	}
	uid := args[0].(uint32)

	player := obj.players[uid]
	if (player.seatID >= 0 && (obj.round.state == msg.GameState_Ready || obj.round.state == msg.GameState_Bet)) ||
		player.seatID < 0 {
		obj.notifyOthers(player.conn,
			&msg.Protocol{
				Msgid: msg.MessageID_LeaveRoom_Notify,
				LeaveRoomNotify: &msg.LeaveRoomNotify{
					Uid: uid,
				}},
		)

		if player.seatID > 0 {
			// 取消下注
			obj.round.unbet(uint32(player.seatID))
		}

		// 将断线玩家踢离房间
		obj.kickPlayer(player.uid)
	} else {
		// 通知其他玩家
		obj.notifyOthers(player.conn,
			&msg.Protocol{
				Msgid: msg.MessageID_Disconnect_Notify,
				DisconnectNotify: &msg.DisconnectNotify{
					Uid: uid,
				}})

		// 设置成断线
		player.conn = nil
	}
}

func (obj *Room) notifyUserReconnect(uid uint32, conn *userConnection) {
	obj.eventChan <- &roomEvent{event: roomEventUserReconnect, args: []interface{}{uid, conn}}
}

func (obj *Room) handleEventUserReconnect(args []interface{}) {
	uid := args[0].(uint32)
	conn := args[1].(*userConnection)

	obj.notifyOthers(conn,
		&msg.Protocol{
			Msgid: msg.MessageID_Reconnect_Notify,
			ReconnectNotify: &msg.ReconnectNotify{
				Uid: uid,
			}})
	obj.players[uid].conn = conn
}

func (obj *Room) handleEventGameStateTimeout(args []interface{}) {
	state := args[0].(msg.GameState)
	obj.round.HandleTimeout(state)
}

func (obj *Room) notifyCloseRoom(c chan bool) {
	obj.eventChan <- &roomEvent{event: roomEventClose, args: []interface{}{c}}
}

func (obj *Room) handleEventClose(args []interface{}) {
	c := args[0].(chan bool)

	if len(obj.players) == 0 {
		// update db
		obj.closed = true
		if err := db.CloseRoom(obj.roomID, time.Now().Unix()); err != nil {
			base.LogError("Fail to close room. error:", err)
		}
		c <- true
	} else {
		c <- false
	}
}

func (obj *Room) notifyGetSeatPlayers(c chan []*msg.ListRoomPlayerInfo) {
	obj.eventChan <- &roomEvent{event: roomEventGetSeatPlayers, args: []interface{}{c}}
}

func (obj *Room) handleEventGetSeatPlayers(args []interface{}) {
	c := args[0].(chan []*msg.ListRoomPlayerInfo)
	p := make([]*msg.ListRoomPlayerInfo, 0, len(obj.tablePlayers))
	for _, player := range obj.tablePlayers {
		if player == nil {
			continue
		}
		p = append(p, &msg.ListRoomPlayerInfo{
			SeatId: uint32(player.seatID),
			Name:   player.name,
			Avatar: player.avatar,
		})
	}
	c <- p
}
func (obj *Room) notifyOthers(userconn *userConnection, p *msg.Protocol) {
	for uid, player := range obj.players {
		if userconn == nil || uid == userconn.user.uid || player.conn == nil {
			continue
		}

		player.sendProtocol(p)
	}
}

func (obj *Room) notifyAll(p *msg.Protocol) {
	for _, player := range obj.players {
		player.sendProtocol(p)
	}
}

func (obj *Room) handleJoinRoomReq(p *ProtocolConnection) {
	rsp := &msg.Protocol{
		Msgid:       msg.MessageID_JoinRoom_Rsp,
		JoinRoomRsp: &msg.JoinRoomRsp{Ret: msg.ErrorID_Ok},
	}
	defer p.userconn.sendProtocol(rsp)

	if obj.closed {
		rsp.JoinRoomRsp.Ret = msg.ErrorID_JoinRoom_Closed
		return
	}

	isNewPlayer := true
	seatID := int32(0)
	if player, ok := obj.players[p.userconn.user.uid]; ok {
		isNewPlayer = false
		player.conn = p.userconn
		seatID = player.seatID
		// 通知其他玩家：此玩家重连成功
		obj.notifyOthers(
			p.userconn,
			&msg.Protocol{
				Msgid: msg.MessageID_Reconnect_Notify,
				ReconnectNotify: &msg.ReconnectNotify{
					Uid: p.userconn.user.uid,
				}},
		)
	} else {
		// 如果是新加入用户
		if len(obj.players) >= int(gApp.config.Room.MaxPlayers) {
			rsp.JoinRoomRsp.Ret = msg.ErrorID_JoinRoom_Full
			return
		}

		player := &RoomPlayer{
			uid:    p.userconn.user.uid,
			name:   p.userconn.user.name,
			avatar: p.userconn.user.avatar,
			conn:   p.userconn,
			seatID: -1,
		}
		seatID = -1

		obj.players[p.userconn.user.uid] = player
		userManager.enterRoom(p.userconn.user.uid, obj)
	}

	rsp.JoinRoomRsp.Room = &msg.Room{
		RoomId:       obj.roomID,
		Number:       roomNumberGenerator.decode(obj.number),
		Name:         obj.name,
		MinBet:       obj.minBet,
		MaxBet:       obj.maxBet,
		Hands:        obj.hands,
		PlayedHands:  obj.playedHands,
		CreditPoints: obj.creditPoints,
		IsShare:      obj.isShare,
		Players:      make([]*msg.Player, len(obj.players)),
		State:        obj.round.state,
	}
	//cards
	if seatID >= 0 {
		if cards, ok := obj.round.handCards[uint32(seatID)]; ok {
			rsp.JoinRoomRsp.Room.Cards = cards
		}
	}

	//countdown
	if gApp.config.Room.StatesCountdown[obj.round.state] == 0 {
		rsp.JoinRoomRsp.Room.Countdown = -1
	} else {
		rsp.JoinRoomRsp.Room.Countdown = int32(gApp.config.Room.StatesCountdown[obj.round.state]) - int32(time.Now().Sub(obj.round.stateBeginTime).Seconds()*1000)
	}
	//result
	if len(obj.round.result) > 0 {
		rsp.JoinRoomRsp.Room.Result = make([]*msg.SeatResult, len(obj.round.result))
		i := 0
		for _, v := range obj.round.result {
			rsp.JoinRoomRsp.Room.Result[i] = v
			i++
		}
	}

	i := 0
	for _, player := range obj.players {
		score := obj.scoreboard.GetScore(player.uid)
		p := &msg.Player{
			Uid:    player.uid,
			Name:   player.name,
			Avatar: player.avatar,
			SeatId: player.seatID,
			Score:  score,
		}
		if player.seatID > 0 {
			p.Bet = obj.round.betChips[uint32(player.seatID)]
		}

		rsp.JoinRoomRsp.Room.Players[i] = p
		i++
	}

	if isNewPlayer {
		// 通知房间其他人
		obj.notifyOthers(
			p.userconn,
			&msg.Protocol{
				Msgid: msg.MessageID_JoinRoom_Notify,
				JoinRoomNotify: &msg.JoinRoomNotify{
					Uid:    p.userconn.user.uid,
					Name:   p.userconn.user.name,
					Avatar: p.userconn.user.avatar,
				}},
		)
	}

	p.userconn.user.room = obj
}

func (obj *Room) handleLeaveRoomReq(p *ProtocolConnection) {
	rsp := &msg.Protocol{
		Msgid:        msg.MessageID_LeaveRoom_Rsp,
		LeaveRoomRsp: &msg.LeaveRoomRsp{Ret: msg.ErrorID_Ok},
	}
	defer p.userconn.sendProtocol(rsp)

	player, ok := obj.players[p.userconn.user.uid]
	if !ok {
		rsp.LeaveRoomRsp.Ret = msg.ErrorID_LeaveRoom_Not_In
		return
	}

	// 检查入座和游戏中
	if player.seatID >= 0 && obj.round.state >= msg.GameState_Confirm_Bet && obj.round.state < msg.GameState_Result {
		rsp.LeaveRoomRsp.Ret = msg.ErrorID_LeaveRoom_Playing
		return
	}

	needSwitchToReady := false
	if obj.round.state == msg.GameState_Bet && player.seatID == 0 {
		needSwitchToReady = true
	}

	obj.kickPlayer(player.uid)
	debug("leave room. uid:", p.userconn.user.uid)

	// 通知房间其他人
	obj.notifyAll(
		&msg.Protocol{
			Msgid: msg.MessageID_LeaveRoom_Notify,
			LeaveRoomNotify: &msg.LeaveRoomNotify{
				Uid: p.userconn.user.uid,
			}},
	)

	if needSwitchToReady {
		obj.round.switchGameState(msg.GameState_Ready)
	}
}

func (obj *Room) handleSitDownReq(p *ProtocolConnection) {
	rsp := &msg.Protocol{
		Msgid:      msg.MessageID_SitDown_Rsp,
		SitDownRsp: &msg.SitDownRsp{Ret: msg.ErrorID_Ok},
	}

	defer p.userconn.sendProtocol(rsp)

	rspProto := rsp.SitDownRsp
	seatID := p.p.SitDownReq.SeatId

	// 检查 seatID 合法性
	if seatID >= gApp.config.Room.MaxTablePlayers {
		rspProto.Ret = msg.ErrorID_SitDown_Invalid_Seat_Id
		return
	}

	// todo: 检查状态

	player := obj.players[p.userconn.user.uid]
	if player == nil {
		base.LogError("[Room][handleSitDownReq] cannot find this player in the room. uid:", p.userconn.user.uid, ". room:", obj.roomID)
		rspProto.Ret = msg.ErrorID_Internal_Error
		return
	}

	if p := obj.tablePlayers[seatID]; p == player {
		// 自己已经在此座位上了
		rspProto.Ret = msg.ErrorID_SitDown_Already_Sit
		return
	} else if p != nil {
		// 此座位已经有人了
		rspProto.Ret = msg.ErrorID_SitDown_Already_Exist_Player
		return
	}

	// credit points
	if obj.creditPoints > 0 && -obj.scoreboard.GetScore(p.userconn.user.uid) > int32(obj.creditPoints) {
		rspProto.Ret = msg.ErrorID_SitDown_CreditPoints_Out
		return
	}

	var sitDownType msg.SitDownType
	if player.seatID == -1 {
		// 新入座
		sitDownType = msg.SitDownType_Sit
	} else {
		// 换座
		obj.tablePlayers[player.seatID] = nil
		sitDownType = msg.SitDownType_Swap
	}
	oldSeatID := player.seatID
	obj.tablePlayers[seatID] = player
	player.seatID = int32(seatID)
	player.handsNoBet = 0

	if seatID == 0 {
		obj.autoBanker = true
		rspProto.Autobanker = obj.autoBanker
	}

	// 通知房间其他人
	obj.notifyOthers(p.userconn,
		&msg.Protocol{
			Msgid: msg.MessageID_SitDown_Notify,
			SitDownNotify: &msg.SitDownNotify{
				Type:      sitDownType,
				Uid:       p.userconn.user.uid,
				SeatId:    seatID,
				OldSeatId: int32(oldSeatID),
				Score:     obj.scoreboard.GetScore(p.userconn.user.uid),
			}},
	)
}

func (obj *Room) handleStandUpReq(p *ProtocolConnection) {
	rsp := &msg.Protocol{
		Msgid:      msg.MessageID_StandUp_Rsp,
		StandUpRsp: &msg.StandUpRsp{Ret: msg.ErrorID_Ok},
	}
	rspProto := rsp.StandUpRsp

	// 检查状态
	if !obj.round.canStandUp() {
		rspProto.Ret = msg.ErrorID_StandUp_Cannot_Stand_Up
		p.userconn.sendProtocol(rsp)
		return
	}

	player := obj.players[p.userconn.user.uid]
	if player == nil {
		base.LogError("[Room][handleStandUpReq] cannot find this player in the room. uid:", p.userconn.user.uid, ". room:", obj.roomID)
		rspProto.Ret = msg.ErrorID_Internal_Error
		p.userconn.sendProtocol(rsp)
		return
	}

	if player.seatID == -1 {
		rspProto.Ret = msg.ErrorID_StandUp_Not_Sit
		p.userconn.sendProtocol(rsp)
		return
	}

	p.userconn.sendProtocol(rsp)

	// 通知房间其他人
	obj.notifyOthers(p.userconn,
		&msg.Protocol{
			Msgid: msg.MessageID_StandUp_Notify,
			StandUpNotify: &msg.StandUpNotify{
				Uid:    p.userconn.user.uid,
				SeatId: uint32(player.seatID),
				Reason: msg.StandUpReason_Request,
			}},
	)

	needSwitchToReady := false
	if player.seatID == 0 && obj.round.state == msg.GameState_Bet {
		needSwitchToReady = true
	}

	// 此座位置空
	obj.tablePlayers[player.seatID] = nil
	player.seatID = -1

	if needSwitchToReady {
		obj.round.switchGameState(msg.GameState_Ready)
	}
}

func (obj *Room) handleAutoBankerReq(p *ProtocolConnection) {
	rsp := &msg.Protocol{
		Msgid:         msg.MessageID_AutoBanker_Rsp,
		AutoBankerRsp: &msg.AutoBankerRsp{Ret: msg.ErrorID_Ok},
	}

	defer p.userconn.sendProtocol(rsp)

	// 检查状态
	if obj.round.state != msg.GameState_Ready && obj.round.state != msg.GameState_Bet && obj.round.state != msg.GameState_Result {
		rsp.AutoBankerRsp.Ret = msg.ErrorID_AutoBanker_Invalid_State
		return
	}

	// 检查是不是庄家
	if obj.tablePlayers[0] == nil || obj.tablePlayers[0].uid != p.userconn.user.uid {
		rsp.AutoBankerRsp.Ret = msg.ErrorID_AutoBanker_Not_Banker
		return
	}

	obj.autoBanker = p.p.AutoBankerReq.AutoBanker
}

func (obj *Room) handleStartGameReq(p *ProtocolConnection) {
	handle := func() bool {
		rsp := &msg.Protocol{
			Msgid:        msg.MessageID_StartGame_Rsp,
			StartGameRsp: &msg.StartGameRsp{Ret: msg.ErrorID_Ok},
		}

		defer p.userconn.sendProtocol(rsp)

		if obj.round.state != msg.GameState_Ready {
			rsp.StartGameRsp.Ret = msg.ErrorID_StartGame_Not_Ready_State
			return false
		}

		// 检查是不是庄家
		if obj.tablePlayers[0] == nil || obj.tablePlayers[0].uid != p.userconn.user.uid {
			rsp.StartGameRsp.Ret = msg.ErrorID_StartGame_Not_Banker
			return false
		}

		// 检查有几个入座
		uids := make([]uint32, 0, gApp.config.Room.MaxTablePlayers)
		for _, player := range obj.tablePlayers {
			if player != nil {
				uids = append(uids, player.uid)
			}
		}
		num := len(uids)
		if num <= 1 {
			rsp.StartGameRsp.Ret = msg.ErrorID_StartGame_Not_Enough_Players
			return false
		}

		// 扣钻
		// 如果是aa制，开局时平摊
		if obj.isShare {
			totalDiamonds := obj.hands * gApp.config.Room.RoomRate
			diamonds := uint32(math.Ceil(float64(totalDiamonds) / float64(num)))
			if !userManager.consumeUsersDiamonds(uids, diamonds, "start a sharing room") {
				rsp.StartGameRsp.Ret = msg.ErrorID_StartGame_Not_Enough_Diamonds
				return false
			}
		}
		return true
	}
	if handle() {
		obj.round.switchGameState(msg.GameState_Bet)
	}
}

func (obj *Room) handleBetReq(p *ProtocolConnection) {
	req := p.p.BetReq
	rsp := &msg.Protocol{
		Msgid:  msg.MessageID_Bet_Rsp,
		BetRsp: &msg.BetRsp{Ret: msg.ErrorID_Ok},
	}
	rspProto := rsp.BetRsp
	defer p.userconn.sendProtocol(rsp)

	// game state
	if obj.round.state != msg.GameState_Bet && obj.round.state != msg.GameState_Confirm_Bet {
		rspProto.Ret = msg.ErrorID_Bet_Not_Bet_State
		return
	}

	// chips
	if req.Chips < obj.minBet || req.Chips > obj.maxBet {
		rspProto.Ret = msg.ErrorID_Bet_Invalid_Chips
		return
	}

	// 是否入座
	player := obj.players[p.userconn.user.uid]
	if player == nil {
		base.LogError("[Room][handleBetReq] cannot find this player in the room. uid:", p.userconn.user.uid, ". room:", obj.roomID)
		rspProto.Ret = msg.ErrorID_Internal_Error
		return
	}
	if player.seatID < 0 {
		rspProto.Ret = msg.ErrorID_Bet_Not_A_Player_On_Seat
		return
	}
	if player.seatID == 0 {
		rspProto.Ret = msg.ErrorID_Bet_Banker_Cannot_Bet
		return
	}

	// 是否已经下注
	if !obj.round.bet(uint32(player.seatID), req.Chips) {
		rspProto.Ret = msg.ErrorID_Bet_Already_Bet
		return
	}

	// 通知其他人
	obj.notifyOthers(p.userconn,
		&msg.Protocol{
			Msgid: msg.MessageID_Bet_Notify,
			BetNotify: &msg.BetNotify{
				SeatId: uint32(player.seatID),
				Chips:  req.Chips,
			}},
	)

	// 如果全部下注，则进入下一阶段
	if obj.round.isAllBet() {
		if obj.round.state == msg.GameState_Bet && obj.round.stopStateTimeout() {
			obj.round.switchGameState(msg.GameState_Confirm_Bet)
		}
	}
}

func (obj *Room) handleCombineReq(p *ProtocolConnection) {
	req := p.p.CombineReq
	rsp := &msg.Protocol{
		Msgid:      msg.MessageID_Combine_Rsp,
		CombineRsp: &msg.CombineRsp{Ret: msg.ErrorID_Ok},
	}
	rspProto := rsp.CombineRsp
	defer p.userconn.sendProtocol(rsp)

	// game state
	if obj.round.state != msg.GameState_Combine {
		rsp.CombineRsp.Ret = msg.ErrorID_Combine_Not_Combine_State
		return
	}

	// 是否参与本局
	player := obj.players[p.userconn.user.uid]
	if player == nil {
		base.LogError("[Room][handleCombineReq] cannot find this player in the room. uid:", p.userconn.user.uid, ". room:", obj.roomID)
		rspProto.Ret = msg.ErrorID_Internal_Error
		return
	}
	seatID := uint32(player.seatID)
	cards, ok := obj.round.handCards[seatID]
	if !ok {
		rspProto.Ret = msg.ErrorID_Combine_Not_In_This_Round
		return
	}

	// 是否已经提交牌组
	if _, ok := obj.round.result[seatID]; ok {
		rspProto.Ret = msg.ErrorID_Combine_Already_Done
		return
	}

	if !req.Autowin {

		// 验证请求数据
		if len(req.CardGroups) != 3 {
			rspProto.Ret = msg.ErrorID_Combine_Invalid_Request_Data
			return
		}
		for ndx, group := range req.CardGroups {
			if group == nil || ((ndx == 0 && len(group.Cards) > 3) ||
				(ndx != 0 && len(group.Cards) > 5)) {
				rspProto.Ret = msg.ErrorID_Combine_Invalid_Request_Data
				return
			}
		}

		// 验证牌是否为手牌
		autoCombine := false
		for ndx, group := range req.CardGroups {
			// 检查是否组牌完成
			if ndx == 0 && len(group.Cards) < 3 {
				autoCombine = true
				break
			} else if len(group.Cards) < 5 {
				autoCombine = true
				break
			}
		}
		cardsLeft := make(map[uint32]bool)
		if autoCombine {
			for _, v := range cards {
				cardsLeft[v] = true
			}
		}

		f := func(cards []uint32, card uint32) bool {
			for _, c := range cards {
				if c == card {
					return true
				}
			}
			return false
		}
		for _, group := range req.CardGroups {
			for _, c := range group.Cards {
				if !f(cards, c) {
					rspProto.Ret = msg.ErrorID_Combine_Invalid_Request_Data
					return
				}
				if autoCombine {
					delete(cardsLeft, c)
				}
			}
		}

		if autoCombine {
			c := make([]uint32, len(cardsLeft))
			i := 0
			for v := range cardsLeft {
				c[i] = v
				i++
			}
			obj.round.leftCards[seatID] = c
		}
	} else {
		// 检查是否满足autowin条件
		cards := make([]uint32, 0, 5)
		_, rank, ok := findCardRank(obj.round.handCards[seatID], cards, 5)
		if !ok || rank < msg.CardRank_Four_Of_A_Kind {
			rspProto.Ret = msg.ErrorID_Combine_Not_Lucky
			return
		}
		obj.round.leftCards[seatID] = obj.round.handCards[seatID]
	}

	obj.round.result[seatID] = &msg.SeatResult{
		SeatId:     seatID,
		CardGroups: req.CardGroups,
		Autowin:    req.Autowin,
		Uid:        player.uid,
	}

	// 通知其他人
	obj.notifyOthers(p.userconn,
		&msg.Protocol{
			Msgid: msg.MessageID_Combine_Notify,
			CombineNotify: &msg.CombineNotify{
				SeatId: uint32(player.seatID),
			}},
	)

	// 如果全部完成组牌，则进入下一阶段
	if obj.round.isAllCombine() {
		if obj.round.stopStateTimeout() {
			obj.round.switchGameState(msg.GameState_Show)
		}
	}
}

func (obj *Room) handleGetScoreboardReq(p *ProtocolConnection) {
	req := p.p.GetScoreboardReq
	rsp := &msg.Protocol{
		Msgid:            msg.MessageID_GetScoreboard_Rsp,
		GetScoreboardRsp: &msg.GetScoreboardRsp{Ret: msg.ErrorID_Ok},
	}
	rspProto := rsp.GetScoreboardRsp
	defer p.userconn.sendProtocol(rsp)

	l := uint32(len(obj.scoreboard.Uids))
	if req.Pos > l {
		rspProto.Ret = msg.ErrorID_GetScoreboard_Pos_Exceed_Range
		return
	}
	rspProto.Total = l
	num := uint32(math.Min(float64(l-req.Pos), float64(gApp.config.Room.ScoreboardCountPerTime)))
	end := req.Pos + num
	rspProto.Items = make([]*msg.ScoreboardItem, num)
	for i := req.Pos; i < end; i++ {
		// rspProto.Items[i-req.Pos] = &msg.ScoreboardItem{
		// 	Uid:   obj.scoreboard.Uids[i],
		// 	Score: obj.scoreboard.List[obj.scoreboard.Uids[i]].Score,
		// }
		rspProto.Items[i-req.Pos] = obj.scoreboard.List[obj.scoreboard.Uids[i]]
	}
}

func (obj *Room) getTablePlayersCount() int {
	ret := 0
	for _, player := range obj.tablePlayers {
		if player != nil {
			ret++
		}
	}
	return ret
}

func (obj *Room) updateScoreboard(seatID uint32, score int32) {
	player := obj.tablePlayers[seatID]
	obj.scoreboard.Update(player.uid, player.name, player.avatar, score)
}

func (obj *Room) handleGetRoundHistoryReq(p *ProtocolConnection) {
	req := p.p.GetRoundHistoryReq
	rsp := &msg.Protocol{
		Msgid:              msg.MessageID_GetRoundHistory_Rsp,
		GetRoundHistoryRsp: &msg.GetRoundHistoryRsp{Ret: msg.ErrorID_Ok},
	}
	rspProto := rsp.GetRoundHistoryRsp
	defer p.userconn.sendProtocol(rsp)

	data, err := db.GetRoundResult(obj.roomID, req.Round)
	if err != nil {
		rspProto.Ret = msg.ErrorID_GetRoundHistory_Round_Not_Exist
		return
	}

	results := &msg.DBResults{}
	if err := proto.Unmarshal([]byte(data), results); err != nil {
		rspProto.Ret = msg.ErrorID_Internal_Error
		base.LogError("Failed to unmarshall data. error:", err, ", roomid:", obj.roomID, ", round:", req.Round)
		return
	}

	rspProto.Results = make([]*msg.PlayerRoundHistory, len(results.Results))
	i := 0
	for _, result := range results.Results {
		r, err := db.GetUsersNameAvatar([]uint32{result.Uid})
		if err != nil {
			rspProto.Ret = msg.ErrorID_DB_Error
			base.LogError("Failed to get username and avatar. error:", err, ", roomid:", obj.roomID, ", uid:", result.Uid)
			return
		}
		rspProto.Results[i] = &msg.PlayerRoundHistory{
			Uid:    result.Uid,
			Name:   r[0].Name,
			Avatar: r[0].Avatar,
			Result: result,
		}
		i++
	}
}

func (obj *Room) kickPlayer(uid uint32) {
	player, ok := obj.players[uid]
	if !ok {
		return
	}

	if player.seatID >= 0 {
		obj.tablePlayers[player.seatID] = nil
	}

	userManager.leaveRoom(player.uid, obj)
	delete(obj.players, player.uid)
}
