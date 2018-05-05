package main

import (
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"./msg"
	"github.com/lzhig/rapidgo/base"
)

func Test_calculateCardRank(t *testing.T) {
	type args struct {
		cards []uint32
	}
	tests := []struct {
		name string
		args args
		want msg.CardRank
	}{
		{
			name: "Straight_Flush",
			args: args{cards: []uint32{3, 5, 2, 4, 1}},
			want: msg.CardRank_Straight_Flush,
		},
		{
			name: "Straight_Flush_1",
			args: args{cards: []uint32{3, 1, 2, 0, 12}},
			want: msg.CardRank_Straight_Flush,
		},

		{
			name: "Straight_Flush_2",
			args: args{cards: []uint32{12, 11, 10, 9, 8}},
			want: msg.CardRank_Straight_Flush,
		},

		{
			name: "2",
			args: args{cards: []uint32{8, 5, 2, 4, 1}},
			want: msg.CardRank_Flush,
		},

		{
			name: "3",
			args: args{cards: []uint32{8, 5, 18, 31, 44}},
			want: msg.CardRank_Four_Of_A_Kind,
		},

		{
			name: "4",
			args: args{cards: []uint32{8, 21, 18, 31, 44}},
			want: msg.CardRank_Full_House,
		},

		{
			name: "5",
			args: args{cards: []uint32{8, 2, 9, 6, 1}},
			want: msg.CardRank_Flush,
		},

		{
			name: "6",
			args: args{cards: []uint32{5, 2, 43, 29, 14}},
			want: msg.CardRank_Straight,
		},

		{
			name: "7",
			args: args{cards: []uint32{5, 2, 18, 29, 31}},
			want: msg.CardRank_Three_Of_A_Kind,
		},

		{
			name: "8",
			args: args{cards: []uint32{5, 2, 18, 29, 28}},
			want: msg.CardRank_Two_Pair,
		},

		{
			name: "9",
			args: args{cards: []uint32{5, 2, 43, 18, 14}},
			want: msg.CardRank_One_Pair,
		},

		{
			name: "10",
			args: args{cards: []uint32{5, 0, 43, 29, 14}},
			want: msg.CardRank_High_Card,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := calculateCardRank(tt.args.cards); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("calculateCardRank() = %v, want %v", got, tt.want)
			}
			fmt.Println(tt.name, tt.args)
		})
	}
}

func BenchmarkLoops(b *testing.B) {
	a := []int{5, 2, 3, 4, 1}
	for i := 0; i < b.N; i++ {
		loops2(a)
	}
}

func loops1(a []int) bool {
	l := len(a)
	s := 0
	for _, v := range a {
		s += v
	}

	if s != (a[0]+a[l-1])*l/2 {
		return false
	}
	return true
}

func loops2(a []int) bool {
	l := len(a)
	for i := 0; i < l-1; i++ {
		if a[i] != a[i+1]-1 {
			return false
		}
	}
	return true
}

