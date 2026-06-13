package status_test

import (
	"testing"

	"github.com/sund3RRR/gonix/internal/status"
)

func TestErrorCode_String(t *testing.T) {
	tests := []struct {
		name string
		code status.ErrorCode
		want string
	}{
		{
			name: "ok",
			code: status.ErrorCodeOK,
			want: "NIX_OK",
		},
		{
			name: "unknown",
			code: status.ErrorCodeUnknown,
			want: "NIX_ERR_UNKNOWN",
		},
		{
			name: "overflow",
			code: status.ErrorCodeOverflow,
			want: "NIX_ERR_OVERFLOW",
		},
		{
			name: "key",
			code: status.ErrorCodeKey,
			want: "NIX_ERR_KEY",
		},
		{
			name: "nix error",
			code: status.ErrorCodeNix,
			want: "NIX_ERR_NIX_ERROR",
		},
		{
			name: "recoverable",
			code: status.ErrorCodeRecoverable,
			want: "NIX_ERR_RECOVERABLE",
		},
		{
			name: "unrecognized code",
			code: status.ErrorCode(12345),
			want: "NIX_ERR_UNKNOWN",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.code.String()
			if got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}
