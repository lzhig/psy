package main

import (
	"./msg"
	"github.com/lzhig/rapidgo/base"
)

var busyMessageHandlers = map[msg.MessageID]func(conn *Connection){
	msg.MessageID_Login_Req: func(conn *Connection) {
		conn.sendProtocol(&msg.Protocol{Msgid: msg.MessageID_Login_Rsp, LoginRsp: &msg.LoginRsp{Ret: msg.ErrorID_System_Busy}})
	},
	msg.MessageID_GetProfile_Req: func(conn *Connection) {
		conn.sendProtocol(&msg.Protocol{Msgid: msg.MessageID_GetProfile_Rsp, GetProfileRsp: &msg.GetProfileRsp{Ret: msg.ErrorID_System_Busy}})
	},
	msg.MessageID_GetPlayingRoom_Req: func(conn *Connection) {
		conn.sendProtocol(&msg.Protocol{Msgid: msg.MessageID_GetPlayingRoom_Rsp, GetPlayingRoomRsp: &msg.GetPlayingRoomRsp{Ret: msg.ErrorID_System_Busy}})
	},
	msg.MessageID_CreateRoom_Req: func(conn *Connection) {
		conn.sendProtocol(&msg.Protocol{Msgid: msg.MessageID_CreateRoom_Rsp, CreateRoomRsp: &msg.CreateRoomRsp{Ret: msg.ErrorID_System_Busy}})
	},
	msg.MessageID_JoinRoom_Req: func(conn *Connection) {
		conn.sendProtocol(&msg.Protocol{Msgid: msg.MessageID_JoinRoom_Rsp, JoinRoomRsp: &msg.JoinRoomRsp{Ret: msg.ErrorID_System_Busy}})
	},
	msg.MessageID_LeaveRoom_Req: func(conn *Connection) {
		conn.sendProtocol(&msg.Protocol{Msgid: msg.MessageID_LeaveRoom_Rsp, LeaveRoomRsp: &msg.LeaveRoomRsp{Ret: msg.ErrorID_System_Busy}})
	},
	msg.MessageID_ListRooms_Req: func(conn *Connection) {
		conn.sendProtocol(&msg.Protocol{Msgid: msg.MessageID_ListRooms_Rsp, ListRoomsRsp: &msg.ListRoomsRsp{Ret: msg.ErrorID_System_Busy}})
	},
	msg.MessageID_CloseRoom_Req: func(conn *Connection) {
		conn.sendProtocol(&msg.Protocol{Msgid: msg.MessageID_CloseRoom_Rsp, CloseRoomRsp: &msg.CloseRoomRsp{Ret: msg.ErrorID_System_Busy}})
	},
	msg.MessageID_SitDown_Req: func(conn *Connection) {
		conn.sendProtocol(&msg.Protocol{Msgid: msg.MessageID_SitDown_Rsp, SitDownRsp: &msg.SitDownRsp{Ret: msg.ErrorID_System_Busy}})
	},
	msg.MessageID_StandUp_Req: func(conn *Connection) {
		conn.sendProtocol(&msg.Protocol{Msgid: msg.MessageID_StandUp_Rsp, StandUpRsp: &msg.StandUpRsp{Ret: msg.ErrorID_System_Busy}})
	},
	msg.MessageID_AutoBanker_Req: func(conn *Connection) {
		conn.sendProtocol(&msg.Protocol{Msgid: msg.MessageID_AutoBanker_Rsp, AutoBankerRsp: &msg.AutoBankerRsp{Ret: msg.ErrorID_System_Busy}})
	},
	msg.MessageID_StartGame_Req: func(conn *Connection) {
		conn.sendProtocol(&msg.Protocol{Msgid: msg.MessageID_StartGame_Rsp, StartGameRsp: &msg.StartGameRsp{Ret: msg.ErrorID_System_Busy}})
	},
	msg.MessageID_Bet_Req: func(conn *Connection) {
		conn.sendProtocol(&msg.Protocol{Msgid: msg.MessageID_Bet_Rsp, BetRsp: &msg.BetRsp{Ret: msg.ErrorID_System_Busy}})
	},
	msg.MessageID_Combine_Req: func(conn *Connection) {
		conn.sendProtocol(&msg.Protocol{Msgid: msg.MessageID_Combine_Rsp, CombineRsp: &msg.CombineRsp{Ret: msg.ErrorID_System_Busy}})
	},
	msg.MessageID_GetScoreboard_Req: func(conn *Connection) {
		conn.sendProtocol(&msg.Protocol{Msgid: msg.MessageID_GetScoreboard_Rsp, GetScoreboardRsp: &msg.GetScoreboardRsp{Ret: msg.ErrorID_System_Busy}})
	},
	msg.MessageID_GetRoundHistory_Req: func(conn *Connection) {
		conn.sendProtocol(&msg.Protocol{Msgid: msg.MessageID_GetRoundHistory_Rsp, GetRoundHistoryRsp: &msg.GetRoundHistoryRsp{Ret: msg.ErrorID_System_Busy}})
	},
	msg.MessageID_SendDiamonds_Req: func(conn *Connection) {
		conn.sendProtocol(&msg.Protocol{Msgid: msg.MessageID_SendDiamonds_Rsp, SendDiamondsRsp: &msg.SendDiamondsRsp{Ret: msg.ErrorID_System_Busy}})
	},
	msg.MessageID_DiamondsRecords_Req: func(conn *Connection) {
		conn.sendProtocol(&msg.Protocol{Msgid: msg.MessageID_DiamondsRecords_Rsp, DiamondsRecordsRsp: &msg.DiamondsRecordsRsp{Ret: msg.ErrorID_System_Busy}})
	},
	msg.MessageID_CareerWinLoseData_Req: func(conn *Connection) {
		conn.sendProtocol(&msg.Protocol{Msgid: msg.MessageID_CareerWinLoseData_Rsp, CareerWinLoseDataRsp: &msg.CareerWinLoseDataRsp{Ret: msg.ErrorID_System_Busy}})
	},
	msg.MessageID_CareerRoomRecords_Req: func(conn *Connection) {
		conn.sendProtocol(&msg.Protocol{Msgid: msg.MessageID_CareerRoomRecords_Rsp, CareerRoomRecordsRsp: &msg.CareerRoomRecordsRsp{Ret: msg.ErrorID_System_Busy}})
	},
	msg.MessageID_GetNotices_Req: func(conn *Connection) {
		conn.sendProtocol(&msg.Protocol{Msgid: msg.MessageID_GetNotices_Rsp, GetNoticesRsp: &msg.GetNoticesRsp{Ret: msg.ErrorID_System_Busy}})
	},
}

