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

func TestQuerySimpleWithEmbeddedLatestCommit(t *testing.T) {
	test := testUtils.RequestTestCase{
		Description: "Embedded latest commits query within object query",
		Request: `query {
					users {
						Name
						Age
						_version {
							cid
							links {
								cid
								name
							}
						}
					}
				}`,
		Docs: map[int][]string{
			0: {
				`{
					"Name": "John",
					"Age": 21
				}`,
			},
		},
		Results: []map[string]any{
			{
				"Name": "John",
				"Age":  uint64(21),
				"_version": []map[string]any{
					{
						"cid": "bafybeiaahzxsfz55nuqnsll42wxrbdjmy5si222l4ydbrwb53tpxnzdmwq",
						"links": []map[string]any{
							{
								"cid":  "bafybeibelbquwus6f4lek2uclnmt5o525qbg46pzgxkjmrfjg3hwg63r4i",
								"name": "Age",
							},
							{
								"cid":  "bafybeih7josq7krvco7jc5z2mhiox4dge77zyn3v7gmemfjkkh3xkqwpni",
								"name": "Name",
							},
						},
					},
				},
			},
		},
	}

	executeTestCase(t, test)
}

func TestQuerySimpleWithEmbeddedLatestCommitWithSchemaVersionId(t *testing.T) {
	test := testUtils.RequestTestCase{
		Description: "Embedded commits query within object query with schema version id",
		Request: `query {
					users {
						Name
						_version {
							schemaVersionId
						}
					}
				}`,
		Docs: map[int][]string{
			0: {
				`{
					"Name": "John",
					"Age": 21
				}`,
			},
		},
		Results: []map[string]any{
			{
				"Name": "John",
				"_version": []map[string]any{
					{
						"schemaVersionId": "bafkreicl25inntuxlx4fbbcwb2tzbja4iyvujjwhk25pen7ljjdzcpgup4",
					},
				},
			},
		},
	}

	executeTestCase(t, test)
}

func TestQuerySimpleWithEmbeddedLatestCommitWithDockey(t *testing.T) {
	const dockey = "bae-52b9170d-b77a-5887-b877-cbdbb99b009f"

	test := testUtils.RequestTestCase{
		Description: "Embedded commits query within object query with dockey",
		Request: `query {
					users {
						Name
						_key
						_version {
							dockey
						}
					}
				}`,
		Docs: map[int][]string{
			0: {
				`{
					"Name": "John",
					"Age": 21
				}`,
			},
		},
		Results: []map[string]any{
			{
				"Name": "John",
				"_key": dockey,
				"_version": []map[string]any{
					{
						"dockey": dockey,
					},
				},
			},
		},
	}

	executeTestCase(t, test)
}

func TestQuerySimpleWithMultipleAliasedEmbeddedLatestCommit(t *testing.T) {
	test := testUtils.RequestTestCase{
		Description: "Embedded, aliased, latest commits query within object query",
		Request: `query {
					users {
						Name
						Age
						_version {
							cid
							L1: links {
								cid
								name
							}
							L2: links {
								name
							}
						}
					}
				}`,
		Docs: map[int][]string{
			0: {
				`{
					"Name": "John",
					"Age": 21
				}`,
			},
		},
		Results: []map[string]any{
			{
				"Name": "John",
				"Age":  uint64(21),
				"_version": []map[string]any{
					{
						"cid": "bafybeiaahzxsfz55nuqnsll42wxrbdjmy5si222l4ydbrwb53tpxnzdmwq",
						"L1": []map[string]any{
							{
								"cid":  "bafybeibelbquwus6f4lek2uclnmt5o525qbg46pzgxkjmrfjg3hwg63r4i",
								"name": "Age",
							},
							{
								"cid":  "bafybeih7josq7krvco7jc5z2mhiox4dge77zyn3v7gmemfjkkh3xkqwpni",
								"name": "Name",
							},
						},
						"L2": []map[string]any{
							{
								"name": "Age",
							},
							{
								"name": "Name",
							},
						},
					},
				},
			},
		},
	}

	executeTestCase(t, test)
}
