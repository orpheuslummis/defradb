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

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAddSchemaFromFile(t *testing.T) {
	conf := NewDefraNodeDefaultConfig(t)
	stopDefra := runDefraNode(t, conf)

	fname := schemaFileFixture(t, "schema.graphql", `
	type User {
		id: ID
		name: String
	}`)

	stdout, _ := runDefraCommand(t, conf, []string{"client", "schema", "add", "-f", fname})

	nodeLog := stopDefra()

	assert.Contains(t, stdout, `{"data":{"result":"success"}}`)
	assertNotContainsSubstring(t, nodeLog, "ERROR")
}

func TestAddSchemaWithDuplicateType(t *testing.T) {
	conf := NewDefraNodeDefaultConfig(t)
	stopDefra := runDefraNode(t, conf)

	fname1 := schemaFileFixture(t, "schema1.graphql", `type Post { id: ID title: String }`)
	fname2 := schemaFileFixture(t, "schema2.graphql", `type Post { id: ID author: String }`)

	stdout1, _ := runDefraCommand(t, conf, []string{"client", "schema", "add", "-f", fname1})
	stdout2, _ := runDefraCommand(t, conf, []string{"client", "schema", "add", "-f", fname2})

	_ = stopDefra()

	assertContainsSubstring(t, stdout1, `{"data":{"result":"success"}}`)
	assertContainsSubstring(t, stdout2, `schema type already exists. Name: Post`)
}

/* disabled because current implementation doesn't support this currently
func TestAddSchemaWithMultipleFiles(t *testing.T) {
	conf := NewDefraNodeDefaultConfig(t)
	stopDefra := runDefraNode(t, conf)

	fname1 := schemaFileFixture(t, "schema1.graphql", `type Post { id: ID title: String }`)
	fname2 := schemaFileFixture(t, "schema2.graphql", `type User { id: ID name: String }`)

	stdout, _ := runDefraCommand(t, conf, []string{"client", "schema", "add", "-f", fname1, "-f", fname2})

	nodeLog := stopDefra()

	assertContainsSubstring(t, stdout, `{"data":{"result":"success"}}`)
	assertNotContainsSubstring(t, nodeLog, "ERROR")

	stdout, _ = runDefraCommand(t, conf, []string{"client", "query",
		`query IntrospectionQuery { __schema { types { name } } }`})
	assertContainsSubstring(t, stdout, `{"name":"Post"}`)
	assertContainsSubstring(t, stdout, `{"name":"User"}`)
}
*/

/* disabled because current implementation doesn't support this currently
func TestAddSchemaWithMultipleFilesWithIntertwinedSchemas(t *testing.T) {
	conf := NewDefraNodeDefaultConfig(t)
	stopDefra := runDefraNode(t, conf)

	fname1 := schemaFileFixture(t, "schema1.graphql", `type Post { id: ID title: String }`)
	fname2 := schemaFileFixture(t, "schema2.graphql", `type User { id: ID posts: [Post] }`)

	stdout, _ := runDefraCommand(t, conf, []string{"client", "schema", "add", "-f", fname1, "-f", fname2})

	nodeLog := stopDefra()

	assertContainsSubstring(t, stdout, `{"data":{"result":"success"}}`)
	assertNotContainsSubstring(t, nodeLog, "ERROR")

	stdout, _ = runDefraCommand(t, conf, []string{"client", "query",
		`query IntrospectionQuery { __schema { types { name } } }`})
	assertContainsSubstring(t, stdout, `{"name":"Post"}`)
	assertContainsSubstring(t, stdout, `{"name":"User"}`)
}
*/
