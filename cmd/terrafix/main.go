package main

import (
	"context"
	"flag"
	"log"
	"os/exec"
	"path/filepath"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/magodo/terrafix/internal/ctrl"
	"github.com/magodo/terrafix/internal/fixer"
	"github.com/magodo/terrafix/internal/terraform/find"
	"github.com/magodo/terraform-client-go/tfclient"
)

type FlagSet struct {
	ModulePath   string
	ProviderPath string
	ProviderAddr string
	LogLevel     string
}

func main() {
	var fset FlagSet
	flag.StringVar(&fset.ModulePath, "path", ".", "root module path")
	flag.StringVar(&fset.ProviderAddr, "provider-addr", "", "fully qualified provider address")
	flag.StringVar(&fset.ProviderPath, "provider-path", "", "path to the target provider executable")
	flag.StringVar(&fset.LogLevel, "log-level", hclog.Error.String(), "log level")
	flag.Parse()

	if fset.ProviderAddr == "" {
		log.Fatal(`"--provider-addr" is not specified`)
	}
	if fset.ProviderPath == "" {
		log.Fatal(`"--provider-path" is not specified`)
	}

	ctx := context.Background()

	tfpath, err := find.FindTF(context.Background(), version.MustConstraints(version.NewConstraint(">=1.0.0")))
	if err != nil {
		log.Fatalf("finding terraform executable: %v", err)
	}
	tf, err := tfexec.NewTerraform(fset.ModulePath, tfpath)
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

	ctrl, err := ctrl.NewController(tf, fset.ModulePath, fset.ProviderAddr, fx)
	if err != nil {
		log.Fatal(err)
	}

	if err := ctrl.FixReferenceOrigins(ctx); err != nil {
		log.Fatal(err)
	}

	if err := ctrl.UpdateRootState(); err != nil {
		log.Fatal(err)
	}

	if err := ctrl.FixDefinition(ctx); err != nil {
		log.Fatal(err)
	}
}
