// Copyright 2022 Democratized Data Foundation
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package one_to_many_multiple

import (
	"testing"

	testUtils "github.com/sourcenetwork/defradb/tests/integration"
)

var bookAuthorGQLSchema = (`
	type article {
		name: String
		author: author
		rating: Int
	}

	type book {
		name: String
		author: author
		score: Int
	}

	type author {
		name: String
		age: Int
		verified: Boolean
		books: [book]
		articles: [article]
	}
`)

func executeTestCase(t *testing.T, test testUtils.RequestTestCase) {
	testUtils.ExecuteRequestTestCase(t, bookAuthorGQLSchema, []string{"article", "book", "author"}, test)
}
