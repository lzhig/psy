package main

import (
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// RoomNumberGenerator type
type RoomNumberGenerator struct {
	sync.Mutex
	chars []byte
	pools []int
	used  int
	rand  *rand.Rand
	len   int
}

func (obj *RoomNumberGenerator) init() error {
	obj.rand = rand.New(rand.NewSource(time.Now().UnixNano()))

	obj.chars = make([]byte, 36, 36)
	for i := 0; i < 10; i++ {
		obj.chars[i] = '0' + byte(i)
	}
	for i := 0; i < 26; i++ {
		obj.chars[10+i] = 'a' + byte(i)
	}

	obj.len = 36 * 36 * 36 * 36
	obj.pools = make([]int, obj.len)
	for i := 0; i < obj.len; i++ {
		obj.pools[i] = i
	}

	// used
	usedNum, err := db.getRoomNumberUsed()
	if err != nil {
		return err
	}
	for _, v := range usedNum {
		obj.used++
		obj.pools[v], obj.pools[obj.len-obj.used] = obj.pools[obj.len-obj.used], obj.pools[v]
	}

	return nil
}

func (obj *RoomNumberGenerator) get() (string, error) {
	obj.Lock()
	defer obj.Unlock()

	if obj.used >= obj.len {
		return "", errors.New("no more number can be used")
	}
	num := obj.rand.Int31n(int32(obj.len - obj.used))
	obj.used++
	obj.pools[num], obj.pools[obj.len-obj.used] = obj.pools[obj.len-obj.used], obj.pools[num]
	return obj.decode(int(num)), nil
}

func (obj *RoomNumberGenerator) put(code string) {
	obj.Lock()
	defer obj.Unlock()

	num := obj.encode(code)
	for i := obj.len - obj.used; i < obj.len; i++ {
		if obj.pools[i] == num {
			obj.pools[i], obj.pools[obj.len-obj.used] = obj.pools[obj.len-obj.used], obj.pools[i]
			obj.used--
			return
		}
	}
	logError("[RoomNumberGenerator][put] cannot find num:", num, obj)
}

func (obj *RoomNumberGenerator) decode(num int) string {
	return fmt.Sprintf("%c%c%c%c",
		obj.chars[(num/(36*36*36))%36],
		obj.chars[(num/(36*36))%36],
		obj.chars[(num/36)%36],
		obj.chars[num%36])
}

func (obj *RoomNumberGenerator) encode(code string) int {
	b := []byte(code)
	l := len(b)
	if l > 4 {
		l = 4
	}
	ret := 0
	for i := 0; i < l; i++ {
		c := b[i]
		v := 0
		if c < 'a' {
			v = int(c - '0')
		} else {
			v = int(c-'a') + 10
		}
		ret = ret*36 + v
	}
	return ret
}
