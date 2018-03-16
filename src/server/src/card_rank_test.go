package main

import (
	"fmt"
	"reflect"
	"testing"

	"./msg"
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
			args:  args{cards: []uint32{1, 2, 3, 0, 9, 8, 10, 11, 12}, form: []uint32{}, n: 5},
			want:  []uint32{12, 11, 10, 9, 8},
			want1: msg.CardRank_Straight_Flush,
			want2: true,
		},
		{
			name:  "four_of_a_kind",
			args:  args{cards: []uint32{2, 5, 45, 6, 7, 8, 19, 20, 21, 22, 32}, form: []uint32{}, n: 5},
			want:  []uint32{22, 45, 32, 19, 6},
			want1: msg.CardRank_Four_Of_A_Kind,
			want2: true,
		},
		{
			name:  "full_house",
			args:  args{cards: []uint32{2, 5, 45, 6, 7, 8, 19, 20, 21, 22, 15}, form: []uint32{}, n: 5},
			want:  []uint32{21, 8, 45, 19, 6},
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
			want:  []uint32{23, 21, 45, 32, 6},
			want1: msg.CardRank_Three_Of_A_Kind,
			want2: true,
		},
		{
			name:  "two_pair",
			args:  args{cards: []uint32{2, 45, 16, 18, 20, 21, 8, 32}, form: []uint32{}, n: 5},
			want:  []uint32{21, 8, 20, 45, 32},
			want1: msg.CardRank_Two_Pair,
			want2: true,
		},
		{
			name:  "one_pair",
			args:  args{cards: []uint32{2, 45, 16, 18, 20, 21, 32}, form: []uint32{}, n: 5},
			want:  []uint32{21, 20, 45, 32, 18},
			want1: msg.CardRank_One_Pair,
			want2: true,
		},
		{
			name:  "high_card",
			args:  args{cards: []uint32{2, 45, 16, 18, 20, 21, 33}, form: []uint32{}, n: 5},
			want:  []uint32{21, 33, 20, 45, 18},
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
			fmt.Println(got)
		})
	}
}
