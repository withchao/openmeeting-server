package livekit

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"github.com/golang/protobuf/proto"
	lksdk "github.com/livekit/server-sdk-go"
	"github.com/openimsdk/openmeeting-server/pkg/rpcclient"
	"github.com/openimsdk/openmeeting-server/pkg/rtc"
	"github.com/openimsdk/protocol/constant"
	"github.com/openimsdk/protocol/msg"
	pbmeeting "github.com/openimsdk/protocol/openmeeting/meeting"
	"github.com/openimsdk/protocol/openmeeting/signal"
	"github.com/openimsdk/protocol/sdkws"
	"github.com/openimsdk/tools/log"
	"github.com/openimsdk/tools/utils/datautil"
	"math/rand"
	"strconv"
	"time"
)

func NewSignalRoomCallback(ctx context.Context, roomID, sID string,
	cb rtc.CallbackInterface, userRpc *rpcclient.User, invitationInfo *signal.InvitationInfo, fn func(ctx context.Context, roomID string) ([]*pbmeeting.ParticipantMetaData, []string, error), msgClient msg.MsgClient) *signalRoomCallback {
	return &signalRoomCallback{
		ctx:             ctx,
		roomID:          roomID,
		sID:             sID,
		cb:              cb,
		userRpc:         userRpc,
		invitationInfo:  invitationInfo,
		getParticipants: fn,
		msg:             msgClient,
	}
}

type signalRoomCallback struct {
	userJoin        bool
	sID             string
	roomID          string
	ctx             context.Context
	cb              rtc.CallbackInterface
	userRpc         *rpcclient.User
	invitationInfo  *signal.InvitationInfo
	getParticipants func(ctx context.Context, roomID string) ([]*pbmeeting.ParticipantMetaData, []string, error)
	msg             msg.MsgClient
}

func (r *signalRoomCallback) OnParticipantConnected(rp *lksdk.RemoteParticipant) {
	log.ZWarn(r.ctx, "OnParticipantConnected", nil)
	r.cb.OnRoomParticipantConnected(r.ctx, rp.Identity())
	if r.invitationInfo != nil && r.invitationInfo.SessionType != constant.SingleChatType {
		metadatas, _, err := r.getParticipants(r.ctx, r.invitationInfo.RoomID)
		if err != nil {
			log.ZError(r.ctx, "getParticipants failed", err, "roomID", r.invitationInfo.RoomID)
		}
		r.msgOnRoomParticipantConnected(r.ctx, r.invitationInfo, metadatas)
	}
}

func (r *signalRoomCallback) OnParticipantDisconnected(rp *lksdk.RemoteParticipant) {
	log.ZWarn(r.ctx, "OnParticipantDisconnected", nil, "kind:", rp.Kind().String())
	r.cb.OnRoomParticipantDisconnected(r.ctx, rp.Identity())
	if r.invitationInfo != nil && r.invitationInfo.SessionType != constant.SingleChatType {
		metadatas, _, err := r.getParticipants(r.ctx, r.invitationInfo.RoomID)
		if err != nil {
			log.ZError(r.ctx, "getParticipants failed", err, "roomID", r.invitationInfo.RoomID)
		}
		r.msgOnRoomParticipantDisconnected(r.ctx, r.invitationInfo, metadatas)
	}

	// clear user token
	//if _, err := r.userRpc.Client.ClearUserToken(r.ctx, &pbuser.ClearUserTokenReq{UserID: rp.Identity()}); err != nil {
	//	log.ZWarn(r.ctx, "clear user token failed", err, "userID", rp.Identity())
	//}
}

func (r *signalRoomCallback) OnDisconnected() {
	log.ZWarn(r.ctx, "OnDisconnected", nil)
	r.cb.OnRoomDisconnected(r.ctx)
}

func (r *signalRoomCallback) OnReconnected() {
	log.ZWarn(r.ctx, "OnReconnected", nil)
}

