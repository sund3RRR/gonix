package utils

import (
	"strings"
	"testing"

	raw "github.com/sund3RRR/nix-go-bindings"
)

func newTestContext(t *testing.T) *raw.NixCContext {
	t.Helper()

	ctx := raw.CContextCreate()
	if ctx == nil {
		t.Fatal("CContextCreate returned nil")
	}
	t.Cleanup(func() {
		raw.CContextFree(ctx)
	})

	return ctx
}

func TestTakeCString(t *testing.T) {
	tests := []struct {
		name         string
		ptr          func(t *testing.T) *byte
		wantContains string
		wantNonEmpty bool
	}{
		{
			name: "nil pointer",
			ptr: func(t *testing.T) *byte {
				t.Helper()

				return nil
			},
			wantContains: "",
		},
		{
			name: "version string",
			ptr: func(t *testing.T) *byte {
				t.Helper()

				ptr := raw.VersionGet()
				if ptr == nil {
					t.Fatal("VersionGet returned nil")
				}

				return ptr
			},
			wantNonEmpty: true,
		},
		{
			name: "context error message",
			ptr: func(t *testing.T) *byte {
				t.Helper()

				ctx := newTestContext(t)
				if got := raw.SetErrMsg(ctx, raw.NixErrUnknown, "gonix utils test error"); got != raw.NixErrUnknown {
					t.Fatalf("SetErrMsg = %v, want %v", got, raw.NixErrUnknown)
				}

				ptr := raw.ErrMsg(nil, ctx)
				if ptr == nil {
					t.Fatal("ErrMsg returned nil")
				}

				return ptr
			},
			wantContains: "gonix utils test error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TakeCString(tt.ptr(t))
			if tt.wantNonEmpty {
				if strings.TrimSpace(got) == "" {
					t.Fatal("TakeCString() returned an empty string")
				}
				return
			}
			if !strings.Contains(got, tt.wantContains) {
				t.Errorf("TakeCString() = %q, want it to contain %q", got, tt.wantContains)
			}
		})
	}
}
