// Copyright (c) 2020 Gitpod GmbH. All rights reserved.
// Licensed under the GNU Affero General Public License (AGPL).
// See License-AGPL.txt in the project root for license information.

package workspace_test

import (
	"context"
	"testing"
	"time"

	"github.com/gitpod-io/gitpod/test/pkg/integration"
	git "github.com/gitpod-io/gitpod/test/tests/workspace/git"
	wsapi "github.com/gitpod-io/gitpod/ws-manager/api"
)

type ContextTest struct {
	Name           string
	ContextURL     string
	WorkspaceRoot  string
	ExpectedBranch string
}

func TestGitHubContexts(t *testing.T) {
	tests := []ContextTest{
		{
			Name:           "open repository",
			ContextURL:     "github.com/gitpod-io/gitpod",
			WorkspaceRoot:  "/workspace/gitpod",
			ExpectedBranch: "main",
		},
		{
			Name:           "open issue",
			ContextURL:     "github.com/gitpod-io/gitpod-test-repo/issues/88",
			WorkspaceRoot:  "/workspace/gitpod-test-repo",
			ExpectedBranch: "main",
		},
		{
			Name:           "open tag",
			ContextURL:     "github.com/gitpod-io/gitpod-test-repo/tree/integration-test-context-tag",
			WorkspaceRoot:  "/workspace/gitpod-test-repo",
			ExpectedBranch: "HEAD",
		},
	}
	runContextTests(t, tests)
}

func runContextTests(t *testing.T, tests []ContextTest) {
	for _, test := range tests {
		t.Run(test.ContextURL, func(t *testing.T) {
			t.Parallel()

			it := integration.NewTest(t)
			defer it.Done()

			nfo := integration.LaunchWorkspaceFromContextURL(it, test.ContextURL)

			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
			defer cancel()
			it.WaitForWorkspace(ctx, nfo.LatestInstance.ID)

			rsa, err := it.Instrument(integration.ComponentWorkspace, "workspace", integration.WithInstanceID(nfo.LatestInstance.ID))
			if err != nil {
				t.Fatal(err)
			}
			defer rsa.Close()

			// get actual from workspace
			actBranch := git.GetBranch(t, rsa, test.WorkspaceRoot)
			rsa.Close()

			ctx, cancel = context.WithTimeout(ctx, 10*time.Second)
			defer cancel()
			_, err = it.API().WorkspaceManager().StopWorkspace(ctx, &wsapi.StopWorkspaceRequest{
				Id: nfo.LatestInstance.ID,
			})
			if err != nil {
				t.Fatal(err)
				return
			}

			// perform actual comparion
			if actBranch != test.ExpectedBranch {
				t.Fatalf("expected branch '%s', got '%s'!", test.ExpectedBranch, actBranch)
			}
		})
	}
}