func (obj *NetworkEngine) handleBusy(msgid msg.MessageID, conn *Connection) {
	if handler, ok := busyMessageHandlers[msgid]; ok {
		handler(conn)
	} else {
		base.LogError("No busy handler for msg:", msgid)
	}
}

func (obj *NetworkEngine) handle(msgid msg.MessageID, p *ProtocolConnection) {
	if p.userconn.user != nil {
		base.LogInfo("received msg:", p.p, ", uid:", p.userconn.user.uid)
	} else {
		base.LogInfo("received msg:", p.p)
	}

	switch msgid {
	case msg.MessageID_Login_Req,
		msg.MessageID_GetProfile_Req:

		if !loginService.Send(loginEventNetworkPacket, []interface{}{p}) {
			obj.handleBusy(msgid, p.userconn)
		}

	case msg.MessageID_GetNotices_Req:
		if !userManager.Send(userManagerEventNetworkPacket, []interface{}{p}) {
			obj.handleBusy(msgid, p.userconn)
		}

	case msg.MessageID_GetPlayingRoom_Req,
		msg.MessageID_CreateRoom_Req,
		msg.MessageID_JoinRoom_Req,
		msg.MessageID_LeaveRoom_Req,
		msg.MessageID_ListRooms_Req,
		msg.MessageID_CloseRoom_Req:

		if !roomManager.Send(roomManagerEventNetworkPacket, []interface{}{p}) {
			obj.handleBusy(msgid, p.userconn)
		}

	case msg.MessageID_SitDown_Req,
		msg.MessageID_StandUp_Req,
		msg.MessageID_AutoBanker_Req,
		msg.MessageID_StartGame_Req,
		msg.MessageID_Bet_Req,
		msg.MessageID_Combine_Req,
		msg.MessageID_GetScoreboard_Req,
		msg.MessageID_GetRoundHistory_Req,
		msg.MessageID_CloseResult_Req:

		if p.userconn.user.GetRoom() != nil {
			if !p.userconn.user.room.Send(roomEventNetworkPacket, []interface{}{p}) {
				obj.handleBusy(msgid, p.userconn)
			}
		} else {
			base.LogError("Cannot find room. proto:", p.p, ", uid:", p.userconn.user.uid)
		}

	case msg.MessageID_SendDiamonds_Req,
		msg.MessageID_DiamondsRecords_Req:
		if !diamondsCenter.Send(diamondsEventNetworkPacket, []interface{}{p}) {
			obj.handleBusy(msgid, p.userconn)
		}

	case msg.MessageID_CareerWinLoseData_Req,
		msg.MessageID_CareerRoomRecords_Req:
		if !careerCenter.Send(careerEventNetworkPacket, []interface{}{p}) {
			obj.handleBusy(msgid, p.userconn)
		}

	default:
		base.LogError("Cannot find dispatcher for msgid:", msg.MessageID_name[int32(p.p.Msgid)])
		p.userconn.Disconnect()
	}
}
