// Copyright 2022 Democratized Data Foundation
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package commits

import (
	testUtils "github.com/sourcenetwork/defradb/tests/integration"
)

const userCollectionGQLSchema = (`
	type users {
		Name: String
		Age: Int
		Verified: Boolean
	}
`)

const companiesCollectionGQLSchema = (`
	type companies {
		Name: String
	}
`)

func updateUserCollectionSchema() testUtils.SchemaUpdate {
	return testUtils.SchemaUpdate{
		Schema: userCollectionGQLSchema,
	}
}

func updateCompaniesCollectionSchema() testUtils.SchemaUpdate {
	return testUtils.SchemaUpdate{
		Schema: companiesCollectionGQLSchema,
	}
}
