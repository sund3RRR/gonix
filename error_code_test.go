package gonix_test

import (
	"testing"

	"github.com/sund3RRR/gonix"
)

func TestErrorCode_String(t *testing.T) {
	tests := []struct {
		name string
		code gonix.ErrorCode
		want string
	}{
		{
			name: "ok",
			code: gonix.ErrorCodeOK,
			want: "NIX_OK",
		},
		{
			name: "unknown",
			code: gonix.ErrorCodeUnknown,
			want: "NIX_ERR_UNKNOWN",
		},
		{
			name: "overflow",
			code: gonix.ErrorCodeOverflow,
			want: "NIX_ERR_OVERFLOW",
		},
		{
			name: "key",
			code: gonix.ErrorCodeKey,
			want: "NIX_ERR_KEY",
		},
		{
			name: "nix error",
			code: gonix.ErrorCodeNix,
			want: "NIX_ERR_NIX_ERROR",
		},
		{
			name: "recoverable",
			code: gonix.ErrorCodeRecoverable,
			want: "NIX_ERR_RECOVERABLE",
		},
		{
			name: "unrecognized code",
			code: gonix.ErrorCode(12345),
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
