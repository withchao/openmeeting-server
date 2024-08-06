package user

import (
	"context"
	"github.com/openimsdk/protocol/openmeeting/user"
	"github.com/openimsdk/protocol/sdkws"
	openimUser "github.com/openimsdk/protocol/user"
	"github.com/openimsdk/tools/discovery"
	"github.com/openimsdk/tools/system/program"
	"github.com/openimsdk/tools/utils/datautil"
)

func NewOpenIMU(discov discovery.SvcDiscoveryRegistry, rpcRegisterName string) User {
	conn, err := discov.GetConn(context.Background(), rpcRegisterName)
	if err != nil {
		program.ExitWithError(err)
	}
	return &openim{user: openimUser.NewUserClient(conn)}
}

type openim struct {
	user openimUser.UserClient
}

func (m *openim) GetUsersInfos(ctx context.Context, userIDs []string) ([]*user.UserInfo, error) {
	if len(userIDs) == 0 {
		return []*user.UserInfo{}, nil
	}
	resp, err := m.user.GetDesignateUsers(ctx, &openimUser.GetDesignateUsersReq{
		UserIDs: userIDs,
	})
	if err != nil {
		return nil, err
	}
	return datautil.Slice(resp.UsersInfo, func(val *sdkws.UserInfo) *user.UserInfo {
		return &user.UserInfo{
			UserID:   val.UserID,
			Nickname: val.Nickname,
		}
	}), nil
}
