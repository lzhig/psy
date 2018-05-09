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
	uid    uint32
	name   string // 名字
	avatar string // 头像

	diamondsMutex sync.RWMutex
	diamonds      uint32 // 钻石
	diamondsChan  chan struct{}

	prePayDiamonds uint32 // 预扣钻石，在处理多人扣钻石时，先预扣钻石，等所有人都预扣钻石成功后，再进行真实扣除

	platformUser PlatformUser

	conn *Connection // 用户当前的连接
	room *Room       // 所在房间
}

// GetName 返回名字
func (obj *User) GetName() string {
	obj.RLock()
	defer obj.RUnlock()
	return obj.name
}

// GetAvatar 返回头像
func (obj *User) GetAvatar() string {
	obj.RLock()
	defer obj.RUnlock()
	return obj.avatar
}

// GetRoom 返回房间
func (obj *User) GetRoom() *Room {
	obj.RLock()
	defer obj.RUnlock()
	return obj.room
}

// SendProtocol 发送消息
func (obj *User) SendProtocol(p *msg.Protocol) {
	obj.RLock()
	if obj.conn != nil {
		obj.conn.sendProtocol(p)
	}
	obj.RUnlock()
}

// EnterRoom 进入房间
func (obj *User) EnterRoom(room *Room) bool {
	obj.Lock()
	defer obj.Unlock()
	if obj.room != nil {
		base.LogError("the user is still in the room, roomid: ", obj.room.roomID)
		return false

	}
	obj.room = room
	return true
}

func (obj *User) LeaveRoom() {
	obj.Lock()
	defer obj.Unlock()
	if obj.room == nil {
		base.LogError("the user is not yet in a room.")
	}
	obj.room = nil

	if obj.conn == nil {
		userManager.DeleteUser(obj.uid)
	}
}

// KickUser 踢离用户
func (obj *User) KickUser(kickType msg.KickType) {
	obj.Lock()
	defer obj.Unlock()

	if obj.conn != nil {
		// 发送被踢消息
		p := &msg.Protocol{
			Msgid:      msg.MessageID_Kick_Notify,
			KickNotify: &msg.KickNotify{Type: kickType},
		}
		obj.conn.sendProtocol(p)

		if obj.room != nil {
			// 向房间发送此用户断线消息
			obj.room.Send(roomEventUserDisconnect, []interface{}{obj.uid})
		}

		oldConn := obj.conn
		obj.conn = nil

		// 5秒后断开连接
		time.AfterFunc(time.Second*5, func() {
			oldConn.Disconnect()
		})
	}
}

func (obj *User) Disconnect(conn *Connection) {
	obj.Lock()
	defer obj.Unlock()

	if obj.conn == conn {
		if obj.room != nil {
			base.LogInfo("disconnected. uid:", obj.uid, ", room_id:", obj.room.roomID)
			// 向房间发送此用户断线消息
			obj.room.Send(roomEventUserDisconnect, []interface{}{obj.uid})
			obj.conn = nil
		} else {
			userManager.DeleteUser(obj.uid)
		}
	}
}

func (obj *User) BeginConsumeDiamonds() bool {
	select {
	case obj.diamondsChan <- struct{}{}:
		return true
	default:
		return false
	}
}

func (obj *User) EndConsumeDiamonds() {
	<-obj.diamondsChan
}

func (obj *User) GetDiamonds() uint32 {
	obj.diamondsMutex.RLock()
	defer obj.diamondsMutex.RUnlock()
	return obj.diamonds
}

func (obj *User) AddDiamonds(diamonds uint32) {
	obj.diamondsMutex.Lock()
	defer obj.diamondsMutex.Unlock()
	obj.diamonds += diamonds
	base.LogInfo("AddDiamonds, uid:", obj.uid, ", diamonds to add:", diamonds, ", new diamonds:", obj.diamonds)
}

func (obj *User) SubDiamonds(diamonds uint32) bool {
	obj.diamondsMutex.Lock()
	defer obj.diamondsMutex.Unlock()
	if obj.diamonds < diamonds {
		base.LogError("SubDiamonds - not enough diamonds. uid:", obj.uid, ", user's diamonds:", obj.diamonds, ", diamonds to sub:", diamonds)
		return false
	}
	obj.diamonds -= diamonds
	base.LogInfo("SubDiamonds, uid:", obj.uid, ", diamonds to sub:", diamonds, ", new diamonds:", obj.diamonds)
	return true
}

