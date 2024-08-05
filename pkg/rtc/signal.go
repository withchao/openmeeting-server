package rtc

import (
	"context"
	"github.com/openimsdk/protocol/msg"
	"github.com/openimsdk/protocol/openmeeting/meeting"
	"github.com/openimsdk/protocol/openmeeting/signal"
)

type SignalInviteResp struct {
	*signal.SignalInviteResp
	Sid string
}

type SignalInviteInGroupResp struct {
	*signal.SignalInviteInGroupResp
	Sid string
}

type SignalCancelResp struct {
	*signal.SignalCancelResp
	Sid string
}

type SignalAcceptResp struct {
	*signal.SignalAcceptResp
	Sid string
}

type SignalHungUpResp struct {
	*signal.SignalHungUpResp
	Sid string
}

type SignalRejectResp struct {
	*signal.SignalRejectResp
	Sid string
}

type SignalRtc interface {
	InviteInUsers(ctx context.Context, req *signal.SignalInviteReq, metadata *meeting.ParticipantMetaData, inviationInfo *signal.InvitationInfo, msgClient msg.MsgClient) (*SignalInviteResp, error)
	InviteInGroup(ctx context.Context, req *signal.SignalInviteInGroupReq, roomMetadata *meeting.MeetingMetadata, metadata *meeting.ParticipantMetaData, inviationInfo *signal.InvitationInfo, msgClient msg.MsgClient) (*SignalInviteInGroupResp, error)
	Cancel(ctx context.Context, req *signal.SignalCancelReq) (*SignalCancelResp, error)
	Accept(ctx context.Context, req *signal.SignalAcceptReq, metadata *meeting.ParticipantMetaData) (*SignalAcceptResp, error)
	HungUp(ctx context.Context, req *signal.SignalHungUpReq) (*SignalHungUpResp, error)
	Reject(ctx context.Context, req *signal.SignalRejectReq) (*SignalRejectResp, error)
	GetTokenByRoomID(ctx context.Context, req *signal.SignalGetTokenByRoomIDReq, metadata *meeting.ParticipantMetaData) (*signal.SignalGetTokenByRoomIDResp, error)
	GetRoomByGroupID(ctx context.Context, req *signal.SignalGetRoomByGroupIDReq) (*signal.SignalGetRoomByGroupIDResp, error)
	GetRooms(ctx context.Context, req *signal.SignalGetRoomsReq) (*signal.SignalGetRoomsResp, error)
}
