package main

import (
	"math"
	"time"

	"./msg"
	"github.com/lzhig/rapidgo/base"
)

const (
	diamondsEventNetworkPacket base.EventID = iota
)

// DiamondsCenter 钻石交易
type DiamondsCenter struct {
	base.EventSystem

	networkPacketHandler base.MessageHandlerImpl
	freeDiamonds         FreeDiamonds
}

func (obj *DiamondsCenter) init() {
	obj.EventSystem.Init(1024, false)
	obj.SetEventHandler(diamondsEventNetworkPacket, obj.handleEventNetworkPacket)

	obj.networkPacketHandler.Init()
	obj.networkPacketHandler.SetMessageHandler(msg.MessageID_SendDiamonds_Req, obj.handleSendDiamondsReq)
	obj.networkPacketHandler.SetMessageHandler(msg.MessageID_DiamondsRecords_Req, obj.handleDiamondsRecordsReq)
}

func (obj *DiamondsCenter) handleEventNetworkPacket(args []interface{}) {
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

func (obj *DiamondsCenter) handleSendDiamondsReq(arg interface{}) {
	p := arg.(*ProtocolConnection)

	// p.userconn.mxJoinroom.Lock()
	// defer p.userconn.mxJoinroom.Unlock()

	// if p.userconn.conn == nil || p.userconn.user == nil {
	// 	return
	// }

	rsp := &msg.Protocol{
		Msgid:           msg.MessageID_SendDiamonds_Rsp,
		SendDiamondsRsp: &msg.SendDiamondsRsp{Ret: msg.ErrorID_Ok},
	}
	defer p.userconn.sendProtocol(rsp)

	req := p.p.SendDiamondsReq

	if p.userconn.user.uid == req.Uid {
		rsp.SendDiamondsRsp.Ret = msg.ErrorID_SendDiamonds_Cannot_Self
		return
	}

	exist, err := db.ExistUser(req.Uid)
	if err != nil {
		rsp.SendDiamondsRsp.Ret = msg.ErrorID_DB_Error
		return
	}
	if !exist {
		rsp.SendDiamondsRsp.Ret = msg.ErrorID_SendDiamonds_No_User
		return
	}

	diamondsWithFee := uint32(math.Ceil(float64(req.Diamonds) * (float64(1) + gApp.config.Diamonds.SendDiamondsFee)))
	diamondsFee := diamondsWithFee - req.Diamonds
	base.LogInfo("diamonds:", req.Diamonds, ", fee rate:", gApp.config.Diamonds.SendDiamondsFee, "Fee:", diamondsFee)

	newDiamonds, err := db.PayDiamonds(p.userconn.user.uid, req.Uid, req.Diamonds, diamondsFee, obj.freeDiamonds.GetDiamondsKept())
	if err == ErrorNotEnoughDiamonds {
		rsp.SendDiamondsRsp.Ret = msg.ErrorID_SendDiamonds_Not_Enough_Diamonds
		return
	} else if err != nil {
		rsp.SendDiamondsRsp.Ret = msg.ErrorID_DB_Error
		return
	}
	rsp.SendDiamondsRsp.Diamonds = newDiamonds

	// 向to玩家发送通知
	conn := userManager.GetUserConn(req.Uid)
	if conn != nil {
		conn.sendProtocol(
			&msg.Protocol{
				Msgid: msg.MessageID_ReceiveDiamonds_Notify,
				ReceiveDiamondsNotify: &msg.ReceiveDiamondsNotify{Diamonds: req.Diamonds}})
	}
}

func (obj *DiamondsCenter) handleDiamondsRecordsReq(arg interface{}) {
	p := arg.(*ProtocolConnection)

	rsp := &msg.Protocol{
		Msgid:              msg.MessageID_DiamondsRecords_Rsp,
		DiamondsRecordsRsp: &msg.DiamondsRecordsRsp{Ret: msg.ErrorID_Ok},
	}
	defer p.userconn.sendProtocol(rsp)

	req := p.p.DiamondsRecordsReq
	// '00:00:00'
	begin, err := time.Parse("2006-1-2", req.BeginTime)
	if err != nil {
		rsp.DiamondsRecordsRsp.Ret = msg.ErrorID_DiamondsRecords_Invalid_Date_Format
		return
	}

	// '23:59:59'
	end, err := time.Parse("2006-1-2", req.EndTime)
	end = end.AddDate(0, 0, 1)
	if err != nil {
		rsp.DiamondsRecordsRsp.Ret = msg.ErrorID_DiamondsRecords_Invalid_Date_Format
		return
	}

	if end.Before(begin) {
		rsp.DiamondsRecordsRsp.Ret = msg.ErrorID_DiamondsRecords_Invalid_End_Time
		return
	}

	if end.After(begin.AddDate(0, 0, 30)) {
		rsp.DiamondsRecordsRsp.Ret = msg.ErrorID_DiamondsRecords_Exceed_30Days
		return
	}

	records, err := db.GetDiamondRecords(p.userconn.user.uid, begin.Unix(), end.Unix())
	if err != nil {
		rsp.DiamondsRecordsRsp.Ret = msg.ErrorID_DB_Error
		return
	}
	rsp.DiamondsRecordsRsp.Records = records

	// user info
	users := make(map[uint32]*msg.UserNameAvatar)
	uids := make([]uint32, 0, len(records))
	for _, record := range records {
		if _, ok := users[record.Uid]; !ok {
			// 获取用户信息
			if name, avatar, ok := userManager.GetUserNameAvatar(record.Uid); ok {
				users[record.Uid] = &msg.UserNameAvatar{
					Uid:    record.Uid,
					Name:   name,
					Avatar: avatar,
				}
				continue
			}

			uids = append(uids, record.Uid)
		}
	}

	result, err := db.GetUsersNameAvatar(uids)
	if err != nil {
		rsp.DiamondsRecordsRsp.Ret = msg.ErrorID_DB_Error
		return
	}
	rsp.DiamondsRecordsRsp.Users = make([]*msg.UserNameAvatar, len(result)+len(users))
	i := 0
	for _, user := range users {
		rsp.DiamondsRecordsRsp.Users[i] = user
		i++
	}
	for _, user := range result {
		rsp.DiamondsRecordsRsp.Users[i] = user
		i++
	}
}

// FreeDiamonds 类型负责赠送免费钻石
// 1. 注册时赠送钻石
// 2. 每天第一次登录时，若用户钻石低于某个数值时，赠送钻石
// 3. 发送钻石时，用户需要保留若干数值的钻石不能发送
type FreeDiamonds struct {
}

// GetFreeDiamondsWhenRegister 返回注册时赠送钻石数
func (obj *FreeDiamonds) GetFreeDiamondsWhenRegister() uint32 {
	return gApp.config.Diamonds.InitDiamonds
}

// GiveFreeDiamondsEveryDay 每天第一次登录时，若用户钻石低于某个数值时，赠送钻石
func (obj *FreeDiamonds) GiveFreeDiamondsEveryDay(uid uint32) {
	db.GiveFreeDiamonds(uid, gApp.config.Diamonds.InitDiamonds)
}

// GetDiamondsKept 返回用户发送钻石时，需要保留的钻石数
func (obj *FreeDiamonds) GetDiamondsKept() uint32 {
	return gApp.config.Diamonds.InitDiamonds
}
