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
	defer func() {
		_ = client.Close()
	}()

	f, err := client.NewFlake("github:NixOS/nixpkgs/nixos-unstable")
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = f.Close()
	}()

	pkg, err := f.FetchPackage("hello", gonix.WithFetchPackageSystem(gonix.DefaultSystem()))
	if err != nil {
		log.Fatal(err)
	}

	outputs, err := f.DownloadPackage(pkg)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("package=%s version=%s outputs=%+v\n", pkg.Name, pkg.Version, outputs)
}
