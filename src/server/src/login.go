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

	limitation base.Limitation
}

func (obj *LoginService) init() {
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
	obj.limitation.Acquire()
	defer obj.limitation.Release()

	rsp := &msg.Protocol{
		Msgid:    msg.MessageID_Login_Rsp,
		LoginRsp: &msg.LoginRsp{Ret: msg.ErrorID_Invalid_Params},
	}
	defer userconn.sendProtocol(rsp)

	errHandle := func(id msg.ErrorID) {
		rsp.LoginRsp.Ret = id
		// 5秒后断开连接
		time.AfterFunc(time.Second*5, func() {
			userconn.Disconnect()
		})
	}

	if p.LoginReq == nil {
		errHandle(msg.ErrorID_Invalid_Params)
		return
	}

	switch p.LoginReq.Type {

	case msg.LoginType_Facebook:
		req := p.LoginReq.Fb

		if !gApp.config.Debug {
			// call interface to verify valid account
			return
		}

		//query if the account is exist in db
		uid, err := userManager.fbUserExists(req.FbId, req.Name)
		if err != nil {
			errHandle(msg.ErrorID_DB_Error)
			return
		}
		if uid == 0 {
			// if doesn't exist, create account in db
			uid, err = userManager.fbUserCreate(req.FbId, req.Name, userconn)
			if err != nil {
				errHandle(msg.ErrorID_DB_Error)
				return
			}
		} else {
			// 如果通过验证，检查此用户是否在线，如果在线，将原连接断开，如果用户在房间内，给房间发送用户断线的消息
			if userManager.userIsConnected(uid) {
				userManager.setUserConnection(uid, userconn)
			} else if room, err := userManager.getRoomUserPlaying(uid); err == nil {
				// 如果是用户断线重连
				rsp.LoginRsp.RoomId = room.roomID
			} else if err := userManager.createUser(uid, userconn); err != nil {
				// create user
				errHandle(msg.ErrorID_DB_Error)
				return
			}
		}

		//uid := uint32(111)

		userconn.uid = uid
		userconn.name = req.Name
		rsp.LoginRsp.Ret = msg.ErrorID_Ok
		rsp.LoginRsp.Uid = uint32(uid)

	default:
		errHandle(msg.ErrorID_Invalid_Params)
		return
	}
}

func sendProtocol(conn *rapidnet.Connection, p *msg.Protocol) {
	data, err := proto.Marshal(p)
	if err != nil {
		logError("Failed to marshal. p:", p, "error:", err)
		return
	}
	conn.Send(data)
}
