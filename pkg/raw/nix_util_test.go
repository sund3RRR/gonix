package raw

import (
	"strings"
	"testing"
	"unsafe"
)

func ownedCString(t *testing.T, s *byte) string {
	t.Helper()

	if s == nil {
		t.Fatal("expected non-nil C string")
	}
	defer StringFree(s)

	ptr := unsafe.Pointer(s)
	n := 0
	for *(*byte)(unsafe.Add(ptr, n)) != 0 {
		n++
	}

	return string(unsafe.Slice(s, n))
}

func newTestContext(t *testing.T) *NixCContext {
	t.Helper()

	ctx := CContextCreate()
	if ctx == nil {
		t.Fatal("CContextCreate returned nil")
	}
	t.Cleanup(func() {
		CContextFree(ctx)
	})

	return ctx
}

func TestNixUtilContextAndInit(t *testing.T) {
	ctx := newTestContext(t)

	if got := ErrCode(ctx); got != NixOk {
		t.Fatalf("new context error code = %v, want %v", got, NixOk)
	}
	if got := LibutilInit(ctx); got != NixOk {
		t.Fatalf("LibutilInit = %v, want %v", got, NixOk)
	}
	if got := ErrCode(ctx); got != NixOk {
		t.Fatalf("post-init error code = %v, want %v", got, NixOk)
	}
}

func TestNixUtilConstants(t *testing.T) {
	if NixOk != 0 {
		t.Fatalf("NixOk = %d, want 0", NixOk)
	}
	if NixErrUnknown != -1 {
		t.Fatalf("NixErrUnknown = %d, want -1", NixErrUnknown)
	}
	if NixErrKey != -3 {
		t.Fatalf("NixErrKey = %d, want -3", NixErrKey)
	}
	if NixErrRecoverable != -5 {
		t.Fatalf("NixErrRecoverable = %d, want -5", NixErrRecoverable)
	}
	if NixLvlError != 0 {
		t.Fatalf("NixLvlError = %d, want 0", NixLvlError)
	}
	if NixLvlWarn != 1 {
		t.Fatalf("NixLvlWarn = %d, want 1", NixLvlWarn)
	}
}

func TestNixUtilVersionGet(t *testing.T) {
	version := ownedCString(t, VersionGet())
	if strings.TrimSpace(version) == "" {
		t.Fatal("VersionGet returned an empty string")
	}
}

func TestNixUtilSettingGet(t *testing.T) {
	ctx := newTestContext(t)
	if got := LibutilInit(ctx); got != NixOk {
		t.Fatalf("LibutilInit = %v, want %v", got, NixOk)
	}

	candidates := []string{
		"keep-failed",
		"print-missing",
		"use-xdg-base-directories",
		"use-sqlite-wal",
		"connect-timeout",
		"http2",
		"experimental-features",
	}
	for _, key := range candidates {
		value := SettingGet(ctx, key)
		if value == nil {
			ClearErr(ctx)
			continue
		}

		settingValue := ownedCString(t, value)
		if strings.TrimSpace(settingValue) == "" && key != "experimental-features" {
			t.Fatalf("SettingGet(%q) returned an unexpectedly empty string", key)
		}
		if got := SettingSet(ctx, key, settingValue); got != NixOk {
			t.Fatalf("SettingSet(%q) = %v, want %v", key, got, NixOk)
		}
		return
	}

	t.Fatal("none of the candidate Nix settings were registered")
}

func TestNixUtilUnknownSettingSetsContextError(t *testing.T) {
	ctx := newTestContext(t)
	if got := LibutilInit(ctx); got != NixOk {
		t.Fatalf("LibutilInit = %v, want %v", got, NixOk)
	}

	value := SettingGet(ctx, "go-bindings-test-setting-that-does-not-exist")
	if value != nil {
		StringFree(value)
		t.Fatal("SettingGet for unknown setting returned a value")
	}
	if got := ErrCode(ctx); got != NixErrKey {
		t.Fatalf("ErrCode after unknown setting = %v, want %v", got, NixErrKey)
	}

	msg := ownedCString(t, ErrMsg(nil, ctx))
	if strings.TrimSpace(msg) == "" {
		t.Fatal("ErrMsg after unknown setting returned an empty string")
	}

	ClearErr(ctx)
	if got := ErrCode(ctx); got != NixOk {
		t.Fatalf("ErrCode after ClearErr = %v, want %v", got, NixOk)
	}
}

func TestNixUtilSetErrMsgAndClear(t *testing.T) {
	ctx := newTestContext(t)

	if got := SetErrMsg(ctx, NixErrUnknown, "go bindings test error"); got != NixErrUnknown {
		t.Fatalf("SetErrMsg = %v, want %v", got, NixErrUnknown)
	}
	if got := ErrCode(ctx); got != NixErrUnknown {
		t.Fatalf("ErrCode after SetErrMsg = %v, want %v", got, NixErrUnknown)
	}

	msg := ownedCString(t, ErrMsg(nil, ctx))
	if !strings.Contains(msg, "go bindings test error") {
		t.Fatalf("ErrMsg = %q, want it to contain test message", msg)
	}

	ClearErr(ctx)
	if got := ErrCode(ctx); got != NixOk {
		t.Fatalf("ErrCode after ClearErr = %v, want %v", got, NixOk)
	}
}

func TestNixUtilSetVerbosity(t *testing.T) {
	ctx := newTestContext(t)

	if got := SetVerbosity(ctx, NixLvlWarn); got != NixOk {
		t.Fatalf("SetVerbosity = %v, want %v", got, NixOk)
	}
}

func TestNixUtilInterruptState(t *testing.T) {
	InterruptClear()
	t.Cleanup(InterruptClear)

	if InterruptRequested() {
		t.Fatal("InterruptRequested after clear = true, want false")
	}

	InterruptRequest()
	if !InterruptRequested() {
		t.Fatal("InterruptRequested after request = false, want true")
	}

	InterruptClear()
	if InterruptRequested() {
		t.Fatal("InterruptRequested after second clear = true, want false")
	}
}
