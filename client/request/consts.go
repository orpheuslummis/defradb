// Copyright 2022 Democratized Data Foundation
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package request

const (
	// GQL special field, returns the host object's type name
	// https://spec.graphql.org/October2021/#sec-Type-Name-Introspection
	TypeNameFieldName = "__typename"

	Cid         = "cid"
	Data        = "data"
	DocKey      = "dockey"
	DocKeys     = "dockeys"
	FieldName   = "field"
	Id          = "id"
	Ids         = "ids"
	ShowDeleted = "showDeleted"

	FilterClause  = "filter"
	GroupByClause = "groupBy"
	LimitClause   = "limit"
	OffsetClause  = "offset"
	OrderClause   = "order"
	DepthClause   = "depth"

	AverageFieldName = "_avg"
	CountFieldName   = "_count"
	KeyFieldName     = "_key"
	GroupFieldName   = "_group"
	DeletedFieldName = "_deleted"
	SumFieldName     = "_sum"
	VersionFieldName = "_version"

	ExplainLabel = "explain"

	LatestCommitsName = "latestCommits"
	CommitsName       = "commits"

	CommitTypeName           = "Commit"
	LinksFieldName           = "links"
	HeightFieldName          = "height"
	CidFieldName             = "cid"
	DockeyFieldName          = "dockey"
	CollectionIDFieldName    = "collectionID"
	SchemaVersionIDFieldName = "schemaVersionId"
	DeltaFieldName           = "delta"

	LinksNameFieldName = "name"
	LinksCidFieldName  = "cid"

	ASC  = OrderDirection("ASC")
	DESC = OrderDirection("DESC")
)

var (
	NameToOrderDirection = map[string]OrderDirection{
		string(ASC):  ASC,
		string(DESC): DESC,
	}

	ReservedFields = map[string]bool{
		TypeNameFieldName: true,
		VersionFieldName:  true,
		GroupFieldName:    true,
		CountFieldName:    true,
		SumFieldName:      true,
		AverageFieldName:  true,
		KeyFieldName:      true,
		DeletedFieldName:  true,
	}

	Aggregates = map[string]struct{}{
		CountFieldName:   {},
		SumFieldName:     {},
		AverageFieldName: {},
	}

	CommitQueries = map[string]struct{}{
		LatestCommitsName: {},
		CommitsName:       {},
	}

	VersionFields = []string{
		HeightFieldName,
		CidFieldName,
		DockeyFieldName,
		CollectionIDFieldName,
		SchemaVersionIDFieldName,
		DeltaFieldName,
	}

	LinksFields = []string{
		LinksNameFieldName,
		LinksCidFieldName,
	}
)
