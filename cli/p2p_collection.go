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
	"github.com/spf13/cobra"
)

func MakeP2PCollectionCommand() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "p2pcollection",
		Short: "Interact with the P2P collection system",
		Long:  "Add, delete, or get the list of P2P collections",
	}
	return cmd
}
