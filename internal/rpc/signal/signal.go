package signal

import (
	"context"
	"fmt"
	"github.com/openimsdk/openmeeting-server/pkg/common/storage/model"
	"github.com/openimsdk/protocol/constant"
	"github.com/openimsdk/protocol/openmeeting/meeting"
	"github.com/openimsdk/protocol/openmeeting/signal"
	"github.com/openimsdk/tools/errs"
	"github.com/openimsdk/tools/log"
)

func (m *signalServer) SignalMessageAssemble(ctx context.Context, req *signal.SignalMessageAssembleReq) (*signal.SignalMessageAssembleResp, error) {
	switch payload := req.SignalReq.Payload.(type) {
	case *signal.SignalReq_Invite:
		if err := validateInvitation(payload.Invite.Invitation); err != nil {
			return nil, err
		}
		userInfo, err := m.userRpc.GetUserInfo(ctx, payload.Invite.Invitation.InviterUserID)
		if err != nil {
			return nil, errs.WrapMsg(err, "get user info failed")
		}
		participant := generateParticipantMetaData(userInfo)
		resp, err := m.rtc.InviteInUsers(ctx, payload.Invite, participant, payload.Invite.Invitation)
		if err != nil {
			return nil, err
		}
		signalModel := signalInvitationPb2DB(payload.Invite, resp.Sid)
		// db record the roomid
		if err := m.signalDB.CreateSignal(ctx, signalModel, payload.Invite.Invitation.InviteeUserIDList); err != nil {
			return nil, err
		}
		// genarate the msgData
		data, err := wrapperMsgData(ctx, payload.Invite.Invitation.InviterUserID, payload.Invite.Invitation.InviteeUserIDList[0], "",
			payload.Invite.Invitation.SessionType, constant.SignalingNotification,
			payload.Invite.Invitation.PlatformID, payload.Invite.OfflinePushInfo, &signal.SignalReq{Payload: payload})
		if err != nil {
			return nil, err
		}
		return &signal.SignalMessageAssembleResp{
			SignalResp: &signal.SignalResp{
				Payload: &signal.SignalResp_Invite{Invite: resp.SignalInviteResp}},
			MsgData: data,
		}, nil
	case *signal.SignalReq_InviteInGroup:
		if err := validateInvitation(payload.InviteInGroup.Invitation); err != nil {
			return nil, err
		}
		userInfo, err := m.userRpc.GetUserInfo(ctx, payload.InviteInGroup.Invitation.InviterUserID)
		if err != nil {
			return nil, errs.WrapMsg(err, "SignalReq_InviteInGroup get user info failed")
		}

		meetingDBInfo, err := m.generateMeetingDBData4Signal(ctx, &signal.SignalInviteInGroupReq{
			Invitation: payload.InviteInGroup.Invitation,
		})
		if err != nil {
			return nil, errs.WrapMsg(err, "generate group meeting data failed")
		}
		metaData, err := m.generateMeetingMetaData4Signal(ctx, meetingDBInfo, &meeting.MeetingSetting{
			DisableCameraOnJoin: payload.InviteInGroup.Invitation.MediaType != "video",
		})
		if err != nil {
			return nil, errs.WrapMsg(err, "generate group meeting meta data failed")
		}
		participant := generateParticipantMetaData(userInfo)
		resp, err := m.rtc.InviteInGroup(ctx, payload.InviteInGroup, metaData, participant, payload.InviteInGroup.Invitation)
		if err != nil {
			return nil, err
		}
		signalModel := signalGroupInvitationPb2DB(payload.InviteInGroup, resp.Sid)
		if err := m.signalDB.CreateSignal(ctx, signalModel, payload.InviteInGroup.Invitation.InviteeUserIDList); err != nil {
			return nil, err
		}
		// genarate the msgData
		data, err := wrapperMsgData(ctx, payload.InviteInGroup.Invitation.InviterUserID, "",
			payload.InviteInGroup.Invitation.GroupID, payload.InviteInGroup.Invitation.SessionType, constant.SignalingNotification,
			payload.InviteInGroup.Invitation.PlatformID, payload.InviteInGroup.OfflinePushInfo, &signal.SignalReq{Payload: payload})
		if err != nil {
			return nil, err
		}
		return &signal.SignalMessageAssembleResp{
			SignalResp: &signal.SignalResp{
				Payload: &signal.SignalResp_InviteInGroup{InviteInGroup: resp.SignalInviteInGroupResp}},
			MsgData: data,
		}, nil
	case *signal.SignalReq_Cancel:
		// only the inviter has right to cancel, if cancel happened, the inviter and invitee will receive the cancel signal callback
		log.ZDebug(ctx, "SignalMessageAssemble SignalReq_Cancel")
		resp, err := m.rtc.Cancel(ctx, payload.Cancel)
		if err != nil {
			return nil, err
		}
		err = m.signalDB.CancelSignalInvitation(ctx, resp.Sid, payload.Cancel.Invitation.RoomID, payload.Cancel.Invitation.InviterUserID)
		if err != nil {
			return nil, err
		}
		// genarate the msgData
		data, err := wrapperMsgData(ctx, payload.Cancel.Invitation.InviterUserID,
			payload.Cancel.Invitation.InviteeUserIDList[0], payload.Cancel.Invitation.GroupID,
			payload.Cancel.Invitation.SessionType, constant.SignalingNotification,
			payload.Cancel.Invitation.PlatformID, payload.Cancel.OfflinePushInfo, &signal.SignalReq{Payload: payload})
		if err != nil {
			return nil, err
		}
		return &signal.SignalMessageAssembleResp{
			SignalResp: &signal.SignalResp{
				Payload: &signal.SignalResp_Cancel{Cancel: resp.SignalCancelResp}},
			MsgData: data,
		}, nil
	case *signal.SignalReq_Accept:
		log.ZDebug(ctx, "SignalMessageAssemble SignalReq_Accept")
		userInfo, err := m.userRpc.GetUserInfo(ctx, payload.Accept.UserID)
		if err != nil {
			return nil, errs.WrapMsg(err, "SignalReq_Accept get user info failed")
		}
		//payload.Accept.Participant = m.generateParticipantMetaData(userInfo)
		participant := generateParticipantMetaData(userInfo)
		resp, err := m.rtc.Accept(ctx, payload.Accept, participant)
		if err != nil {
			return nil, err
		}
		if err := m.signalDB.AcceptSignalInvitation(ctx, resp.Sid, payload.Accept.UserID); err != nil {
			return nil, err
		}
		// send the accept msg to inviter and invitee, both receive the callback msg
		data, err := wrapperMsgData(ctx, payload.Accept.UserID, payload.Accept.Invitation.InviterUserID,
			payload.Accept.Invitation.GroupID, payload.Accept.Invitation.SessionType, constant.SignalingNotification,
			payload.Accept.Invitation.PlatformID, payload.Accept.OfflinePushInfo, &signal.SignalReq{Payload: payload})
		if err != nil {
			return nil, err
		}
		return &signal.SignalMessageAssembleResp{
			SignalResp: &signal.SignalResp{
				Payload: &signal.SignalResp_Accept{Accept: resp.SignalAcceptResp}},
			MsgData: data,
		}, nil
	case *signal.SignalReq_HungUp:
		// after hungon, 接通之后，双方的操作都是hungUp
		log.ZDebug(ctx, "SignalMessageAssemble SignalReq_HungUp")
		resp, err := m.rtc.HungUp(ctx, payload.HungUp)
		if err != nil {
			return nil, err
		}
		var recvID string
		if payload.HungUp.Invitation.SessionType == constant.SingleChatType {
			if payload.HungUp.UserID == payload.HungUp.Invitation.InviterUserID {
				recvID = payload.HungUp.Invitation.InviteeUserIDList[0]
			} else {
				recvID = payload.HungUp.Invitation.InviterUserID
			}
		}
		if err := m.signalDB.HungUpSignalInvitation(ctx, resp.Sid, payload.HungUp.UserID); err != nil {
			return nil, err
		}
		data, err := wrapperMsgData(ctx, payload.HungUp.UserID, recvID, payload.HungUp.Invitation.GroupID, payload.HungUp.Invitation.SessionType, constant.SignalingNotification,
			payload.HungUp.Invitation.PlatformID, payload.HungUp.OfflinePushInfo, &signal.SignalReq{Payload: payload})
		if err != nil {
			return nil, err
		}
		return &signal.SignalMessageAssembleResp{
			SignalResp: &signal.SignalResp{
				Payload: &signal.SignalResp_HungUp{HungUp: resp.SignalHungUpResp}},
			MsgData: data,
		}, nil
	case *signal.SignalReq_Reject:
		log.ZDebug(ctx, "SignalMessageAssemble SignalReq_Reject")
		// 拒绝只能被邀请者的人在没有打通的时候调用
		resp, err := m.rtc.Reject(ctx, payload.Reject)
		if err != nil {
			return nil, err
		}
		if err := m.signalDB.RejectSignalInvitation(ctx, resp.Sid, payload.Reject.UserID); err != nil {
			return nil, err
		}
		data, err := wrapperMsgData(ctx, payload.Reject.UserID, payload.Reject.Invitation.InviterUserID,
			payload.Reject.Invitation.GroupID, payload.Reject.Invitation.SessionType, constant.SignalingNotification,
			payload.Reject.Invitation.PlatformID, payload.Reject.OfflinePushInfo, &signal.SignalReq{Payload: payload})
		if err != nil {
			return nil, err
		}
		return &signal.SignalMessageAssembleResp{
			SignalResp: &signal.SignalResp{
				Payload: &signal.SignalResp_Reject{Reject: resp.SignalRejectResp}},
			MsgData: data,
		}, nil
	case *signal.SignalReq_GetTokenByRoomID:
		log.ZDebug(ctx, "SignalMessageAssemble SignalReq_GetTokenByRoomID")
		userInfo, err := m.userRpc.GetUserInfo(ctx, payload.GetTokenByRoomID.UserID)
		if err != nil {
			return nil, errs.WrapMsg(err, "SignalReq_GetTokenByRoomID get user info failed")
		}
		//payload.GetTokenByRoomID.Participant = m.generateParticipantMetaData(userInfo)
		participant := generateParticipantMetaData(userInfo)
		resp, err := m.rtc.GetTokenByRoomID(ctx, payload.GetTokenByRoomID, participant)
		if err != nil {
			return nil, err
		}
		return &signal.SignalMessageAssembleResp{
			SignalResp: &signal.SignalResp{Payload: &signal.SignalResp_GetTokenByRoomID{GetTokenByRoomID: resp}},
		}, nil
	default:
		return nil, errs.ErrInternalServer.WrapMsg(fmt.Sprintf("unknown payload type %T", payload))
	}
}

