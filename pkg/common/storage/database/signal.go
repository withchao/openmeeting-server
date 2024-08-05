package database

import (
	"context"
	"github.com/openimsdk/openmeeting-server/pkg/common/storage/model"
	"github.com/openimsdk/tools/db/pagination"
	"time"
)

type SignalInterface interface {
	Find(ctx context.Context, sids []string) ([]*model.SignalModel, error)
	CreateSignal(ctx context.Context, signalModel *model.SignalModel) error
	Update(ctx context.Context, sid string, update map[string]any) error
	UpdateSignalFileURL(ctx context.Context, sID, fileURL string) error
	UpdateSignalEndTime(ctx context.Context, sID string, endTime time.Time) error
	Delete(ctx context.Context, sids []string) error
	PageSignal(ctx context.Context, sesstionType int32, sendID string, startTime, endTime time.Time, pagination pagination.Pagination) (int64, []*model.SignalModel, error)
}
