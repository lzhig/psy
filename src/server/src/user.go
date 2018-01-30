/**
 * @author [Bruce]
 * @email [lzhig@outlook.com]
 * @create date 2018-01-19 11:24:28
 * @modify date 2018-01-19 11:24:28
 * @desc [description]
 */

package main

import (
	"errors"
	"sync"
	"time"

	"./msg"
)

// The User type represents a player
type User struct {
	sync.RWMutex
	uid      uint32
	name     string // 名字
	diamonds uint32 // 钻石

	conn *userConnection // 用户的连接
	room *Room           // 所在房间
}

// UserManager type
type UserManager struct {
	users sync.Map
	//users map[uint32]*User
}

func (obj *UserManager) init() {
	//obj.users = make(map[uint32]*User)
}

func (obj *UserManager) fbUserExists(fbID, name string) (uint32, error) {
	return db.getUIDFacebook(fbID, name)
}

func (obj *UserManager) fbUserCreate(fbID, name string, conn *userConnection) (uint32, error) {
	uid, err := db.createFacebookUser(fbID, name)
	if err != nil {
		return 0, err
	}
	if err := obj.createUser(uid, conn); err != nil {
		return 0, err
	}

	return uid, nil
}

func (obj *UserManager) createUser(uid uint32, conn *userConnection) error {
	name, diamonds, err := db.getUserData(uid)
	if err != nil {
		return err
	}

	obj.users.Store(uid, &User{uid: uid, name: name, diamonds: diamonds, conn: conn})
	return nil
}

func (obj *UserManager) userIsConnected(uid uint32) bool {
	if v, ok := obj.users.Load(uid); ok {
		user := v.(*User)

		user.RLock()
		defer user.RUnlock()

		return user.conn != nil
	}
	return false
}

func (obj *UserManager) setUserConnection(uid uint32, conn *userConnection) {
	if v, ok := obj.users.Load(uid); ok {
		user := v.(*User)
		user.Lock()
		defer user.Unlock()

		if user.conn != nil {
			// 发送被踢消息
			p := &msg.Protocol{
				Msgid:      msg.MessageID_Kick_Notify,
				KickNotify: &msg.KickNotify{},
			}
			conn.sendProtocol(p)

			if user.room != nil {
				// 向房间发送此用户断线消息
				roomManager.notifyUserDisconnect(uid, user.room.roomID)
			}

			oldConn := user.conn
			// 5秒后断开连接
			time.AfterFunc(time.Second*5, func() {
				oldConn.Disconnect()
			})
		}

		user.conn = conn
	}
}

func (obj *UserManager) getRoomIDUserPlaying(uid uint32) (uint32, error) {
	if v, ok := obj.users.Load(uid); ok {
		user := v.(*User)
		user.RLock()
		defer user.RUnlock()

		if user.room != nil {
			return user.room.roomID, nil
		}
	}
	return 0, errors.New("not in a room")
}

func (obj *UserManager) consumeDiamonds(uid uint32, diamonds uint32, reason string) bool {
	if v, ok := obj.users.Load(uid); ok {
		user := v.(*User)
		user.Lock()
		defer user.Unlock()

		if user.diamonds < diamonds {
			return false
		}

		user.diamonds -= diamonds

		if err := db.saveUserDiamonds(uid, user.diamonds); err != nil {
			logError("[UserManager] [consumeDiamonds] save to db. error:", err)
		} else {
			logInfo("[UserManager] [consumeDiamonds] uid:", uid, ", consume diamonds:", diamonds, ", diamonds:", user.diamonds, "reason:", reason)
		}

		// 发送扣除钻石通知
		sendProtocol(user.conn.conn,
			&msg.Protocol{
				Msgid: msg.MessageID_ConsumeDiamonds_Notify,
				ConsumeDiamondsNotify: &msg.ConsumeDiamondsNotify{Diamonds: diamonds}})

		return true
	}
	return false
}
