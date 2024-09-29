package fixer

import (
	"context"
	"fmt"

	"github.com/magodo/terraform-client-go/tfclient"
	"github.com/magodo/terraform-client-go/tfclient/typ"
	"github.com/zclconf/go-cty/cty"
)

type ProviderFixer struct {
	tfc tfclient.Client
}

var _ Fixer = ProviderFixer{}

func NewProviderFixer(c tfclient.Client) (*ProviderFixer, error) {
	return &ProviderFixer{tfc: c}, nil
}

var _ Fixer = ProviderFixer{}

func (p ProviderFixer) FixDefinition(ctx context.Context, req FixDefinitionRequest) (*FixDefinitionResponse, error) {
	resp, diags := p.tfc.CallFunction(ctx, typ.CallFunctionRequest{
		FunctionName: "terrafix_config_definition",
		Arguments: []cty.Value{
			cty.StringVal(string(req.BlockType)),
			cty.StringVal(req.BlockName),
			cty.NumberIntVal(int64(req.Version)),
			cty.StringVal(string(req.RawContent)),
			cty.StringVal(string(req.RawState)),
		},
	})
	if diags.HasErrors() {
		return nil, diags.Err()
	}
	if resp.Err != nil {
		return nil, diags.Err()
	}
	return &FixDefinitionResponse{RawContent: []byte(resp.Result.AsString())}, nil
}

func (p ProviderFixer) FixReferenceOrigins(ctx context.Context, req FixReferenceOriginsRequest) (*FixReferenceOriginsResponse, error) {
	var contents []cty.Value
	for _, content := range req.RawContents {
		contents = append(contents, cty.StringVal(string(content)))
	}
	resp, diags := p.tfc.CallFunction(ctx, typ.CallFunctionRequest{
		FunctionName: "terrafix_config_references",
		Arguments: []cty.Value{
			cty.StringVal(string(req.BlockType)),
			cty.StringVal(req.BlockName),
			cty.NumberIntVal(int64(req.Version)),
			cty.ListVal(contents),
		},
	})
	if diags.HasErrors() {
		return nil, diags.Err()
	}
	if resp.Err != nil {
		return nil, diags.Err()
	}
	if resp.Result.IsNull() {
		return nil, fmt.Errorf("the provider returns null result, which is a provider bug.")
	}
	l := resp.Result.AsValueSlice()
	if len(l) != len(req.RawContents) {
		return nil, fmt.Errorf("the provider's response length doesn't match the request length %d, got=%d, which is a provider bug.", len(req.RawContents), len(l))
	}
	var updatedContents [][]byte
	for _, content := range l {
		updatedContents = append(updatedContents, []byte(content.AsString()))
	}
	return &FixReferenceOriginsResponse{RawContents: updatedContents}, nil
}
