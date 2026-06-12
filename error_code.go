package gonix

import raw "github.com/sund3RRR/nix-go-bindings"

// ErrorCode identifies a status code returned by the Nix C API.
//
// The numeric values intentionally mirror nix-go-bindings so callers can log
// or compare gonix errors without importing raw generated types. Unknown raw
// values are preserved as numbers, but String reports them as NIX_ERR_UNKNOWN.
type ErrorCode int32

const (
	// ErrorCodeOK reports that a Nix C API operation completed successfully.
	ErrorCodeOK ErrorCode = ErrorCode(raw.NixOk)

	// ErrorCodeUnknown reports a generic failure not described by a narrower code.
	ErrorCodeUnknown ErrorCode = ErrorCode(raw.NixErrUnknown)

	// ErrorCodeOverflow reports that a value did not fit in the requested representation.
	ErrorCodeOverflow ErrorCode = ErrorCode(raw.NixErrOverflow)

	// ErrorCodeKey reports a missing or invalid key, setting, attribute, or lookup name.
	ErrorCodeKey ErrorCode = ErrorCode(raw.NixErrKey)

	// ErrorCodeNix reports that the context contains a structured upstream Nix error.
	//
	// This is the only code for which NixError.Name and NixError.Info may carry
	// structured Nix exception details. A context may still contain this code
	// without detailed info when the code was set manually by lower-level code.
	ErrorCodeNix ErrorCode = ErrorCode(raw.NixErrNixError)

	// ErrorCodeRecoverable reports a failure that Nix classified as recoverable.
	ErrorCodeRecoverable ErrorCode = ErrorCode(raw.NixErrRecoverable)
)

// String returns the canonical Nix C API symbol for code.
//
// Unrecognized values are rendered as NIX_ERR_UNKNOWN. The original numeric
// value is still retained in the ErrorCode itself and in formatted NixError
// output.
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
