package cmd

import (
	"context"
	"github.com/openimsdk/openmeeting-server/internal/rpc/signal"
	"github.com/openimsdk/openmeeting-server/pkg/common/config"
	"github.com/openimsdk/openmeeting-server/pkg/common/prommetrics"
	"github.com/openimsdk/openmeeting-server/pkg/common/startrpc"
	"github.com/openimsdk/tools/system/program"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/cobra"
)

type SignalRpcCmd struct {
	*RootCmd
	ctx           context.Context
	configMap     map[string]any
	meetingConfig *signal.Config
}

func NewSignalRpcCmd() *SignalRpcCmd {
	var signalConfig signal.Config
	ret := &SignalRpcCmd{meetingConfig: &signalConfig}
	ret.configMap = map[string]any{
		OpenMeetingRPCSignalCfgFileName: &signalConfig.Rpc,
		RedisConfigFileName:             &signalConfig.Redis,
		MongodbConfigFileName:           &signalConfig.Mongo,
		ShareFileName:                   &signalConfig.Share,
		DiscoveryConfigFilename:         &signalConfig.Discovery,
		LiveKitConfigFilename:           &signalConfig.Rtc,
	}
	ret.RootCmd = NewRootCmd(program.GetProcessName(), WithConfigMap(ret.configMap))
	ret.ctx = context.WithValue(context.Background(), "version", config.Version)
	ret.Command.RunE = func(cmd *cobra.Command, args []string) error {
		return ret.runE()
	}
	return ret
}

func (a *SignalRpcCmd) Exec() error {
	return a.Execute()
}

func (a *SignalRpcCmd) runE() error {
	return startrpc.Start(a.ctx, &a.meetingConfig.Discovery, &a.meetingConfig.Rpc.Prometheus, a.meetingConfig.Rpc.RPC.ListenIP,
		a.meetingConfig.Rpc.RPC.RegisterIP, a.meetingConfig.Rpc.RPC.Ports,
		a.Index(), a.meetingConfig.Share.RpcRegisterName.Signal, a.meetingConfig, signal.Start, []prometheus.Collector{prommetrics.SignalCreatedCounter})
}
