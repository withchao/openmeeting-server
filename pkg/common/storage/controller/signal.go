package controller

import (
	"context"
	"github.com/openimsdk/openmeeting-server/pkg/common/constant"
	"github.com/openimsdk/openmeeting-server/pkg/common/storage/cache"
	"github.com/openimsdk/openmeeting-server/pkg/common/storage/database"
	"github.com/openimsdk/openmeeting-server/pkg/common/storage/model"
	"github.com/openimsdk/tools/db/pagination"
	"github.com/openimsdk/tools/db/tx"
	"github.com/openimsdk/tools/log"
	"github.com/openimsdk/tools/utils/datautil"
	"time"
)

type SignalTempModel struct {
	Signal *model.SignalModel
	Users  []*model.SignalInvitationModel
}

type SignalDatabase interface {
	CreateSignal(ctx context.Context, signalModel *model.SignalModel, inviteUserIDs []string) error
	UpdateSignalFileURL(ctx context.Context, sID, videoURL string) error

	GetSignalInvitationInfoByRoomID(ctx context.Context, roomID string) (cacheModel *model.SignalCacheModel, err error)
	GetAvailableSignalInvitationInfo(ctx context.Context, userID string) (cacheModel *model.SignalCacheModel, err error)

	AcceptSignalInvitation(ctx context.Context, sID, userID string) error
	RejectSignalInvitation(ctx context.Context, sID, userID string) error
	HungUpSignalInvitation(ctx context.Context, sID, userID string) error
	CancelSignalInvitation(ctx context.Context, sID, roomID, userID string) error
	UpdateSignalEndTime(ctx context.Context, sid string, endTime time.Time) error

	DelRoomSignal(ctx context.Context, roomID string) error

	// cms
	GetSignalInvitationRecords(ctx context.Context, sesstionType int32, sendID, recvID string, startTime, endTime time.Time, pagination pagination.Pagination) (int64, []*SignalTempModel, error)
	DeleteSignalRecords(ctx context.Context, sIDs []string) error
}

type singalDatabase struct {
	signalDB           database.SignalInterface
	signalInvitationDB database.SignalInvitationInterface
	tx                 tx.Tx
	cache              cache.SignalCache
}

func NewSignalDatabase(signalDB database.SignalInterface, signalInvitDB database.SignalInvitationInterface, cache cache.SignalCache, tx tx.Tx) SignalDatabase {
	return &singalDatabase{signalDB: signalDB, signalInvitationDB: signalInvitDB, cache: cache, tx: tx}
}

func (s *singalDatabase) CreateSignal(ctx context.Context, signalModel *model.SignalModel, inviteUserIDs []string) (err error) {
	_, err = s.cache.CreateSignalInvite(ctx, signalModel, inviteUserIDs)
	if err != nil {
		return err
	}
	return s.tx.Transaction(ctx, func(ctx context.Context) error {
		if err := s.signalDB.CreateSignal(ctx, signalModel); err != nil && (!model.IsDuplicate(err)) {
			log.ZDebug(ctx, "CreateSignal ng")
			return err
		}
		if err := s.signalInvitationDB.CreateSignalInvitation(ctx, signalModel.SID, append([]string{signalModel.InviterUserID}, inviteUserIDs...)); err != nil {
			log.ZDebug(ctx, "CreateSignalInvitation ng")
			return err
		}
		return nil
	})
}

func (s *singalDatabase) UpdateSignalFileURL(ctx context.Context, sID, fileURL string) error {
	return s.signalDB.UpdateSignalFileURL(ctx, sID, fileURL)
}

func (s *singalDatabase) UpdateSignalEndTime(ctx context.Context, sid string, endTime time.Time) error {
	return s.signalDB.UpdateSignalEndTime(ctx, sid, endTime)
}

func (s *singalDatabase) GetSignalInvitationInfoByRoomID(ctx context.Context, roomID string) (signalModel *model.SignalCacheModel, err error) {
	return s.cache.GetSignalInvitationInfoByRoomID(ctx, roomID)
}

