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

	"./msg"
	"github.com/golang/protobuf/proto"
	"github.com/lzhig/rapidgo/rapidnet"
)

// LoginService 登录服务
type LoginService struct {
	protoChan chan *ProtocolConnection
}

func (obj *LoginService) init() {
	obj.protoChan = make(chan *ProtocolConnection, 128)
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
			obj.handle(p.conn, p.p)
		}
	}
}

func (obj *LoginService) handle(conn *rapidnet.Connection, p *msg.Protocol) {
	rsp := &msg.Protocol{
		Msgid:    msg.MessageID_Login_Rsp,
		LoginRsp: &msg.LoginRsp{Ret: msg.ErrorID_Invalid_Params},
	}

	switch p.LoginReq.Type {

	case msg.LoginType_Facebook:
		req := p.LoginReq.Fb

		if !gApp.config.Debug {
			// call interface to verify valid account
			return
		}

		//query if the account is exist in db
		uid := userManager.fbUserExists(req.FbId, req.Name)
		if uid == 0 {
			// if doesn't exist, create account in db
			uid = userManager.fbUserCreate(req.FbId, req.Name)
		} else {
			// load data from db
			userManager.fbUserLoad(uid)
		}
		if uid == 0 {
			rsp.LoginRsp.Ret = msg.ErrorID_DB_Error
		} else {
			rsp.LoginRsp.Ret = msg.ErrorID_Ok
		}
		rsp.LoginRsp.Uid = uint32(uid)

	default:
	}
	sendProtocol(conn, rsp)
}

func sendProtocol(conn *rapidnet.Connection, p *msg.Protocol) {
	data, err := proto.Marshal(p)
	if err != nil {
		logError("Failed to marshal. p:", p, "error:", err)
		return
	}
	conn.Send(data)
}
