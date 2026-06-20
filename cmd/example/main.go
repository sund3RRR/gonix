package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/sund3RRR/gonix"
	"github.com/sund3RRR/gonix/store"
)

func main() {
	client, err := gonix.NewClient(gonix.ClientConfig{
		Verbosity: gonix.VerbosityInfo,
		LogFormat: gonix.LogFormatBar,
		Store: gonix.StoreConfig{
			URI: store.Daemon,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close() //nolint:errcheck

	var evaluation struct {
		Message string `nix:"message" validate:"required"`
		Answer  int    `nix:"answer" validate:"required"`
	}
	if err := client.Eval(`{ message = "hello from Nix"; answer = 6 * 7; }`, &evaluation); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("evaluation=%+v\n", evaluation)

	ref := "github:NixOS/nixpkgs/nixpkgs-unstable"
	f, err := client.OpenFlake(ref)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close() //nolint:errcheck

	fmt.Printf("flake='%s' fragment='%s' fingerprint='%s' \n", ref, f.Fragment(), f.Fingerprint())

	lockInfo, err := f.LockInfo()
	if err != nil {
		log.Fatal(err)
	}
	lockJSON, err := json.Marshal(lockInfo)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("flake_meta=%v\n", string(lockJSON))

	system := gonix.DefaultSystem()
	packageValue, err := f.OutputValue([]string{"legacyPackages", system, "hello"})
	if err != nil {
		log.Fatal(err)
	}
	defer packageValue.Close() //nolint:errcheck

	var pkg struct {
		Name    string `nix:"name" validate:"required"`
		DrvPath string `nix:"drvPath" validate:"required"`
	}
	if err := client.Unmarshal(packageValue, &pkg); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("package=%s drvPath=%s\n", pkg.Name, pkg.DrvPath)

	outputs, err := client.Realize(pkg.DrvPath)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("realized outputs=%+v\n", outputs)
}
