package main

import (
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/sund3RRR/gonix"
	"github.com/sund3RRR/gonix/store"
)

func main() {
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
	defer closeResource("runtime", r.Close)

	features, err := r.Setting("experimental-features")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("experimental-features: %s\n", features)

	s, err := r.OpenStore(gonix.DefaultStoreDir, store.WithReadOnly(true))
	if err != nil {
		log.Fatal(err)
	}

	uri, err := s.URI()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("store uri: %s\n", uri)

	storeDir, err := s.StoreDir()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("store dir: %s\n", storeDir)

	version, err := s.Version()
	if err != nil {
		log.Fatal(err)
	}
	if version == "" {
		fmt.Println("store version: unavailable")
	} else {
		fmt.Printf("store version: %s\n", version)
	}

	pathText := gonix.DefaultStoreDir + "/00000000000000000000000000000000-demo"
	if len(os.Args) > 1 {
		pathText = os.Args[1]
	}

	path, err := s.ParsePath(pathText)
	if err != nil {
		log.Fatal(err)
	}
	defer closeResource("store path", path.Close)

	name, err := path.Name()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("path name: %s\n", name)

	hash, err := path.Hash()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("path hash: %s\n", hex.EncodeToString(hash[:]))

	realPath, err := s.RealPath(path)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("real path: %s\n", realPath)

	valid, err := s.IsValidPath(path)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("valid in local store: %t\n", valid)

	_, err = s.ParsePath("/not/a/nix/store/path")
	if err != nil {
		var nixErr *gonix.Error
		if errors.As(err, &nixErr) {
			fmt.Printf("structured nix error: code=%s message=%q\n", nixErr.Code, nixErr.Message)
		} else {
			fmt.Printf("parse error: %v\n", err)
		}
	}
}

func closeResource(name string, close func() error) {
	if err := close(); err != nil {
		log.Printf("failed to close %s: %v", name, err)
	}
}
