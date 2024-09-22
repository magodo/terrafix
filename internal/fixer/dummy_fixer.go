package fixer

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

type DummyFixer struct{}

var _ Fixer = DummyFixer{}

func (d DummyFixer) FixDefinition(req FixDefinitionRequest) FixDefinitionResponse {
	sf, diags := hclsyntax.ParseConfig(req.Definition.RawContent, "", hcl.InitialPos)
	if diags.HasErrors() {
		err := diags.Error()
		return FixDefinitionResponse{Error: &err}
	}
	_ = sf

	wf, diags := hclwrite.ParseConfig(req.Definition.RawContent, "", hcl.InitialPos)
	if diags.HasErrors() {
		err := diags.Error()
		return FixDefinitionResponse{Error: &err}
	}
	wf.Body().Blocks()[0].Body().SetAttributeValue("undefined", cty.StringVal("foo"))

	return FixDefinitionResponse{Definition: HCLContent{RawContent: wf.Bytes(), Range: req.Definition.Range}}
}

func (d DummyFixer) FixReferenceOrigins(req FixReferenceOriginsRequest) FixReferenceOriginsResponse {
	var contents []HCLContent
	for _, origin := range req.ReferenceOrigins {
		origin.RawContent = []byte(fmt.Sprintf(`"${%s.undefined}"`, origin.RawContent))
		contents = append(contents, origin)
	}
	return FixReferenceOriginsResponse{ReferenceOrigins: contents}
}
