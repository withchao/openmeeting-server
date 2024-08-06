package livekit

import (
	"context"
	"encoding/json"
	"github.com/livekit/protocol/livekit"
	"github.com/openimsdk/openmeeting-server/pkg/rpcclient"
	"github.com/openimsdk/openmeeting-server/pkg/rtc"
	"github.com/openimsdk/protocol/msg"
	"github.com/openimsdk/protocol/openmeeting/meeting"
	"github.com/openimsdk/protocol/openmeeting/signal"
	"github.com/openimsdk/tools/errs"
	"github.com/openimsdk/tools/log"
	"github.com/openimsdk/tools/utils/datautil"
)

type signalLiveKit struct {
	livekit *LiveKit
	user    *rpcclient.User
	msg     msg.MsgClient
}

func (x *signalLiveKit) validateInvitation(invitation *signal.InvitationInfo) (err error) {
	if len(invitation.InviteeUserIDList) == 0 {
		return errs.ErrArgs.WrapMsg("invitee user id list is empty")
	}
	return nil
}

func (x *signalLiveKit) CheckUsersIsBusyLine(ctx context.Context, userIDs []string) (busyLineUserIDs []string, err error) {
	roomsResp, err := x.livekit.roomClient.ListRooms(ctx, &livekit.ListRoomsRequest{})
	if err != nil {
		log.ZError(ctx, "CheckUsersIsBusyLine list rooms err:", err)
		return nil, errs.Wrap(err)
	}
	for _, v := range roomsResp.Rooms {
		resp, err := x.livekit.roomClient.ListParticipants(ctx, &livekit.ListParticipantsRequest{Room: v.Name})
		if err != nil {
			log.ZError(ctx, "CheckUsersIsBusyLine ListParticipants err:", err)
			return nil, errs.Wrap(err)
		}
		for _, v := range resp.Participants {
			if datautil.Contain(v.Identity, userIDs...) {
				busyLineUserIDs = append(busyLineUserIDs, v.Identity)
			}
		}
	}
	return busyLineUserIDs, nil
}

func (x *signalLiveKit) InviteInUsers(ctx context.Context, req *signal.SignalInviteReq, metadata *meeting.ParticipantMetaData, inviationInfo *signal.InvitationInfo) (*rtc.SignalInviteResp, error) {
	if err := x.validateInvitation(req.Invitation); err != nil {
		log.ZDebug(ctx, "InviteInUsers validateInvitation error")
		return nil, err
	}
	// 占线
	busyLineUserIDList, err := x.CheckUsersIsBusyLine(ctx, req.Invitation.InviteeUserIDList)
	if err != nil {
		log.ZDebug(ctx, "InviteInUsers busy error")
		return nil, err
	}
	log.ZDebug(ctx, "get busyLineUserIDs success", "busyLineUserIDList", busyLineUserIDList)
	req.Invitation.BusyLineUserIDList = busyLineUserIDList
	// checke room is exist, if exist, return seesion id
	sid, err := x.livekit.RoomIsExist(ctx, req.Invitation.RoomID)
	var token, liveURL string
	if err != nil {
		// if room is not exist, create room
		roomMateData := &meeting.MeetingMetadata{}

		sid, token, liveURL, err = x.livekit.CreateRoom(ctx, req.Invitation.RoomID, req.Invitation.InviterUserID, roomMateData, metadata, nil)
		//sid, token, liveURL, err = x.CreateRoom(ctx, req.Invitation.RoomID, req.Invitation.InviterUserID, nil, metadata, nil, inviationInfo, msgClient)
		log.ZDebug(ctx, "InviteInUsers CreateRoom", "token", token)
	} else {
		//if room is exist, get the token\liveUrl
		token, liveURL, err = x.livekit.GetJoinToken(ctx, req.Invitation.RoomID, req.Invitation.InviterUserID, metadata, false)
		log.ZDebug(ctx, "InviteInUsers CreateRoom room exist", "token", token)
	}
	if err != nil {
		log.ZDebug(ctx, "InviteInUsers error happened")
		return nil, err
	}

	return &rtc.SignalInviteResp{
		Sid: sid,
		SignalInviteResp: &signal.SignalInviteResp{
			Token:              token,
			RoomID:             req.Invitation.RoomID,
			LiveURL:            liveURL,
			BusyLineUserIDList: busyLineUserIDList,
		},
	}, nil
}

