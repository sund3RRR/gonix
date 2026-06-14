package store

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Config contains typed Nix store settings.
//
// Nix receives non-zero settings as store URL query parameters.
type Config struct {
	// StoreDir is the logical Nix store directory, usually /nix/store.
	StoreDir string `json:"store,omitempty"`

	// PathInfoCacheSize is the in-memory store path metadata cache size.
	PathInfoCacheSize int `json:"path-info-cache-size,omitempty"`

	// Trusted allows paths from this store to be used as substitutes even when
	// they are not signed by a key listed in trusted-public-keys.
	Trusted bool `json:"trusted,omitempty"`

	// Priority is this store's substituter priority; lower values have higher priority.
	Priority int `json:"priority,omitempty"`

	// WantMassQuery reports whether this store can efficiently answer bulk path
	// validity queries when used as a substituter.
	WantMassQuery bool `json:"want-mass-query,omitempty"`

	// SystemFeatures lists system features available for builds on this store,
	// such as "kvm".
	SystemFeatures []string `json:"system-features,omitempty"`

	// ReadOnly opens the store in read-only mode when the backend supports it.
	ReadOnly bool `json:"read-only,omitempty"`

	// BuildDir is the host directory used for temporary derivation build directories.
	BuildDir string `json:"build-dir,omitempty"`

	// LogDir is the directory where Nix stores log files.
	LogDir string `json:"log,omitempty"`

	// RealStoreDir is the physical filesystem path of the Nix store.
	RealStoreDir string `json:"real,omitempty"`

	// RequireSignatures controls whether store paths copied into this store must
	// have a trusted signature.
	RequireSignatures bool `json:"require-sigs,omitempty"`

	// RootDir is the directory prefixed to other local filesystem store paths.
	RootDir string `json:"root,omitempty"`

	// StateDir is the directory where Nix stores state.
	StateDir string `json:"state,omitempty"`
}

// Params serializes c as Nix store parameters.
func (c Config) Params() (map[string]string, error) {
	data, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()

	var values map[string]any
	if err := decoder.Decode(&values); err != nil {
		return nil, err
	}

	params := make(map[string]string, len(values))
	for key, value := range values {
		params[key] = stringParam(value)
	}

	return params, nil
}

func stringParam(value any) string {
	switch value := value.(type) {
	case nil:
		return ""
	case string:
		return value
	case bool:
		return strconv.FormatBool(value)
	case json.Number:
		return value.String()
	case []any:
		parts := make([]string, 0, len(value))
		for _, item := range value {
			parts = append(parts, stringParam(item))
		}
		return strings.Join(parts, " ")
	default:
		return fmt.Sprintf("%s", value)
	}
}
