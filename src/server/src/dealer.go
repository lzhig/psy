package main

import (
	"context"
	"math/rand"
	"time"
)

// Dealer type
type Dealer struct {
	s *rand.Rand
	c chan []uint32
}

func (obj *Dealer) init() {
	obj.s = rand.New(rand.NewSource(time.Now().UnixNano()))
	obj.c = make(chan []uint32)

	ctx, _ := gApp.CreateCancelContext()

	gApp.GoRoutine(ctx, obj.loop)
}

func (obj *Dealer) deal() []uint32 {
	return <-obj.c
}

func (obj *Dealer) loop(ctx context.Context) {
	defer debug("exit Dealer goroutine")
	for {

		//generate
		t := obj.s.Perm(52)
		c := make([]uint32, 52)
		for ndx, v := range t {
			c[ndx] = uint32(v)
		}

		select {
		case <-ctx.Done():
			return
		case obj.c <- c:
		}
	}
}
