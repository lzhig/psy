package main

import (
	"math"
	"time"

	"github.com/golang/protobuf/proto"

	"./msg"
	"github.com/lzhig/rapidgo/base"
)

// Round 表示一局
type Round struct {
	room           *Room
	state          msg.GameState          // 游戏状态
	stateBeginTime time.Time              // 状态开始时间
	players        map[uint32]*RoomPlayer // 本局参与的玩家
	handCards      map[uint32][]uint32    // 各个座位的发牌
	//leftCards      map[uint32][]uint32        // 各个座位组牌剩下的牌
	betChips    map[uint32]uint32          // 各个座位的下注
	result      map[uint32]*msg.SeatResult // 各个座位的结算
	closeResult map[uint32]bool            // 各个座位是否已经关闭结算

	stateTimeout *time.Timer
}

// Init 初始化
func (obj *Round) Init(room *Room) {
	obj.room = room
	obj.state = msg.GameState_Ready

	obj.Begin()
}

// Begin 一局开始
func (obj *Round) Begin() {
	obj.players = make(map[uint32]*RoomPlayer)
	obj.betChips = make(map[uint32]uint32)
	obj.handCards = make(map[uint32][]uint32)
	obj.result = make(map[uint32]*msg.SeatResult)
	obj.closeResult = make(map[uint32]bool)
}

func (obj *Round) bet(seatID uint32, chips uint32) bool {
	_, alreadBet := obj.betChips[seatID]
	if alreadBet {
		return false
	}
	obj.betChips[seatID] = chips
	return true
}

func (obj *Round) unbet(seatID uint32) {
	delete(obj.betChips, seatID)
}

func (obj *Round) isAllBet() bool {
	//return len(obj.betChips) == int(gApp.config.Room.MaxTablePlayers)
	return len(obj.betChips) == obj.room.getTablePlayersCount()-1
}

func (obj *Round) isAllCombine() bool {
	return len(obj.result) == len(obj.handCards)
}

func (obj *Round) CloseResult(seatID uint32) {
	obj.closeResult[seatID] = true
}

func (obj *Round) isAllCloseResult() bool {
	return len(obj.closeResult) == len(obj.handCards)
}

