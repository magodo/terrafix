package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-exec/tfexec"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	"github.com/magodo/terrafix/internal/ctrl"
	"github.com/magodo/terrafix/internal/fixer"
	"github.com/magodo/terrafix/internal/terraform/find"
	"github.com/magodo/terraform-client-go/tfclient"
)

type FlagSet struct {
	ProviderPath      string
	ProviderAddr      string
	Output            string
	LogLevel          string
	SkipFixReference  bool
	SkipFixDefinition bool
}

func main() {
	var fset FlagSet

	flag.StringVar(&fset.ProviderAddr, "provider-addr", "", "The fully qualified provider address (e.g. registry.terraform.io/hashicorp/azurerm)")
	flag.StringVar(&fset.ProviderPath, "provider-path", "", "The path to the target provider executable")
	flag.StringVar(&fset.Output, "output", "", "The output folder where the updated configs will be written to (by default writes to the stdout)")
	flag.StringVar(&fset.LogLevel, "log-level", hclog.Error.String(), "The log level")
	flag.BoolVar(&fset.SkipFixReference, "skip-fix-reference", false, "Whether to skip fixing the reference")
	flag.BoolVar(&fset.SkipFixDefinition, "skip-fix-definition", false, "Whether to skip fixing the definition")

	flag.Usage = func() {
		fmt.Fprint(os.Stderr, `usage: terrafix [options] root-module-path

terrafix fixes user's terraform configurations to match the targeting provider's schema.
`)
		flag.PrintDefaults()
	}
	flag.Parse()

	if l := len(flag.Args()); l != 1 {
		log.Fatalf("expects one argument, got=%d", l)
	}
	if fset.ProviderPath == "" {
		log.Fatal(`"--provider-path" is not specified`)
	}

	ctx := context.Background()

	modulePath := flag.Arg(0)
	if fset.ProviderAddr == "" {
		// Deduce the provider address via the provider executable name,
		// and assuming it is namespaced by hashicorp.
		// This is a shorthand only for hashicorp owned providers.
		fset.ProviderAddr = "registry.terraform.io/hashicorp/" +
			strings.TrimPrefix(filepath.Base(fset.ProviderPath), "terraform-provider-")
	}

	tfpath, err := find.FindTF(context.Background(), version.MustConstraints(version.NewConstraint(">=1.0.0")))
	if err != nil {
		log.Fatalf("finding terraform executable: %v", err)
	}
	tf, err := tfexec.NewTerraform(modulePath, tfpath)
	if err != nil {
		log.Fatalf("error running NewTerraform: %s", err)
	}

	var fx fixer.Fixer
	// Test purpose
	if fset.ProviderPath == "terrafix-dummy" {
		fx = &fixer.DummyFixer{}
	} else {
		opts := tfclient.Option{
			Cmd: exec.Command(fset.ProviderPath),
			Logger: hclog.New(&hclog.LoggerOptions{
				Output: hclog.DefaultOutput,
				Level:  hclog.LevelFromString(fset.LogLevel),
				Name:   filepath.Base(fset.ProviderPath),
			}),
		}
		c, err := tfclient.New(opts)
		if err != nil {
			log.Fatal(err)
		}
		defer c.Close()

		fx, err = fixer.NewProviderFixer(c)
		if err != nil {
			log.Fatalf("new provider fixer: %v", err)
		}
	}

	paddr, err := tfaddr.ParseProviderSource(fset.ProviderAddr)
	if err != nil {
		log.Fatalf("failed to parse provider addr %q: %v", fset.ProviderAddr, err)
	}

	ctrl, err := ctrl.NewController(ctrl.Option{
		Path:         modulePath,
		ProviderAddr: paddr,
		TF:           tf,
		Fixer:        fx,
	})
	if err != nil {
		log.Fatal(err)
	}

	if !fset.SkipFixReference {
		if err := ctrl.FixReferenceOrigins(ctx); err != nil {
			log.Fatal(err)
		}

		if err := ctrl.UpdateRootState(); err != nil {
			log.Fatal(err)
		}
	}

	if !fset.SkipFixDefinition {
		if err := ctrl.FixDefinition(ctx); err != nil {
			log.Fatal(err)
		}
	}

	var odir *string
	if fset.Output != "" {
		odir = &fset.Output
	}

	if err := ctrl.Write(odir); err != nil {
		log.Fatal(err)
	}
}
