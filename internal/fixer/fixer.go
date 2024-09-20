package fixer

import (
	"github.com/hashicorp/hcl/v2"
)

type Fixer interface {
	FixReferenceOrigins(FixReferenceOriginsRequest) FixReferenceOriginsResponse
	FixDefinition(FixDefinitionRequest) FixDefinitionResponse
}

type BlockType int

const (
	BlockTypeProvider   BlockType = 0
	BlockTypeResource   BlockType = 1
	BlockTypeDataSource BlockType = 2
)

type HCLContent struct {
	RawContent []byte
	Range      hcl.Range
}

type FixReferenceOriginsRequest struct {
	BlockType        BlockType
	BlockName        string
	Version          int
	ReferenceOrigins []HCLContent
}

type FixReferenceOriginsResponse struct {
	Error            *string
	ReferenceOrigins []HCLContent
}

type FixDefinitionRequest struct {
	BlockType  BlockType
	BlockName  string
	Version    int
	Definition HCLContent
}

type FixDefinitionResponse struct {
	Error      *string
	Definition HCLContent
}
