package store

import (
	"encoding/json"
	"fmt"

	"github.com/sund3RRR/gonix/internal/status"
	"github.com/sund3RRR/gonix/nixcontext"
	"github.com/sund3RRR/gonix/pkg/raw"
	"github.com/sund3RRR/gonix/pkg/utils"
)

// DerivationData is the Go representation of Nix derivation JSON.
type DerivationData struct {
	Name            string                      `json:"name"`
	Version         uint64                      `json:"version"`
	Outputs         map[string]DerivationOutput `json:"outputs"`
	Inputs          DerivationInputs            `json:"inputs"`
	System          string                      `json:"system"`
	Builder         string                      `json:"builder"`
	Args            []string                    `json:"args"`
	Environment     map[string]string           `json:"env"`
	StructuredAttrs json.RawMessage             `json:"structuredAttrs,omitempty"`
}

// DerivationOutput describes one Nix derivation output.
//
// Nix identifies the output variant by the combination of populated fields.
// An empty DerivationOutput represents a deferred output.
type DerivationOutput struct {
	Path     string `json:"path,omitempty"`
	Method   string `json:"method,omitempty"`
	Hash     string `json:"hash,omitempty"`
	HashAlgo string `json:"hashAlgo,omitempty"`
	Impure   bool   `json:"impure,omitempty"`
}

// DerivationInputs describes source and derivation inputs.
type DerivationInputs struct {
	Sources     []string                   `json:"srcs"`
	Derivations map[string]DerivationInput `json:"drvs"`
}

// DerivationInput describes selected outputs of an input derivation.
//
// DynamicOutputs recursively describes outputs of dynamic derivations.
type DerivationInput struct {
	Outputs        []string                   `json:"outputs"`
	DynamicOutputs map[string]DerivationInput `json:"dynamicOutputs"`
}

// Derivation owns a Nix derivation handle.
//
// A derivation describes how Nix can build one or more outputs: builder,
// arguments, environment, input sources, input derivations, and output paths.
// This wrapper owns the underlying Nix derivation object, not the store path of
// a .drv file. Use Store.AddDerivation to write a derivation to a store and get
// its store path.
//
// A Derivation must be closed when the caller is done with it. Methods needing
// the raw handle require the original Nix context to remain valid. Cached JSON
// serialization and deserialization remain available after Close or Context
// closure.
type Derivation struct {
	ctx  *nixcontext.Context
	ptr  *raw.NixDerivation
	json []byte
}

// NewDerivation wraps an owned raw Nix derivation handle.
//
// NewDerivation takes ownership of ptr, caches its Nix-normalized JSON, and
// releases ptr if initialization fails. A nil ptr is rejected. The ctx argument
// is borrowed and must remain valid for operations requiring the raw handle.
func NewDerivation(ctx *nixcontext.Context, ptr *raw.NixDerivation) (*Derivation, error) {
	if ptr == nil {
		return nil, fmt.Errorf("derivation: nil raw derivation")
	}

	rawCtx, err := ctx.Borrow()
	if err != nil {
		raw.DerivationFree(ptr)
		return nil, fmt.Errorf("derivation: failed to borrow context: %w", err)
	}

	rawJSON := raw.DerivationToJson(rawCtx, ptr)
	if rawJSON == nil {
		raw.DerivationFree(ptr)
		if err := status.FromContext(rawCtx); err != nil {
			return nil, fmt.Errorf("derivation: failed to serialize to json: %w", err)
		}
		return nil, fmt.Errorf("derivation: failed to serialize to json")
	}

	return &Derivation{
		json: []byte(utils.TakeCString(rawJSON)),
		ctx:  ctx,
		ptr:  ptr,
	}, nil
}

// SerializeJSON returns a Go-owned copy of d's cached Nix derivation JSON.
//
// The cached serialization remains available after Derivation or Context
// closure.
func (d *Derivation) SerializeJSON() []byte {
	return append([]byte(nil), d.json...)
}

// Deserialize decodes d's cached Nix derivation JSON into Go-native data.
//
// Deserialize does not validate Nix derivation semantics. The cached
// serialization remains available after Derivation or Context closure.
func (d *Derivation) Deserialize() (DerivationData, error) {
	var data DerivationData
	if err := json.Unmarshal(d.json, &data); err != nil {
		return DerivationData{}, fmt.Errorf("derivation: failed to deserialize cached json: %w", err)
	}
	return data, nil
}

// Clone returns an independently owned copy of d.
//
// The caller must close the returned derivation independently from d.
func (d *Derivation) Clone() (*Derivation, error) {
	if d.ptr == nil {
		return nil, status.ErrClosed
	}

	rawCtx, err := d.ctx.Borrow()
	if err != nil {
		return nil, fmt.Errorf("derivation: failed to borrow context: %w", err)
	}

	clone := raw.DerivationClone(d.ptr)
	if clone == nil {
		return nil, fmt.Errorf("derivation: failed to clone derivation: %w", status.FromContext(rawCtx))
	}

	derivation, err := NewDerivation(d.ctx, clone)
	if err != nil {
		return nil, fmt.Errorf("derivation: failed to create cloned derivation: %w", err)
	}

	return derivation, nil
}

// Borrow returns d's borrowed raw Nix derivation handle.
//
// Callers must not free the returned pointer and must not retain it beyond the
// immediate raw Nix call that needs it. This is an escape hatch for integration
// with lower-level bindings.
func (d *Derivation) Borrow() (*raw.NixDerivation, error) {
	if d.ptr == nil {
		return nil, status.ErrClosed
	}

	return d.ptr, nil
}

// Close releases the owned Nix derivation handle.
//
// Close is safe to call more than once. Once Close returns, methods that need
// the raw derivation handle report status.ErrClosed.
func (d *Derivation) Close() error {
	if d == nil || d.ptr == nil {
		return nil
	}

	raw.DerivationFree(d.ptr)
	d.ptr = nil

	return nil
}
