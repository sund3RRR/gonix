// Package nixcontext owns and initializes Nix C API contexts.
//
// A Context is the lifetime root for gonix resources. Stores, evaluators,
// values, flake settings, and flakes borrow a Context and must be closed before
// it. Context does not track or close child resources.
package nixcontext

import (
	"fmt"

	"github.com/sund3RRR/gonix/internal/status"
	"github.com/sund3RRR/gonix/pkg/utils"
	nix "github.com/sund3RRR/nix-go-bindings"
)

// Config configures Context creation.
type Config struct {
	// LoadConfig makes Nix load the user's configuration while initializing the
	// store library. The zero value uses the no-load-config initialization path.
	LoadConfig bool
}

// Verbosity is a Go-native Nix verbosity level.
type Verbosity int

const (
	// VerbosityDefault leaves the current Nix verbosity unchanged.
	VerbosityDefault Verbosity = iota
	// VerbosityError shows only errors.
	VerbosityError
	// VerbosityWarn shows warnings and errors.
	VerbosityWarn
	// VerbosityNotice shows notices, warnings, and errors.
	VerbosityNotice
	// VerbosityInfo shows informational messages.
	VerbosityInfo
	// VerbosityTalkative shows talkative Nix logs.
	VerbosityTalkative
	// VerbosityChatty shows chatty Nix logs.
	VerbosityChatty
	// VerbosityDebug shows debug Nix logs.
	VerbosityDebug
	// VerbosityVomit shows the most verbose Nix logs.
	VerbosityVomit
)

// LogFormat is a Go-native Nix log format.
type LogFormat string

const (
	// LogFormatRaw writes raw log messages.
	LogFormatRaw LogFormat = "raw"
	// LogFormatRawWithLogs writes raw log messages including log records.
	LogFormatRawWithLogs LogFormat = "raw-with-logs"
	// LogFormatInternalJSON writes Nix's internal JSON log format.
	LogFormatInternalJSON LogFormat = "internal-json"
	// LogFormatBar writes progress-bar formatted logs.
	LogFormatBar LogFormat = "bar"
	// LogFormatBarWithLogs writes progress-bar formatted logs including log records.
	LogFormatBarWithLogs LogFormat = "bar-with-logs"
)

// Context owns an initialized Nix C API context.
//
// Context is not goroutine-safe. Resources created with a Context borrow it and
// must be closed before the Context. Close is idempotent.
type Context struct {
	ptr *nix.NixCContext
}

// New creates a Nix context and initializes libutil, libstore, and libexpr.
func New(cfg Config) (*Context, error) {
	ptr := nix.CContextCreate()
	if ptr == nil {
		return nil, fmt.Errorf("nixcontext: create context")
	}

	ctx := &Context{ptr: ptr}
	var err error
	defer func() {
		if err != nil {
			_ = ctx.Close()
		}
	}()

	if code := nix.LibutilInit(ptr); status.ErrorCode(code) != status.ErrorCodeOK {
		err = status.FromContext(ptr)
		return nil, fmt.Errorf("nixcontext: initialize util library: %w", err)
	}

	libstoreInit := nix.LibstoreInitNoLoadConfig
	if cfg.LoadConfig {
		libstoreInit = nix.LibstoreInit
	}
	if code := libstoreInit(ptr); status.ErrorCode(code) != status.ErrorCodeOK {
		err = status.FromContext(ptr)
		return nil, fmt.Errorf("nixcontext: initialize store library: %w", err)
	}

	if code := nix.LibexprInit(ptr); status.ErrorCode(code) != status.ErrorCodeOK {
		err = status.FromContext(ptr)
		return nil, fmt.Errorf("nixcontext: initialize expression library: %w", err)
	}

	return ctx, nil
}

// Borrow returns the borrowed raw Nix context.
//
// Callers must not free the returned pointer and must not retain it beyond the
// immediate raw Nix call that needs it. Borrow returns status.ErrClosed after
// Close.
func (c *Context) Borrow() (*nix.NixCContext, error) {
	if c.ptr == nil {
		return nil, status.ErrClosed
	}

	return c.ptr, nil
}

// Setting returns the current value of a Nix setting.
func (c *Context) Setting(key string) (string, error) {
	ptr, err := c.Borrow()
	if err != nil {
		return "", err
	}

	value := nix.SettingGet(ptr, key)
	if value == nil {
		return "", fmt.Errorf("nixcontext: failed to get setting %q: %w", key, status.FromContext(ptr))
	}

	return utils.TakeCString(value), nil
}

// SetSetting sets a Nix setting.
func (c *Context) SetSetting(key, value string) error {
	ptr, err := c.Borrow()
	if err != nil {
		return err
	}

	if code := nix.SettingSet(ptr, key, value); status.ErrorCode(code) != status.ErrorCodeOK {
		return fmt.Errorf("nixcontext: failed to set setting %q: %w", key, status.FromContext(ptr))
	}

	return nil
}

// SetVerbosity sets the Nix verbosity level.
//
// VerbosityDefault leaves the current verbosity unchanged.
func (c *Context) SetVerbosity(level Verbosity) error {
	ptr, err := c.Borrow()
	if err != nil {
		return err
	}
	if level == VerbosityDefault {
		return nil
	}
	if level < VerbosityError || level > VerbosityVomit {
		return fmt.Errorf("nixcontext: invalid verbosity %d", level)
	}

	rawLevel := nix.NixVerbosity(level - VerbosityError)
	if code := nix.SetVerbosity(ptr, rawLevel); status.ErrorCode(code) != status.ErrorCodeOK {
		return fmt.Errorf("nixcontext: failed to set verbosity: %w", status.FromContext(ptr))
	}

	return nil
}

// SetLogFormat sets the Nix log format.
//
// An empty format leaves the current log format unchanged.
func (c *Context) SetLogFormat(format LogFormat) error {
	ptr, err := c.Borrow()
	if err != nil {
		return err
	}
	if format == "" {
		return nil
	}

	if code := nix.SetLogFormat(ptr, string(format)); status.ErrorCode(code) != status.ErrorCodeOK {
		return fmt.Errorf("nixcontext: failed to set log format: %w", status.FromContext(ptr))
	}

	return nil
}

// Close releases the owned Nix context.
//
// Child resources must be closed before Close. Close is idempotent.
func (c *Context) Close() error {
	if c == nil || c.ptr == nil {
		return nil
	}

	nix.CContextFree(c.ptr)
	c.ptr = nil

	return nil
}
