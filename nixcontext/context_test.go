package nixcontext_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/sund3RRR/gonix"
	"github.com/sund3RRR/gonix/nixcontext"
)

func TestContextBootstrapAndSettings(t *testing.T) {
	for _, cfg := range []nixcontext.Config{
		{},
		{LoadConfig: true},
	} {
		ctx, err := nixcontext.New(cfg)
		if err != nil {
			t.Fatalf("nixcontext.New(%+v) error = %v", cfg, err)
		}

		if err := ctx.SetSetting("experimental-features", "nix-command flakes"); err != nil {
			t.Fatalf("Context.SetSetting() error = %v", err)
		}
		got, err := ctx.Setting("experimental-features")
		if err != nil {
			t.Fatalf("Context.Setting() error = %v", err)
		}
		if !strings.Contains(got, "nix-command") || !strings.Contains(got, "flakes") {
			t.Fatalf("experimental-features = %q, want nix-command and flakes", got)
		}

		if err := ctx.SetVerbosity(nixcontext.VerbosityWarn); err != nil {
			t.Fatalf("Context.SetVerbosity() error = %v", err)
		}
		if err := ctx.SetLogFormat(nixcontext.LogFormatRaw); err != nil {
			t.Fatalf("Context.SetLogFormat() error = %v", err)
		}

		if err := ctx.Close(); err != nil {
			t.Fatalf("Context.Close() error = %v", err)
		}
	}
}

func TestContextErrorsAndClose(t *testing.T) {
	ctx, err := nixcontext.New(nixcontext.Config{})
	if err != nil {
		t.Fatalf("nixcontext.New() error = %v", err)
	}

	if err := ctx.SetSetting("gonix-test-setting-that-does-not-exist", "x"); err == nil {
		t.Fatal("Context.SetSetting(invalid) error = nil")
	}
	if err := ctx.SetVerbosity(nixcontext.Verbosity(100)); err == nil {
		t.Fatal("Context.SetVerbosity(invalid) error = nil")
	}
	if err := ctx.SetLogFormat(nixcontext.LogFormat("invalid-gonix-format")); err == nil {
		t.Fatal("Context.SetLogFormat(invalid) error = nil")
	}

	if err := ctx.Close(); err != nil {
		t.Fatalf("Context.Close() error = %v", err)
	}
	if err := ctx.Close(); err != nil {
		t.Fatalf("second Context.Close() error = %v", err)
	}

	_, err = ctx.Borrow()
	requireClosed(t, err)
	_, err = ctx.Setting("cores")
	requireClosed(t, err)
	requireClosed(t, ctx.SetSetting("cores", "2"))
	requireClosed(t, ctx.SetVerbosity(nixcontext.VerbosityWarn))
	requireClosed(t, ctx.SetLogFormat(nixcontext.LogFormatRaw))
}

func requireClosed(t *testing.T, err error) {
	t.Helper()

	if !errors.Is(err, gonix.ErrClosed) {
		t.Fatalf("error = %v, want gonix.ErrClosed", err)
	}
}
