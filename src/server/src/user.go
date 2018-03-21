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
	"github.com/lzhig/rapidgo/base"
)

// PlatformUser 平台用户接口
type PlatformUser interface {
	GetPlatformID() uint32
	GetAvatarURL() string
	GetUID() (uint32, error)
	GetName() string
	SaveToDB(uint32) error
}

type LocalUser struct {
	id   string
	name string
}

func (obj *LocalUser) GetAvatarURL() string { return "" }
func (obj *LocalUser) GetUID() (uint32, error) {
	return db.getUIDFacebook(obj.id)
}
func (obj *LocalUser) SaveToDB(uid uint32) error {
	return db.AddFacebookUser(obj.id, uid)
}

func (obj *LocalUser) GetPlatformID() uint32 { return 0 }
func (obj *LocalUser) GetName() string       { return obj.name }

// The User type represents a player
type User struct {
	sync.RWMutex
	uid      uint32
	name     string // 名字
	diamonds uint32 // 钻石

	platformUser PlatformUser

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

func (obj *UserManager) loadUser(user PlatformUser) error {
	return nil
}

// func (obj *UserManager) fbUserExists(fbID, name string) (uint32, error) {
// 	return db.getUIDFacebook(fbID, name)
// }

func (obj *UserManager) CreateUser(pu PlatformUser, conn *userConnection) (*User, error) {
	uid, err := db.CreateUser(pu.GetName(), pu.GetPlatformID(), gApp.config.User.InitDiamonds)
	if err != nil {
		return nil, err
	}
	if err := pu.SaveToDB(uid); err != nil {
		base.LogError("[UserManager][CreateUser] Failed to Save to db. uid:", uid, ", user:", pu)
		return nil, err
	}
	user, err := obj.createUser(uid, conn)
	if err == nil {
		user.platformUser = pu
		return user, err
	}
	return nil, err
}

// func (obj *UserManager) fbUserCreate(fbID, name string, conn *userConnection) (uint32, error) {
// 	uid, err := db.createFacebookUser(fbID, name, gApp.config.User.InitDiamonds)
// 	if err != nil {
// 		return 0, err
// 	}
// 	if err := obj.createUser(uid, conn); err != nil {
// 		return 0, err
// 	}

// 	return uid, nil
// }

func (obj *UserManager) LoadUser(pu PlatformUser, uid uint32, conn *userConnection) (*User, error) {
	user, err := obj.createUser(uid, conn)
	if err == nil {
		user.platformUser = pu
		user.name = pu.GetName()
		return user, err
	}
	return nil, err
}

func (obj *UserManager) createUser(uid uint32, conn *userConnection) (*User, error) {
	name, diamonds, err := db.getUserData(uid)
	if err != nil {
		return nil, err
	}

	user := &User{uid: uid, name: name, diamonds: diamonds, conn: conn}
	obj.users.Store(uid, user)
	return user, nil
}

func (obj *UserManager) GetUser(uid uint32) *User {
	if v, ok := obj.users.Load(uid); ok {
		return v.(*User)
	}
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
			user.conn.sendProtocol(p)

			if user.room != nil {
				// 向房间发送此用户断线消息
				user.room.notifyUserDisconnect(uid)
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

func (obj *UserManager) userDisconnect(uid uint32, conn *userConnection) {
	if v, ok := obj.users.Load(uid); ok {
		user := v.(*User)
		user.Lock()
		defer user.Unlock()

		if user.conn == conn {
			if user.room != nil {
				// 向房间发送此用户断线消息
				user.room.notifyUserDisconnect(uid)
			}
			user.conn = nil
		}
	}
}

func (obj *UserManager) getRoomUserPlaying(uid uint32) (*Room, error) {
	if v, ok := obj.users.Load(uid); ok {
		user := v.(*User)
		user.RLock()
		defer user.RUnlock()

		if user.room != nil {
			return user.room, nil
		}
	}
	return nil, errors.New("not in a room")
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
			base.LogError("[UserManager] [consumeDiamonds] save to db. error:", err)
		} else {
			base.LogInfo("[UserManager] [consumeDiamonds] uid:", uid, ", consume diamonds:", diamonds, ", diamonds:", user.diamonds, ", reason:", reason)
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

func (obj *UserManager) consumeUsersDiamonds(uids []uint32, diamonds uint32, reason string) bool {
	ret := false
	usersDone := make([]*User, 0, len(uids))
	defer func() {
		if ret {
			for _, user := range usersDone {
				if err := db.saveUserDiamonds(user.uid, user.diamonds); err != nil {
					base.LogError("[UserManager] [consumeDiamonds] save to db. error:", err)
				} else {
					base.LogInfo("[UserManager] [consumeDiamonds] uid:", user.uid, ", consume diamonds:", diamonds, ", diamonds:", user.diamonds, "reason:", reason)
				}

				// 发送扣除钻石通知
				sendProtocol(user.conn.conn,
					&msg.Protocol{
						Msgid: msg.MessageID_ConsumeDiamonds_Notify,
						ConsumeDiamondsNotify: &msg.ConsumeDiamondsNotify{Diamonds: diamonds}})

			}
			return
		}
		for _, user := range usersDone {
			if v, ok := obj.users.Load(user.uid); ok {
				user := v.(*User)
				user.Lock()
				defer user.Unlock()
				user.diamonds += diamonds
			}
		}
	}()

	for _, uid := range uids {
		if v, ok := obj.users.Load(uid); ok {
			user := v.(*User)
			user.Lock()
			defer user.Unlock()

			if user.diamonds < diamonds {
				return false
			}
			user.diamonds -= diamonds

			usersDone = append(usersDone, user)
		}
	}
	ret = true
	return ret
}

func (obj *UserManager) enterRoom(uid uint32, room *Room) {
	if v, ok := obj.users.Load(uid); ok {
		user := v.(*User)
		user.Lock()
		defer user.Unlock()

		if user.room != nil {
			base.LogError("user.room should be nil when a user enters room")
		}
		user.room = room
	}
}

func (obj *UserManager) leaveRoom(uid uint32, room *Room) {
	if v, ok := obj.users.Load(uid); ok {
		user := v.(*User)
		user.Lock()
		defer user.Unlock()

		if user.room == nil {
			panic("user.room should not be nil when a user leaves room")
		}
		user.room = nil
	}
}

func (obj *UserManager) getUserRoom(uid uint32) *Room {
	if v, ok := obj.users.Load(uid); ok {
		user := v.(*User)
		user.RLock()
		defer user.RUnlock()

		return user.room
	}

	return nil
}
