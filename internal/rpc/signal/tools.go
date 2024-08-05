package signal

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	constant2 "github.com/openimsdk/openmeeting-server/pkg/common/constant"
	"github.com/openimsdk/openmeeting-server/pkg/common/storage/model"
	"github.com/openimsdk/protocol/constant"
	"github.com/openimsdk/protocol/openmeeting/meeting"
	"github.com/openimsdk/protocol/openmeeting/signal"
	"github.com/openimsdk/protocol/openmeeting/user"
	"github.com/openimsdk/protocol/sdkws"
	"github.com/openimsdk/tools/errs"
	"github.com/openimsdk/tools/utils/datautil"
	"github.com/openimsdk/tools/utils/timeutil"
	"google.golang.org/protobuf/proto"
	"math/rand"
	"strconv"
	"time"
)

func validateInvitation(invitation *signal.InvitationInfo) error {
	if len(invitation.InviteeUserIDList) == 0 {
		return errs.ErrArgs.WrapMsg("invitee user id list is empty")
	}
	return nil
}

func generateParticipantMetaData(userInfo *user.UserInfo) *meeting.ParticipantMetaData {
	return &meeting.ParticipantMetaData{
		UserInfo: &meeting.UserInfo{
			UserID:   userInfo.UserID,
			Nickname: userInfo.Nickname,
			Account:  userInfo.Account,
		},
	}
}

// add for signal begin{
func signalInvitationPb2DB(payload *signal.SignalInviteReq, sid string) *model.SignalModel {
	val := &model.SignalModel{
		InviterUserID: payload.Invitation.InviterUserID,
		CustomData:    payload.Invitation.CustomData,
		GroupID:       payload.Invitation.GroupID,
		RoomID:        payload.Invitation.RoomID,
		Timeout:       payload.Invitation.Timeout,
		MediaType:     payload.Invitation.MediaType,
		PlatformID:    payload.Invitation.PlatformID,
		SessionType:   payload.Invitation.SessionType,
		InitiateTime:  time.Unix(payload.Invitation.InitiateTime, 0),
		SID:           sid,
	}
	if payload.OfflinePushInfo != nil {
		val.Title = payload.OfflinePushInfo.Title
		val.Desc = payload.OfflinePushInfo.Desc
		val.Ex = payload.OfflinePushInfo.Ex
		val.IOSPushSound = payload.OfflinePushInfo.IOSPushSound
		val.IOSBadgeCount = payload.OfflinePushInfo.IOSBadgeCount
		val.SignalInfo = payload.OfflinePushInfo.SignalInfo
	}
	return val
}

func signalGroupInvitationPb2DB(payload *signal.SignalInviteInGroupReq, sid string) *model.SignalModel {
	val := &model.SignalModel{
		InviterUserID: payload.Invitation.InviterUserID,
		CustomData:    payload.Invitation.CustomData,
		GroupID:       payload.Invitation.GroupID,
		RoomID:        payload.Invitation.RoomID,
		Timeout:       payload.Invitation.Timeout,
		MediaType:     payload.Invitation.MediaType,
		PlatformID:    payload.Invitation.PlatformID,
		SessionType:   payload.Invitation.SessionType,
		InitiateTime:  time.Unix(payload.Invitation.InitiateTime, 0),
		SID:           sid,
	}
	if payload.OfflinePushInfo != nil {
		val.Title = payload.OfflinePushInfo.Title
		val.Desc = payload.OfflinePushInfo.Desc
		val.Ex = payload.OfflinePushInfo.Ex
		val.IOSPushSound = payload.OfflinePushInfo.IOSPushSound
		val.IOSBadgeCount = payload.OfflinePushInfo.IOSBadgeCount
		val.SignalInfo = payload.OfflinePushInfo.SignalInfo
	}
	return val
}

func signalModelDB2Pb(model *model.SignalModel) (*signal.InvitationInfo, *sdkws.OfflinePushInfo) {
	invitation := signal.InvitationInfo{
		InviterUserID: model.InviterUserID,
		CustomData:    model.CustomData,
		GroupID:       model.GroupID,
		RoomID:        model.RoomID,
		Timeout:       model.Timeout,
		MediaType:     model.MediaType,
		PlatformID:    model.PlatformID,
		SessionType:   model.SessionType,
		InitiateTime:  model.InitiateTime.Unix(),
	}
	offlinePushInfo := sdkws.OfflinePushInfo{
		Title:         model.Title,
		Desc:          model.Desc,
		Ex:            model.Ex,
		IOSPushSound:  model.IOSPushSound,
		IOSBadgeCount: model.IOSBadgeCount,
	}
	return &invitation, &offlinePushInfo
}

func wrapperMsgData(ctx context.Context, sendID, recvID, groupID string, sesstionType int32, contentType int32, platformID int32, offlinePushInfo *sdkws.OfflinePushInfo, req proto.Message) (*sdkws.MsgData, error) {
	reqData, err := proto.Marshal(req)
	if err != nil {
		return nil, err
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
	return &msgData, nil
}

func getMsgID(sendID string) string {
	str := sendID + "-" + strconv.FormatInt(time.Now().UnixNano(), 10) + "-" + strconv.FormatUint(rand.Uint64(), 10)
	sum := md5.Sum([]byte(str))
	return hex.EncodeToString(sum[:])
}

func generateDefaultPersonalData(userID string) *meeting.PersonalData {
	return &meeting.PersonalData{
		UserID: userID,
		PersonalSetting: &meeting.PersonalMeetingSetting{
			CameraOnEntry:     false,
			MicrophoneOnEntry: false,
		},
		LimitSetting: &meeting.PersonalMeetingSetting{
			CameraOnEntry:     true,
			MicrophoneOnEntry: true,
		},
	}
}

func (m *signalServer) generateMeetingDBData4Signal(ctx context.Context, req *signal.SignalInviteInGroupReq) (*model.MeetingInfo, error) {
	return &model.MeetingInfo{
		MeetingID:     req.Invitation.GroupID,
		StartTime:     timeutil.GetCurrentTimestampBySecond(),
		Status:        constant2.InProgress,
		CreatorUserID: req.Invitation.InviterUserID,
	}, nil
}

func (m *signalServer) generateMeetingMetaData4Signal(ctx context.Context, info *model.MeetingInfo, setting *meeting.MeetingSetting) (*meeting.MeetingMetadata, error) {
	userInfo, err := m.userRpc.GetUserInfo(ctx, info.CreatorUserID)
	if err != nil {
		return nil, err
	}

	metaData := &meeting.MeetingMetadata{}
	metaData.PersonalData = []*meeting.PersonalData{generateDefaultPersonalData(info.CreatorUserID)}
	systemInfo := &meeting.SystemGeneratedMeetingInfo{
		CreatorUserID:   info.CreatorUserID,
		Status:          info.Status,
		StartTime:       info.StartTime,
		MeetingID:       info.MeetingID,
		CreatorNickname: userInfo.Nickname,
	}
	creatorInfo := &meeting.CreatorDefinedMeetingInfo{
		HostUserID: info.CreatorUserID,
	}
	meetingInfo := &meeting.MeetingInfo{
		SystemGenerated:       systemInfo,
		CreatorDefinedMeeting: creatorInfo,
	}
	metaData.Detail = &meeting.MeetingInfoSetting{
		Info:    meetingInfo,
		Setting: setting,
	}
	return metaData, nil
}
