package fixer

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

type DummyFixer struct{}

var _ Fixer = DummyFixer{}

func (d DummyFixer) FixDefinition(_ context.Context, req FixDefinitionRequest) (*FixDefinitionResponse, error) {
	sf, diags := hclsyntax.ParseConfig(req.RawContent, "", hcl.InitialPos)
	if diags.HasErrors() {
		return nil, fmt.Errorf(diags.Error())
	}
	_ = sf

	wf, diags := hclwrite.ParseConfig(req.RawContent, "", hcl.InitialPos)
	if diags.HasErrors() {
		return nil, fmt.Errorf(diags.Error())
	}
	wf.Body().Blocks()[0].Body().SetAttributeValue("undefined", cty.StringVal("foo"))

	return &FixDefinitionResponse{RawContent: wf.Bytes()}, nil
}

func (d DummyFixer) FixReferenceOrigins(_ context.Context, req FixReferenceOriginsRequest) (*FixReferenceOriginsResponse, error) {
	var contents [][]byte
	for _, origin := range req.RawContents {
		origin = []byte(fmt.Sprintf(`"${%s.undefined}"`, origin))
		contents = append(contents, origin)
	}
	return &FixReferenceOriginsResponse{RawContents: contents}, nil
}
