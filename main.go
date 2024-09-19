package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/hcl-lang/reference"
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/magodo/terrafix/internal/state"
	"github.com/magodo/terrafix/internal/terraform/find"
)

func main() {
	rootModPath := "testdata/module"
	tfpath, err := find.FindTF(context.Background(), version.MustConstraints(version.NewConstraint(">=1.0.0")))
	if err != nil {
		log.Fatalf("finding terraform executable: %v", err)
	}
	tf, err := tfexec.NewTerraform(rootModPath, tfpath)
	if err != nil {
		log.Fatalf("error running NewTerraform: %s", err)
	}
	root, err := state.NewRootState(tf, rootModPath)
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

			fmt.Println("\t\tTargets:")
			targets, err := d.ReferenceTargetsForOriginAtPos(lang.Path{Path: modPath, LanguageID: "terraform"}, ref.OriginRange().Filename, ref.OriginRange().Start)
			if err != nil {
				log.Fatal(err)
			}
			for _, ref := range targets {
				fmt.Printf("\t\t- %s %s\n", ref.Path, ref.Range)
			}
		}

		fmt.Println("Targets:")
		printTargetRefs(1, modState.TargetRefs)
	}
}

func printTargetRefs(indent int, targets reference.Targets) {
	prefix := strings.Repeat("\t", indent)
	for _, ref := range targets {
		fmt.Printf("%s- %s %s %s\n", prefix, ref.Addr, ref.RangePtr, ref.Name)
		printTargetRefs(indent+1, ref.NestedTargets)
	}
}
