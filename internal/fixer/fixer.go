package fixer

import (
	"github.com/hashicorp/hcl-lang/lang"
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

type FixReferenceOriginsRequest struct {
	BlockType        BlockType
	BlockName        string
	Version          int
	ReferenceOrigins []ReferenceOrigin
}

type FixReferenceOriginsResponse struct {
	ReferenceOrigins []ReferenceOrigin
}

type ReferenceOrigin struct {
	Addr    lang.Address
	Range   hcl.Range
	Content []byte
}

type FixConfigRequest struct {
	TypeName string
	Version  int
	Range    hcl.Range
	Content  []byte
}

type FixConfigResponse struct {
	Range   hcl.Range
	Content []byte
}
