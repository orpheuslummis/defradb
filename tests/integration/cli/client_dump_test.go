// Copyright 2022 Democratized Data Foundation
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package clitest

import "testing"

func TestClientDumpSimple(t *testing.T) {
	conf := NewDefraNodeDefaultConfig(t)
	stopDefra := runDefraNode(t, conf)

	stdout, _ := runDefraCommand(t, conf, []string{"client", "dump"})

	nodeLog := stopDefra()

	assertContainsSubstring(t, stdout, `{"data":{"response":"ok"}}`)
	assertContainsSubstring(t, nodeLog, "/db/system/seq/collection/names")
}