func (r *signalRoomCallback) OnReconnecting() {
	log.ZWarn(r.ctx, "OnReconnecting", nil)
}

func (r *signalRoomCallback) msgOnRoomParticipantConnected(ctx context.Context, invitation *signal.InvitationInfo, metadatas []*pbmeeting.ParticipantMetaData) {
	if metadatas != nil && invitation != nil {
		req := &signal.SignalOnRoomParticipantConnectedReq{GroupID: invitation.GroupID, Participant: metadatas, Invitation: invitation}
		err := r.sendSignalMsg(ctx, invitation.InviterUserID, "", invitation.GroupID, constant.SuperGroupChatType, constant.RoomParticipantsConnectedNotification, invitation.PlatformID, nil, req)
		if err != nil {
			log.ZError(ctx, "OnRoomParticipantConnected sendSignalMsg failed", err)
		}
	}
}

func (r *signalRoomCallback) msgOnRoomParticipantDisconnected(ctx context.Context, invitation *signal.InvitationInfo, metadatas []*pbmeeting.ParticipantMetaData) {
	if metadatas != nil && invitation != nil {
		req := &signal.SignalOnRoomParticipantConnectedReq{GroupID: invitation.GroupID, Participant: metadatas, Invitation: invitation}
		err := r.sendSignalMsg(ctx, invitation.InviterUserID, "", invitation.GroupID, constant.SuperGroupChatType, constant.RoomParticipantsDisconnectedNotification, invitation.PlatformID, nil, req)
		if err != nil {
			log.ZError(ctx, "OnRoomParticipantDisconnected sendSignalMsg failed", err)
		}
	}
}

func (r *signalRoomCallback) sendSignalMsg(ctx context.Context, sendID, recvID, groupID string, sesstionType int32, contentType int32, platformID int32, offlinePushInfo *sdkws.OfflinePushInfo, req proto.Message) error {
	reqData, err := proto.Marshal(req)
	if err != nil {
		return err
	}
	msgData := sdkws.MsgData{
		MsgFrom:    constant.UserMsgType,
		CreateTime: time.Now().UnixNano(),
		Options:    make(map[string]bool, 9),

		ContentType:      contentType,
		Content:          reqData,
		SenderPlatformID: platformID,
		SessionType:      sesstionType,
		SendID:           sendID,
		ClientMsgID:      getMsgID(sendID),
		RecvID:           recvID,
		GroupID:          groupID,
		OfflinePushInfo:  offlinePushInfo,
	}
	datautil.SetSwitchFromOptions(msgData.Options, constant.IsHistory, false)
	datautil.SetSwitchFromOptions(msgData.Options, constant.IsPersistent, false)
	datautil.SetSwitchFromOptions(msgData.Options, constant.IsSenderSync, true)
	datautil.SetSwitchFromOptions(msgData.Options, constant.IsConversationUpdate, false)
	datautil.SetSwitchFromOptions(msgData.Options, constant.IsSenderConversationUpdate, false)
	datautil.SetSwitchFromOptions(msgData.Options, constant.IsUnreadCount, true)
	datautil.SetSwitchFromOptions(msgData.Options, constant.IsOfflinePush, true)
	datautil.SetSwitchFromOptions(msgData.Options, constant.IsNotNotification, false)
	datautil.SetSwitchFromOptions(msgData.Options, constant.IsSendMsg, false)
	_, err = r.msg.SendMsg(ctx, &msg.SendMsgReq{MsgData: &msgData})
	return err
}

func Md5(s string) string {
	h := md5.New()
	h.Write([]byte(s))
	cipher := h.Sum(nil)
	return hex.EncodeToString(cipher)
}

func int64ToString(i int64) string {
	return strconv.FormatInt(i, 10)
}

func getMsgID(sendID string) string {
	t := int64ToString(time.Now().UnixNano())
	return Md5(t + sendID + int64ToString(rand.Int63n(time.Now().UnixNano())))
}
