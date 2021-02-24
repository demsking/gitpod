// Copyright (c) 2021 Gitpod GmbH. All rights reserved.
// Licensed under the GNU Affero General Public License (AGPL).
// See License-AGPL.txt in the project root for license information.

package cmd

import (
	"context"
	"time"

	"github.com/gitpod-io/gitpod/common-go/log"
	"github.com/gitpod-io/gitpod/supervisor/api"
	"github.com/spf13/cobra"
)

var notifyCmd = &cobra.Command{
	Use:   "notify <title> <message> <actions...>",
	Short: "Notifies the user of an external event",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		client := api.NewNotificationServiceClient(dialSupervisor())

		var (
			ctx, cancel = context.WithTimeout(context.Background(), 15*time.Second)
			title       = args[0]
			message     = args[1]
			actions     = args[2:]
		)
		defer cancel()
		response, err := client.Notify(ctx, &api.NotifyRequest{
			Title:   title,
			Message: message,
			Actions: actions,
		})
		if err != nil {
			log.WithError(err).Fatal("cannot notify client")
		}
		log.Info("User answered with '%s'", response.Action)
	},
}

func init() {
	rootCmd.AddCommand(notifyCmd)
}
