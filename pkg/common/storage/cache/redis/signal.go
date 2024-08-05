package redis

import (
	"context"
	"encoding/json"
	"github.com/dtm-labs/rockscache"
	"github.com/openimsdk/openmeeting-server/pkg/common/storage/cache"
	"github.com/openimsdk/openmeeting-server/pkg/common/storage/database"
	"github.com/openimsdk/openmeeting-server/pkg/common/storage/model"
	"github.com/openimsdk/tools/errs"
	"github.com/redis/go-redis/v9"
	"time"
)

const (
	signalCache           = "SIGNAL:"
	userSignalCache       = "USER_SIGNAL:"
	signalInviteUserCache = "SIGNAL_INVITE_USER:"
)

type SignalCacheRedis struct {
	cache.Meta
	SignalDB   database.SignalInterface
	expireTime time.Duration
	rcClient   *rockscache.Client
	rdb        redis.UniversalClient
}

func NewSignal(rdb redis.UniversalClient, signalDB database.SignalInterface, options rockscache.Options) cache.SignalCache {
	rcClient := rockscache.NewClient(rdb, options)
	mc := NewMetaCacheRedis(rcClient)
	mc.SetRawRedisClient(rdb)
	return &SignalCacheRedis{
		rdb:        rdb,
		Meta:       NewMetaCacheRedis(rcClient),
		SignalDB:   signalDB,
		expireTime: meetingExpireTime,
		rcClient:   rcClient,
	}
}

func (s *SignalCacheRedis) CloneFriendCache() cache.SignalCache {
	return &SignalCacheRedis{
		//BatchDeleter: s.BatchDeleter.Clone(),
		SignalDB:   s.SignalDB,
		expireTime: s.expireTime,
		rcClient:   s.rcClient,
		rdb:        s.rdb,
	}
}

func (s *SignalCacheRedis) getRoomSignalCacheKey(roomID string) string {
	return signalCache + roomID
}

func (s *SignalCacheRedis) getUserSignalCache(userID string) string {
	return userSignalCache + userID
}

func (s *SignalCacheRedis) getSignalInviteUserCacheKey(roomID string) string {
	return signalInviteUserCache + roomID
}

func (s *SignalCacheRedis) GetSignalInvite(ctx context.Context, userID string) (cacheModel *model.SignalCacheModel, err error) {

	roomID, err := s.rdb.Get(ctx, s.getUserSignalCache(userID)).Result()
	if err != nil {
		return nil, errs.Wrap(err)
	}
	key := s.getRoomSignalCacheKey(roomID)
	bytes, err := s.rdb.Get(ctx, key).Bytes()
	if err != nil {
		return nil, errs.Wrap(err)
	}
	cacheModel = &model.SignalCacheModel{}
	if err := json.Unmarshal(bytes, cacheModel); err != nil {
		return nil, errs.Wrap(err)
	}
	return cacheModel, nil
}

func (s *SignalCacheRedis) IsUnhandle(ctx context.Context, userID string) (isUnhandle bool, err error) {
	err = s.rdb.Get(ctx, s.getUserSignalCache(userID)).Err()
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		return false, errs.Wrap(err)
	}
	return true, nil
}

func (s *SignalCacheRedis) CreateSignalInvite(ctx context.Context, signalModel *model.SignalModel, inviteeUserIDs []string) (unHandleUserIDs []string, err error) {
	cacheModel := &model.SignalCacheModel{SignalModel: *signalModel, InviteeUserIDList: inviteeUserIDs}
	bytes, err := json.Marshal(cacheModel)
	if err != nil {
		return nil, errs.Wrap(err)
	}
	pipe := s.rdb.Pipeline()
	for _, userID := range inviteeUserIDs {
		isUnhandle, _ := s.IsUnhandle(ctx, userID)
		if isUnhandle {
			unHandleUserIDs = append(unHandleUserIDs, userID)
			continue
		}
		err = pipe.Set(ctx, s.getUserSignalCache(userID), cacheModel.RoomID, time.Duration(cacheModel.Timeout)*time.Second).Err()
		if err != nil {
			return nil, errs.Wrap(err)
		}
	}
	err = pipe.Set(ctx, s.getRoomSignalCacheKey(cacheModel.RoomID), string(bytes), time.Duration(cacheModel.Timeout)*time.Second).Err()
	if err != nil {
		return nil, errs.Wrap(err)
	}
	// err = pipe.SAdd(ctx, s.getSignalInviteUserCacheKey(cacheModel.RoomID), inviteeUserIDs).Err()
	// if err != nil {
	// 	return nil, errs.Wrap(err)
	// }
	// err = pipe.Expire(ctx, s.getSignalInviteUserCacheKey(cacheModel.RoomID), time.Duration(cacheModel.Timeout)*time.Second).Err()
	// if err != nil {
	// 	return nil, errs.Wrap(err)
	// }
	_, err = pipe.Exec(ctx)
	if err != nil {
		return nil, errs.Wrap(err)
	}
	return unHandleUserIDs, nil
}

func (s *SignalCacheRedis) GetSignalInvitationInfoByRoomID(ctx context.Context, roomID string) (cacheModel *model.SignalCacheModel, err error) {
	bytes, err := s.rdb.Get(ctx, s.getRoomSignalCacheKey(roomID)).Bytes()
	if err != nil {
		return nil, errs.Wrap(err)
	}
	cacheModel = &model.SignalCacheModel{}
	if err := json.Unmarshal(bytes, cacheModel); err != nil {
		return nil, errs.Wrap(err)
	}
	return cacheModel, nil
}

func (s *SignalCacheRedis) GetAvailableSignalInvitationInfo(ctx context.Context, userID string) (cacheModel *model.SignalCacheModel, err error) {
	roomID, err := s.rdb.Get(ctx, s.getUserSignalCache(userID)).Result()
	if err != nil {
		return nil, errs.Wrap(err)
	}
	cacheModel, err = s.GetSignalInvitationInfoByRoomID(ctx, roomID)
	if err != nil {
		return nil, errs.Wrap(err)
	}
	return cacheModel, nil
}

func (s *SignalCacheRedis) DelUserSignal(ctx context.Context, userID string) error {
	return s.rdb.Del(ctx, s.getUserSignalCache(userID)).Err()
}

func (s *SignalCacheRedis) DelRoomSignal(ctx context.Context, roomID string) error {
	return s.rdb.Del(ctx, s.getRoomSignalCacheKey(roomID)).Err()
}
