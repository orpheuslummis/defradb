// Copyright 2023 Democratized Data Foundation
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package field

import (
	"testing"

	testUtils "github.com/sourcenetwork/defradb/tests/integration"
)

func TestSchemaUpdatesCopyFieldErrors(t *testing.T) {
	test := testUtils.TestCase{
		Description: "Test schema update, copy field",
		Actions: []any{
			testUtils.SchemaUpdate{
				Schema: `
					type Users {
						Name: String
						Email: String
					}
				`,
			},
			testUtils.SchemaPatch{
				Patch: `
					[
						{ "op": "copy", "from": "/Users/Schema/Fields/1", "path": "/Users/Schema/Fields/2" }
					]
				`,
				ExpectedError: "duplicate field. Name: Email",
			},
			testUtils.Request{
				Request: `query {
					Users {
						Name
						Email
					}
				}`,
				Results: []map[string]any{},
			},
		},
	}
	testUtils.ExecuteTestCase(t, []string{"Users"}, test)
}

func TestSchemaUpdatesCopyFieldWithRemoveIDAndReplaceName(t *testing.T) {
	test := testUtils.TestCase{
		Description: "Test schema update, copy field, rename and remove IDs",
		Actions: []any{
			testUtils.SchemaUpdate{
				Schema: `
					type Users {
						Name: String
						Email: String
					}
				`,
			},
			testUtils.SchemaPatch{
				// Here we esentially use Email as a template, copying it, clearing the ID, and renaming the
				// clone.
				Patch: `
					[
						{ "op": "copy", "from": "/Users/Schema/Fields/1", "path": "/Users/Schema/Fields/3" },
						{ "op": "remove", "path": "/Users/Schema/Fields/3/ID" },
						{ "op": "replace", "path": "/Users/Schema/Fields/3/Name", "value": "Fax" }
					]
				`,
			},
			testUtils.Request{
				Request: `query {
					Users {
						Name
						Email
						Fax
					}
				}`,
				Results: []map[string]any{},
			},
		},
	}
	testUtils.ExecuteTestCase(t, []string{"Users"}, test)
}

// This is an odd test, but still a possibility and we should still cover it.
func TestSchemaUpdatesCopyFieldWithRemoveIDAndReplaceNameAndKindSubstitution(t *testing.T) {
	test := testUtils.TestCase{
		Description: "Test schema update, copy field, rename, re-type, and remove IDs",
		Actions: []any{
			testUtils.SchemaUpdate{
				Schema: `
					type Users {
						Name: String
					}
				`,
			},
			testUtils.SchemaPatch{
				// Here we esentially use Name as a template, copying it, clearing the ID, and renaming and
				// re-typing the clone.
				Patch: `
					[
						{ "op": "copy", "from": "/Users/Schema/Fields/1", "path": "/Users/Schema/Fields/2" },
						{ "op": "remove", "path": "/Users/Schema/Fields/2/ID" },
						{ "op": "replace", "path": "/Users/Schema/Fields/2/Name", "value": "Age" },
						{ "op": "replace", "path": "/Users/Schema/Fields/2/Kind", "value": "Integer" }
					]
				`,
			},
			testUtils.CreateDoc{
				CollectionID: 0,
				Doc: `{
					"Name": "John",
					"Age": 3
				}`,
			},
			testUtils.Request{
				Request: `query {
					Users {
						Name
						Age
					}
				}`,
				Results: []map[string]any{
					{
						"Name": "John",
						// It is important to test this with data, to ensure the type has been substituted correctly
						"Age": uint64(3),
					},
				},
			},
		},
	}
	testUtils.ExecuteTestCase(t, []string{"Users"}, test)
}

// This is an odd test, but still a possibility and we should still cover it.
func TestSchemaUpdatesCopyFieldWithRemoveIDAndReplaceNameAndInvalidKindSubstitution(t *testing.T) {
	test := testUtils.TestCase{
		Description: "Test schema update, copy field, rename, re-type to invalid, and remove ID",
		Actions: []any{
			testUtils.SchemaUpdate{
				Schema: `
					type Users {
						Name: String
					}
				`,
			},
			testUtils.SchemaPatch{
				// Here we esentially use Name as a template, copying it, clearing the ID, and renaming and
				// re-typing the clone.
				Patch: `
					[
						{ "op": "copy", "from": "/Users/Schema/Fields/1", "path": "/Users/Schema/Fields/2" },
						{ "op": "remove", "path": "/Users/Schema/Fields/2/ID" },
						{ "op": "replace", "path": "/Users/Schema/Fields/2/Name", "value": "Age" },
						{ "op": "replace", "path": "/Users/Schema/Fields/2/Kind", "value": "NotAValidKind" }
					]
				`,
				ExpectedError: "no type found for given name. Kind: NotAValidKind",
			},
		},
	}
	testUtils.ExecuteTestCase(t, []string{"Users"}, test)
}