func Test_findCardRank(t *testing.T) {
	type args struct {
		cards []uint32
		form  []uint32
		n     int
	}
	tests := []struct {
		name  string
		args  args
		want  []uint32
		want1 msg.CardRank
		want2 bool
	}{
		{
			name:  "straight_flush",
			args:  args{cards: []uint32{2, 5, 4, 6, 7, 8, 19, 20, 21, 22, 18}, form: []uint32{}, n: 5},
			want:  []uint32{22, 21, 20, 19, 18},
			want1: msg.CardRank_Straight_Flush,
			want2: true,
		},
		{
			name:  "straight_flush_A",
			args:  args{cards: []uint32{1, 2, 3, 0, 12}, form: []uint32{}, n: 5},
			want:  []uint32{3, 2, 1, 0, 12},
			want1: msg.CardRank_Straight_Flush,
			want2: true,
		},
		{
			name:  "straight_flush_TJQKA",
			args:  args{cards: []uint32{1, 2, 3, 0, 9, 8, 10, 7, 6, 12}, form: []uint32{}, n: 5},
			want:  []uint32{10, 9, 8, 7, 6},
			want1: msg.CardRank_Straight_Flush,
			want2: true,
		},
		{
			name:  "four_of_a_kind",
			args:  args{cards: []uint32{2, 5, 45, 6, 7, 8, 19, 20, 21, 22, 32}, form: []uint32{}, n: 5},
			want:  []uint32{45, 32, 19, 6, 22},
			want1: msg.CardRank_Four_Of_A_Kind,
			want2: true,
		},
		{
			name:  "four_of_a_kind_1",
			args:  args{cards: []uint32{5, 45, 19, 6, 32}, form: []uint32{}, n: 5},
			want:  []uint32{45, 32, 19, 6, 5},
			want1: msg.CardRank_Four_Of_A_Kind,
			want2: true,
		},
		{
			name:  "full_house",
			args:  args{cards: []uint32{2, 5, 45, 6, 7, 8, 19, 20, 21, 22, 15}, form: []uint32{}, n: 5},
			want:  []uint32{45, 19, 6, 21, 8},
			want1: msg.CardRank_Full_House,
			want2: true,
		},
		{
			name:  "flush",
			args:  args{cards: []uint32{2, 5, 35, 6, 7, 8, 19, 20, 21, 22, 15}, form: []uint32{}, n: 5},
			want:  []uint32{22, 21, 20, 19, 15},
			want1: msg.CardRank_Flush,
			want2: true,
		},
		{
			name:  "Flush_A",
			args:  args{cards: []uint32{12, 11, 10, 1, 0}, form: []uint32{}, n: 5},
			want:  []uint32{12, 11, 10, 1, 0},
			want1: msg.CardRank_Flush,
			want2: true,
		},
		{
			name:  "straight",
			args:  args{cards: []uint32{7, 8, 19, 33, 21, 22, 18}, form: []uint32{}, n: 5},
			want:  []uint32{22, 21, 33, 19, 18},
			want1: msg.CardRank_Straight,
			want2: true,
		},
		{
			name:  "straight_A",
			args:  args{cards: []uint32{1, 2, 3, 13, 12}, form: []uint32{}, n: 5},
			want:  []uint32{3, 2, 1, 13, 12},
			want1: msg.CardRank_Straight,
			want2: true,
		},
		{
			name:  "three_of_a_kind",
			args:  args{cards: []uint32{2, 45, 6, 18, 20, 21, 23, 32}, form: []uint32{}, n: 5},
			want:  []uint32{45, 32, 6, 23, 21},
			want1: msg.CardRank_Three_Of_A_Kind,
			want2: true,
		},
		{
			name:  "two_pair",
			args:  args{cards: []uint32{2, 45, 16, 18, 20, 21, 8, 32}, form: []uint32{}, n: 5},
			want:  []uint32{21, 8, 45, 32, 20},
			want1: msg.CardRank_Two_Pair,
			want2: true,
		},
		{
			name:  "one_pair",
			args:  args{cards: []uint32{2, 45, 16, 18, 20, 21, 32}, form: []uint32{}, n: 5},
			want:  []uint32{45, 32, 21, 20, 18},
			want1: msg.CardRank_One_Pair,
			want2: true,
		},
		{
			name:  "one_pair_1",
			args:  args{cards: []uint32{2, 45, 16, 18, 20, 21, 33}, form: []uint32{}, n: 5},
			want:  []uint32{33, 20, 21, 45, 18},
			want1: msg.CardRank_One_Pair,
			want2: true,
		},
		{
			name:  "high_card_1",
			args:  args{cards: []uint32{12, 11, 10, 1, 13}, form: []uint32{}, n: 5},
			want:  []uint32{12, 11, 10, 1, 13},
			want1: msg.CardRank_High_Card,
			want2: true,
		},
		{
			name:  "test",
			args:  args{cards: []uint32{23, 34, 7, 32, 42, 41, 14, 0}, form: []uint32{}, n: 5},
			want:  []uint32{23, 34, 7, 32, 42},
			want1: msg.CardRank_High_Card,
			want2: true,
		},
		{
			name:  "test-1",
			args:  args{cards: []uint32{9, 27, 4, 25, 21}, form: []uint32{28, 50}, n: 5},
			want:  []uint32{25, 50, 9, 21, 28},
			want1: msg.CardRank_High_Card,
			want2: true,
		},
		{
			name:  "test-2",
			args:  args{cards: []uint32{}, form: []uint32{23, 0, 45, 32, 6}, n: 5},
			want:  []uint32{45, 32, 6, 23, 0},
			want1: msg.CardRank_Three_Of_A_Kind,
			want2: true,
		},
		{
			name:  "test-3",
			args:  args{cards: nil, form: []uint32{49, 23, 26, 0, 15}, n: 5},
			want:  []uint32{49, 23, 26, 0, 15},
			want1: msg.CardRank_Two_Pair,
			want2: true,
		},
		{
			name:  "test-4",
			args:  args{cards: nil, form: []uint32{50, 35, 48}, n: 3},
			want:  []uint32{48, 35, 50},
			want1: msg.CardRank_One_Pair,
			want2: true,
		},
		{
			name:  "test-5",
			args:  args{cards: nil, form: []uint32{12, 11, 7, 3, 0}, n: 5},
			want:  []uint32{12, 11, 7, 3, 0},
			want1: msg.CardRank_Flush,
			want2: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, got2 := findCardRank(tt.args.cards, tt.args.form, tt.args.n)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("findCardRank() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("findCardRank() got1 = %v, want %v", got1, tt.want1)
			}
			if got2 != tt.want2 {
				t.Errorf("findCardRank() got2 = %v, want %v", got2, tt.want2)
			}
			fmt.Println(got, got1)
		})
	}
}

