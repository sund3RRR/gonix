package flake

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// LockInfo is the Go representation of a Nix flake lock graph.
type LockInfo struct {
	// Version is the Nix lock-file schema version.
	Version uint64 `json:"version"`

	// Root names the root node in Nodes.
	Root string `json:"root"`

	// Nodes contains the resolved lock graph keyed by Nix-assigned node name.
	Nodes map[string]LockNode `json:"nodes"`
}

// LockNode describes one node in a Nix flake lock graph.
type LockNode struct {
	// Inputs maps input names to direct node or follows edges.
	Inputs map[string]LockInput `json:"inputs,omitempty"`

	// Original contains the scheme-specific attributes of the requested input.
	Original LockReference `json:"original,omitempty"`

	// Locked contains the scheme-specific attributes of the resolved input.
	Locked LockReference `json:"locked,omitempty"`

	// Flake reports whether this node is a flake.
	//
	// Nix omits the JSON field when the value is true and writes it only for
	// false. LockNode normalizes that wire-format default during unmarshalling.
	Flake bool `json:"flake"`

	// Parent is the optional parent input path recorded for overridden inputs.
	Parent []string `json:"parent,omitempty"`
}

// UnmarshalJSON decodes a lock node and applies Nix's omitted-true flake default.
func (n *LockNode) UnmarshalJSON(data []byte) error {
	type lockNode LockNode
	var decoded struct {
		lockNode
		Flake *bool `json:"flake"`
	}
	decoded.lockNode.Flake = true

	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	if decoded.Flake != nil {
		decoded.lockNode.Flake = *decoded.Flake
	}

	*n = LockNode(decoded.lockNode)
	return nil
}

// LockReference contains scheme-specific Nix flake reference attributes.
//
// Attribute values remain raw JSON because fetcher schemes may define their own
// string, unsigned integer, boolean, or future-compatible attributes.
type LockReference map[string]json.RawMessage

// LockInput describes either a direct lock node edge or a follows path.
//
// Nix encodes direct edges as JSON strings and follows edges as string arrays.
// Use GetNode and GetFollows to inspect which variant is present.
type LockInput struct {
	node    string
	follows []string
}

// GetNode returns the direct target node when i is a direct edge.
func (i LockInput) GetNode() (string, bool) {
	if i.follows != nil {
		return "", false
	}

	return i.node, true
}

// GetFollows returns a copy of the target path when i is a follows edge.
//
// An explicitly empty follows path returns a non-nil empty slice and true.
func (i LockInput) GetFollows() ([]string, bool) {
	if i.follows == nil {
		return nil, false
	}

	path := make([]string, len(i.follows))
	copy(path, i.follows)
	return path, true
}

// MarshalJSON encodes direct edges as strings and follows edges as arrays.
func (i LockInput) MarshalJSON() ([]byte, error) {
	if i.follows != nil {
		return json.Marshal(i.follows)
	}

	return json.Marshal(i.node)
}

// UnmarshalJSON decodes a direct node string or follows path array.
func (i *LockInput) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return fmt.Errorf("flake: empty lock input")
	}

	switch data[0] {
	case '"':
		var node string
		if err := json.Unmarshal(data, &node); err != nil {
			return fmt.Errorf("flake: failed to decode direct lock input: %w", err)
		}
		i.node = node
		i.follows = nil
		return nil
	case '[':
		var follows []string
		if err := json.Unmarshal(data, &follows); err != nil {
			return fmt.Errorf("flake: failed to decode follows lock input: %w", err)
		}
		if follows == nil {
			follows = []string{}
		}
		i.node = ""
		i.follows = follows
		return nil
	default:
		return fmt.Errorf("flake: lock input must be a node string or follows path array")
	}
}
