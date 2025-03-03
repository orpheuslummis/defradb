// Copyright 2022 Democratized Data Foundation
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package simple

import (
	"testing"

	testUtils "github.com/sourcenetwork/defradb/tests/integration"
)

func TestQuerySimpleWithSumWithFilter(t *testing.T) {
	test := testUtils.RequestTestCase{
		Description: "Simple query, sum with filter",
		Request: `query {
					_sum(users: {field: Age, filter: {Age: {_gt: 26}}})
				}`,
		Docs: map[int][]string{
			0: {
				`{
					"Name": "John",
					"Age": 21
				}`,
				`{
					"Name": "Bob",
					"Age": 30
				}`,
				`{
					"Name": "Alice",
					"Age": 32
				}`,
			},
		},
		Results: []map[string]any{
			{
				"_sum": int64(62),
			},
		},
	}

	executeTestCase(t, test)
}
