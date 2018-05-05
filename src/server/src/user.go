/**
 * @author [Bruce]
 * @email [lzhig@outlook.com]
 * @create date 2018-01-19 11:24:28
 * @modify date 2018-01-19 11:24:28
 * @desc [description]
 */

package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"time"

	"./msg"
	"github.com/lzhig/rapidgo/base"
)

var errUserBanned = errors.New("This account has been banned")

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

	conn *userConnection // 用户当前的连接
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
	userManagerEventKickUser
	userManagerEventKickUsersNotInRoom
	userManagerEventNotifyAllUsers
	userManagerEventNetworkPacket
	userManagerEventNoticeTimer
)

// UserManager type
type UserManager struct {
	base.EventSystem

	//users sync.Map
	users map[uint32]*User

	networkPacketHandler base.MessageHandlerImpl

	notices *NoticesConfig
}

// Init 初始化
func (obj *UserManager) Init() error {
	obj.notices = &NoticesConfig{}
	if err := obj.notices.Load(gApp.config.Notice.File); err != nil {
		return err
	}
	obj.notices.BeginTicker(&obj.EventSystem, userManagerEventNoticeTimer)

	obj.EventSystem.Init(1024, true)
	obj.SetEventHandler(userManagerEventNetworkPacket, obj.handleEventNetworkPacket)
	obj.SetEventHandler(userManagerEventCreateUser, obj.handleEventCreateUser)
	obj.SetEventHandler(userManagerEventLoadUser, obj.handleEventLoadUser)
	obj.SetEventHandler(userManagerEventDisconnect, obj.handleEventDisconnect)
	obj.SetEventHandler(userManagerEventConsumeDiamonds, obj.handleEventConsumeDiamonds)
	obj.SetEventHandler(userManagerEventUsersConsumeDiamonds, obj.handleEventUsersConsumeDiamonds)
	obj.SetEventHandler(userManagerEventEnterRoom, obj.handleEventEnterRoom)
	obj.SetEventHandler(userManagerEventLeaveRoom, obj.handleEventLeaveRoom)
	obj.SetEventHandler(userManagerEventGetRoom, obj.handleEventGetRoom)
	//obj.SetEventHandler(userManagerEventSetRoom, obj.handleEventSetRoom)
	obj.SetEventHandler(userManagerEventGetNameAvatar, obj.handleEventGetNameAvatar)
	obj.SetEventHandler(userManagerEventKickUser, obj.handleEventKickUser)
	obj.SetEventHandler(userManagerEventKickUsersNotInRoom, obj.handleEventKickUsersNotInRoom)
	obj.SetEventHandler(userManagerEventNotifyAllUsers, obj.handleEventNotifyAllUsers)
	obj.SetEventHandler(userManagerEventNoticeTimer, obj.handleEventNoticeTimer)

	obj.users = make(map[uint32]*User)

	obj.networkPacketHandler.Init()
	obj.networkPacketHandler.SetMessageHandler(msg.MessageID_GetNotices_Req, obj.handleGetNotices)
	return nil
}

func (obj *UserManager) handleEventNetworkPacket(args []interface{}) {
	p := args[0].(*ProtocolConnection)
	if p == nil {
		base.LogError("args[0] isn't a ProtocolConnection object.")
		return
	}

	if !obj.networkPacketHandler.Handle(p.p.Msgid, p) {
		base.LogError("cannot find handler for msgid:", msg.MessageID_name[int32(p.p.Msgid)])
		p.userconn.Disconnect()
	}
}

func (obj *UserManager) handleGetNotices(arg interface{}) {
	p := arg.(*ProtocolConnection)
	userconn := p.userconn

	rsp := &msg.Protocol{
		Msgid: msg.MessageID_GetNotices_Rsp,
		GetNoticesRsp: &msg.GetNoticesRsp{Ret: msg.ErrorID_Ok,
			Notices: obj.notices.GetAvailableNotices(false),
		},
	}

	userconn.sendProtocol(rsp)
}

// func (obj *UserManager) fbUserExists(fbID, name string) (uint32, error) {
// 	return db.getUIDFacebook(fbID, name)
// }

// CreateUser 创建用户对象
func (obj *UserManager) CreateUser(pu PlatformUser, conn *userConnection) (*User, error) {
	userC := make(chan []interface{})
	obj.Send(userManagerEventCreateUser, []interface{}{pu, conn, userC})
	ret := <-userC
	return ret[0].(*User), ret[1].(error)
}

func (obj *UserManager) handleEventCreateUser(args []interface{}) {
	pu := args[0].(PlatformUser)
	conn := args[1].(*userConnection)
	userC := args[2].(chan []interface{})
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
		userC <- []interface{}{user, nil}
		return
	}
	userC <- []interface{}{nil, err}
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

