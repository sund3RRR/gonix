// Package store wraps Nix store handles and store-backed operations.
package store

import (
	"reflect"
	"testing"

	nix "github.com/sund3RRR/nix-go-bindings"
)

func TestNew(t *testing.T) {
	type args struct {
		ctx  *nix.NixCContext
		uri  string
		opts []Option
	}
	tests := []struct {
		name    string
		args    args
		want    *Store
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.args.ctx, tt.args.uri, tt.args.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("New() = %v, want %v", got, tt.want)
			}
		})
	}
}