func (m *signalServer) SignalGetRoomByGroupID(ctx context.Context, req *signal.SignalGetRoomByGroupIDReq) (*signal.SignalGetRoomByGroupIDResp, error) {
	return m.rtc.GetRoomByGroupID(ctx, req)
}

func (m *signalServer) SignalGetTokenByRoomID(ctx context.Context, req *signal.SignalGetTokenByRoomIDReq) (*signal.SignalGetTokenByRoomIDResp, error) {
	return m.rtc.GetTokenByRoomID(ctx, req, nil)
}

func (m *signalServer) SignalGetRooms(ctx context.Context, req *signal.SignalGetRoomsReq) (*signal.SignalGetRoomsResp, error) {
	return m.rtc.GetRooms(ctx, req)
}

func (m *signalServer) GetSignalInvitationInfo(ctx context.Context, req *signal.GetSignalInvitationInfoReq) (*signal.GetSignalInvitationInfoResp, error) {
	signalCacheModel, err := m.signalDB.GetSignalInvitationInfoByRoomID(ctx, req.RoomID)
	if err != nil {
		return nil, err
	}
	invitationInfo, offlinePushInfo := signalModelDB2Pb(&signalCacheModel.SignalModel)
	invitationInfo.InviteeUserIDList = signalCacheModel.InviteeUserIDList
	return &signal.GetSignalInvitationInfoResp{InvitationInfo: invitationInfo, OfflinePushInfo: offlinePushInfo}, nil
}

func (m *signalServer) GetSignalInvitationInfoStartApp(ctx context.Context, req *signal.GetSignalInvitationInfoStartAppReq) (*signal.GetSignalInvitationInfoStartAppResp, error) {
	signalCacheModel, err := m.signalDB.GetAvailableSignalInvitationInfo(ctx, req.UserID)
	if err != nil {
		if model.IsNotFound(err) {
			return &signal.GetSignalInvitationInfoStartAppResp{}, nil
		}
		return nil, err
	}
	invitationInfo, offlinePushInfo := signalModelDB2Pb(&signalCacheModel.SignalModel)
	invitationInfo.InviteeUserIDList = signalCacheModel.InviteeUserIDList
	return &signal.GetSignalInvitationInfoStartAppResp{Invitation: invitationInfo, OfflinePushInfo: offlinePushInfo}, nil
}
