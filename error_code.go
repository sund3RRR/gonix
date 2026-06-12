package gonix

import raw "github.com/sund3RRR/nix-go-bindings"

// ErrorCode is a Nix C API error code.
type ErrorCode int32

const (
	ErrorCodeOK          ErrorCode = ErrorCode(raw.NixOk)
	ErrorCodeUnknown     ErrorCode = ErrorCode(raw.NixErrUnknown)
	ErrorCodeOverflow    ErrorCode = ErrorCode(raw.NixErrOverflow)
	ErrorCodeKey         ErrorCode = ErrorCode(raw.NixErrKey)
	ErrorCodeNix         ErrorCode = ErrorCode(raw.NixErrNixError)
	ErrorCodeRecoverable ErrorCode = ErrorCode(raw.NixErrRecoverable)
)

func (code ErrorCode) String() string {
	switch code {
	case ErrorCodeOK:
		return "NIX_OK"
	case ErrorCodeOverflow:
		return "NIX_ERR_OVERFLOW"
	case ErrorCodeKey:
		return "NIX_ERR_KEY"
	case ErrorCodeNix:
		return "NIX_ERR_NIX_ERROR"
	case ErrorCodeRecoverable:
		return "NIX_ERR_RECOVERABLE"
	default:
		return "NIX_ERR_UNKNOWN"
	}
}
