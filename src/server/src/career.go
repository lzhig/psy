package main

import (
	"time"

	"./msg"
	"github.com/lzhig/rapidgo/base"
)

type CareerCenter struct {
	base.MessageHandlerImpl
}

func (obj *CareerCenter) Init() {
	obj.MessageHandlerImpl.Init(16)

	obj.AddMessageHandler(msg.MessageID_CareerWinLoseData_Req, obj.handleCareerWinLoseData)
	obj.AddMessageHandler(msg.MessageID_CareerRoomRecords_Req, obj.handleCareerRoomRecords)
	obj.AddBusyHandler(msg.MessageID_CareerWinLoseData_Req, obj.handleBusyCareerWinLoseData)
	obj.AddBusyHandler(msg.MessageID_CareerRoomRecords_Req, obj.handleBusyCareerRoomRecords)
}

func (obj *CareerCenter) Handle(arg interface{}) {
	p := arg.(*ProtocolConnection)
	obj.MessageHandlerImpl.Handle(p.p.Msgid, p)
}

func (obj *CareerCenter) handleCareerWinLoseData(arg interface{}) {
	p := arg.(*ProtocolConnection)
	rsp := &msg.Protocol{
		Msgid:                msg.MessageID_CareerWinLoseData_Rsp,
		CareerWinLoseDataRsp: &msg.CareerWinLoseDataRsp{Ret: msg.ErrorID_Ok},
	}
	defer p.userconn.sendProtocol(rsp)

	req := p.p.CareerWinLoseDataReq

	if len(req.Days) == 0 {
		rsp.CareerWinLoseDataRsp.Ret = msg.ErrorID_Invalid_Params
		return
	}

	uid := p.userconn.user.uid

	t, _ := time.ParseInLocation("2006-1-2 15:4:5", "0001-1-1 0:0:0", time.Local)
	y, m, d := time.Now().AddDate(0, 0, 1).Date()
	end := t.AddDate(y-1, int(m)-1, d-1)
	rsp.CareerWinLoseDataRsp.Data = make([]*msg.CareerWinLoseDataItem, len(req.Days))
	for ndx, days := range req.Days {
		begin := end.AddDate(0, 0, -int(days))

		r, err := db.GetCareerWinLoseData(uid, begin.Unix(), end.Unix())
		if err != nil {
			rsp.CareerWinLoseDataRsp.Ret = msg.ErrorID_DB_Error
			return
		}
		item := &msg.CareerWinLoseDataItem{}
		for _, score := range r {
			if score > 0 {
				item.Win += uint32(score)
			} else {
				item.Lose += uint32(-score)
			}
		}
		rsp.CareerWinLoseDataRsp.Data[ndx] = item
	}
}

func (obj *CareerCenter) handleBusyCareerWinLoseData(arg interface{}) {
	p := arg.(*ProtocolConnection)
	rsp := &msg.Protocol{
		Msgid:                msg.MessageID_CareerWinLoseData_Rsp,
		CareerWinLoseDataRsp: &msg.CareerWinLoseDataRsp{Ret: msg.ErrorID_System_Busy},
	}
	p.userconn.sendProtocol(rsp)
}

func (obj *CareerCenter) handleCareerRoomRecords(arg interface{}) {
	p := arg.(*ProtocolConnection)
	rsp := &msg.Protocol{
		Msgid:                msg.MessageID_CareerRoomRecords_Rsp,
		CareerRoomRecordsRsp: &msg.CareerRoomRecordsRsp{Ret: msg.ErrorID_Ok},
	}
	defer p.userconn.sendProtocol(rsp)

	uid := p.userconn.user.uid

	req := p.p.CareerRoomRecordsReq
	t, _ := time.ParseInLocation("2006-1-2 15:4:5", "0001-1-1 0:0:0", time.Local)
	y, m, d := time.Now().AddDate(0, 0, 1).Date()
	end := t.AddDate(y-1, int(m)-1, d-1)
	begin := end.AddDate(0, 0, -int(req.Days))
	r, err := db.GetCareerRooms(uid, begin.Unix(), end.Unix())
	if err != nil {
		rsp.CareerRoomRecordsRsp.Ret = msg.ErrorID_DB_Error
		return
	}
	rsp.CareerRoomRecordsRsp.Records = r
	for _, room := range r {
		scoreboard, err := db.loadScoreboard(room.RoomId)
		if err != nil {
			rsp.CareerRoomRecordsRsp.Ret = msg.ErrorID_DB_Error
			return
		}
		room.Items = scoreboard
	}
}

func (obj *CareerCenter) handleBusyCareerRoomRecords(arg interface{}) {
	p := arg.(*ProtocolConnection)
	rsp := &msg.Protocol{
		Msgid:                msg.MessageID_CareerRoomRecords_Rsp,
		CareerRoomRecordsRsp: &msg.CareerRoomRecordsRsp{Ret: msg.ErrorID_System_Busy},
	}
	p.userconn.sendProtocol(rsp)
}
