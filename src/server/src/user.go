/**
 * @author [Bruce]
 * @email [lzhig@outlook.com]
 * @create date 2018-01-19 11:24:28
 * @modify date 2018-01-19 11:24:28
 * @desc [description]
 */

package main

import (
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

// LocalUser 测试用户
type LocalUser struct {
	id   string
	name string
}

// GetAvatarURL 返回用户avatar url
func (obj *LocalUser) GetAvatarURL() string { return "" }

// GetUID 返回用户uid
func (obj *LocalUser) GetUID() (uint32, error) {
	return db.getUIDFacebook(obj.id)
}

// SaveToDB 保存到数据库
func (obj *LocalUser) SaveToDB(uid uint32) error {
	return db.AddFacebookUser(obj.id, uid)
}

// GetPlatformID 返回平台id
func (obj *LocalUser) GetPlatformID() uint32 { return 0 }

// GetName 返回用户名字
func (obj *LocalUser) GetName() string { return obj.name }

// The User type represents a player
type User struct {
	sync.RWMutex
	uid      uint32
	name     string // 名字
	avatar   string // 头像
	diamonds uint32 // 钻石

	platformUser PlatformUser

	conn *userConnection // 用户的连接
	room *Room           // 所在房间
}

const (
	userManagerEventCreateUser base.EventID = iota
	userManagerEventLoadUser
	userManagerEventDisconnect
	userManagerEventConsumeDiamonds
	userManagerEventUsersConsumeDiamonds
	userManagerEventEnterRoom
	userManagerEventLeaveRoom
	userManagerEventGetRoom
	//userManagerEventSetRoom
	userManagerEventGetNameAvatar
)

// UserManager type
type UserManager struct {
	base.EventSystem

	//users sync.Map
	users map[uint32]*User
}

func (obj *UserManager) init() {
	obj.EventSystem.Init(1024, true)
	obj.EventSystem.SetEventHandler(userManagerEventCreateUser, obj.handleEventCreateUser)
	obj.EventSystem.SetEventHandler(userManagerEventLoadUser, obj.handleEventLoadUser)
	obj.EventSystem.SetEventHandler(userManagerEventDisconnect, obj.handleEventDisconnect)
	obj.EventSystem.SetEventHandler(userManagerEventConsumeDiamonds, obj.handleEventConsumeDiamonds)
	obj.EventSystem.SetEventHandler(userManagerEventUsersConsumeDiamonds, obj.handleEventUsersConsumeDiamonds)
	obj.EventSystem.SetEventHandler(userManagerEventEnterRoom, obj.handleEventEnterRoom)
	obj.EventSystem.SetEventHandler(userManagerEventLeaveRoom, obj.handleEventLeaveRoom)
	obj.EventSystem.SetEventHandler(userManagerEventGetRoom, obj.handleEventGetRoom)
	//obj.EventSystem.SetEventHandler(userManagerEventSetRoom, obj.handleEventSetRoom)
	obj.EventSystem.SetEventHandler(userManagerEventGetNameAvatar, obj.handleEventGetNameAvatar)

	obj.users = make(map[uint32]*User)
}

// func (obj *UserManager) fbUserExists(fbID, name string) (uint32, error) {
// 	return db.getUIDFacebook(fbID, name)
// }

// CreateUser 创建用户对象
func (obj *UserManager) CreateUser(pu PlatformUser, conn *userConnection) *User {
	userC := make(chan *User)
	obj.Send(userManagerEventCreateUser, []interface{}{pu, conn, userC})
	return <-userC
}

func (obj *UserManager) handleEventCreateUser(args []interface{}) {
	pu := args[0].(PlatformUser)
	conn := args[1].(*userConnection)
	userC := args[2].(chan *User)
	uid, err := db.CreateUser(pu.GetName(), pu.GetAvatarURL(), diamondsCenter.freeDiamonds.GetFreeDiamondsWhenRegister(), pu.GetPlatformID())
	if err != nil {
		userC <- nil
		return
	}
	if err := pu.SaveToDB(uid); err != nil {
		base.LogError("[UserManager][CreateUser] Failed to Save to db. uid:", uid, ", user:", pu)
		userC <- nil
		return
	}
	user, err := obj.createUser(uid, conn)
	if err == nil {
		user.platformUser = pu
		userC <- user
		return
	}
	userC <- nil
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

// LoadUser 加载用户
func (obj *UserManager) LoadUser(pu PlatformUser, uid uint32, conn *userConnection) *User {
	userC := make(chan *User)
	obj.Send(userManagerEventLoadUser, []interface{}{pu, uid, conn, userC})
	return <-userC
}

func (obj *UserManager) handleEventLoadUser(args []interface{}) {
	pu := args[0].(PlatformUser)
	uid := args[1].(uint32)
	conn := args[2].(*userConnection)
	userC := args[3].(chan *User)

	// 如果通过验证，检查此用户是否在线，如果在线，将原连接断开，如果用户在房间内，给房间发送用户断线的消息
	user, ok := obj.users[uid]
	if ok {
		if user.conn != nil {
			// 发送被踢消息
			p := &msg.Protocol{
				Msgid:      msg.MessageID_Kick_Notify,
				KickNotify: &msg.KickNotify{},
			}
			user.conn.sendProtocol(p)

			if user.room != nil {
				// 向房间发送此用户断线消息
				user.room.Send(roomEventUserDisconnect, []interface{}{uid})
			}

			oldConn := user.conn

			// 5秒后断开连接
			time.AfterFunc(time.Second*5, func() {
				oldConn.Disconnect()
			})
		}

		user.platformUser = pu
		user.name = pu.GetName()
	} else {
		user = obj.loadUser(pu, uid, conn)
		if user == nil {
			userC <- nil
			return
		}
	}

	user.conn = conn
	db.UpdateName(uid, user.name)
	userC <- user
}

func (obj *UserManager) createUser(uid uint32, conn *userConnection) (*User, error) {
	diamondsCenter.freeDiamonds.GiveFreeDiamondsEveryDay(uid)
	name, _, avatar, diamonds, err := db.GetUserProfile(uid)
	if err != nil {
		return nil, err
	}

	user := &User{uid: uid, name: name, avatar: avatar, diamonds: diamonds, conn: conn}
	obj.users[uid] = user
	return user, nil
}

func (obj *UserManager) loadUser(pu PlatformUser, uid uint32, conn *userConnection) *User {
	user, err := obj.createUser(uid, conn)
	if err == nil {
		user.platformUser = pu
		user.name = pu.GetName()
		return user
	}
	return nil
}

// GetUser 返回用户对象
func (obj *UserManager) GetUserNameAvatar(uid uint32) (string, string, bool) {
	ret := make(chan []interface{})
	obj.Send(userManagerEventGetNameAvatar, []interface{}{uid, ret})
	result := <-ret
	return result[0].(string), result[1].(string), result[2].(bool)
}

func (obj *UserManager) handleEventGetNameAvatar(args []interface{}) {
	uid := args[0].(uint32)
	ret := args[1].(chan []interface{})
	if user, ok := obj.users[uid]; ok {
		ret <- []interface{}{user.name, user.avatar, true}
		return
	}
	ret <- []interface{}{"", "", false}
}

func (obj *UserManager) userDisconnect(uid uint32, conn *userConnection) {
	ret := make(chan struct{})
	obj.Send(userManagerEventDisconnect, []interface{}{uid, conn, ret})
	<-ret
}

func (obj *UserManager) handleEventDisconnect(args []interface{}) {
	uid := args[0].(uint32)
	conn := args[1].(*userConnection)
	ret := args[2].(chan struct{})

	if user, ok := obj.users[uid]; ok {
		if user.conn == conn {
			if user.room != nil {
				base.LogInfo("disconnected. uid:", user.uid, ", room_id:", user.room.roomID)
				// 向房间发送此用户断线消息
				user.room.Send(roomEventUserDisconnect, []interface{}{uid})
				user.conn = nil
			} else {
				delete(obj.users, uid)
				base.LogInfo("delete uid:", uid)
			}
		}
	}
	ret <- struct{}{}
}

func (obj *UserManager) consumeDiamonds(uid uint32, diamonds uint32, reason string) bool {
	ret := make(chan bool)
	obj.Send(userManagerEventConsumeDiamonds, []interface{}{uid, diamonds, reason, ret})
	return <-ret
}

func (obj *UserManager) handleEventConsumeDiamonds(args []interface{}) {
	uid := args[0].(uint32)
	diamonds := args[1].(uint32)
	reason := args[2].(string)
	ret := args[3].(chan bool)
	if user, ok := obj.users[uid]; ok {
		if user.diamonds < diamonds {
			ret <- false
			return
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

		ret <- true
		return
	}
	ret <- false
}

func (obj *UserManager) consumeUsersDiamonds(uids []uint32, diamonds uint32, reason string) bool {
	ret := make(chan bool)
	obj.Send(userManagerEventUsersConsumeDiamonds, []interface{}{uids, diamonds, reason, ret})
	return <-ret
}

func (obj *UserManager) handleEventUsersConsumeDiamonds(args []interface{}) {
	uids := args[0].([]uint32)
	diamonds := args[1].(uint32)
	reason := args[2].(string)
	result := args[3].(chan bool)

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

		// 如果失败，把之前减的钻石再加上
		for _, user := range usersDone {
			user.diamonds += diamonds
		}
	}()

	for _, uid := range uids {
		if user, ok := obj.users[uid]; ok {

			if user.diamonds < diamonds {
				result <- false
				return
			}

			user.diamonds -= diamonds
			usersDone = append(usersDone, user)
		}
	}
	ret = true
	result <- true
	return
}

func (obj *UserManager) EnterRoom(uid uint32, room *Room) bool {
	ret := make(chan bool)
	obj.Send(userManagerEventEnterRoom, []interface{}{uid, room, ret})
	return <-ret
}

func (obj *UserManager) handleEventEnterRoom(args []interface{}) {
	uid := args[0].(uint32)
	room := args[1].(*Room)
	ret := args[2].(chan bool)
	if user, ok := obj.users[uid]; ok {
		if user.room != nil {
			base.LogError("user.room should be nil when a user enters room. roomid:", user.room.roomID)
		}
		user.room = room
		ret <- true
	} else {
		ret <- false
	}
}

func (obj *UserManager) LeaveRoom(uid uint32) {
	ret := make(chan struct{})
	obj.Send(userManagerEventLeaveRoom, []interface{}{uid, ret})
	<-ret
}

func (obj *UserManager) handleEventLeaveRoom(args []interface{}) {
	uid := args[0].(uint32)
	ret := args[1].(chan struct{})
	if user, ok := obj.users[uid]; ok {
		if user.room == nil {
			base.LogError("user.room should not be nil when a user leaves room")
		}
		user.room = nil

		if user.conn == nil {
			delete(obj.users, uid)
			base.LogInfo("delete user:", uid)
		}
	}
	ret <- struct{}{}
}

func (obj *UserManager) GetUserRoom(uid uint32) *Room {
	ret := make(chan *Room)
	obj.Send(userManagerEventGetRoom, []interface{}{uid, ret})
	return <-ret
}

func (obj *UserManager) handleEventGetRoom(args []interface{}) {
	uid := args[0].(uint32)
	ret := args[1].(chan *Room)
	if user, ok := obj.users[uid]; ok {
		ret <- user.room
		return
	}
	ret <- nil
	return
}

// func (obj *UserManager) SetUserRoom(uid uint32, room *Room) bool {
// 	ret := make(chan bool)
// 	obj.Send(userManagerEventSetRoom, []interface{}{uid, room, ret})
// 	return <-ret
// }

// func (obj *UserManager) handleEventSetRoom(args []interface{}) {
// 	uid := args[0].(uint32)
// 	room := args[1].(*Room)
// 	ret := args[2].(chan bool)
// 	if user, ok := obj.users[uid]; ok {
// 		user.room = room
// 		ret <- true
// 	} else {
// 		ret <- false
// 	}
// }