func Test_find(t *testing.T) {
	f := func(cards []uint32, card uint32) int {
		low := 0
		high := len(cards) - 1
		for low <= high {
			median := (low + high) / 2
			if cards[median] > card {
				low = median + 1
			} else {
				high = median - 1
			}
		}

		if low == len(cards) || cards[low] != card {
			return -1
		}

		return low
	}

	cards := []uint32{9, 8, 7, 6, 5, 4, 3, 2, 1}
	card := uint32(4)
	fmt.Println(f(cards, card))
}

func Test_time(tt *testing.T) {
	a := time.Now()
	fmt.Println("今天0点:", base.GetTodayZeroClockTime(&a).Unix())
	fmt.Println("明天0点:", base.GetTodayZeroClockTime(&a).AddDate(0, 0, 1).Unix())
	fmt.Println(time.Now().Unix())
	t, err := time.ParseInLocation("2006-1-2 15:4:5", time.Now().Format("2006-1-2 15:4:5"), time.Local)
	fmt.Println(t.Unix(), err)
	t, err = time.ParseInLocation("2006-1-2 15:4:5", "0001-1-1 0:0:0", time.Local)
	fmt.Println(t.Unix(), err)
	fmt.Println(t.Date())
	y, m, d := time.Now().AddDate(0, 0, 0).Date()
	t = t.AddDate(y-1, int(m)-1, d-1)
	fmt.Println(t.Format("2006-1-2 15:4:5"))
	fmt.Println("今天0点：", t.Unix())
	{
		t, _ := time.ParseInLocation("2006-1-2 15:4:5", "0001-1-1 0:0:0", time.Local)
		y, m, d := time.Now().AddDate(0, 0, 1).Date()
		end := t.AddDate(y-1, int(m)-1, d-1)
		fmt.Println("明天0点：", end.Unix())
	}
}

func Test_time1(t *testing.T) {
	now := time.Now()
	year, mon, day := now.UTC().Date()
	hour, min, sec := now.UTC().Clock()
	zone, _ := now.UTC().Zone()
	fmt.Printf("UTC 时间是 %d-%d-%d %02d:%02d:%02d %s\n",
		year, mon, day, hour, min, sec, zone)

	year, mon, day = now.Date()
	hour, min, sec = now.Clock()
	zone, _ = now.Zone()
	fmt.Printf("本地时间是 %d-%d-%d %02d:%02d:%02d %s\n",
		year, mon, day, hour, min, sec, zone)
}

func Test_defer(t *testing.T) {
	a := true
	defer func() {
		if a {
			fmt.Println("a is true")
		} else {
			fmt.Println("a is false")
		}
	}()
	a = false
}

func Test_use_chan(t *testing.T) {
	eventsystem := &struct {
		base.EventSystem
	}{}
	eventsystem.Init(1024, true)

	v := 0
	eventsystem.SetEventHandler(1, func(args []interface{}) {
		value := args[0].(int)
		v = value
		c := args[1].(chan struct{})
		c <- struct{}{}
	})
	b := time.Now()
	w := sync.WaitGroup{}
	for i := 0; i < 10000; i++ {
		w.Add(1)
		go func() {
			c := make(chan struct{})
			for i := 0; i < 500; i++ {
				eventsystem.Send(1, []interface{}{1, c})
				<-c
			}
			w.Done()
		}()
	}
	w.Wait()
	fmt.Println(time.Now().Sub(b))
}

func Test_use_mutex(t *testing.T) {
	m := sync.Mutex{}
	v := 0
	// getValue := func() int {
	// 	m.Lock()
	// 	defer m.Unlock()
	// 	return 0
	// }
	setValue := func(value int) {
		m.Lock()
		defer m.Unlock()
		v = value
	}
	b := time.Now()
	w := sync.WaitGroup{}
	for i := 0; i < 10000; i++ {
		w.Add(1)
		go func() {
			for i := 0; i < 500; i++ {
				//getValue()
				setValue(1)
			}
			w.Done()
		}()
	}
	w.Wait()
	fmt.Println(time.Now().Sub(b))
}

func Test_Slice(t *testing.T) {
	a := []uint32{1}
	b := a[1:]
	fmt.Println(a, b)
}
