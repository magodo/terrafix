package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/magodo/terrafix/internal/ctrl"
	"github.com/magodo/terrafix/internal/fixer"
	"github.com/magodo/terrafix/internal/terraform/find"
)

func main() {
	rootModulePath := flag.String("p", "", "root module path")
	flag.Parse()
	rootModPath := *rootModulePath

	tfpath, err := find.FindTF(context.Background(), version.MustConstraints(version.NewConstraint(">=1.0.0")))
	if err != nil {
		log.Fatalf("finding terraform executable: %v", err)
	}
	tf, err := tfexec.NewTerraform(rootModPath, tfpath)
	if err != nil {
		log.Fatalf("error running NewTerraform: %s", err)
	}

	ctrl, err := ctrl.NewController(tf, rootModPath, "registry.terraform.io/hashicorp/azurerm", fixer.DummyFixer{})
	if err != nil {
		log.Fatal(err)
	}

	if err := ctrl.FixReferenceOrigins(); err != nil {
		log.Fatal(err)
	}
}
