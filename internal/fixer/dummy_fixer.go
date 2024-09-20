package fixer

import "fmt"

type DummyFixer struct{}

var _ Fixer = DummyFixer{}

// FixConfig implements Fixer.
func (d DummyFixer) FixConfig(req FixConfigRequest) FixConfigResponse {
	panic("unimplemented")
}

// FixReferenceOrigins implements Fixer.
func (d DummyFixer) FixReferenceOrigins(req FixReferenceOriginsRequest) FixReferenceOriginsResponse {
	var contents []HCLContent
	for _, origin := range req.ReferenceOrigins {
		origin.RawContent = []byte(fmt.Sprintf(`"${%s}-updated"`, origin.RawContent))
		contents = append(contents, origin)
	}
	return FixReferenceOriginsResponse{ReferenceOrigins: contents}
}
