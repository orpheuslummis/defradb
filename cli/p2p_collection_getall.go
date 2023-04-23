// Copyright 2022 Democratized Data Foundation
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package cli

import (
	"context"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/sourcenetwork/defradb/config"
	"github.com/sourcenetwork/defradb/errors"
	"github.com/sourcenetwork/defradb/logging"
	netclient "github.com/sourcenetwork/defradb/net/api/client"
)

func MakeP2PCollectionGetallCommand(cfg *config.Config) *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "getall",
		Short: "Get all P2P collections",
		Long:  `Use this command if you wish to get all P2P collections in the pubsub topics`,
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.NoArgs(cmd, args); err != nil {
				return errors.New("must specify no argument")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cred := insecure.NewCredentials()
			client, err := netclient.NewClient(cfg.Net.RPCAddress, grpc.WithTransportCredentials(cred))
			if err != nil {
				return errors.Wrap("failed to create RPC client", err)
			}

			rpcTimeoutDuration, err := cfg.Net.RPCTimeoutDuration()
			if err != nil {
				return errors.Wrap("failed to parse RPC timeout duration", err)
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), rpcTimeoutDuration)
			defer cancel()

			collections, err := client.GetAllP2PCollections(ctx)
			if err != nil {
				return errors.Wrap("failed to add p2p collections, request failed", err)
			}

			if len(collections) > 0 {
				log.FeedbackInfo(ctx, "Successfully got all P2P collections")
				for _, col := range collections {
					log.FeedbackInfo(ctx, col.Name, logging.NewKV("CollectionID", col.ID))
				}
			} else {
				log.FeedbackInfo(ctx, "No P2P collection found")
			}

			return nil
		},
	}
	return cmd
}