func (s *singalDatabase) GetAvailableSignalInvitationInfo(ctx context.Context, userID string) (signalModel *model.SignalCacheModel, err error) {
	return s.cache.GetAvailableSignalInvitationInfo(ctx, userID)
}

func (s *singalDatabase) AcceptSignalInvitation(ctx context.Context, sID, userID string) error {
	if err := s.cache.DelUserSignal(ctx, userID); err != nil {
		return err
	}
	if err := s.signalInvitationDB.HandleSignalInvitation(ctx, sID, userID, constant.SignalAccept); err != nil { //SignalAccept
		return err
	}
	return nil
}

func (s *singalDatabase) RejectSignalInvitation(ctx context.Context, sID, userID string) error {
	if err := s.cache.DelUserSignal(ctx, userID); err != nil {
		return err
	}
	if err := s.signalInvitationDB.HandleSignalInvitation(ctx, sID, userID, constant.SignalReject); err != nil { //SignalReject
		return err
	}
	return nil

}
func (s *singalDatabase) HungUpSignalInvitation(ctx context.Context, sID, userID string) error {
	if err := s.cache.DelUserSignal(ctx, userID); err != nil {
		return err
	}
	if err := s.signalInvitationDB.HandleSignalInvitation(ctx, sID, userID, constant.SignalHungUp); err != nil { //SignalHungUp
		return err
	}
	return nil
}

func (s *singalDatabase) CancelSignalInvitation(ctx context.Context, sID, roomID, userID string) error {
	if err := s.cache.DelRoomSignal(ctx, roomID); err != nil {
		return err
	}
	if err := s.signalInvitationDB.HandleSignalInvitation(ctx, sID, userID, constant.SignalCancel); err != nil {
		return err
	}
	return nil
}

func (s *singalDatabase) DelRoomSignal(ctx context.Context, roomID string) error {
	return s.cache.DelRoomSignal(ctx, roomID)
}

func (s *singalDatabase) GetSignalInvitationRecords(ctx context.Context, sesstionType int32, sendID, recvID string, startTime, endTime time.Time, pagination pagination.Pagination) (int64, []*SignalTempModel, error) {
	var (
		total   int64
		signals []*model.SignalModel
	)
	if recvID == "" {
		var err error
		total, signals, err = s.signalDB.PageSignal(ctx, sesstionType, sendID, startTime, endTime, pagination)
		if err != nil {
			return 0, nil, err
		}
	} else {
		var (
			sids []string
			err  error
		)
		total, sids, err = s.signalInvitationDB.PageSID(ctx, recvID, startTime, endTime, pagination)
		if err != nil {
			return 0, nil, err
		}
		if len(sids) > 0 {
			temp, err := s.signalDB.Find(ctx, sids)
			if err != nil {
				return 0, nil, err
			}
			signalMap := datautil.SliceToMap(temp, func(signal *model.SignalModel) string {
				return signal.SID
			})
			signals = make([]*model.SignalModel, 0, len(sids))
			for _, sid := range sids {
				signal, ok := signalMap[sid]
				if !ok {
					continue
				}
				signals = append(signals, signal)
			}
		}
	}
	var res []*SignalTempModel
	if len(signals) > 0 {
		res = make([]*SignalTempModel, 0, len(signals))
		for _, signal := range signals {
			invitations, err := s.signalInvitationDB.Find(ctx, signal.SID)
			if err != nil {
				return 0, nil, err
			}
			res = append(res, &SignalTempModel{Signal: signal, Users: invitations})
		}
	}
	return total, res, nil
}

func (s *singalDatabase) DeleteSignalRecords(ctx context.Context, sids []string) error {
	return s.tx.Transaction(ctx, func(ctx context.Context) error {
		if err := s.signalDB.Delete(ctx, sids); err != nil {
			return err
		}
		if err := s.signalInvitationDB.Delete(ctx, sids); err != nil {
			return err
		}
		return nil
	})
}
