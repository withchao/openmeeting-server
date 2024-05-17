// Copyright © 2023 OpenIM. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package api

import (
	"context"
	"fmt"
	"github.com/openimsdk/openmeeting-server/pkg/common/config"
	"github.com/openimsdk/tools/utils/datautil"
	"github.com/openimsdk/tools/utils/network"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	kdisc "github.com/openimsdk/openmeeting-server/pkg/common/discoveryregister"
	ginprom "github.com/openimsdk/openmeeting-server/pkg/common/ginprometheus"
	"github.com/openimsdk/openmeeting-server/pkg/common/prommetrics"
	"github.com/openimsdk/tools/discovery"
	"github.com/openimsdk/tools/errs"
	"github.com/openimsdk/tools/log"
	"github.com/openimsdk/tools/system/program"
)

type Config struct {
	API       config.API
	Discovery config.Discovery
	Share     config.Share
}

func Start(ctx context.Context, index int, config *Config) error {
	apiPort, err := datautil.GetElemByIndex(config.API.Api.Ports, index)
	if err != nil {
		return err
	}
	prometheusPort, err := datautil.GetElemByIndex(config.API.Prometheus.Ports, index)
	if err != nil {
		return err
	}

	var client discovery.SvcDiscoveryRegistry

	// Determine whether zk is passed according to whether it is a clustered deployment
	client, err = kdisc.NewDiscoveryRegister(&config.Discovery)
	if err != nil {
		return errs.WrapMsg(err, "failed to register discovery service")
	}

	var (
		netDone = make(chan struct{}, 1)
		netErr  error
	)

	router := newGinRouter(client, config)
	if config.API.Prometheus.Enable {
		go func() {
			p := ginprom.NewPrometheus("app", prommetrics.GetGinCusMetrics("Api"))
			p.SetListenAddress(fmt.Sprintf(":%d", prometheusPort))
			if err = p.Use(router); err != nil && err != http.ErrServerClosed {
				netErr = errs.WrapMsg(err, fmt.Sprintf("prometheus start err: %d", prometheusPort))
				netDone <- struct{}{}
			}
		}()

	}
	address := net.JoinHostPort(network.GetListenIP(config.API.Api.ListenIP), strconv.Itoa(apiPort))

	server := http.Server{Addr: address, Handler: router}
	log.CInfo(ctx, "API server is initializing", "address", address, "apiPort", apiPort, "prometheusPort", prometheusPort)
	go func() {
		err = server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			netErr = errs.WrapMsg(err, fmt.Sprintf("api start err: %s", server.Addr))
			netDone <- struct{}{}

		}
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	select {
	case <-sigs:
		program.SIGTERMExit()
		err := server.Shutdown(ctx)
		if err != nil {
			return errs.WrapMsg(err, "shutdown err")
		}
	case <-netDone:
		close(netDone)
		return netErr
	}
	return nil
}