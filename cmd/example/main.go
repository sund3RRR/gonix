package main

import (
	"fmt"
	"log"

	"github.com/sund3RRR/gonix"
	"github.com/sund3RRR/gonix/store"
)

func main() {
	const storeURI = store.Auto
	const flakeRef = "github:NixOS/nixpkgs/nixos-unstable"
	const readOnly = true
	var system = gonix.DefaultSystem()

	// Create a new gonix Runtime
	r, err := gonix.NewRuntime(
		gonix.WithExperimentalFeatures(
			gonix.ExperimentalFeatureNixCommand,
			gonix.ExperimentalFeatureFlakes,
			gonix.ExperimentalFeature("read-only-local-store"),
		),
		gonix.WithVerbosity(gonix.VerbosityWarn),
		gonix.WithLogFormat(gonix.LogFormatRaw),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = r.Close()
	}()

	// Open a read-only store
	s, err := r.OpenStore(storeURI, store.WithReadOnly(readOnly))
	if err != nil {
		log.Fatal(err)
	}

	// Create a new gonix Client with a read-only store
	c, err := gonix.NewClient(r, gonix.WithClientStore(s))
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = c.Close()
	}()

	// Parse and lock the flake reference
	ref, err := c.ParseFlakeRef(flakeRef)
	if err != nil {
		log.Fatal(err)
	}

	locked, err := c.LockFlake(ref)
	if err != nil {
		log.Fatal(err)
	}

	// Fetch and print packages
	fmt.Printf("flake=%s system=%s store=%s readOnly=%t\n", flakeRef, system, storeURI, readOnly)
	for _, name := range []string{"hello", "git", "kubectl", "openssl"} {
		pkg, err := c.FetchPackage(locked, name, gonix.WithFetchPackageSystem(system))
		if err != nil {
			fmt.Printf("FAIL %-18s %v\n", name, err)
			return
		}

		fmt.Printf("OK   %-18s name=%q version=%q drvPath=%q outPath=%q outputs=%v\n",
			name, pkg.Name, pkg.Version, pkg.DrvPath, pkg.OutPath, pkg.Outputs)
	}
}
