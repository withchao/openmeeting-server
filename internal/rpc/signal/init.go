package signal

import (
	"context"
	"github.com/openimsdk/openmeeting-server/pkg/common/config"
	"github.com/openimsdk/openmeeting-server/pkg/common/storage/cache/redis"
	"github.com/openimsdk/openmeeting-server/pkg/common/storage/controller"
	"github.com/openimsdk/openmeeting-server/pkg/common/storage/database/mgo"
	"github.com/openimsdk/openmeeting-server/pkg/rpcclient"
	"github.com/openimsdk/openmeeting-server/pkg/rtc"
	"github.com/openimsdk/openmeeting-server/pkg/rtc/livekit"
	userfind "github.com/openimsdk/openmeeting-server/pkg/user"
	"github.com/openimsdk/protocol/msg"
	"github.com/openimsdk/protocol/openmeeting/signal"
	"github.com/openimsdk/tools/db/mongoutil"
	"github.com/openimsdk/tools/db/redisutil"
	registry "github.com/openimsdk/tools/discovery"
	"google.golang.org/grpc"
)

type Config struct {
	Rpc       config.Meeting
	Redis     config.Redis
	Mongo     config.Mongo
	Discovery config.Discovery
	Share     config.Share
	Rtc       config.RTC
}

func Start(ctx context.Context, config *Config, client registry.SvcDiscoveryRegistry, server *grpc.Server) error {
	mgoCli, err := mongoutil.NewMongoDB(ctx, config.Mongo.Build())
	if err != nil {
		return err
	}
	rdb, err := redisutil.NewRedisClient(ctx, config.Redis.Build())
	if err != nil {
		return err
	}

	signalDB, err := mgo.NewSignal(mgoCli.GetDB())
	if err != nil {
		return err
	}
	signalInvitationDB, err := mgo.NewSignalInvitation(mgoCli.GetDB())
	if err != nil {
		return err
	}
	signalCache := redis.NewSignal(rdb, signalDB, redis.GetDefaultOpt())
	db := controller.NewSignalDatabase(signalDB, signalInvitationDB, signalCache, mgoCli.GetTx())

	user := userfind.NewOpenIMU(client, config.Share.RpcRegisterName.User)
	msgConn, err := client.GetConn(context.Background(), config.Share.RpcRegisterName.Msg)
	if err != nil {
		return err
	}
	userRpc := rpcclient.NewUser(user)
	msgClient := msg.NewMsgClient(msgConn)
	signalRtc := livekit.NewSignalLiveKit(&config.Rtc, userRpc, msgClient)

	// init rpc client here
	u := &signalServer{
		userRpc:  userRpc,
		rtc:      signalRtc,
		msg:      msgClient,
		signalDB: db,
	}

	signal.RegisterSignalServer(server, u)
	return nil
}

type signalServer struct {
	userRpc  *rpcclient.User
	rtc      rtc.SignalRtc
	msg      msg.MsgClient
	signalDB controller.SignalDatabase
}
