package database

import (
	"context"
	"github.com/openimsdk/openmeeting-server/pkg/common/storage/model"
	"github.com/openimsdk/tools/db/pagination"
	"time"
)

type SignalInvitationInterface interface {
	Find(ctx context.Context, sid string) ([]*model.SignalInvitationModel, error)
	CreateSignalInvitation(ctx context.Context, sid string, inviteeUserIDs []string) error
	HandleSignalInvitation(ctx context.Context, sID, InviteeUserID string, status int32) error
	PageSID(ctx context.Context, recvID string, startTime, endTime time.Time, pagination pagination.Pagination) (int64, []string, error)
	Delete(ctx context.Context, sids []string) error
}
