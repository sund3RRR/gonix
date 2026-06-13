package status_test

import (
	"strings"
	"testing"

	"github.com/sund3RRR/gonix/internal/status"
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

func TestFromContext(t *testing.T) {
	tests := []struct {
		name             string
		setup            func(t *testing.T) *raw.NixCContext
		want             *status.NixError
		wantMsgSubstring string
		wantContextCode  raw.NixErr
		checkContextCode bool
	}{
		{
			name: "nil context",
			setup: func(t *testing.T) *raw.NixCContext {
				t.Helper()

				return nil
			},
			want: nil,
		},
		{
			name: "ok context returns nil",
			setup: func(t *testing.T) *raw.NixCContext {
				t.Helper()

				return newTestContext(t)
			},
			want:             nil,
			wantContextCode:  raw.NixOk,
			checkContextCode: true,
		},
		{
			name: "copies error and clears context",
			setup: func(t *testing.T) *raw.NixCContext {
				t.Helper()

				ctx := newTestContext(t)
				if got := raw.SetErrMsg(ctx, raw.NixErrUnknown, "gonix test error"); got != raw.NixErrUnknown {
					t.Fatalf("SetErrMsg = %v, want %v", got, raw.NixErrUnknown)
				}

				return ctx
			},
			want: &status.NixError{
				Code: status.ErrorCodeUnknown,
			},
			wantMsgSubstring: "gonix test error",
			wantContextCode:  raw.NixOk,
			checkContextCode: true,
		},
		{
			name: "copies nix error code",
			setup: func(t *testing.T) *raw.NixCContext {
				t.Helper()

				ctx := newTestContext(t)
				if got := raw.SetErrMsg(ctx, raw.NixErrNixError, "gonix nix error"); got != raw.NixErrNixError {
					t.Fatalf("SetErrMsg = %v, want %v", got, raw.NixErrNixError)
				}

				return ctx
			},
			want: &status.NixError{
				Code: status.ErrorCodeNix,
			},
			wantMsgSubstring: "gonix nix error",
			wantContextCode:  raw.NixOk,
			checkContextCode: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setup(t)
			got := status.FromContext(ctx)
			if tt.checkContextCode {
				if got := raw.ErrCode(ctx); got != tt.wantContextCode {
					t.Errorf("ErrCode after NewNixError = %v, want %v", got, tt.wantContextCode)
				}
			}

			if tt.want == nil {
				if got != nil {
					t.Fatalf("NewNixError() = %v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Errorf("NewNixError() = %v, want %v", got, tt.want)
				return
			}
			if got.Code != tt.want.Code {
				t.Errorf("NewNixError().Code = %v, want %v", got.Code, tt.want.Code)
			}
			if !strings.Contains(got.Message, tt.wantMsgSubstring) {
				t.Errorf("NewNixError().Message = %q, want it to contain %q", got.Message, tt.wantMsgSubstring)
			}
			if got.Name != tt.want.Name {
				t.Errorf("NewNixError().Name = %q, want %q", got.Name, tt.want.Name)
			}
			if got.Info != tt.want.Info {
				t.Errorf("NewNixError().Info = %q, want %q", got.Info, tt.want.Info)
			}
		})
	}
}
