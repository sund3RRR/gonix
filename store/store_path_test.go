package store

import (
	"reflect"
	"testing"

	"github.com/sund3RRR/gonix/storepath"
	nix "github.com/sund3RRR/nix-go-bindings"
)

func TestStore_URI(t *testing.T) {
	type fields struct {
		ctx *nix.NixCContext
		ptr *nix.Store
	}
	tests := []struct {
		name    string
		fields  fields
		want    string
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
			got, err := s.URI()
			if (err != nil) != tt.wantErr {
				t.Errorf("Store.URI() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Store.URI() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStore_StoreDir(t *testing.T) {
	type fields struct {
		ctx *nix.NixCContext
		ptr *nix.Store
	}
	tests := []struct {
		name    string
		fields  fields
		want    string
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
			got, err := s.StoreDir()
			if (err != nil) != tt.wantErr {
				t.Errorf("Store.StoreDir() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Store.StoreDir() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStore_Version(t *testing.T) {
	type fields struct {
		ctx *nix.NixCContext
		ptr *nix.Store
	}
	tests := []struct {
		name    string
		fields  fields
		want    string
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
			got, err := s.Version()
			if (err != nil) != tt.wantErr {
				t.Errorf("Store.Version() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Store.Version() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStore_ParsePath(t *testing.T) {
	type fields struct {
		ctx *nix.NixCContext
		ptr *nix.Store
	}
	type args struct {
		path string
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
			got, err := s.ParsePath(tt.args.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("Store.ParsePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Store.ParsePath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStore_PathFromHash(t *testing.T) {
	type fields struct {
		ctx *nix.NixCContext
		ptr *nix.Store
	}
	type args struct {
		hashPart []byte
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
			got, err := s.PathFromHash(tt.args.hashPart)
			if (err != nil) != tt.wantErr {
				t.Errorf("Store.PathFromHash() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Store.PathFromHash() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStore_RealPath(t *testing.T) {
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
		want    string
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
			got, err := s.RealPath(tt.args.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("Store.RealPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Store.RealPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStore_IsValidPath(t *testing.T) {
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
		want    bool
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
			got, err := s.IsValidPath(tt.args.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("Store.IsValidPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Store.IsValidPath() = %v, want %v", got, tt.want)
			}
		})
	}
}