func (x *signalLiveKit) InviteInGroup(ctx context.Context, req *signal.SignalInviteInGroupReq, roomMetadata *meeting.MeetingMetadata, participantMetadata *meeting.ParticipantMetaData, inviationInfo *signal.InvitationInfo, msgClient msg.MsgClient, userRpc *rpcclient.User) (*rtc.SignalInviteInGroupResp, error) {
	if err := x.validateInvitation(req.Invitation); err != nil {
		return nil, err
	}
	// 占线
	busyLineUserIDList, err := x.CheckUsersIsBusyLine(ctx, req.Invitation.InviteeUserIDList)
	if err != nil {
		return nil, err
	}
	log.ZDebug(ctx, "get busy users", "busyLineUserIDList", busyLineUserIDList)
	req.Invitation.BusyLineUserIDList = busyLineUserIDList
	var sid, token, liveURL string
	sid, err = x.livekit.RoomIsExist(ctx, req.Invitation.RoomID)
	if err != nil {
		sid, token, liveURL, err = x.livekit.CreateRoom(ctx, req.Invitation.RoomID, req.Invitation.InviterUserID, roomMetadata, participantMetadata, userRpc)
		log.ZDebug(ctx, "InviteInGroup CreateRoom", "token", token)
	} else {
		token, liveURL, err = x.livekit.GetJoinToken(ctx, req.Invitation.RoomID, req.Invitation.InviterUserID, participantMetadata, false)
		log.ZDebug(ctx, "InviteInGroup CreateRoom room exist", "token", token)
	}
	if err != nil {
		return nil, errs.ErrInternalServer.WrapMsg(err.Error())
	}
	return &rtc.SignalInviteInGroupResp{
		Sid: sid,
		SignalInviteInGroupResp: &signal.SignalInviteInGroupResp{
			Token:              token,
			RoomID:             req.Invitation.RoomID,
			LiveURL:            liveURL,
			BusyLineUserIDList: busyLineUserIDList,
		},
	}, nil
}

func (x *signalLiveKit) Cancel(ctx context.Context, req *signal.SignalCancelReq) (*rtc.SignalCancelResp, error) {
	sID, err := x.livekit.RoomIsExist(ctx, req.Invitation.RoomID)
	if err != nil {
		return nil, err
	}
	_, err = x.livekit.roomClient.RemoveParticipant(ctx, &livekit.RoomParticipantIdentity{Room: req.Invitation.RoomID, Identity: req.UserID})
	if err != nil && !x.livekit.IsNotFound(err) {
		return nil, err
	}
	return &rtc.SignalCancelResp{Sid: sID, SignalCancelResp: &signal.SignalCancelResp{}}, nil
}

func (x *signalLiveKit) Accept(ctx context.Context, req *signal.SignalAcceptReq, metadata *signal.ParticipantMetaData) (*rtc.SignalAcceptResp, error) {
	sid, err := x.livekit.RoomIsExist(ctx, req.Invitation.RoomID)
	if err != nil {
		return nil, err
	}
	token, liveURL, err := x.livekit.GetJoinToken(ctx, req.Invitation.RoomID, req.UserID, metadata, false)
	if err != nil {
		return nil, errs.Wrap(err)
	}
	return &rtc.SignalAcceptResp{
		Sid: sid,
		SignalAcceptResp: &signal.SignalAcceptResp{
			Token:   token,
			RoomID:  req.Invitation.RoomID,
			LiveURL: liveURL,
		},
	}, nil
}

func (x *signalLiveKit) HungUp(ctx context.Context, req *signal.SignalHungUpReq) (*rtc.SignalHungUpResp, error) {
	sid, err := x.livekit.RoomIsExist(ctx, req.Invitation.RoomID)
	if err != nil {
		return nil, err
	}
	_, err = x.livekit.roomClient.RemoveParticipant(ctx, &livekit.RoomParticipantIdentity{Room: req.Invitation.RoomID, Identity: req.UserID})
	if err != nil && !x.livekit.IsNotFound(err) {
		return nil, err
	}
	return &rtc.SignalHungUpResp{Sid: sid, SignalHungUpResp: &signal.SignalHungUpResp{}}, nil
}

func (x *signalLiveKit) Reject(ctx context.Context, req *signal.SignalRejectReq) (*rtc.SignalRejectResp, error) {
	sid, err := x.livekit.RoomIsExist(ctx, req.Invitation.RoomID)
	if err != nil {
		return nil, err
	}
	_, err = x.livekit.roomClient.RemoveParticipant(ctx, &livekit.RoomParticipantIdentity{Room: req.Invitation.RoomID, Identity: req.UserID})
	if err != nil && !x.livekit.IsNotFound(err) {
		return nil, err
	}
	return &rtc.SignalRejectResp{Sid: sid, SignalRejectResp: &signal.SignalRejectResp{}}, nil
}

