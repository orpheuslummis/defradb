// Copyright 2022 Democratized Data Foundation
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package one_to_many

import (
	"testing"

	testUtils "github.com/sourcenetwork/defradb/tests/integration"
)

func TestQueryOneToManyWithSingleChildLimit(t *testing.T) {
	test := testUtils.RequestTestCase{
		Description: "One-to-many relation query from many side with limit",
		Request: `query {
			author {
				name
				published (limit: 1) {
					name
					rating
				}
			}
		}`,
		Docs: map[int][]string{
			//books
			0: { // bae-fd541c25-229e-5280-b44b-e5c2af3e374d
				`{
					"name": "Painted House",
					"rating": 4.9,
					"author_id": "bae-41598f0c-19bc-5da6-813b-e80f14a10df3"
				}`,
				`{
					"name": "A Time for Mercy",
					"rating": 4.5,
					"author_id": "bae-41598f0c-19bc-5da6-813b-e80f14a10df3"
					}`,
				`{
					"name": "Theif Lord",
					"rating": 4.8,
					"author_id": "bae-b769708d-f552-5c3d-a402-ccfd7ac7fb04"
				}`,
			},
			//authors
			1: {
				// bae-41598f0c-19bc-5da6-813b-e80f14a10df3
				`{
					"name": "John Grisham",
					"age": 65,
					"verified": true
				}`,
				// bae-b769708d-f552-5c3d-a402-ccfd7ac7fb04
				`{
					"name": "Cornelia Funke",
					"age": 62,
					"verified": false
				}`,
			},
		},
		Results: []map[string]any{
			{
				"name": "John Grisham",
				"published": []map[string]any{
					{
						"name":   "Painted House",
						"rating": 4.9,
					},
				},
			},
			{
				"name": "Cornelia Funke",
				"published": []map[string]any{
					{
						"name":   "Theif Lord",
						"rating": 4.8,
					},
				},
			},
		},
	}

	executeTestCase(t, test)
}

func TestQueryOneToManyWithMultipleChildLimits(t *testing.T) {
	test := testUtils.RequestTestCase{
		Description: "One-to-many relation query from many side with limit",
		Request: `query {
			author {
				name
				p1: published (limit: 1) {
					name
					rating
				}
				p2: published (limit: 2) {
					name
					rating
				}
			}
		}`,
		Docs: map[int][]string{
			//books
			0: { // bae-fd541c25-229e-5280-b44b-e5c2af3e374d
				`{
					"name": "Painted House",
					"rating": 4.9,
					"author_id": "bae-41598f0c-19bc-5da6-813b-e80f14a10df3"
				}`,
				`{
					"name": "A Time for Mercy",
					"rating": 4.5,
					"author_id": "bae-41598f0c-19bc-5da6-813b-e80f14a10df3"
					}`,
				`{
					"name": "Theif Lord",
					"rating": 4.8,
					"author_id": "bae-b769708d-f552-5c3d-a402-ccfd7ac7fb04"
				}`,
			},
			//authors
			1: {
				// bae-41598f0c-19bc-5da6-813b-e80f14a10df3
				`{
					"name": "John Grisham",
					"age": 65,
					"verified": true
				}`,
				// bae-b769708d-f552-5c3d-a402-ccfd7ac7fb04
				`{
					"name": "Cornelia Funke",
					"age": 62,
					"verified": false
				}`,
			},
		},
		Results: []map[string]any{
			{
				"name": "John Grisham",
				"p1": []map[string]any{
					{
						"name":   "Painted House",
						"rating": 4.9,
					},
				},
				"p2": []map[string]any{
					{
						"name":   "Painted House",
						"rating": 4.9,
					},
					{
						"name":   "A Time for Mercy",
						"rating": 4.5,
					},
				},
			},
			{
				"name": "Cornelia Funke",
				"p1": []map[string]any{
					{
						"name":   "Theif Lord",
						"rating": 4.8,
					},
				},
				"p2": []map[string]any{
					{
						"name":   "Theif Lord",
						"rating": 4.8,
					},
				},
			},
		},
	}

	executeTestCase(t, test)
}
