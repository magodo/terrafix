package fixer

import (
	"github.com/hashicorp/hcl/v2"
)

type Fixer interface {
	FixReferenceOrigins(FixReferenceOriginsRequest) FixReferenceOriginsResponse
	FixConfig(FixConfigRequest) FixConfigResponse
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
	ReferenceOrigins []HCLContent
}

type FixConfigRequest struct {
	TypeName string
	Version  int
	Config   HCLContent
}

type FixConfigResponse struct {
	Config HCLContent
}
