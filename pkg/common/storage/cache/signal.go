package cache

import (
	"context"
	"github.com/openimsdk/openmeeting-server/pkg/common/storage/model"
)

type SignalCache interface {
	//BatchDeleter
	//NewSignalCache(rdb redis.UniversalClient) *SignalCache
	CloneFriendCache() SignalCache
	CreateSignalInvite(ctx context.Context, signalModel *model.SignalModel, inviteeUserIDs []string) (unHandleUserIDs []string, err error)
	GetSignalInvitationInfoByRoomID(ctx context.Context, roomID string) (cacheModel *model.SignalCacheModel, err error)
	GetAvailableSignalInvitationInfo(ctx context.Context, userID string) (cacheModel *model.SignalCacheModel, err error)
	DelUserSignal(ctx context.Context, userID string) error
	DelRoomSignal(ctx context.Context, roomID string) error
}
