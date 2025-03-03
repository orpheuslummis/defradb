// Copyright 2022 Democratized Data Foundation
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package tests

import (
	"github.com/sourcenetwork/immutable"

	"github.com/sourcenetwork/defradb/config"
)

// TestCase contains the details of the test case to execute.
type TestCase struct {
	// Test description, optional.
	Description string

	// Actions contains the set of actions and their expected results that
	// this test should execute.  They will execute in the order that they
	// are provided.
	Actions []any
}

// SetupComplete is a flag to explicitly notify the change detector at which point
// setup is complete so that it may split actions across database code-versions.
//
// If a SetupComplete action is not provided the change detector will split before
// the first item that is neither a SchemaUpdate, CreateDoc or UpdateDoc action.
type SetupComplete struct{}

// ConfigureNode allows the explicit configuration of new Defra nodes.
//
// If no nodes are explicitly configured, a default one will be setup.  There is no
// upper limit to the number that can be configured.
//
// Nodes may be explicitly referenced by index by other actions using `NodeID` properties.
// If the action has a `NodeID` property and it is not specified, the action will be
// effected on all nodes.
type ConfigureNode struct {
	config.Config
}

// SchemaUpdate is an action that will update the database schema.
type SchemaUpdate struct {
	// NodeID may hold the ID (index) of a node to apply this update to.
	//
	// If a value is not provided the update will be applied to all nodes.
	NodeID immutable.Option[int]

	// The schema update.
	Schema string

	// Any error expected from the action. Optional.
	//
	// String can be a partial, and the test will pass if an error is returned that
	// contains this string.
	ExpectedError string
}

type SchemaPatch struct {
	// NodeID may hold the ID (index) of a node to apply this patch to.
	//
	// If a value is not provided the patch will be applied to all nodes.
	NodeID immutable.Option[int]

	Patch         string
	ExpectedError string
}

// CreateDoc will attempt to create the given document in the given collection
// using the collection api.
type CreateDoc struct {
	// NodeID may hold the ID (index) of a node to apply this create to.
	//
	// If a value is not provided the document will be created in all nodes.
	NodeID immutable.Option[int]

	// The collection in which this document should be created.
	CollectionID int

	// The document to create, in JSON string format.
	Doc string

	// Any error expected from the action. Optional.
	//
	// String can be a partial, and the test will pass if an error is returned that
	// contains this string.
	ExpectedError string
}

// DeleteDoc will attempt to delete the given document in the given collection
// using the collection api.
type DeleteDoc struct {
	// NodeID may hold the ID (index) of a node to apply this create to.
	//
	// If a value is not provided the document will be created in all nodes.
	NodeID immutable.Option[int]

	// The collection in which this document should be deleted.
	CollectionID int

	// The index-identifier of the document within the collection.  This is based on
	// the order in which it was created, not the ordering of the document within the
	// database.
	DocID int

	// Any error expected from the action. Optional.
	//
	// String can be a partial, and the test will pass if an error is returned that
	// contains this string.
	ExpectedError string

	// Setting DontSync to true will prevent waiting for that delete.
	DontSync bool
}

// UpdateDoc will attempt to update the given document in the given collection
// using the collection api.
type UpdateDoc struct {
	// NodeID may hold the ID (index) of a node to apply this update to.
	//
	// If a value is not provided the update will be applied to all nodes.
	NodeID immutable.Option[int]

	// The collection in which this document exists.
	CollectionID int

	// The index-identifier of the document within the collection.  This is based on
	// the order in which it was created, not the ordering of the document within the
	// database.
	DocID int

	// The document update, in JSON string format. Will only update the properties
	// provided.
	Doc string

	// Any error expected from the action. Optional.
	//
	// String can be a partial, and the test will pass if an error is returned that
	// contains this string.
	ExpectedError string

	// Setting DontSync to true will prevent waiting for that update.
	DontSync bool
}

// Request represents a standard Defra (GQL) request.
type Request struct {
	// NodeID may hold the ID (index) of a node to execute this request on.
	//
	// If a value is not provided the request will be executed against all nodes,
	// in which case the expected results must all match across all nodes.
	NodeID immutable.Option[int]

	// The request to execute.
	Request string

	// The expected (data) results of the issued request.
	Results []map[string]any

	// Any error expected from the action. Optional.
	//
	// String can be a partial, and the test will pass if an error is returned that
	// contains this string.
	ExpectedError string
}

// TransactionRequest2 represents a transactional request.
//
// A new transaction will be created for the first TransactionRequest2 of any given
// TransactionId. TransactionRequest2s will be submitted to the database in the order
// in which they are recieved (interleaving amongst other actions if provided), however
// they will not be commited until a TransactionCommit of matching TransactionId is
// provided.
type TransactionRequest2 struct {
	// Used to identify the transaction for this to run against.
	TransactionID int

	// The request to run against the transaction.
	Request string

	// The expected (data) results of the issued request.
	Results []map[string]any

	// Any error expected from the action. Optional.
	//
	// String can be a partial, and the test will pass if an error is returned that
	// contains this string.
	ExpectedError string
}

// TransactionCommit represents a commit request for a transaction of the given id.
type TransactionCommit struct {
	// Used to identify the transaction to commit.
	TransactionID int

	// Any error expected from the action. Optional.
	//
	// String can be a partial, and the test will pass if an error is returned that
	// contains this string.
	ExpectedError string
}

// SubscriptionRequest represents a subscription request.
//
// The subscription will remain active until shortly after all actions have been processed.
// The results of the subscription will then be asserted upon.
type SubscriptionRequest struct {
	// The subscription request to submit.
	Request string

	// The expected (data) results yielded through the subscription across its lifetime.
	Results []map[string]any

	// Any error expected from the action. Optional.
	//
	// String can be a partial, and the test will pass if an error is returned that
	// contains this string.
	ExpectedError string
}

type IntrospectionRequest struct {
	// The introspection request to use when fetching schema state.
	//
	// Available properties can be found in the GQL spec:
	// https://spec.graphql.org/October2021/#sec-Introspection
	Request string

	// The full data expected to be returned from the introspection request.
	ExpectedData map[string]any

	// If [ExpectedData] is nil and this is populated, the test framework will assert
	// that the value given exists in the actual results.
	//
	// If this contains nested maps it only requires the last (i.e. non-map) value to
	// be present along the given path.  If an array/slice is present in this chain,
	// it will assert that the items in the expected-array have exact matches in the
	// corresponding result-array (inner maps are not traversed beyond the array,
	// the full array-item must match exactly).
	ContainsData map[string]any

	// Any error expected from the action. Optional.
	//
	// String can be a partial, and the test will pass if an error is returned that
	// contains this string.
	ExpectedError string
}
