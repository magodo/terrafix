package fixer

import "context"

type Fixer interface {
	FixReferenceOrigins(context.Context, FixReferenceOriginsRequest) (*FixReferenceOriginsResponse, error)
	FixDefinition(context.Context, FixDefinitionRequest) (*FixDefinitionResponse, error)
}

type BlockType string

const (
	BlockTypeProvider   BlockType = "provider"
	BlockTypeResource   BlockType = "resource"
	BlockTypeDataSource BlockType = "datasource"
)

type FixReferenceOriginsRequest struct {
	BlockType BlockType
	BlockName string
	Version   int
	// The raw HCL contents of each reference origin
	RawContents [][]byte
}

type FixReferenceOriginsResponse struct {
	// The updated raw HCL contents of each reference origin
	RawContents [][]byte
}

type FixDefinitionRequest struct {
	BlockType BlockType
	BlockName string
	Version   int
	// The raw HCL content of this block definition
	RawContent []byte
}

type FixDefinitionResponse struct {
	// The updated raw HCL content of this block definition
	RawContent []byte
}
