// Copyright 2022 Democratized Data Foundation
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

// defradb is a decentralized peer-to-peer, user-centric, privacy-focused document database.
package main

import (
	"context"
	"os"

	"github.com/sourcenetwork/defradb/cli"
	"github.com/sourcenetwork/defradb/config"
)

// Execute adds all child commands to the root command and sets flags appropriately.
func main() {
	cfg := config.DefaultConfig()
	ctx := context.Background()
	defraCmd := cli.NewDefraCommand(cfg)
	if err := defraCmd.Execute(ctx); err != nil {
		os.Exit(1)
	}
}
