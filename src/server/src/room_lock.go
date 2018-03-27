package main

import (
	"sync"
)

type RoomLocker struct {
	m    sync.Mutex
	lock map[uint32]*sync.RWMutex
}

func (obj *RoomLocker) Init() {
	obj.lock = make(map[uint32]*sync.RWMutex)
}

func (obj *RoomLocker) RLock(roomid uint32) {
	obj.m.Lock()
	defer obj.m.Unlock()

	if m, ok := obj.lock[roomid]; ok {
		m.RLock()
	} else {
		m := &sync.RWMutex{}
		obj.lock[roomid] = m
		m.RLock()
	}
}

func (obj *RoomLocker) RUnlock(roomid uint32) {
	obj.m.Lock()
	defer obj.m.Unlock()

	if m, ok := obj.lock[roomid]; ok {
		m.RUnlock()
	} else {
		panic("need call RLock() first.")
	}
}

func (obj *RoomLocker) Lock(roomid uint32) {
	obj.m.Lock()
	defer obj.m.Unlock()

	if m, ok := obj.lock[roomid]; ok {
		m.Lock()
	} else {
		m := &sync.RWMutex{}
		obj.lock[roomid] = m
		m.Lock()
	}
}

func (obj *RoomLocker) Unlock(roomid uint32) {
	obj.m.Lock()
	defer obj.m.Unlock()

	if m, ok := obj.lock[roomid]; ok {
		m.Unlock()
	} else {
		panic("need call Lock() first.")
	}
}
