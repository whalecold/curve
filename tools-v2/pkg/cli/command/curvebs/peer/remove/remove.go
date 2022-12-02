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

package remove

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"

	cmderror "github.com/opencurve/curve/tools-v2/internal/error"
	basecmd "github.com/opencurve/curve/tools-v2/pkg/cli/command"
	utils2 "github.com/opencurve/curve/tools-v2/pkg/cli/command/curvebs/peer/utils"
	"github.com/opencurve/curve/tools-v2/pkg/config"
	"github.com/opencurve/curve/tools-v2/pkg/output"
	"github.com/opencurve/curve/tools-v2/proto/proto/common"
	"github.com/opencurve/curve/tools-v2/proto/proto/cli"
)

const (
	removeExample = `$ curve bs peer remove --logicalpoolid=1 --copysetid=10001 --peer=127.0.0.1:8080:0 
 --curconf=127.0.0.1:8080:0,127.0.0.1:8081:1,127.0.0.1:8082:2 --rpcretrytimes=1 --rpctimeout=10s`
)

// RPCClient the rpc client for the rpc function RemovePeer
type RPCClient struct {
	Info    *basecmd.Rpc
	Request *cli.RemovePeerRequest
	cli     cli.CliServiceClient
}

func (rpp *RPCClient) NewRpcClient(cc grpc.ClientConnInterface) {
	rpp.cli = cli.NewCliServiceClient(cc)
}

func (rpp *RPCClient) Stub_Func(ctx context.Context) (interface{}, error) {
	return rpp.cli.RemovePeer(ctx, rpp.Request)
}

var _ basecmd.RpcFunc = (*RPCClient)(nil) // check interface

// Command the command to perform remove peer.
type Command struct {
	basecmd.FinalCurveCmd

	// request parameters
	opts          utils2.Options
	logicalPoolID uint32
	copysetID     uint32

	conf       utils2.Configuration
	removePeer *common.Peer

	removeCopySet bool
}

var _ basecmd.FinalCurveCmdFunc = (*Command)(nil) // check interface
// NewCommand ...
func NewCommand() *cobra.Command {
	cCmd := &Command{
		FinalCurveCmd: basecmd.FinalCurveCmd{
			Use:     "remove",
			Short:   "remove the peer from the copyset",
			Example: removeExample,
		},
	}
	basecmd.NewFinalCurveCli(&cCmd.FinalCurveCmd, cCmd)
	return cCmd.Cmd
}

func (cCmd *Command) AddFlags() {
	config.AddRpcRetryTimesFlag(cCmd.Cmd)
	config.AddRpcTimeoutFlag(cCmd.Cmd)

	config.AddLogicalPoolIdFlag(cCmd.Cmd)
	config.AddCopysetIdFlag(cCmd.Cmd)

	config.AddPeerFlag(cCmd.Cmd)
	config.AddCurConfFlag(cCmd.Cmd)
}

func (cCmd *Command) Init(cmd *cobra.Command, args []string) error {
	cCmd.opts = utils2.Options{}

	var err error
	cCmd.opts.Timeout = config.GetFlagDuration(cCmd.Cmd, config.RPCTIMEOUT)
	cCmd.opts.RetryTimes = config.GetFlagInt32(cCmd.Cmd, config.RPCRETRYTIMES)

	cCmd.copysetID, err = config.GetBsFlagUint32(cCmd.Cmd, config.CURVEBS_COPYSET_ID)
	if err != nil {
		return err
	}
	cCmd.logicalPoolID, err = config.GetBsFlagUint32(cCmd.Cmd, config.CURVEBS_LOGIC_POOL_ID)
	if err != nil {
		return err
	}

	// parse config
	curConf := config.GetBsFlagString(cCmd.Cmd, config.CURVEBS_CURRENT_CONFADDRESS)
	c, err := utils2.ParseConfiguration(curConf)
	if err != nil {
		return err
	}
	cCmd.conf = *c

	// parse conf
	peer := config.GetBsFlagString(cCmd.Cmd, config.CURVEBS_PEER)
	cCmd.removePeer, err = utils2.ParsePeer(peer)
	if err != nil {
		return err
	}
	return nil
}

func (cCmd *Command) Print(cmd *cobra.Command, args []string) error {
	return output.FinalCmdOutput(&cCmd.FinalCurveCmd, cCmd)
}

func (cCmd *Command) RunCommand(cmd *cobra.Command, args []string) error {

	// 1. acquire leader peer info.
	leader, err := utils2.GetLeader(cCmd.logicalPoolID, cCmd.copysetID, cCmd.conf, cCmd.opts)
	if err != nil {
		return err
	}

	// 2. remove peer
	prefix := fmt.Sprintf("Remove peer (%s:%v)for copyset(%v,%v) ", cCmd.removePeer.GetAddress(),
		cCmd.removePeer.GetId(), cCmd.logicalPoolID, cCmd.copysetID)
	err = cCmd.execRemovePeer(leader)
	if err != nil {
		fmt.Println(prefix, "fail, detail:", err.Error())
		return nil
	}

	if !cCmd.removeCopySet {
		return nil
	}
	// 3. delete broken copyset.
	prefix = fmt.Sprintf("Delete copyset (%v,%v)", cCmd.logicalPoolID, cCmd.copysetID)
	err = utils2.DeleteBrokenCopyset(cCmd.logicalPoolID, cCmd.copysetID, cCmd.removePeer, cCmd.opts)
	if err != nil {
		fmt.Println(prefix, "fail, detail:", err.Error())
		return nil
	}
	fmt.Println(prefix, "success")
	return nil
}

func (cCmd *Command) ResultPlainOutput() error {
	return nil
}

func (cCmd *Command) execRemovePeer(leader *common.Peer) error {
	peer, err := utils2.ParsePeer(leader.GetAddress())
	if err != nil {
		return err
	}
	leaderID := leader.GetAddress()
	peerID := fmt.Sprintf("%s:%v", cCmd.removePeer.GetAddress(), cCmd.removePeer.GetId())
	rpcCli := &RPCClient{
		Info: basecmd.NewRpc([]string{peer.GetAddress()}, cCmd.opts.Timeout, cCmd.opts.RetryTimes, "RemovePeer"),
		Request: &cli.RemovePeerRequest{
			LogicPoolId: &cCmd.logicalPoolID,
			CopysetId:   &cCmd.copysetID,
			LeaderId:    &leaderID,
			PeerId:      &peerID,
		},
	}

	response, errCmd := basecmd.GetRpcResponse(rpcCli.Info, rpcCli)
	if errCmd.TypeCode() != cmderror.CODE_SUCCESS {
		return errors.New("failed to remove the peer error:" + errCmd.Message)
	}
	resp, ok := response.(*cli.RemovePeerResponse)
	if !ok {
		return errors.New("error type interface when remove peer info")
	}
	fmt.Printf("configuration of replication group changed from %v to %v", resp.OldPeers, resp.NewPeers)
	return nil
}
