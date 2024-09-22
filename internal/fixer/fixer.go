package fixer

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

type FixReferenceOriginsRequest struct {
	BlockType BlockType
	BlockName string
	Version   int
	// The raw HCL contents of each reference origin
	RawContents [][]byte
}

type FixReferenceOriginsResponse struct {
	Error *string
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
	Error *string
	// The updated raw HCL content of this block definition
	RawContent []byte
}
