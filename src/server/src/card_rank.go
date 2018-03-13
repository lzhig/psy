package main

import (
	"sort"

	"./msg"
)

const (
	Card_A_Value = 12
	Card_2_Value = 0
)

// Card type
type Card struct {
	value uint32
	color msg.CardColorSuit
}

type matchFunc func([]Card, []int, int, bool) bool

func calculateCardRank(cards []uint32) msg.CardRank {
	num := len(cards)
	c := make([]Card, num)
	value := make([]int, num)
	for ndx, v := range cards {
		c[ndx].value = v % 13
		c[ndx].color = msg.CardColorSuit(v / 13)
		value[ndx] = ndx
	}
	sortCards(c, value)
	// sort.Slice(value,
	// 	func(i, j int) bool {
	// 		if c[value[i]].value == c[value[j]].value {
	// 			return c[value[i]].color > c[value[j]].color
	// 		}
	// 		return c[value[i]].value > c[value[j]].value
	// 	},
	// )

	// 统计相同的个数,key为牌面大小
	sameCards := make(map[uint32]*struct {
		count uint32 // 个数
		cards []int  // 对应的索引
	})

	for _, ndx := range value {
		v := c[ndx].value
		if s, ok := sameCards[v]; ok {
			s.cards = append(s.cards, ndx)
			s.count++
		} else {
			sameCards[v] = &struct {
				count uint32
				cards []int
			}{count: 1, cards: make([]int, 0, 5)}
			sameCards[v].cards = append(sameCards[v].cards, ndx)
		}
	}
	// 按个数从多到少排序,
	if len(sameCards) < 5 {
		s := make([]uint32, len(sameCards))
		ndx := 0
		for k := range sameCards {
			s[ndx] = k
			ndx++
		}

		sort.Slice(s, func(i, j int) bool { return sameCards[s[i]].count > sameCards[s[j]].count })
		defer func() {
			i := 0
			for _, k := range s {
				for _, ndx := range sameCards[k].cards {
					cards[i] = c[ndx].value + uint32(c[ndx].color)*13
					i++
				}
			}
		}()

		if num == 3 {
			if sameCards[s[0]].count == 3 {
				return msg.CardRank_Three_Of_A_Kind
			}
			if sameCards[s[0]].count == 2 {
				return msg.CardRank_One_Pair
			}

			return msg.CardRank_High_Card
		}

		if sameCards[s[0]].count == 4 {
			return msg.CardRank_Four_Of_A_Kind
		}
		if sameCards[s[0]].count == 3 {
			if sameCards[s[1]].count == 2 {
				return msg.CardRank_Full_House
			}
			return msg.CardRank_Three_Of_A_Kind
		}
		if sameCards[s[0]].count == 2 {
			if sameCards[s[1]].count == 2 {
				return msg.CardRank_Two_Pair
			}
			return msg.CardRank_One_Pair
		}
	}

	defer func() {
		i := 0
		for _, k := range value {
			cards[i] = c[k].value + uint32(c[k].color)*13
			i++
		}
	}()

	if calculateStraightFlush(c, value, num) {
		return msg.CardRank_Straight_Flush
	}
	if calculateFlush(c, value, num) {
		return msg.CardRank_Flush
	}
	if calculateStraight(c, value, num) {
		return msg.CardRank_Straight
	}
	return msg.CardRank_High_Card
}

func sortCards(cards []Card, value []int) {
	sort.Slice(value,
		func(i, j int) bool {
			if cards[value[i]].value == cards[value[j]].value {
				return cards[value[i]].color > cards[value[j]].color
			}
			return cards[value[i]].value > cards[value[j]].value
		})
}

func calculateStraightFlush(cards []Card, value []int, num int) bool {
	for i := 0; i < num-1; i++ {
		if cards[value[i]].value != cards[value[i+1]].value+1 || cards[value[i]].color != cards[value[i+1]].color {
			return false
		}
	}

	return true
}

// func calculateFourOfAKind(cards []Card, value []int, num int) bool {
// 	diff := 0
// 	for i := 0; i < num-1; i++ {
// 		if cards[value[i]].value != cards[value[i+1]].value {
// 			diff++
// 			if diff > 1 {
// 				return false
// 			}
// 		}
// 	}