// KickUser 踢离用户
func (obj *UserManager) KickUser(uid uint32) error {
	c := make(chan error)
	obj.Send(userManagerEventKickUser, []interface{}{uid, c})
	return <-c
}

func (obj *UserManager) handleEventKickUser(args []interface{}) {
	uid := args[0].(uint32)
	c := args[1].(chan error)
	user, ok := obj.users[uid]
	if !ok {
		c <- fmt.Errorf("The user: %d not exist", uid)
		return
	}

	obj.kickUser(user, msg.KickType_GM)
	c <- nil
}

// KickUsersNotInRoom 踢离不在房间中的所有玩家
func (obj *UserManager) KickUsersNotInRoom() {
	c := make(chan struct{})
	obj.Send(userManagerEventKickUsersNotInRoom, []interface{}{c})
	<-c
}

func (obj *UserManager) handleEventKickUsersNotInRoom(args []interface{}) {
	c := args[0].(chan struct{})

	for _, user := range obj.users {
		if user.room == nil {
			obj.kickUser(user, msg.KickType_StopServer)
		}
	}
	c <- struct{}{}
}

func (obj *UserManager) kickUser(user *User, t msg.KickType) {
	if user.conn != nil {
		// 发送被踢消息
		p := &msg.Protocol{
			Msgid:      msg.MessageID_Kick_Notify,
			KickNotify: &msg.KickNotify{Type: t},
		}
		user.conn.sendProtocol(p)

		if user.room != nil {
			// 向房间发送此用户断线消息
			user.room.Send(roomEventUserDisconnect, []interface{}{user.uid})
		}

		oldConn := user.conn
		user.conn = nil

		// 5秒后断开连接
		time.AfterFunc(time.Second*5, func() {
			oldConn.Disconnect()
		})
	}
}

