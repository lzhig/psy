package main

import (
	"math"
	"time"

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
	obj.room.playedHands++
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
			// 如果没有人下注，则流局
			if len(obj.betChips) == 0 {
				obj.switchGameState(msg.GameState_Ready)
			} else {
				obj.switchGameState(msg.GameState_Confirm_Bet)
			}
		case msg.GameState_Confirm_Bet:
			obj.switchGameState(msg.GameState_Deal)
		case msg.GameState_Deal:
			obj.switchGameState(msg.GameState_Combine)
		case msg.GameState_Combine:
			obj.switchGameState(msg.GameState_Show)
		case msg.GameState_Show:
			obj.switchGameState(msg.GameState_Result)
		case msg.GameState_Result:
			// 如果是自动连庄
			if obj.room.playedHands < obj.room.hands {
				if obj.room.autoBanker {
					obj.switchGameState(msg.GameState_Bet)
				} else {
					// 庄家站起
					obj.switchGameState(msg.GameState_Ready)
				}
			} else {
				// todo: 如果局数已满
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
		obj.room.notifyAll(notify)
	case msg.GameState_Bet, msg.GameState_Combine:
		obj.room.notifyAll(notify)

	case msg.GameState_Confirm_Bet:
		// 确定参与玩家
		for _, player := range obj.room.tablePlayers {
			if player == nil || player.seatID < 0 {
				continue
			}
			seatID := uint32(player.seatID)
			if seatID == 0 {
				obj.players[seatID] = player
				continue
			}

			if _, ok := obj.betChips[seatID]; ok {
				obj.players[seatID] = player
			}
		}
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

		notify.GameStateNotify.Result = make([]*msg.SeatResult, len(obj.handCards))
		i := 0
		for _, v := range obj.result {
			notify.GameStateNotify.Result[i] = v
			i++
		}
		base.LogInfo(notify)
		obj.room.notifyAll(notify)
	case msg.GameState_Result:
		obj.room.notifyAll(notify)
	}

	if notify.GameStateNotify.Countdown > 0 {
		obj.stateTimeout = time.AfterFunc(time.Duration(notify.GameStateNotify.Countdown)*time.Millisecond,
			func() {
				obj.room.eventChan <- &roomEvent{
					event: roomEventGameStateTimeout,
					args:  []interface{}{state},
				}
			})
	}
}

func (obj *Round) deal() {
	base.LogInfo("[Round][deal]")
	cards := dealer.deal()

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
	for seatID := range obj.players {
		if _, ok := obj.result[seatID]; !ok {
			obj.result[seatID] = &msg.SeatResult{
				SeatId: seatID,
				CardGroups: []*msg.CardGroup{
					&msg.CardGroup{Cards: make([]uint32, 0, 3)},
					&msg.CardGroup{Cards: make([]uint32, 0, 5)},
					&msg.CardGroup{Cards: make([]uint32, 0, 5)},
				},
				Autowin: false,
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
			if ndx == 0 && len(group.Cards) < 3 {
				autoCombine = true
			} else if ndx != 0 && len(group.Cards) < 5 {
				autoCombine = true
				num = 5
			}
			if autoCombine {
				formCards, rank, ok := findCardRank(leftCards, group.Cards, num)
				base.LogInfo(formCards, rank, ok)
				if !ok {
					base.LogError("[Round][calculateResult]failed to find card rank. left cards=", obj.leftCards[result.SeatId], ",init cards=", group.Cards)
					rank = msg.CardRank_High_Card
				} else {
					// 移去用掉的牌
					pos := 0
					for _, v := range formCards {
						for ndx, c := range leftCards {
							if v == c {
								leftCards[pos], leftCards[ndx] = leftCards[ndx], leftCards[pos]
								pos++
							}
						}
					}
					leftCards = leftCards[pos:]
					group.Cards = formCards
				}
				result.Ranks[ndx] = rank
				continue
			}

			result.Ranks[ndx] = calculateCardRank(group.Cards)
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
						if result.CardGroups[i].Cards[j]%13 > result.CardGroups[i+1].Cards[j]%13 {
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
			result.Scores, result.TotalScore = obj.compareCardGroup(result, bankerResult)
		}

		// 计算输赢积分
		result.Bet = obj.betChips[result.SeatId]
		result.Win = result.TotalScore * int32(obj.betChips[result.SeatId])
		bankerWin += result.Win
	}
	obj.result[0].Win = -bankerWin
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

				// 判断A 2 3 4 5
				if a.Ranks[i] == msg.CardRank_Straight_Flush || a.Ranks[i] == msg.CardRank_Straight {
					if a.CardGroups[i].Cards[0] == Card_A_Value && a.CardGroups[i].Cards[4] == Card_2_Value {

					}
				}
			}
			base.LogInfo(a.CardGroups, b.CardGroups)
			for j := 0; j < num; j++ {
				base.LogInfo(a.CardGroups[i], b.CardGroups[i])
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
