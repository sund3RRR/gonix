package main

import (
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

	f, err := client.NewFlake("github:sund3RRR/tuxedo-nixos")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close() //nolint:errcheck

	system := gonix.MakeSystem(gonix.OSLinux, gonix.ArchX86_64)

	packages, err := f.ListPackages(gonix.WithListPackagesSystem(system))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("flake exposes %d top-level packages for %s: %+v\n", len(packages), system, packages)

	packageAttr := "default"

	var packageName string
	if err := f.Output([]string{"packages", system, packageAttr, "name"}, &packageName); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("flake package %s has name=%s\n", packageAttr, packageName)

	pkg, err := f.FetchPackage(packageAttr, gonix.WithFetchPackageSystem(system))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("package=%s version=%s\n", pkg.Name, pkg.Version)

	outputs, err := f.RealizePackage(pkg)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("realized outputs=%+v\n", outputs)
}
