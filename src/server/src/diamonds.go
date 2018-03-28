package main

import (
	"context"
	"time"

	"./msg"
	"github.com/lzhig/rapidgo/base"
)

// DiamondsCenter 钻石交易
type DiamondsCenter struct {
	protoChan chan *ProtocolConnection
}

func (obj *DiamondsCenter) init() {
	obj.protoChan = make(chan *ProtocolConnection, 16)
	ctx, _ := gApp.CreateCancelContext()
	gApp.GoRoutine(ctx, obj.loop)
}

func (obj *DiamondsCenter) loop(ctx context.Context) {
	defer debug("exit DiamondsCenter goroutine")
	for {
		select {
		case <-ctx.Done():
			return

		case p := <-obj.protoChan:
			switch p.p.Msgid {
			case msg.MessageID_SendDiamonds_Req:
				obj.handleSendDiamondsReq(p)
			case msg.MessageID_DiamondsRecords_Req:
				obj.handleDiamondsRecordsReq(p)
			default:
				base.LogError("cannot find dispatcher for msgid:", msg.MessageID_name[int32(p.p.Msgid)])
			}
		}
	}
}

func (obj *DiamondsCenter) handle(p *ProtocolConnection) {
	select {
	case obj.protoChan <- p:
	default:
		obj.handleBusy(p)
	}
}

func (obj *DiamondsCenter) handleBusy(p *ProtocolConnection) {
	rsp := &msg.Protocol{}
	switch p.p.Msgid {
	case msg.MessageID_SendDiamonds_Req:
		rsp.Msgid = msg.MessageID_SendDiamonds_Rsp
		rsp.SendDiamondsRsp = &msg.SendDiamondsRsp{Ret: msg.ErrorID_System_Busy}
	case msg.MessageID_DiamondsRecords_Req:
		rsp.Msgid = msg.MessageID_DiamondsRecords_Rsp
		rsp.DiamondsRecordsRsp = &msg.DiamondsRecordsRsp{Ret: msg.ErrorID_System_Busy}
	default:
		base.LogError("cannot find dispatcher for msgid:", msg.MessageID_name[int32(p.p.Msgid)])
	}
}

// Pay function
func (obj *DiamondsCenter) Pay(from, to uint32, diamonds uint32) error {
	return db.PayDiamonds(from, to, diamonds)
}

func (obj *DiamondsCenter) handleSendDiamondsReq(p *ProtocolConnection) {
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

	err = obj.Pay(p.userconn.user.uid, req.Uid, req.Diamonds)
	if err == ErrorNotEnoughDiamonds {
		rsp.SendDiamondsRsp.Ret = msg.ErrorID_SendDiamonds_Not_Enough_Diamonds
		return
	} else if err != nil {
		rsp.SendDiamondsRsp.Ret = msg.ErrorID_DB_Error
		return
	}
}

func (obj *DiamondsCenter) handleDiamondsRecordsReq(p *ProtocolConnection) {
	rsp := &msg.Protocol{
		Msgid:              msg.MessageID_DiamondsRecords_Rsp,
		DiamondsRecordsRsp: &msg.DiamondsRecordsRsp{Ret: msg.ErrorID_Ok},
	}
	defer p.userconn.sendProtocol(rsp)

	req := p.p.DiamondsRecordsReq
	begin, err := time.Parse("2006-1-2", req.BeginTime)
	if err != nil {
		rsp.DiamondsRecordsRsp.Ret = msg.ErrorID_DiamondsRecords_Invalid_Date_Format
		return
	}

	end, err := time.Parse("2006-1-2", req.EndTime)
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
			if user := userManager.GetUser(record.Uid); user != nil {
				users[record.Uid] = &msg.UserNameAvatar{
					Uid:    record.Uid,
					Name:   user.name,
					Avatar: user.avatar,
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