// HandleTimeout function
func (obj *Round) HandleTimeout(state msg.GameState) {
	if obj.state == state {
		switch state {
		case msg.GameState_Bet:
			obj.switchGameState(msg.GameState_Confirm_Bet)

		case msg.GameState_Confirm_Bet:
			// 如果没有人下注，则流局
			if len(obj.betChips) == 0 {
				if obj.room.closed {
					obj.switchGameState(msg.GameState_CloseRoom)
					return
				}

				obj.switchGameState(msg.GameState_Ready)
				return
			}

			// 确定参与玩家
			hasBanker := false
			for _, player := range obj.room.tablePlayers {
				if player == nil || player.seatID < 0 {
					continue
				}
				seatID := uint32(player.seatID)
				if seatID == 0 {
					hasBanker = true
					obj.players[seatID] = player
					continue
				}

				if _, ok := obj.betChips[seatID]; ok {
					obj.players[seatID] = player
					player.handsNoBet = 0
				} else {
					player.handsNoBet++
				}
			}
			if !hasBanker {
				obj.switchGameState(msg.GameState_Ready)
			} else {
				obj.switchGameState(msg.GameState_Deal)
			}

		case msg.GameState_Deal:
			obj.switchGameState(msg.GameState_Combine)
		case msg.GameState_Combine:
			obj.switchGameState(msg.GameState_Confirm_Combine)
		case msg.GameState_Confirm_Combine:
			obj.switchGameState(msg.GameState_Show)
		case msg.GameState_Show:
			obj.switchGameState(msg.GameState_Result)
		case msg.GameState_Result:
			if obj.room.playedHands < obj.room.hands-1 {

				// check if closed
				if obj.room.closed {
					obj.switchGameState(msg.GameState_CloseRoom)
					return
				}

				// check credit points
				if obj.room.creditPoints > 0 {
					for seatID, player := range obj.room.tablePlayers {
						if player != nil && -obj.room.scoreboard.GetScore(player.uid) > int32(obj.room.creditPoints) {
							obj.room.tablePlayers[seatID] = nil
							player.seatID = -1

							obj.room.notifyAll(
								&msg.Protocol{
									Msgid: msg.MessageID_StandUp_Notify,
									StandUpNotify: &msg.StandUpNotify{
										Uid:    player.uid,
										SeatId: uint32(seatID),
										Reason: msg.StandUpReason_CreditPointsOut,
									}},
							)
						}
					}
				}

				// check no bet for 3 hands
				for seatID, player := range obj.room.tablePlayers {
					if player == nil || seatID == 0 || player.handsNoBet < gApp.config.Room.KickNoBetForHands {
						continue
					}
					obj.room.tablePlayers[seatID] = nil
					player.seatID = -1

					obj.room.notifyAll(
						&msg.Protocol{
							Msgid: msg.MessageID_StandUp_Notify,
							StandUpNotify: &msg.StandUpNotify{
								Uid:    player.uid,
								SeatId: uint32(seatID),
								Reason: msg.StandUpReason_NoActionFor3Hands,
							}},
					)
				}

				// check disconnected players
				for _, player := range obj.room.tablePlayers {
					if player == nil {
						continue
					}

					if player.conn == nil {
						obj.room.kickPlayer(player.uid)
					}
				}

				// check autobanker
				//base.LogInfo("obj.room.autoBanker=", obj.room.autoBanker)
				if obj.room.autoBanker {
					// 如果是自动连庄
					obj.Begin()

					// 检查座位上有庄家并且有其他玩家
					hasBanker := false
					hasOthers := false
					for _, player := range obj.room.tablePlayers {
						if player != nil {
							if player.seatID == 0 {
								hasBanker = true
							} else {
								hasOthers = true
							}

							if hasBanker && hasOthers {
								break
							}
						}
					}

					if hasBanker && hasOthers {
						// 局数+1
						obj.room.nextRound()
						obj.switchGameState(msg.GameState_Bet)
					} else {
						// 局数+1
						obj.room.nextRound()
						obj.switchGameState(msg.GameState_Ready)
					}
				} else {
					// 庄家站起
					player := obj.room.tablePlayers[0]
					obj.room.tablePlayers[0] = nil
					player.seatID = -1

					obj.room.notifyAll(
						&msg.Protocol{
							Msgid: msg.MessageID_StandUp_Notify,
							StandUpNotify: &msg.StandUpNotify{
								Uid:    player.uid,
								SeatId: uint32(0),
								Reason: msg.StandUpReason_BankerNotAutoBanker,
							}},
					)
					// 局数+1
					obj.room.nextRound()
					obj.switchGameState(msg.GameState_Ready)
				}
			} else {
				// 局数已满
				obj.switchGameState(msg.GameState_CloseRoom)
			}
		}
	}
}

