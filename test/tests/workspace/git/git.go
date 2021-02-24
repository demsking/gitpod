// Copyright (c) 2020 Gitpod GmbH. All rights reserved.
// Licensed under the GNU Affero General Public License (AGPL).
// See License-AGPL.txt in the project root for license information.

package git

import (
	"fmt"
	"net/rpc"
	"testing"

	agent "github.com/gitpod-io/gitpod/test/tests/workspace/workspace_agent/api"
)

func GetBranch(t *testing.T, rsa *rpc.Client, workspaceRoot string) string {
	var resp agent.ExecResponse
	err := rsa.Call("WorkspaceAgent.Exec", &agent.ExecRequest{
		Dir:     workspaceRoot,
		Command: "git",
		Args:    []string{"rev-parse", "--abbrev-ref", "HEAD"},
	}, &resp)
	if err != nil {
		t.Fatal(err)
	}
	if resp.ExitCode != 0 {
		t.Fatal(fmt.Errorf("getBranch returned rc: %d", resp.ExitCode))
	}
	return resp.Stdout
}

func Commit(t *testing.T, rsa *rpc.Client, workspaceRoot string, message string, all bool) {
	args := []string{"commit", "-m", message}
	if all {
		args = append(args, "--all")
	}
	var resp agent.ExecResponse
	err := rsa.Call("WorkspaceAgent.Exec", &agent.ExecRequest{
		Dir:     workspaceRoot,
		Command: "git",
		Args:    args,
	}, &resp)
	if err != nil {
		t.Fatal(err)
	}
	if resp.ExitCode != 0 {
		t.Fatal(fmt.Errorf("commit returned rc: %d", resp.ExitCode))
	}
}
