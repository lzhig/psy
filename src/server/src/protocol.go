package main

import (
	"./msg"
	"github.com/lzhig/rapidgo/base"
)

func (obj *NetworkEngine) handle(msgid msg.MessageID, p *ProtocolConnection) {
	switch msgid {
	case msg.MessageID_Login_Req,
		msg.MessageID_GetProfile_Req:

		loginService.Send(loginEventNetworkPacket, []interface{}{p})

	case msg.MessageID_GetPlayingRoom_Req,
		msg.MessageID_CreateRoom_Req,
		msg.MessageID_JoinRoom_Req,
		msg.MessageID_LeaveRoom_Req,
		msg.MessageID_ListRooms_Req,
		msg.MessageID_CloseRoom_Req:

		roomManager.Send(roomManagerEventNetworkPacket, []interface{}{p})

	case msg.MessageID_SitDown_Req,
		msg.MessageID_StandUp_Req,
		msg.MessageID_AutoBanker_Req,
		msg.MessageID_StartGame_Req,
		msg.MessageID_Bet_Req,
		msg.MessageID_Combine_Req,
		msg.MessageID_GetScoreboard_Req,
		msg.MessageID_GetRoundHistory_Req:

		if p.userconn.user.room != nil {
			p.userconn.user.room.Send(roomEventNetworkPacket, []interface{}{p})
		} else {
			base.LogError("Cannot find room. proto:", p)
		}

	case msg.MessageID_SendDiamonds_Req,
		msg.MessageID_DiamondsRecords_Req:
		diamondsCenter.Send(diamondsEventNetworkPacket, []interface{}{p})

	case msg.MessageID_CareerWinLoseData_Req,
		msg.MessageID_CareerRoomRecords_Req:
		careerCenter.Send(careerEventNetworkPacket, []interface{}{p})

	default:
		base.LogError("Cannot find dispatcher for msgid:", msg.MessageID_name[int32(p.p.Msgid)])
		p.userconn.Disconnect()
	}
}