const (
	userManagerEventNetworkPacket base.EventID = iota
	userManagerEventNoticeTimer
)

// UserManager type
type UserManager struct {
	base.EventSystem

	usersMutex sync.RWMutex
	users      map[uint32]*User

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

// GetUser 获取用户
func (obj *UserManager) GetUser(uid uint32) *User {
	obj.usersMutex.RLock()
	defer obj.usersMutex.RUnlock()
	user, ok := obj.users[uid]
	if ok {
		return user
	}
	return nil
}

// CreateUser 创建用户对象
func (obj *UserManager) CreateUser(pu PlatformUser, conn *Connection) (*User, error) {
	obj.usersMutex.Lock()
	defer obj.usersMutex.Unlock()

	uid, err := db.CreateUser(pu.GetName(), pu.GetAvatarURL(), diamondsCenter.freeDiamonds.GetFreeDiamondsWhenRegister(), pu.GetPlatformID())
	if err != nil {
		return nil, err
	}
	if err := pu.SaveToDB(uid); err != nil {
		base.LogError("[UserManager][CreateUser] Failed to Save to db. uid:", uid, ", user:", pu)
		return nil, err
	}
	user, err := obj.createUser(uid, conn)
	if err != nil {
		return nil, err
	}
	user.platformUser = pu
	return user, nil
}

// KickUser 踢离用户
func (obj *UserManager) KickUser(uid uint32) error {
	obj.usersMutex.RLock()
	defer obj.usersMutex.RUnlock()

	user, ok := obj.users[uid]
	if !ok {
		return fmt.Errorf("The user: %d not exist", uid)
	}

	user.KickUser(msg.KickType_GM)
	return nil
}

// KickUsersNotInRoom 踢离不在房间中的所有玩家
func (obj *UserManager) KickUsersNotInRoom() {
	obj.usersMutex.RLock()
	defer obj.usersMutex.RUnlock()

	for _, user := range obj.users {
		if user.room == nil {
			user.KickUser(msg.KickType_StopServer)
		}
	}
}

// LoadUser 加载用户
func (obj *UserManager) LoadUser(pu PlatformUser, uid uint32, conn *Connection) (*User, error) {
	obj.usersMutex.Lock()
	defer obj.usersMutex.Unlock()

	// 如果通过验证，检查此用户是否在线，如果在线，将原连接断开，如果用户在房间内，给房间发送用户断线的消息
	user, ok := obj.users[uid]
	var err error
	if ok {
		user.KickUser(msg.KickType_Login)

		user.platformUser = pu
		user.name = pu.GetName()
	} else {
		user, err = obj.loadUser(pu, uid, conn)
		if err != nil {
			return nil, err
		}
	}

	user.conn = conn
	db.UpdateName(uid, user.name)
	return user, err
}

func (obj *UserManager) createUser(uid uint32, conn *Connection) (*User, error) {
	diamondsCenter.freeDiamonds.GiveFreeDiamondsEveryDay(uid)
	name, _, avatar, diamonds, status, err := db.GetUserProfile(uid)
	if err != nil {
		return nil, err
	}
	// 账号被封
	if status == 1 {
		return nil, errUserBanned
	}

	user := &User{uid: uid, name: name, avatar: avatar, diamonds: diamonds, conn: conn, diamondsChan: make(chan struct{}, 1)}
	obj.users[uid] = user
	return user, nil
}

func (obj *UserManager) loadUser(pu PlatformUser, uid uint32, conn *Connection) (*User, error) {
	user, err := obj.createUser(uid, conn)
	if err == nil {
		user.platformUser = pu
		user.name = pu.GetName()
		return user, err
	}
	return nil, err
}

func (obj *UserManager) DeleteUser(uid uint32) bool {
	obj.usersMutex.Lock()
	defer obj.usersMutex.Unlock()

	if _, ok := obj.users[uid]; ok {
		delete(obj.users, uid)
		base.LogInfo("delete uid:", uid)
		return true
	}
	return false
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
	obj.usersMutex.RLock()
	defer obj.usersMutex.RUnlock()

	// load notice
	if err := obj.notices.Load(gApp.config.Notice.File); err != nil {
		return err
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
	return nil
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
