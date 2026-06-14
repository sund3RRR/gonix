package store

import (
	"testing"

	"github.com/sund3RRR/gonix/storepath"
	nix "github.com/sund3RRR/nix-go-bindings"
)

func TestStore_CopyClosure(t *testing.T) {
	type fields struct {
		ctx *nix.NixCContext
		ptr *nix.Store
	}
	type args struct {
		dst  *Store
		path *storepath.Path
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
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
			if err := s.CopyClosure(tt.args.dst, tt.args.path); (err != nil) != tt.wantErr {
				t.Errorf("Store.CopyClosure() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStore_CopyPathTo(t *testing.T) {
	type fields struct {
		ctx *nix.NixCContext
		ptr *nix.Store
	}
	type args struct {
		dst  *Store
		path *storepath.Path
		opts []CopyOption
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
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
			if err := s.CopyPathTo(tt.args.dst, tt.args.path, tt.args.opts...); (err != nil) != tt.wantErr {
				t.Errorf("Store.CopyPathTo() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
