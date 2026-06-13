package gonix

import "github.com/sund3RRR/gonix/internal/status"

// Error describes a Nix failure captured by gonix.
//
// It is a stable, Go-native snapshot of a Nix context error: the status code,
// human-readable message, and any structured upstream Nix details that were
// available when the error was converted.
type Error = status.NixError

// ErrorCode identifies a Nix C API status code returned through gonix errors.
type ErrorCode = status.ErrorCode

const (
	// ErrorCodeOK reports that a Nix operation completed successfully.
	ErrorCodeOK ErrorCode = status.ErrorCodeOK

	// ErrorCodeUnknown reports a generic failure not described by a narrower code.
	ErrorCodeUnknown ErrorCode = status.ErrorCodeUnknown

	// ErrorCodeOverflow reports that a value did not fit in the requested representation.
	ErrorCodeOverflow ErrorCode = status.ErrorCodeOverflow

	// ErrorCodeKey reports a missing or invalid key, setting, attribute, or lookup name.
	ErrorCodeKey ErrorCode = status.ErrorCodeKey

	// ErrorCodeNix reports a structured upstream Nix exception.
	ErrorCodeNix ErrorCode = status.ErrorCodeNix

	// ErrorCodeRecoverable reports a failure that Nix classified as recoverable.
	ErrorCodeRecoverable ErrorCode = status.ErrorCodeRecoverable
)
