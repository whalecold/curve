/*
 *  Copyright (c) 2022 NetEase Inc.
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

/*
 * Project: tools-v2
 * Created Date: 2022-11-30
 * Author: zls1129@gmail.com
 */

package utils

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	cmderror "github.com/opencurve/curve/tools-v2/internal/error"
	basecmd "github.com/opencurve/curve/tools-v2/pkg/cli/command"
	common "github.com/opencurve/curve/tools-v2/proto/proto/common"
	"google.golang.org/grpc"
	"log"
	"github.com/opencurve/curve/tools-v2/proto/proto/cli"
)

// LeaderRpc the rpc client for the rpc function GetLeader
type LeaderRpc struct {
	Info    *basecmd.Rpc
	Request *cli.GetLeaderRequest
	Cli     cli.CliServiceClient
}

var _ basecmd.RpcFunc = (*LeaderRpc)(nil) // check interface

// NewRpcClient ...
func (ufRp *LeaderRpc) NewRpcClient(cc grpc.ClientConnInterface) {
	ufRp.Cli = cli.NewCliServiceClient(cc)
}

// Stub_Func ...
func (ufRp *LeaderRpc) Stub_Func(ctx context.Context) (interface{}, error) {
	return ufRp.Cli.GetLeader(ctx, ufRp.Request)
}

// Configuration the configuration for peer.
type Configuration struct {
	Peers []*common.Peer
}

// Options rpc request options.
type Options struct {
	Timeout    time.Duration
	RetryTimes int32
}

// ParsePeer parse the peer string
func ParsePeer(peer string) (*common.Peer, error) {
	cs := strings.Split(peer, ":")
	if len(cs) != 3 {
		return nil, fmt.Errorf("error format for the peer info %s", peer)
	}
	id, err := strconv.ParseUint(cs[2], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("error format for the peer id %s", cs[2])
	}
	cs = cs[:2]
	address := strings.Join(cs, ":")
	return &common.Peer{
		Id:      &id,
		Address: &address,
	}, nil
}

// ParseConfiguration parse the conf string into Configuration
func ParseConfiguration(conf string) (*Configuration, error) {
	if len(conf) == 0 {
		return nil, fmt.Errorf("empty group configuration")
	}

	confs := strings.Split(conf, ",")
	configuration := &Configuration{}
	for _, c := range confs {
		peer, err := ParsePeer(c)
		if err != nil {
			return nil, err
		}
		configuration.Peers = append(configuration.Peers, peer)
	}
	return configuration, nil
}

// GetLeader get leader for the address.
func GetLeader(logicalPoolID, copysetID uint32, conf Configuration, opts Options) (*common.Peer, error) {
	if len(conf.Peers) == 0 {
		return nil, errors.New("empty group configuration")
	}
	for _, peer := range conf.Peers {
		p := "0"
		rpcCli := &LeaderRpc{
			Request: &cli.GetLeaderRequest{
				LogicPoolId: &logicalPoolID,
				CopysetId:   &copysetID,
				PeerId:      &p,
			},
			Info: basecmd.NewRpc([]string{peer.GetAddress()}, opts.Timeout, opts.RetryTimes, "GetLeader"),
		}
		log.Println("pool:", logicalPoolID, "copyset: ", copysetID, "peer: ", peer)
		response, errCmd := basecmd.GetRpcResponse(rpcCli.Info, rpcCli)
		if errCmd.TypeCode() != cmderror.CODE_SUCCESS {
			fmt.Printf("failed to acquire leader peer info %s error : %s", peer.GetAddress(), errCmd.Message)
			continue
		}
		resp, ok := response.(*cli.GetLeaderResponse)
		if !ok {
			fmt.Printf("error type interface when acquire leader peer info %s", peer.GetAddress())
			continue
		}
		if resp.LeaderId != nil {
			fmt.Println("id ", *resp.LeaderId)
			return nil, nil
		}
	}
	return nil, fmt.Errorf("failed to acquire leader peer info")
}
