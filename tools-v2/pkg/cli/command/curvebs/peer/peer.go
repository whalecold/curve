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

package peer

import (
	basecmd "github.com/opencurve/curve/tools-v2/pkg/cli/command"
	"github.com/opencurve/curve/tools-v2/pkg/cli/command/curvebs/peer/remove"
	"github.com/spf13/cobra"
)

// Command the command set for copyset
type Command struct {
	basecmd.MidCurveCmd
}

var _ basecmd.MidCurveCmdFunc = (*Command)(nil) // check interface

// AddSubCommands ...
func (umountCmd *Command) AddSubCommands() {
	umountCmd.Cmd.AddCommand(
		remove.NewCommand(),
	)
}

// NewCommand ...
func NewCommand() *cobra.Command {
	cmd := &Command{
		basecmd.MidCurveCmd{
			Use:   "peer",
			Short: "peer operation",
		},
	}
	return basecmd.NewMidCurveCli(&cmd.MidCurveCmd, cmd)
}