func (obj *Round) switchGameState(state msg.GameState) {
	//base.LogInfo("switchGameState", state, ", room_id:", obj.room.roomID, ", players:", len(obj.room.players))
	obj.state = state
	obj.stateBeginTime = time.Now()
	notify := &msg.Protocol{
		Msgid: msg.MessageID_GameState_Notify,
		GameStateNotify: &msg.GameStateNotify{
			State:     state,
			Countdown: gApp.config.Room.StatesCountdown[state],
		}}
	switch state {
	case msg.GameState_Ready:
		obj.Begin()

		// check disconnected players
		for _, player := range obj.room.tablePlayers {
			if player == nil {
				continue
			}

			if player.conn == nil {
				obj.room.kickPlayer(player.uid)
			}
		}

		obj.room.beginReleaseTimer()

		// 流局不再增加局数
		//obj.room.nextRound()
		notify.GameStateNotify.PlayedHands = obj.room.playedHands

		obj.room.notifyAll(notify)

	case msg.GameState_Bet:
		notify.GameStateNotify.PlayedHands = obj.room.playedHands
		obj.room.notifyAll(notify)

	case msg.GameState_Combine, msg.GameState_Confirm_Combine:
		obj.room.notifyAll(notify)

	case msg.GameState_Confirm_Bet:
		obj.room.notifyAll(notify)

	case msg.GameState_Deal:
		// 对每个参与本轮游戏的玩家发牌
		obj.deal()

		notify.GameStateNotify.DealSeats = make([]uint32, len(obj.players))
		i := 0
		for seatID := range obj.players {
			notify.GameStateNotify.DealSeats[i] = seatID
			i++
		}

		for _, player := range obj.room.players {
			if obj.isJoinPlayer(player.seatID) {
				notify.GameStateNotify.DealCards = make([]uint32, gApp.config.Room.DealCardsNum)
				for ndx, v := range obj.handCards[uint32(player.seatID)] {
					notify.GameStateNotify.DealCards[ndx] = uint32(v)
				}
			} else {
				notify.GameStateNotify.DealCards = nil
			}
			player.sendProtocol(notify)
		}

	case msg.GameState_Show:
		// 结算
		obj.calculateResult()

		resultsToDB := &msg.DBResults{Results: make([]*msg.SeatResult, len(obj.result))}

		notify.GameStateNotify.Result = make([]*msg.SeatResult, len(obj.handCards))
		i := 0
		for _, v := range obj.result {
			notify.GameStateNotify.Result[i] = v
			resultsToDB.Results[i] = v
			i++
		}
		//base.LogInfo(notify)
		obj.room.notifyAll(notify)

		// save to db
		go func() {
			buf, err := proto.Marshal(resultsToDB)
			if err != nil {
				base.LogError("Failed to marshal results. reason:", err)
				return
			}

			if err := db.SaveRoundResult(obj.room.roomID, obj.room.playedHands, string(buf)); err != nil {
				base.LogError("Failed to save results to db. reason:", err, "data:\n", buf)
			}
		}()
	case msg.GameState_Result:
		obj.room.notifyAll(notify)

		if obj.room.playedHands >= obj.room.hands-1 {
			// 先关闭房间
			obj.room.closed = true

			// update db
			if err := db.CloseRoom(obj.room.roomID, time.Now().Unix()); err != nil {
				base.LogError("Failed to close room. error:", err)
			}

			// kick all players
			for _, player := range obj.room.players {
				if player.seatID >= 0 {
					obj.room.tablePlayers[player.seatID] = nil
				}
				userManager.LeaveRoom(player.uid)
				delete(obj.room.players, player.uid)
			}
		}

	case msg.GameState_CloseRoom:
		obj.room.closed = true

		// 关闭房间
		roomManager.Send(roomManagerEventCloseRoom, []interface{}{obj.room})
	}

	if notify.GameStateNotify.Countdown > 0 {
		obj.stateTimeout = time.AfterFunc(time.Duration(notify.GameStateNotify.Countdown)*time.Millisecond,
			func() {
				obj.room.Send(roomEventGameStateTimeout, []interface{}{state})
				// obj.room.eventChan <- &roomEvent{
				// 	event: roomEventGameStateTimeout,
				// 	args:  []interface{}{state},
				// }
			})
	}
}

func (obj *Round) deal() {
	//base.LogInfo("[Round][deal]")
	cards := dealer.deal()
	//cards := []uint32{0, 1, 2, 3, 12, 5, 6, 7, 8, 9, 24, 13, 43, 47, 26, 27, 28, 29, 38, 31, 32, 33, 34, 35, 50, 39}
	//cards := []uint32{12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 26, 25, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50, 51}

	obj.handCards[0] = cards[0:gApp.config.Room.DealCardsNum]
	base.LogInfo("seat 0:", obj.handCards[0])
	i := uint32(1)
	for seatID := range obj.betChips {
		obj.handCards[seatID] = cards[i*gApp.config.Room.DealCardsNum : (i+1)*gApp.config.Room.DealCardsNum]
		base.LogInfo("seat ", seatID, ":", obj.handCards[seatID])
		i++
	}
}

func (obj *Round) canStandUp() bool {
	return obj.state == msg.GameState_Ready || obj.state == msg.GameState_Bet
}

func (obj *Round) stopStateTimeout() bool {
	if !obj.stateTimeout.Stop() {
		select {
		case <-obj.stateTimeout.C:
		default:
		}
		return false
	}
	return true
}

func (obj *Round) isJoinPlayer(seatID int32) bool {
	if seatID < 0 {
		return false
	}

	if _, ok := obj.players[uint32(seatID)]; ok {
		return true
	}
	return false
}

func (obj *Round) autoCombine(already []*msg.CardGroup, leftCards []uint32) ([]*msg.CardGroup, []msg.CardRank) {
	finalGroups := make([]*msg.CardGroup, 3)
	ranks := make([]msg.CardRank, 3)
	for ndx := 2; ndx >= 0; ndx-- {
		num := 5
		if ndx == 0 {
			num = 3
		}
		var groupCards []uint32
		if already != nil && len(already) > ndx {
			groupCards = already[ndx].Cards
		}

		formCards, rank, ok := findCardRank(leftCards, groupCards, num)
		if !ok {
			base.LogError("failed to find card rank. left cards=", leftCards, ", groupCards=", groupCards, ", num:", num, ", ndx:", ndx, ", room_id:", obj.room.roomID)
		} else {
			// 移去用过的牌
			pos := 0
			for _, v := range formCards {
				for i, c := range leftCards {
					if v == c {
						leftCards[pos], leftCards[i] = leftCards[i], leftCards[pos]
						pos++
					}
				}
			}
			leftCards = leftCards[pos:]
		}
		ranks[ndx] = rank
		finalGroups[ndx] = &msg.CardGroup{Cards: formCards}
	}
	return finalGroups, ranks
}

