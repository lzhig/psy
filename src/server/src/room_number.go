package main

import (
	"errors"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/lzhig/rapidgo/base"
)

// RoomNumberGenerator type
type RoomNumberGenerator struct {
	sync.Mutex
	chars []byte
	pools []int
	used  int
	rand  *rand.Rand
	len   int

	charValueMap     map[byte]int
	valueCharMap     map[int]byte
	charsCount       int
	roomNumberLength uint
}

func (obj *RoomNumberGenerator) init(length uint, chars []byte, used []int) error {
	obj.rand = rand.New(rand.NewSource(time.Now().UnixNano()))

	obj.charValueMap = make(map[byte]int)
	obj.valueCharMap = make(map[int]byte)
	obj.charsCount = len(chars)
	obj.roomNumberLength = length

	for i, c := range chars {
		obj.charValueMap[c] = i
		obj.valueCharMap[i] = c
	}

	obj.len = int(math.Pow(float64(obj.charsCount), float64(length)))
	obj.pools = make([]int, obj.len)
	for i := 0; i < obj.len; i++ {
		obj.pools[i] = i
	}

	// used
	for _, v := range used {
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
	base.LogError("[RoomNumberGenerator][put] cannot find num:", num, obj)
}

func (obj *RoomNumberGenerator) decode(num int) string {
	b := make([]byte, obj.roomNumberLength, obj.roomNumberLength)
	obj._decode(num, 0, b)
	return string(b)
}

func (obj *RoomNumberGenerator) _decode(num int, length uint, output []byte) {
	if length < obj.roomNumberLength-1 {
		obj._decode(num/obj.charsCount, length+1, output[:obj.roomNumberLength-length-1])
	}
	output[obj.roomNumberLength-length-1] = obj.valueCharMap[(num)%(obj.charsCount)]
}

func (obj *RoomNumberGenerator) encode(code string) int {
	b := []byte(code)
	l := len(b)
	ret := 0
	for i := 0; i < l; i++ {
		c := b[i]
		ret = ret*obj.charsCount + obj.charValueMap[c]
	}
	return ret
}