// 	return true
// }

func calculateFlush(cards []Card, value []int, num int) bool {
	for i := 0; i < num-1; i++ {
		if cards[value[i]].color != cards[value[i+1]].color {
			return false
		}
	}

	return true
}

func calculateStraight(cards []Card, value []int, num int) bool {
	for i := 0; i < num-1; i++ {
		if cards[value[i]].value != cards[value[i+1]].value+1 {
			return false
		}
	}

	return true
}

///////////////////////////////////////////////////////////////////////////////

func findCardRank(cards, form []uint32, n int) ([]uint32, msg.CardRank, bool) {
	num := len(cards)
	if num+len(form) < n {
		return nil, msg.CardRank_High_Card, false
	}

	c := make([]Card, num)
	value := make([]int, num)
	for ndx, v := range cards {
		c[ndx].value = v % 13
		c[ndx].color = msg.CardColorSuit(v / 13)
		value[ndx] = ndx
	}
	sortCards(c, value)

	f := make([]Card, 0, n)
	fv := make([]int, 0, n)
	for ndx, v := range form {
		f = append(f, Card{value: v % 13, color: msg.CardColorSuit(v / 13)})
		fv = append(fv, ndx)
	}
	sortCards(f, fv)

	h := []struct {
		rank  msg.CardRank
		match matchFunc
	}{
		{rank: msg.CardRank_Straight_Flush, match: matchStraightFlush},
		{rank: msg.CardRank_Four_Of_A_Kind, match: matchFourOfAKind},
		{rank: msg.CardRank_Full_House, match: matchFullHouse},
		{rank: msg.CardRank_Flush, match: matchFlush},
		{rank: msg.CardRank_Straight, match: matchStraight},
		{rank: msg.CardRank_Three_Of_A_Kind, match: matchThreeOfAKind},
		{rank: msg.CardRank_Two_Pair, match: matchTwoPair},
		{rank: msg.CardRank_One_Pair, match: matchOnePair},
	}
	sort := true
	if len(fv) == 0 {
		sort = false
	}

	retFunc := func(cards []Card, value []int, n int) []uint32 {
		r := make([]uint32, n)
		i := 0
		for _, v := range value {
			r[i] = cards[v].value + uint32(cards[v].color)*13
			i++
			if i == n {
				break
			}
		}

		return r
	}

	for _, v := range h {
		if r1, r2, r3 := findCardRankWithMatch(c, value, f, fv, n, v.match, sort); r3 {
			// sortCards(r1, r2)
			return retFunc(r1, r2, n), v.rank, true
		}
	}

	return retFunc(c, value, n), msg.CardRank_High_Card, true
}

func findCardRankWithMatch(cards []Card, value []int, f []Card, fv []int, n int, match matchFunc, needSort bool) ([]Card, []int, bool) {
	if len(value)+len(fv) < n {
		return nil, nil, false
	}

	// 检查f中是否符合条件
	if !match(f, fv, n, needSort) {
		return nil, nil, false
	}

	return findRecursive(cards, value, f, fv, n, match, needSort)
}

func findRecursive(cards []Card, value []int, f []Card, fv []int, n int, match matchFunc, needSort bool) ([]Card, []int, bool) {
	num := len(f)
	ndx := 0
	for _, v := range value {
		c := cards[v]
		fBak := f
		fvBak := fv
		f = append(f, c)
		fv = append([]int{}, fv...)
		fv = append(fv, num)

		if !match(f, fv, n, needSort) {
			f = fBak
			fv = fvBak
			ndx++
			continue
		}

		if len(fv) == n {
			return f, fv, true
		}

		if r1, r2, r3 := findRecursive(cards, value[ndx+1:], f, fv, n, match, needSort); r3 {
			return r1, r2, true
		}
		f = fBak
		fv = fvBak
		ndx++
	}
	return nil, nil, false
}

