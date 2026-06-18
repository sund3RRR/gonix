package main

import (
	"fmt"
	"log"

	"github.com/sund3RRR/gonix"
	"github.com/sund3RRR/gonix/store"
)

func main() {
	client, err := gonix.NewClient(gonix.ClientConfig{
		Verbosity: gonix.VerbosityWarn,
		LogFormat: gonix.LogFormatRaw,
		Store: gonix.StoreConfig{
			URI: store.Auto,
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

	f, err := client.NewFlake("github:NixOS/nixpkgs/nixos-unstable")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close() //nolint:errcheck

	var packageName string
	if err := f.Output([]string{"packages", gonix.DefaultSystem(), "hello", "name"}, &packageName); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("flake package name=%s\n", packageName)

	pkg, err := f.FetchPackage("hello", gonix.WithFetchPackageSystem(gonix.DefaultSystem()))
	if err != nil {
		log.Fatal(err)
	}

	outputs, err := f.RealizePackage(pkg)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("package=%s version=%s outputs=%+v\n", pkg.Name, pkg.Version, outputs)
}
