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
	var origins []ReferenceOrigin
	for _, origin := range req.ReferenceOrigins {
		origin.Content = []byte(fmt.Sprintf(`"${%s}-updated"`, origin.Content))
		origins = append(origins, origin)
	}
	return FixReferenceOriginsResponse{ReferenceOrigins: origins}
}
