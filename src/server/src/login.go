/**
 * @author [Bruce]
 * @email [lzhig@outlook.com]
 * @create date 2018-01-18 03:43:45
 * @modify date 2018-01-18 03:43:45
 * @desc [description]
 */
package main

import (
	"time"

	"./msg"
	"github.com/golang/protobuf/proto"
	"github.com/lzhig/rapidgo/base"
	"github.com/lzhig/rapidgo/rapidnet"
)

const (
	loginEventNetworkPacket base.EventID = iota
)

// LoginService 登录服务
type LoginService struct {
	base.EventSystem

	networkPacketHandler base.MessageHandlerImpl
	fbChecker            FacebookUserCheck

	//limitation base.Limitation
}

func (obj *LoginService) init() {
	obj.EventSystem.Init(1024, true)
	obj.SetEventHandler(loginEventNetworkPacket, obj.handleEventNetworkPacket)

	obj.networkPacketHandler.Init()
	obj.networkPacketHandler.SetMessageHandler(msg.MessageID_Login_Req, obj.handleLogin)
	obj.networkPacketHandler.SetMessageHandler(msg.MessageID_GetProfile_Req, obj.handleGetProfile)

	obj.fbChecker.Init()

	//obj.limitation.Init(16)
}

func (obj *LoginService) handleEventNetworkPacket(args []interface{}) {
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

func (obj *LoginService) handleLogin(arg interface{}) {
	pc := arg.(*ProtocolConnection)

	// pc.userconn.mxJoinroom.Lock()
	// defer pc.userconn.mxJoinroom.Unlock()

	// if pc.userconn.conn == nil {
	// 	return
	// }

	p := pc.p
	userconn := pc.userconn

	rsp := &msg.Protocol{
		Msgid:    msg.MessageID_Login_Rsp,
		LoginRsp: &msg.LoginRsp{Ret: msg.ErrorID_Invalid_Params},
	}

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
				u = userManager.CreateUser(user, userconn)
				// uid, err = userManager.fbUserCreate(fbid, name, userconn)
				if u == nil {
					errHandle(msg.ErrorID_DB_Error)
					userconn.sendProtocol(rsp)
					return
				}
			} else {
				// login不再返回room number
				u = userManager.LoadUser(user, uid, userconn)
				if u == nil {
					errHandle(msg.ErrorID_DB_Error)
					userconn.sendProtocol(rsp)
					return
				}
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

func (obj *LoginService) handleGetProfile(arg interface{}) {
	pc := arg.(*ProtocolConnection)
	userconn := pc.userconn

	// pc.userconn.mxJoinroom.Lock()
	// defer pc.userconn.mxJoinroom.Unlock()

	// if pc.userconn.conn == nil || pc.userconn.user == nil {
	// 	return
	// }

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
