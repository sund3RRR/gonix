package store

import (
	"reflect"
	"testing"

	"github.com/sund3RRR/gonix/storepath"
	nix "github.com/sund3RRR/nix-go-bindings"
)

func TestStore_DerivationFromJSON(t *testing.T) {
	type fields struct {
		ctx *nix.NixCContext
		ptr *nix.Store
	}
	type args struct {
		data []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *Derivation
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Store{
				ctx: tt.fields.ctx,
				ptr: tt.fields.ptr,
			}
			got, err := s.DerivationFromJSON(tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("Store.DerivationFromJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Store.DerivationFromJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStore_DerivationFromPath(t *testing.T) {
	type fields struct {
		ctx *nix.NixCContext
		ptr *nix.Store
	}
	type args struct {
		path *storepath.Path
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *Derivation
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Store{
				ctx: tt.fields.ctx,
				ptr: tt.fields.ptr,
			}
			got, err := s.DerivationFromPath(tt.args.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("Store.DerivationFromPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Store.DerivationFromPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStore_AddDerivation(t *testing.T) {
	type fields struct {
		ctx *nix.NixCContext
		ptr *nix.Store
	}
	type args struct {
		d *Derivation
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *storepath.Path
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Store{
				ctx: tt.fields.ctx,
				ptr: tt.fields.ptr,
			}
			got, err := s.AddDerivation(tt.args.d)
			if (err != nil) != tt.wantErr {
				t.Errorf("Store.AddDerivation() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Store.AddDerivation() = %v, want %v", got, tt.want)
			}
		})
	}
}