func matchStraightFlush(cards []Card, value []int, n int, needSort bool) bool {
	if n < 5 {
		return false
	}

	if needSort {
		sortCards(cards, value)
	}

	l := len(value)
	for i := 0; i < l-1; i++ {
		if cards[value[i]].value == cards[value[i+1]].value {
			return false
		}
		if cards[value[i]].color != cards[value[i+1]].color {
			return false
		}
	}
	// 如果为A，判断A 2 3 4 5的情况
	if l > 1 && cards[value[0]].value == Card_A_Value {
		if cards[value[l-1]].value != uint32(5-l) && cards[value[l-1]].value != uint32(13-l) {
			return false
		}
		return true
	}

	if l > 2 && cards[value[0]].value-cards[value[l-1]].value > 4 {
		return false
	}

	return true
}

func matchFourOfAKind(cards []Card, value []int, n int, needSort bool) bool {
	if n < 5 {
		return false
	}

	if needSort {
		sortCards(cards, value)
	}

	same := 0
	diff := 0
	l := len(value)
	for i := 0; i < l-1; i++ {
		if cards[value[i]].value != cards[value[i+1]].value {
			diff++
			same = 0
			if diff > 1 {
				return false
			}
		} else {
			same++
			if same == 3 {
				return true
			}
		}
	}

	if l == 5 {
		return false
	}

	return true
}

func matchFullHouse(cards []Card, value []int, n int, needSort bool) bool {
	if n < 5 {
		return false
	}

	if needSort {
		sortCards(cards, value)
	}

	same := 0
	diff := 0
	for i := 0; i < len(value)-1; i++ {
		if cards[value[i]].value != cards[value[i+1]].value {
			same = 0
			diff++
			if diff > 1 {
				return false
			}
		} else {
			same++
			if same > 2 {
				return false
			}
		}
	}

	return true
}

func matchFlush(cards []Card, value []int, n int, needSort bool) bool {
	if n < 5 {
		return false
	}

	if needSort {
		sortCards(cards, value)
	}

	for i := 0; i < len(value)-1; i++ {
		if cards[value[i]].color != cards[value[i+1]].color {
			return false
		}
	}
	return true
}

func matchStraight(cards []Card, value []int, n int, needSort bool) bool {
	if n < 5 {
		return false
	}

	if needSort {
		sortCards(cards, value)
	}

	l := len(value)
	for i := 0; i < l-1; i++ {
		if cards[value[i]].value == cards[value[i+1]].value {
			return false
		}
	}
	// 如果为A，判断A 2 3 4 5的情况
	if l > 1 && cards[value[0]].value == Card_A_Value {
		if cards[value[l-1]].value != uint32(5-l) {
			return false
		}
		return true
	}

	if l > 2 && cards[value[0]].value-cards[value[l-1]].value > 4 {
		return false
	}

	return true
}

func matchThreeOfAKind(cards []Card, value []int, n int, needSort bool) bool {
	if needSort {
		sortCards(cards, value)
	}

	same := 0
	diff := 0
	l := len(value)
	for i := 0; i < l-1; i++ {
		if cards[value[i]].value != cards[value[i+1]].value {
			diff++
			same = 0
			if diff > 2 {
				return false
			}
		} else {
			same++
			if same == 2 {
				return true
			}
		}
	}

	if l == n {
		return false
	}

	return true
}

func matchTwoPair(cards []Card, value []int, n int, needSort bool) bool {
	if n < 5 {
		return false
	}
	if needSort {
		sortCards(cards, value)
	}

	same := 0
	sametime := 0
	diff := 0
	l := len(value)
	for i := 0; i < l-1; i++ {
		if cards[value[i]].value != cards[value[i+1]].value {
			diff++
			same = 0
			if diff > 2 {
				return false
			}
		} else {
			same++
			if same == 1 {
				sametime++
				if sametime == 2 {
					return true
				}
			}
		}
	}

	if l == 5 {
		return false
	}

	return true
}

func matchOnePair(cards []Card, value []int, n int, needSort bool) bool {
	if needSort {
		sortCards(cards, value)
	}

	same := 0
	sametime := 0
	diff := 0
	l := len(value)
	for i := 0; i < l-1; i++ {
		if cards[value[i]].value != cards[value[i+1]].value {
			diff++
			same = 0
		} else {
			same++
			if same == 1 {
				sametime++
				if sametime > 1 {
					return false
				}
			}
		}
	}
	if l == n {
		if (n == 3 || n == 5) && sametime == 1 {
			return true
		}
		return false
	}

	return true
}
