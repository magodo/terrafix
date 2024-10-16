package ctrl

import (
	"github.com/hashicorp/terraform-exec/tfexec"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	"github.com/magodo/terrafix/internal/fixer"
)

type Option struct {
	// The root module path
	Path string

	// The target provider's fully qualified address
	ProviderAddr tfaddr.Provider

	TF    *tfexec.Terraform
	Fixer fixer.Fixer
}
