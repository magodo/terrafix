package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/hcl-lang/reference"
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/magodo/terrafix/internal/filesystem"
	"github.com/magodo/terrafix/internal/state"
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
	fs, err := filesystem.NewMemFS(rootModPath, os.Stdout)
	if err != nil {
		log.Fatalf("error new memory filesystem: %s", err)
	}
	root, err := state.NewRootState(tf, fs, rootModPath)
	if err != nil {
		log.Fatal(err)
	}

	d := root.Decoder()

	for modPath, modState := range root.ModuleStates {
		fmt.Printf("\nModule path: %s\n\n", modPath)
		fmt.Println("Origins:")
		for _, ref := range modState.OriginRefs {
			switch ref := ref.(type) {
			case reference.LocalOrigin:
				fmt.Printf("\t- (Local) %s %s\n", ref.Addr, ref.Range)
			case reference.DirectOrigin:
				fmt.Printf("\t- (Direct) %s\n", ref.Range)
			case reference.PathOrigin:
				fmt.Printf("\t- (Path) %s \n", ref.Range)
			}

			targets, err := d.ReferenceTargetsForOriginAtPos(lang.Path{Path: modPath, LanguageID: "terraform"}, ref.OriginRange().Filename, ref.OriginRange().Start)
			if err != nil {
				log.Fatal(err)
			}
			if len(targets) > 0 {
				fmt.Println("\t\tTargets:")
				for _, ref := range targets {
					fmt.Printf("\t\t- %s %s\n", ref.Path, ref.Range)
				}
			}
		}

		fmt.Println("Targets:")
		if err := printTargetRefs(d, modPath, modState.TargetRefs); err != nil {
			log.Fatal(err)
		}
	}
}

func printTargetRefs(d *decoder.Decoder, modPath string, targets reference.Targets) error {
	for _, ref := range targets {
		fmt.Printf("\t- %s %s %s\n", ref.Addr, ref.RangePtr, ref.Name)
		if ref.RangePtr == nil {
			fmt.Println("\t  (nil RangePtr)")
			continue
		}
		origins := d.ReferenceOriginsTargetingPos(lang.Path{Path: modPath, LanguageID: "terraform"}, ref.RangePtr.Filename, ref.RangePtr.Start)
		if len(origins) > 0 {
			fmt.Println("\t\tOrigins:")
			for _, ref := range origins {
				fmt.Printf("\t\t- %s %s\n", ref.Path, ref.Range)
			}
		}
		if err := printTargetRefs(d, modPath, ref.NestedTargets); err != nil {
			return err
		}
	}
	return nil
}
