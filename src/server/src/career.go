package main

import (
	"math"
	"time"

	"./msg"
	"github.com/lzhig/rapidgo/base"
)

const (
	careerEventNetworkPacket base.EventID = iota
)

// CareerCenter type
type CareerCenter struct {
	base.EventSystem

	networkPacketHandler base.MessageHandlerImpl
}

// Init function
func (obj *CareerCenter) Init() {
	obj.EventSystem.Init(1024, false)
	obj.SetEventHandler(careerEventNetworkPacket, obj.handleEventNetworkPacket)

	obj.networkPacketHandler.Init()
	obj.networkPacketHandler.SetMessageHandler(msg.MessageID_CareerWinLoseData_Req, obj.handleCareerWinLoseData)
	obj.networkPacketHandler.SetMessageHandler(msg.MessageID_CareerRoomRecords_Req, obj.handleCareerRoomRecords)
}

func (obj *CareerCenter) handleEventNetworkPacket(args []interface{}) {
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

func (obj *CareerCenter) handleCareerWinLoseData(arg interface{}) {
	p := arg.(*ProtocolConnection)

	// p.userconn.mxJoinroom.Lock()
	// defer p.userconn.mxJoinroom.Unlock()

	// if p.userconn.conn == nil || p.userconn.user == nil {
	// 	return
	// }

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

	now := time.Now()
	end := base.GetTodayZeroClockTime(&now).AddDate(0, 0, 1) // 明天0点
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

func (obj *CareerCenter) handleCareerRoomRecords(arg interface{}) {
	p := arg.(*ProtocolConnection)

	// p.userconn.mxJoinroom.Lock()
	// defer p.userconn.mxJoinroom.Unlock()

	// if p.userconn.conn == nil || p.userconn.user == nil {
	// 	return
	// }

	rsp := &msg.Protocol{
		Msgid:                msg.MessageID_CareerRoomRecords_Rsp,
		CareerRoomRecordsRsp: &msg.CareerRoomRecordsRsp{Ret: msg.ErrorID_Ok},
	}
	defer p.userconn.sendProtocol(rsp)

	uid := p.userconn.user.uid

	req := p.p.CareerRoomRecordsReq

	if req.Count == 0 {
		return
	}

	now := time.Now()
	end := base.GetTodayZeroClockTime(&now).AddDate(0, 0, 1) // 明天0点
	begin := end.AddDate(0, 0, -int(req.Days))
	num := uint32(math.Min(float64(req.Count), float64(gApp.config.Room.CareerRoomRecordCountPerTime)))
	r, err := db.GetCareerRooms(uid, begin.Unix(), end.Unix(), req.Pos, num)
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