// LoadUser 加载用户
func (obj *UserManager) LoadUser(pu PlatformUser, uid uint32, conn *userConnection) (*User, error) {
	userC := make(chan []interface{})
	obj.Send(userManagerEventLoadUser, []interface{}{pu, uid, conn, userC})
	ret := <-userC
	if ret[1] == nil {
		return ret[0].(*User), nil
	}
	err := ret[1].(error)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (obj *UserManager) handleEventLoadUser(args []interface{}) {
	pu := args[0].(PlatformUser)
	uid := args[1].(uint32)
	conn := args[2].(*userConnection)
	userC := args[3].(chan []interface{})

	// 如果通过验证，检查此用户是否在线，如果在线，将原连接断开，如果用户在房间内，给房间发送用户断线的消息
	user, ok := obj.users[uid]
	var err error
	if ok {
		obj.kickUser(user, msg.KickType_Login)

		user.platformUser = pu
		user.name = pu.GetName()
	} else {
		user, err = obj.loadUser(pu, uid, conn)
		if err != nil {
			userC <- []interface{}{nil, err}
			return
		}
	}

	user.conn = conn
	db.UpdateName(uid, user.name)
	userC <- []interface{}{user, err}
}

func (obj *UserManager) createUser(uid uint32, conn *userConnection) (*User, error) {
	diamondsCenter.freeDiamonds.GiveFreeDiamondsEveryDay(uid)
	name, _, avatar, diamonds, status, err := db.GetUserProfile(uid)
	if err != nil {
		return nil, err
	}
	// 账号被封
	if status == 1 {
		return nil, errUserBanned
	}

	user := &User{uid: uid, name: name, avatar: avatar, diamonds: diamonds, conn: conn}
	obj.users[uid] = user
	return user, nil
}

func (obj *UserManager) loadUser(pu PlatformUser, uid uint32, conn *userConnection) (*User, error) {
	user, err := obj.createUser(uid, conn)
	if err == nil {
		user.platformUser = pu
		user.name = pu.GetName()
		return user, err
	}
	return nil, err
}

// GetUserNameAvatar 返回用户对象
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

		if err := db.PayDiamonds(uid, 0, diamonds, 0); err != nil {
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
				if err := db.PayDiamonds(user.uid, 0, user.diamonds, 0); err != nil {
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

func (obj *UserManager) handleEventNoticeTimer(args []interface{}) {
	noticeID := args[0].(uint32)

	notice := obj.notices.GetNotice(noticeID)
	if notice == nil {
		base.LogWarn("Cannot find the notice. notice id:", noticeID)
	} else {
		notice.notified = true
		p := &msg.Protocol{
			Msgid: msg.MessageID_Notices_Notify,
			NoticesNotify: &msg.NoticesNotify{
				Notices: []*msg.Notice{
					&msg.Notice{
						Id:      notice.ID,
						Begin:   notice.beginTime,
						End:     notice.endTime,
						Content: notice.Content,
						Type:    notice.Type,
					},
				},
			},
		}

		for _, user := range obj.users {
			user.conn.sendProtocol(p)
		}
	}

	obj.notices.BeginTicker(&obj.EventSystem, userManagerEventNoticeTimer)
}

// NotifyAllUsers 发送公告
func (obj *UserManager) NotifyAllUsers() error {
	c := make(chan error)
	obj.Send(userManagerEventNotifyAllUsers, []interface{}{c})
	return <-c
}

// 更新公告时，先通知当前时间内的公告，然后定时一个离当前时间最近的公告
func (obj *UserManager) handleEventNotifyAllUsers(args []interface{}) {
	c := args[0].(chan error)

	// load notice
	if err := obj.notices.Load(gApp.config.Notice.File); err != nil {
		c <- err
		return
	}

	notices := obj.notices.GetAvailableNotices(true)
	p := &msg.Protocol{
		Msgid: msg.MessageID_Notices_Notify,
		NoticesNotify: &msg.NoticesNotify{
			Notices: notices,
		},
	}

	for _, user := range obj.users {
		user.conn.sendProtocol(p)
	}

	obj.notices.BeginTicker(&obj.EventSystem, userManagerEventNoticeTimer)
	c <- nil
}

// NoticeConfig 公告配置
type NoticeConfig struct {
	ID      uint32         `json:"id"`
	Begin   string         `json:"begin"`
	End     string         `json:"end"`
	Content string         `json:"content"`
	Type    msg.NoticeType `json:"type"`

	beginTime uint32
	endTime   uint32
	notified  bool
}

// NoticesConfig 公告配置
type NoticesConfig struct {
	Config []*NoticeConfig `json:"notices"`

	notices map[uint32]*NoticeConfig

	timer *time.Timer
}

// GetNotice 返回公告
func (obj *NoticesConfig) GetNotice(noticeID uint32) *NoticeConfig {
	if notice, ok := obj.notices[noticeID]; ok {
		return notice
	}
	return nil
}

// DeleteNotice 删除公告
func (obj *NoticesConfig) DeleteNotice(noticeID uint32) {
	delete(obj.notices, noticeID)
}

// Load 读取配置文件
func (obj *NoticesConfig) Load(filename string) error {
	if obj.timer != nil {
		obj.timer.Stop()
	}

	obj.notices = make(map[uint32]*NoticeConfig)

	// load notice
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	r := bufio.NewReader(f)
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, obj)
	if err != nil {
		return err
	}

	// check format of date
	for _, notice := range obj.Config {
		begin, err := time.ParseInLocation("2006-1-2 15:4:5", notice.Begin, time.Local)
		if err != nil {
			base.LogError("Failed to parse begin time of notice. id:", notice.ID)
			return err
		}
		notice.beginTime = uint32(begin.Unix())

		end, err := time.ParseInLocation("2006-1-2 15:4:5", notice.End, time.Local)
		if err != nil {
			base.LogError("Failed to parse end time of notice. id:", notice.ID)
			return err
		}
		notice.endTime = uint32(end.Unix())

		base.LogInfo(notice)

		obj.notices[notice.ID] = notice
	}

	return nil
}

// BeginTicker 开始定时
func (obj *NoticesConfig) BeginTicker(event *base.EventSystem, eventID base.EventID) {
	if len(obj.notices) == 0 {
		return
	}
	now := uint32(time.Now().Unix())
	// 查找离当前时间最近的公告

	noticeID := uint32(0)
	earlier := uint32(0)
	for id, notice := range obj.notices {
		if notice.beginTime < now || notice.notified {
			continue
		}

		if earlier == 0 || earlier > notice.beginTime {
			earlier = notice.beginTime
			noticeID = id
		}
	}

	obj.timer = time.AfterFunc(time.Duration(earlier-now)*time.Second, func() {
		event.Send(eventID, []interface{}{noticeID})
	})

}

// GetAvailableNotices 返回当前有效的公告
func (obj *NoticesConfig) GetAvailableNotices(isNotify bool) []*msg.Notice {
	now := uint32(time.Now().Unix())

	ret := make([]*msg.Notice, 0, len(obj.notices))
	for _, notice := range obj.notices {
		if notice.beginTime <= now && notice.endTime > now {
			ret = append(ret, &msg.Notice{
				Id:      notice.ID,
				Begin:   notice.beginTime,
				End:     notice.endTime,
				Content: notice.Content,
				Type:    notice.Type,
			})
			if isNotify {
				notice.notified = true
			}
		}
	}
	return ret
}
