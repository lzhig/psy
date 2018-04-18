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
	state          msg.GameState              // 游戏状态
	stateBeginTime time.Time                  // 状态开始时间
	players        map[uint32]*RoomPlayer     // 本局参与的玩家
	handCards      map[uint32][]uint32        // 各个座位的发牌
	leftCards      map[uint32][]uint32        // 各个座位组牌剩下的牌
	betChips       map[uint32]uint32          // 各个座位的下注
	result         map[uint32]*msg.SeatResult // 各个座位的结算

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
	obj.leftCards = make(map[uint32][]uint32)
	obj.result = make(map[uint32]*msg.SeatResult)
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
				base.LogInfo("obj.room.autoBanker=", obj.room.autoBanker)
				if obj.room.autoBanker {
					// 如果是自动连庄
					obj.Begin()

					// 局数+1
					obj.room.nextRound()

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
						obj.switchGameState(msg.GameState_Bet)
					} else {
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
	base.LogInfo("switchGameState", state)
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

		// 流局不再增加局数
		//obj.room.nextRound()

		obj.room.notifyAll(notify)

	case msg.GameState_Bet, msg.GameState_Combine:
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
		base.LogInfo(notify)
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
		// 先关闭房间
		if obj.room.playedHands >= obj.room.hands-1 {
			obj.room.closed = true
			// update db
			if err := db.CloseRoom(obj.room.roomID, time.Now().Unix()); err != nil {
				base.LogError("Fail to close room. error:", err)
			}
		}
		obj.room.notifyAll(notify)

	case msg.GameState_CloseRoom:
		obj.room.closed = true
		obj.room.notifyAll(notify)

		// kick all players
		for _, player := range obj.room.players {
			//delete(obj.room.players, player.uid)
			if player.seatID >= 0 {
				obj.room.tablePlayers[player.seatID] = nil
			}
			userManager.leaveRoom(player.uid, obj.room)
			player.conn.user.room = nil
		}
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
	base.LogInfo("[Round][deal]")
	cards := dealer.deal()
	//cards := []uint32{0, 1, 2, 3, 12, 5, 6, 7, 8, 9, 24, 13, 43, 47, 26, 27, 28, 29, 38, 31, 32, 33, 34, 35, 50, 39}

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

func (obj *Round) calculateResult() {
	// 检查每个参与本局的玩家是否有提交组牌
	for seatID, player := range obj.players {
		if _, ok := obj.result[seatID]; !ok {
			obj.result[seatID] = &msg.SeatResult{
				SeatId: seatID,
				CardGroups: []*msg.CardGroup{
					&msg.CardGroup{Cards: make([]uint32, 0, 3)},
					&msg.CardGroup{Cards: make([]uint32, 0, 5)},
					&msg.CardGroup{Cards: make([]uint32, 0, 5)},
				},
				Autowin: false,
				Uid:     player.uid,
			}
			// leftCards
			obj.leftCards[seatID] = make([]uint32, len(obj.handCards[seatID]))
			copy(obj.leftCards[seatID], obj.handCards[seatID])
		}
	}

	// 计算牌型
	var bankerResult *msg.SeatResult
	for _, result := range obj.result {
		if result.SeatId == 0 {
			bankerResult = result
		}

		// 有lucky时，需要计算组牌
		// if result.Autowin {
		// 	continue
		// }

		result.Ranks = make([]msg.CardRank, 3)
		leftCards := obj.leftCards[result.SeatId]
		base.LogInfo("leftCards:", leftCards)
		for ndx := 2; ndx >= 0; ndx-- {
			group := result.CardGroups[ndx]
			base.LogInfo(ndx, group.Cards)

			// 自动组牌
			autoCombine := false
			num := 3
			if ndx != 0 {
				num = 5
			}
			if ndx == 0 && len(group.Cards) < 3 {
				autoCombine = true
			} else if ndx != 0 && len(group.Cards) < 5 {
				autoCombine = true
			}
			if autoCombine {
				formCards, rank, ok := findCardRank(leftCards, group.Cards, num)
				base.LogInfo(formCards, rank, ok)
				if !ok {
					base.LogError("failed to find card rank. left cards=", obj.leftCards[result.SeatId], ",init cards=", group.Cards)
					rank = msg.CardRank_High_Card
				} else {
					// 移去用掉的牌
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
					group.Cards = formCards
				}
				result.Ranks[ndx] = rank
			} else {
				//result.Ranks[ndx] = calculateCardRank(group.Cards)
				formCards, rank, ok := findCardRank(nil, group.Cards, num)
				base.LogInfo(formCards, rank, ok)
				if !ok {
					base.LogError("failed to find card rank. group.Cards=", group.Cards)
					rank = msg.CardRank_High_Card
				}
				result.Ranks[ndx] = rank
				group.Cards = formCards
			}
		}
	}

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
			base.LogInfo(a.CardGroups[i], b.CardGroups[i])
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