func (x *signalLiveKit) GetTokenByRoomID(ctx context.Context, req *signal.SignalGetTokenByRoomIDReq, metadata *signal.ParticipantMetaData) (*signal.SignalGetTokenByRoomIDResp, error) {
	_, err := x.livekit.RoomIsExist(ctx, req.RoomID)
	if err != nil {
		return nil, err
	}
	token, liveURL, err := x.livekit.GetJoinToken(ctx, req.RoomID, req.UserID, metadata, false)
	if err != nil {
		return nil, err
	}
	return &signal.SignalGetTokenByRoomIDResp{
		Token:   token,
		LiveURL: liveURL,
	}, nil
}

func (x *signalLiveKit) GetRoomByGroupID(ctx context.Context, req *signal.SignalGetRoomByGroupIDReq) (*signal.SignalGetRoomByGroupIDResp, error) {
	participants, usrisList, err := x.livekit.GetParticipantMetaDataS(ctx, req.GroupID)
	if err != nil {
		return nil, err
	}
	if len(participants) == 0 {
		return &signal.SignalGetRoomByGroupIDResp{}, nil
	}
	resplk, err := x.livekit.roomClient.ListRooms(ctx, &livekit.ListRoomsRequest{Names: []string{req.GroupID}})
	if err != nil {
		return nil, err
	}
	var meetingMetadata *meeting.MeetingMetadata
	var invitation *signal.InvitationInfo
	if len(resplk.Rooms) > 0 {
		if resplk.Rooms[0].Metadata != "" {
			meetingMetadata = &signal.MeetingMetadata{}
			if err := json.Unmarshal([]byte(resplk.Rooms[0].Metadata), meetingMetadata); err != nil {
				return nil, err
			}
			var mediatype string
			if meetingMetadata.Detail.Setting.DisableCameraOnJoin {
				mediatype = "audio"
			} else {
				mediatype = "video"
			}
			invitation = &signal.InvitationInfo{
				InviterUserID:     meetingMetadata.Detail.Info.SystemGenerated.CreatorUserID,
				GroupID:           meetingMetadata.Detail.Info.SystemGenerated.MeetingID,
				RoomID:            meetingMetadata.Detail.Info.SystemGenerated.MeetingID,
				MediaType:         mediatype,
				InviteeUserIDList: usrisList,
			}
		} else {
			log.ZError(ctx, "resplk.Rooms[0].Metadata is null", nil, "room", resplk.Rooms[0].String())
			return nil, errs.ErrRecordNotFound.WrapMsg("room metadata is null")
		}
	}
	return &signal.SignalGetRoomByGroupIDResp{
		Invitation:  invitation,
		Participant: participants,
		RoomID:      req.GroupID,
	}, nil
}

func (x *signalLiveKit) GetRooms(ctx context.Context, req *signal.SignalGetRoomsReq) (*signal.SignalGetRoomsResp, error) {
	roomsResp, err := x.livekit.roomClient.ListRooms(ctx, &livekit.ListRoomsRequest{Names: req.RoomIDs})
	if err != nil {
		return nil, err
	}
	rooms := make([]*signal.SignalGetRoomByGroupIDResp, 0, len(roomsResp.Rooms))
	for _, room := range roomsResp.Rooms {
		participantResp, err := x.livekit.roomClient.ListParticipants(ctx, &livekit.ListParticipantsRequest{Room: room.Name})
		if err != nil {
			return nil, err
		}
		var metaDataList []*signal.ParticipantMetaData
		for _, participant := range participantResp.GetParticipants() {
			metadata := &signal.ParticipantMetaData{}
			if err := json.Unmarshal([]byte(participant.Metadata), metadata); err != nil {
				log.ZError(ctx, "Unmarshal err", err, "metadata", participant.Metadata)
				continue
			}
			metaDataList = append(metaDataList, metadata)
		}
		invitationInfo := &signal.InvitationInfo{}
		if err = json.Unmarshal([]byte(room.Metadata), invitationInfo); err != nil {
			log.ZError(ctx, "Unmarshal err", err, "metadata", room.Metadata)
			continue
		}
		rooms = append(rooms, &signal.SignalGetRoomByGroupIDResp{
			Invitation:  invitationInfo,
			Participant: metaDataList,
		})
	}
	return &signal.SignalGetRoomsResp{RoomList: rooms}, nil
}