func (obj *Round) calculateResult() {
	// 检查每个参与本局的玩家是否有提交组牌
	for seatID, player := range obj.players {
		if _, ok := obj.result[seatID]; !ok {
			cardGroups, ranks := obj.autoCombine(nil, obj.handCards[seatID])
			obj.result[seatID] = &msg.SeatResult{
				SeatId:     seatID,
				CardGroups: cardGroups,
				Ranks:      ranks,
				Autowin:    false,
				Uid:        player.uid,
			}
		}
	}

	bankerResult := obj.result[0]

	// 当前没有检测牌型乌龙
	for _, result := range obj.result {
		if result.Autowin {
			continue
		}

		calcFoul := func(result *msg.SeatResult) {
			for i := 0; i < 2; i++ {
				if result.Ranks[i] > result.Ranks[i+1] {
					result.Foul = true
					return
				} else if result.Ranks[i] == result.Ranks[i+1] {
					// 如果牌型一样，比较牌值
					n := int(math.Min(float64(len(result.CardGroups[i].Cards)), float64(len(result.CardGroups[i+1].Cards))))
					for j := 0; j < n; j++ {
						a := result.CardGroups[i].Cards[j] % 13
						b := result.CardGroups[i+1].Cards[j] % 13
						if a < b {
							return
						} else if a > b {
							result.Foul = true
							return
						}
					}
				}
			}
		}

		calcFoul(result)
	}

	bankerWin := int32(0)
	// 计算得分, 分别与庄家比较
	for _, result := range obj.result {
		if result.SeatId == 0 {
			continue
		}

		// autowin
		if bankerResult.Autowin {
			if !result.Autowin {
				result.TotalScore = -2
			}
		} else if result.Autowin {
			result.TotalScore = 2
		} else if bankerResult.Foul {
			if !result.Foul {
				result.TotalScore = 2
			}
		} else if result.Foul {
			result.TotalScore = -2
		} else {
			var score int32
			result.Scores, score = obj.compareCardGroup(result, bankerResult)
			if score == 3 {
				if result.Ranks[0] >= msg.CardRank_Four_Of_A_Kind {
					result.TotalScore = 3
				} else {
					result.TotalScore = 2
				}
			} else if score == 1 {
				result.TotalScore = 1
			} else if score == -1 {
				result.TotalScore = -1
			} else if score == -3 {
				result.TotalScore = -2
			} else {
				base.LogError("Invalid score")
			}
		}

		// 计算输赢积分
		result.Bet = obj.betChips[result.SeatId]
		result.Win = result.TotalScore * int32(obj.betChips[result.SeatId])
		bankerWin += result.Win

		obj.room.updateScoreboard(result.SeatId, result.Win)
	}
	obj.result[0].Win = -bankerWin
	obj.room.updateScoreboard(0, -bankerWin)
}

func (obj *Round) compareCardGroup(a, b *msg.SeatResult) ([]int32, int32) {
	score := make([]int32, 3)
	totalScore := int32(0)
	s := int32(0)
	for i := 0; i < 3; i++ {
		if a.Ranks[i] > b.Ranks[i] {
			s = 1
		} else if a.Ranks[i] < b.Ranks[i] {
			s = -1
		} else {
			// 如果牌型一样，再比较单个牌
			num := 3
			if i != 0 {
				num = 5
			}
			//base.LogInfo(a.CardGroups[i], b.CardGroups[i])
			for j := 0; j < num; j++ {
				v1 := a.CardGroups[i].Cards[j] % 13
				v2 := b.CardGroups[i].Cards[j] % 13
				if v1 > v2 {
					s = 1
					break
				} else if v1 < v2 {
					s = -1
					break
				}
			}
			// 如果大小一样，比较花色
			if s == 0 {
				for j := 0; j < num; j++ {
					c1 := a.CardGroups[i].Cards[j] / 13
					c2 := b.CardGroups[i].Cards[j] / 13
					if c1 > c2 {
						s = 1
						break
					} else {
						s = -1
						break
					}
				}
			}
		}

		score[i] = s
		totalScore += s
	}
	return score, totalScore
}
