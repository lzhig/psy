/**
 * @author [Bruce]
 * @email [lzhig@outlook.com]
 * @create date 2018-01-18 03:43:45
 * @modify date 2018-01-18 03:43:45
 * @desc [description]
 */
package main

import (
	"context"
	"time"

	"./msg"
	"github.com/golang/protobuf/proto"
	"github.com/lzhig/rapidgo/base"
	"github.com/lzhig/rapidgo/rapidnet"
)

// LoginService 登录服务
type LoginService struct {
	protoChan chan *ProtocolConnection

	fbChecker FacebookUserCheck

	limitation base.Limitation
}

func (obj *LoginService) init() {
	obj.fbChecker.Init()

	obj.limitation.Init(16)
	obj.protoChan = make(chan *ProtocolConnection, 16)
	ctx, _ := gApp.CreateCancelContext()
	gApp.GoRoutine(ctx, obj.loop)
}

// GetDispatchChan function
func (obj *LoginService) GetDispatchChan() chan<- *ProtocolConnection {
	return obj.protoChan
}

func (obj *LoginService) loop(ctx context.Context) {
	defer debug("exit LoginService goroutine")
	for {
		select {
		case <-ctx.Done():
			return

		case p := <-obj.protoChan:
			go obj.handle(p.userconn, p.p)
		}
	}
}

func (obj *LoginService) handle(userconn *userConnection, p *msg.Protocol) {
	defer base.LogPanic()
	base.LogInfo(p)
	obj.limitation.Acquire()
	defer obj.limitation.Release()

	switch p.Msgid {
	case msg.MessageID_Login_Req:
		obj.handleLogin(userconn, p)
	case msg.MessageID_GetProfile_Req:
		obj.handleGetProfile(userconn, p)

	}
}

func (obj *LoginService) handleLogin(userconn *userConnection, p *msg.Protocol) {

	rsp := &msg.Protocol{
		Msgid:    msg.MessageID_Login_Rsp,
		LoginRsp: &msg.LoginRsp{Ret: msg.ErrorID_Invalid_Params},
	}
	//defer userconn.sendProtocol(rsp)

	errHandle := func(id msg.ErrorID) {
		rsp.LoginRsp.Ret = id
		// 5秒后断开连接
		time.AfterFunc(time.Second*5, func() {
			userconn.Disconnect()
		})
	}

	if p.LoginReq == nil {
		errHandle(msg.ErrorID_Invalid_Params)
		userconn.sendProtocol(rsp)
		return
	}

	switch p.LoginReq.Type {

	case msg.LoginType_Facebook:
		req := p.LoginReq.Fb

		var u *User
		createUser := func(user PlatformUser) {
			//query if the account is exist in db
			uid, err := user.GetUID()
			if err != nil {
				errHandle(msg.ErrorID_DB_Error)
				userconn.sendProtocol(rsp)
				return
			}

			if uid == 0 {
				// if doesn't exist, create account in db
				u, err = userManager.CreateUser(user, userconn)
				// uid, err = userManager.fbUserCreate(fbid, name, userconn)
				if err != nil {
					errHandle(msg.ErrorID_DB_Error)
					userconn.sendProtocol(rsp)
					return
				}
			} else {
				// 如果通过验证，检查此用户是否在线，如果在线，将原连接断开，如果用户在房间内，给房间发送用户断线的消息
				u = userManager.GetUser(uid)
				if u == nil {
					u, err = userManager.LoadUser(user, uid, userconn)
					if err != nil {
						errHandle(msg.ErrorID_DB_Error)
						userconn.sendProtocol(rsp)
						return
					}
				} else {
					if userManager.userIsConnected(uid) {
						userManager.setUserConnection(uid, userconn)
					} else if room, err := userManager.getRoomUserPlaying(uid); err == nil {
						// 如果是用户断线重连
						rsp.LoginRsp.RoomNumber = roomNumberGenerator.decode(room.number)
					}

					u.platformUser = user
					u.name = user.GetName()

				}

				u.conn = userconn
				db.UpdateName(uid, u.name)
			}

			userconn.user = u

			rsp.LoginRsp.Ret = msg.ErrorID_Ok
			rsp.LoginRsp.Uid = uint32(uid)
			rsp.LoginRsp.Name = u.name
			rsp.LoginRsp.Avatar = u.avatar
			userconn.sendProtocol(rsp)
		}

		if !gApp.config.Debug {
			// call interface to verify valid account
			user := &FacebookUser{Fbid: req.FbId, Token: req.Token}
			obj.fbChecker.Check(user, func(user *FacebookUser, result bool, reason string) {
				if !result {
					errHandle(msg.ErrorID_Login_Facebook_Failed)
					base.LogError(user, "failed to login facebook. reason:", reason)
					userconn.sendProtocol(rsp)
					return
				}

				createUser(user)
			})
			return
		}

		//debug
		user := &LocalUser{id: req.FbId, name: req.FbId}
		createUser(user)

	default:
		errHandle(msg.ErrorID_Invalid_Params)
		return
	}
}

func (obj *LoginService) handleGetProfile(userconn *userConnection, p *msg.Protocol) {

	rsp := &msg.Protocol{
		Msgid:         msg.MessageID_GetProfile_Rsp,
		GetProfileRsp: &msg.GetProfileRsp{Ret: msg.ErrorID_Ok},
	}
	defer userconn.sendProtocol(rsp)

	name, signture, avatar, diamonds, err := db.GetUserProfile(userconn.user.uid)
	if err != nil {
		rsp.GetProfileRsp.Ret = msg.ErrorID_DB_Error
		return
	}
	userconn.user.diamonds = diamonds

	rsp.GetProfileRsp.Uid = userconn.user.uid
	rsp.GetProfileRsp.Name = name
	rsp.GetProfileRsp.Signture = signture
	rsp.GetProfileRsp.Avatar = avatar
	rsp.GetProfileRsp.Diamonds = diamonds
}

func sendProtocol(conn *rapidnet.Connection, p *msg.Protocol) {
	if conn == nil {
		return
	}
	base.LogInfo(p)
	data, err := proto.Marshal(p)
	if err != nil {
		base.LogError("Failed to marshal. p:", p, "error:", err)
		return
	}
	conn.Send(data)
}
