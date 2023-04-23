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

func MakeClientCommand() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "client",
		Short: "Interact with a running DefraDB node as a client",
		Long: `Interact with a running DefraDB node as a client.
Execute queries, add schema types, and run debug routines.`,
	}

	return cmd
}
