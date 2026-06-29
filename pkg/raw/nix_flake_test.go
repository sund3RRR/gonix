package raw

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func chmodTreeWritable(t *testing.T, root string) {
	t.Helper()

	if err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return os.Chmod(path, 0o700)
		}
		return os.Chmod(path, 0o600)
	}); err != nil && !os.IsNotExist(err) {
		t.Fatalf("chmod temp store tree: %v", err)
	}
}

func enableTestFlakes(t *testing.T, ctx *NixCContext) {
	t.Helper()

	if got := SettingSet(ctx, "experimental-features", "nix-command flakes"); got != NixOk {
		t.Fatalf("SettingSet(experimental-features) = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
	}
}

func newTestLocalStore(t *testing.T, ctx *NixCContext) (*Store, string) {
	t.Helper()

	if got := LibstoreInit(ctx); got != NixOk {
		t.Fatalf("LibstoreInit = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
	}

	root := t.TempDir()
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q): %v", root, err)
	}
	root = resolvedRoot
	t.Cleanup(func() {
		chmodTreeWritable(t, root)
	})

	storeDir := filepath.Join(root, "store")
	stateDir := filepath.Join(root, "state")
	logDir := filepath.Join(root, "log")
	params := StoreParams{
		Items: []StoreParam{
			{Key: []byte("store"), KeyLen: 5, Value: []byte(storeDir), ValueLen: uint64(len(storeDir))},
			{Key: []byte("state"), KeyLen: 5, Value: []byte(stateDir), ValueLen: uint64(len(stateDir))},
			{Key: []byte("log"), KeyLen: 3, Value: []byte(logDir), ValueLen: uint64(len(logDir))},
		},
		Len: 3,
	}

	store := StoreOpen(ctx, "local", params)
	if store == nil {
		t.Fatalf("StoreOpen(local) returned nil: err=%v msg=%q", ErrCode(ctx), errMsgString(t, ctx))
	}
	t.Cleanup(func() {
		StoreFree(store)
	})

	return store, root
}

func newTestFlakeSettings(t *testing.T) (*NixCContext, *NixFetchersSettings, *NixFlakeSettings) {
	t.Helper()

	ctx := newTestContext(t)
	if got := LibutilInit(ctx); got != NixOk {
		t.Fatalf("LibutilInit = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
	}
	enableTestFlakes(t, ctx)
	if got := LibstoreInit(ctx); got != NixOk {
		t.Fatalf("LibstoreInit = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
	}
	if got := LibexprInit(ctx); got != NixOk {
		t.Fatalf("LibexprInit = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
	}

	fetchSettings := FetchersSettingsNew(ctx)
	if fetchSettings == nil {
		t.Fatalf("FetchersSettingsNew returned nil: err=%v msg=%q", ErrCode(ctx), errMsgString(t, ctx))
	}
	t.Cleanup(func() {
		FetchersSettingsFree(fetchSettings)
	})

	flakeSettings := FlakeSettingsNew(ctx)
	if flakeSettings == nil {
		t.Fatalf("FlakeSettingsNew returned nil: err=%v msg=%q", ErrCode(ctx), errMsgString(t, ctx))
	}
	t.Cleanup(func() {
		FlakeSettingsFree(flakeSettings)
	})

	return ctx, fetchSettings, flakeSettings
}

func newTestFlakeState(t *testing.T) (*NixCContext, *Store, *NixFetchersSettings, *NixFlakeSettings, *EvalState) {
	t.Helper()

	ctx := newTestContext(t)
	if got := LibutilInit(ctx); got != NixOk {
		t.Fatalf("LibutilInit = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
	}
	enableTestFlakes(t, ctx)
	store, _ := newTestLocalStore(t, ctx)
	if got := LibexprInit(ctx); got != NixOk {
		t.Fatalf("LibexprInit = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
	}

	fetchSettings := FetchersSettingsNew(ctx)
	if fetchSettings == nil {
		t.Fatalf("FetchersSettingsNew returned nil: err=%v msg=%q", ErrCode(ctx), errMsgString(t, ctx))
	}
	t.Cleanup(func() {
		FetchersSettingsFree(fetchSettings)
	})

	flakeSettings := FlakeSettingsNew(ctx)
	if flakeSettings == nil {
		t.Fatalf("FlakeSettingsNew returned nil: err=%v msg=%q", ErrCode(ctx), errMsgString(t, ctx))
	}
	t.Cleanup(func() {
		FlakeSettingsFree(flakeSettings)
	})

	builder := EvalStateBuilderNew(ctx, store)
	if builder == nil {
		t.Fatalf("EvalStateBuilderNew returned nil: err=%v msg=%q", ErrCode(ctx), errMsgString(t, ctx))
	}
	state := EvalStateBuild(ctx, builder)
	EvalStateBuilderFree(builder)
	if state == nil {
		t.Fatalf("EvalStateBuild returned nil: err=%v msg=%q", ErrCode(ctx), errMsgString(t, ctx))
	}
	t.Cleanup(func() {
		StateFree(state)
	})

	return ctx, store, fetchSettings, flakeSettings, state
}

func newFlakeParseFlags(t *testing.T, ctx *NixCContext, settings *NixFlakeSettings, baseDir string) *NixFlakeReferenceParseFlags {
	t.Helper()

	flags := FlakeReferenceParseFlagsNew(ctx, settings)
	if flags == nil {
		t.Fatalf("FlakeReferenceParseFlagsNew returned nil: err=%v msg=%q", ErrCode(ctx), errMsgString(t, ctx))
	}
	t.Cleanup(func() {
		FlakeReferenceParseFlagsFree(flags)
	})

	if baseDir != "" {
		if got := FlakeReferenceParseFlagsSetBaseDirectory(ctx, flags, baseDir, uint64(len(baseDir))); got != NixOk {
			t.Fatalf("FlakeReferenceParseFlagsSetBaseDirectory = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
		}
	}

	return flags
}

func parseTestFlakeReference(
	t *testing.T,
	ctx *NixCContext,
	fetchSettings *NixFetchersSettings,
	flakeSettings *NixFlakeSettings,
	parseFlags *NixFlakeReferenceParseFlags,
	ref string,
) (*NixFlakeReference, string) {
	t.Helper()

	result := FlakeReferenceAndFragmentFromString(
		ctx,
		fetchSettings,
		flakeSettings,
		parseFlags,
		ref,
		uint64(len(ref)),
	)
	if result == nil {
		t.Fatalf("FlakeReferenceAndFragmentFromString(%q) returned nil: err=%v msg=%q", ref, ErrCode(ctx), errMsgString(t, ctx))
	}
	defer FlakeReferenceResultFree(result)

	reference := FlakeReferenceResultTakeReference(result)
	if reference == nil {
		t.Fatalf("FlakeReferenceResultTakeReference returned nil for %q", ref)
	}
	t.Cleanup(func() {
		FlakeReferenceFree(reference)
	})

	fragment := ownedCString(t, FlakeReferenceResultTakeFragment(result))
	return reference, fragment
}

func writeTestFlake(t *testing.T, dir string, contents string) {
	t.Helper()

	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", dir, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "flake.nix"), []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile(%q/flake.nix): %v", dir, err)
	}
}

func lockedFlakeHello(t *testing.T, ctx *NixCContext, settings *NixFlakeSettings, state *EvalState, lockedFlake *NixLockedFlake) string {
	t.Helper()

	value := LockedFlakeGetOutputAttrs(ctx, settings, state, lockedFlake)
	if value == nil {
		t.Fatalf("LockedFlakeGetOutputAttrs returned nil: err=%v msg=%q", ErrCode(ctx), errMsgString(t, ctx))
	}
	defer func() {
		if got := ValueDecref(ctx, value); got != NixOk {
			t.Fatalf("ValueDecref(output attrs) = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
		}
	}()

	hello := GetAttrByname(ctx, value, state, "hello")
	if hello == nil {
		t.Fatalf("GetAttrByname(hello) returned nil: err=%v msg=%q", ErrCode(ctx), errMsgString(t, ctx))
	}
	defer func() {
		if got := ValueDecref(ctx, hello); got != NixOk {
			t.Fatalf("ValueDecref(hello attr) = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
		}
	}()

	return ownedCString(t, GetString(ctx, hello))
}

func lockedFlakeFingerprint(
	t *testing.T,
	ctx *NixCContext,
	store *Store,
	fetchSettings *NixFetchersSettings,
	lockedFlake *NixLockedFlake,
) string {
	t.Helper()

	raw := LockedFlakeGetFingerprint(ctx, store, fetchSettings, lockedFlake)
	if raw == nil {
		if got := ErrCode(ctx); got != NixOk {
			t.Fatalf("LockedFlakeGetFingerprint returned nil: err=%v msg=%q", got, errMsgString(t, ctx))
		}
		t.Fatal("LockedFlakeGetFingerprint returned no fingerprint")
	}

	fingerprint := ownedCString(t, raw)
	if got := ErrCode(ctx); got != NixOk {
		t.Fatalf("ErrCode after LockedFlakeGetFingerprint = %v, want %v", got, NixOk)
	}
	if len(fingerprint) != 64 {
		t.Fatalf("LockedFlakeGetFingerprint length = %d, want 64: %q", len(fingerprint), fingerprint)
	}
	for _, c := range []byte(fingerprint) {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			t.Fatalf("LockedFlakeGetFingerprint = %q, want lowercase base16", fingerprint)
		}
	}

	return fingerprint
}

func TestNixFlakeSettingsAddToEvalStateBuilderAddsGetFlake(t *testing.T) {
	ctx := newTestContext(t)
	if got := LibutilInit(ctx); got != NixOk {
		t.Fatalf("LibutilInit = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
	}
	enableTestFlakes(t, ctx)
	store, _ := newTestLocalStore(t, ctx)
	if got := LibexprInit(ctx); got != NixOk {
		t.Fatalf("LibexprInit = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
	}

	settings := FlakeSettingsNew(ctx)
	if settings == nil {
		t.Fatalf("FlakeSettingsNew returned nil: err=%v msg=%q", ErrCode(ctx), errMsgString(t, ctx))
	}
	t.Cleanup(func() {
		FlakeSettingsFree(settings)
	})

	builder := EvalStateBuilderNew(ctx, store)
	if builder == nil {
		t.Fatalf("EvalStateBuilderNew returned nil: err=%v msg=%q", ErrCode(ctx), errMsgString(t, ctx))
	}

	if got := FlakeSettingsAddToEvalStateBuilder(ctx, settings, builder); got != NixOk {
		EvalStateBuilderFree(builder)
		t.Fatalf("FlakeSettingsAddToEvalStateBuilder = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
	}

	state := EvalStateBuild(ctx, builder)
	EvalStateBuilderFree(builder)
	if state == nil {
		t.Fatalf("EvalStateBuild returned nil: err=%v msg=%q", ErrCode(ctx), errMsgString(t, ctx))
	}
	t.Cleanup(func() {
		StateFree(state)
	})

	value := allocTestValue(t, ctx, state)
	if got := ExprEvalFromString(ctx, state, "builtins.getFlake", ".", value); got != NixOk {
		t.Fatalf("ExprEvalFromString(builtins.getFlake) = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
	}
	if got := GetType(ctx, value); got != NixTypeFunction {
		t.Fatalf("GetType(builtins.getFlake) = %v, want %v", got, NixTypeFunction)
	}
}

func TestNixFlakeReferenceParsing(t *testing.T) {
	ctx, fetchSettings, flakeSettings := newTestFlakeSettings(t)

	parseFlags := newFlakeParseFlags(t, ctx, flakeSettings, "")
	result := FlakeReferenceAndFragmentFromString(
		ctx,
		fetchSettings,
		flakeSettings,
		parseFlags,
		".#legacyPackages.aarch127-unknown...orion",
		uint64(len(".#legacyPackages.aarch127-unknown...orion")),
	)
	if result != nil {
		FlakeReferenceResultFree(result)
		t.Fatal("FlakeReferenceAndFragmentFromString unexpectedly accepted relative ref without base dir")
	}
	if got := ErrCode(ctx); got == NixOk {
		t.Fatalf("ErrCode after relative ref without base dir = %v, want non-OK", got)
	}
	ClearErr(ctx)

	baseDir := t.TempDir()
	writeTestFlake(t, baseDir, `
{
  outputs = { ... }: {
    hello = "potato";
  };
}
`)
	parseFlagsWithBase := newFlakeParseFlags(t, ctx, flakeSettings, baseDir)
	reference, fragment := parseTestFlakeReference(
		t,
		ctx,
		fetchSettings,
		flakeSettings,
		parseFlagsWithBase,
		".#legacyPackages.aarch127-unknown...orion",
	)
	if reference == nil {
		t.Fatal("parsed flake reference is nil")
	}
	if fragment != "legacyPackages.aarch127-unknown...orion" {
		t.Fatalf("parsed fragment = %q, want legacyPackages.aarch127-unknown...orion", fragment)
	}
}

func TestNixFlakeLockAndOutputAttrs(t *testing.T) {
	ctx, store, fetchSettings, flakeSettings, state := newTestFlakeState(t)

	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("EvalSymlinks(flake root): %v", err)
	}
	writeTestFlake(t, filepath.Join(root, "b"), `
{
  outputs = { ... }: {
    hello = "BOB";
  };
}
`)
	writeTestFlake(t, filepath.Join(root, "a"), `
{
  inputs.b.url = "`+filepath.ToSlash(filepath.Join(root, "b"))+`";
  outputs = { b, ... }: {
    hello = b.hello;
  };
}
`)
	writeTestFlake(t, filepath.Join(root, "c"), `
{
  outputs = { ... }: {
    hello = "Claire";
  };
}
`)

	parseFlags := newFlakeParseFlags(t, ctx, flakeSettings, root)
	reference, fragment := parseTestFlakeReference(t, ctx, fetchSettings, flakeSettings, parseFlags, "./a")
	if fragment != "" {
		t.Fatalf("fragment for ./a = %q, want empty", fragment)
	}

	lockFlags := FlakeLockFlagsNew(ctx, flakeSettings)
	if lockFlags == nil {
		t.Fatalf("FlakeLockFlagsNew returned nil: err=%v msg=%q", ErrCode(ctx), errMsgString(t, ctx))
	}
	t.Cleanup(func() {
		FlakeLockFlagsFree(lockFlags)
	})

	if got := FlakeLockFlagsSetModeCheck(ctx, lockFlags); got != NixOk {
		t.Fatalf("FlakeLockFlagsSetModeCheck = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
	}
	lockedFlake := FlakeLock(ctx, fetchSettings, flakeSettings, state, lockFlags, reference)
	if lockedFlake != nil {
		LockedFlakeFree(lockedFlake)
		t.Fatal("FlakeLock in check mode unexpectedly succeeded before lock file exists")
	}
	if got := ErrCode(ctx); got == NixOk {
		t.Fatalf("ErrCode after check-mode lock = %v, want non-OK", got)
	}
	ClearErr(ctx)

	if got := FlakeLockFlagsSetModeVirtual(ctx, lockFlags); got != NixOk {
		t.Fatalf("FlakeLockFlagsSetModeVirtual = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
	}
	lockedFlake = FlakeLock(ctx, fetchSettings, flakeSettings, state, lockFlags, reference)
	if lockedFlake == nil {
		t.Fatalf("FlakeLock in virtual mode returned nil: err=%v msg=%q", ErrCode(ctx), errMsgString(t, ctx))
	}
	if got := lockedFlakeHello(t, ctx, flakeSettings, state, lockedFlake); got != "BOB" {
		t.Fatalf("virtual lock hello = %q, want BOB", got)
	}
	LockedFlakeFree(lockedFlake)

	if _, err := os.Stat(filepath.Join(root, "a", "flake.lock")); !os.IsNotExist(err) {
		t.Fatalf("virtual lock flake.lock stat err = %v, want not exist", err)
	}

	if got := FlakeLockFlagsSetModeCheck(ctx, lockFlags); got != NixOk {
		t.Fatalf("FlakeLockFlagsSetModeCheck after virtual = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
	}
	lockedFlake = FlakeLock(ctx, fetchSettings, flakeSettings, state, lockFlags, reference)
	if lockedFlake != nil {
		LockedFlakeFree(lockedFlake)
		t.Fatal("FlakeLock in check mode unexpectedly succeeded after virtual-only lock")
	}
	if got := ErrCode(ctx); got == NixOk {
		t.Fatalf("ErrCode after second check-mode lock = %v, want non-OK", got)
	}
	ClearErr(ctx)

	if got := FlakeLockFlagsSetModeWriteAsNeeded(ctx, lockFlags); got != NixOk {
		t.Fatalf("FlakeLockFlagsSetModeWriteAsNeeded = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
	}
	lockedFlake = FlakeLock(ctx, fetchSettings, flakeSettings, state, lockFlags, reference)
	if lockedFlake == nil {
		t.Fatalf("FlakeLock in write-as-needed mode returned nil: err=%v msg=%q", ErrCode(ctx), errMsgString(t, ctx))
	}
	if got := lockedFlakeHello(t, ctx, flakeSettings, state, lockedFlake); got != "BOB" {
		t.Fatalf("write-as-needed lock hello = %q, want BOB", got)
	}

	if got := SetErrMsg(ctx, NixErrUnknown, "stale lock JSON error"); got != NixErrUnknown {
		t.Fatalf("SetErrMsg before LockedFlakeGetLockJson = %v, want %v", got, NixErrUnknown)
	}
	lockJSONRaw := LockedFlakeGetLockJson(ctx, lockedFlake)
	if lockJSONRaw == nil {
		t.Fatalf("LockedFlakeGetLockJson returned nil: err=%v msg=%q", ErrCode(ctx), errMsgString(t, ctx))
	}
	lockJSON := ownedCString(t, lockJSONRaw)
	if got := ErrCode(ctx); got != NixOk {
		t.Fatalf("ErrCode after LockedFlakeGetLockJson = %v, want %v", got, NixOk)
	}

	writtenLockJSON, err := os.ReadFile(filepath.Join(root, "a", "flake.lock"))
	if err != nil {
		t.Fatalf("ReadFile(written flake.lock): %v", err)
	}
	if got, want := mustParseJSON(t, lockJSON), mustParseJSON(t, string(writtenLockJSON)); !reflect.DeepEqual(got, want) {
		t.Fatalf("LockedFlakeGetLockJson = %#v, want written flake.lock %#v", got, want)
	}

	if got := SetErrMsg(ctx, NixErrUnknown, "stale fingerprint error"); got != NixErrUnknown {
		t.Fatalf("SetErrMsg before LockedFlakeGetFingerprint = %v, want %v", got, NixErrUnknown)
	}
	writtenFingerprint := lockedFlakeFingerprint(t, ctx, store, fetchSettings, lockedFlake)
	if got := lockedFlakeFingerprint(t, ctx, store, fetchSettings, lockedFlake); got != writtenFingerprint {
		t.Fatalf("repeated fingerprint = %q, want %q", got, writtenFingerprint)
	}
	LockedFlakeFree(lockedFlake)

	if _, err := os.Stat(filepath.Join(root, "a", "flake.lock")); err != nil {
		t.Fatalf("written flake.lock stat: %v", err)
	}

	if got := FlakeLockFlagsSetModeCheck(ctx, lockFlags); got != NixOk {
		t.Fatalf("FlakeLockFlagsSetModeCheck after write = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
	}
	lockedFlake = FlakeLock(ctx, fetchSettings, flakeSettings, state, lockFlags, reference)
	if lockedFlake == nil {
		t.Fatalf("FlakeLock in check mode after written lock returned nil: err=%v msg=%q", ErrCode(ctx), errMsgString(t, ctx))
	}
	if got := lockedFlakeHello(t, ctx, flakeSettings, state, lockedFlake); got != "BOB" {
		t.Fatalf("check lock hello = %q, want BOB", got)
	}
	LockedFlakeFree(lockedFlake)

	if got := FlakeLockFlagsSetModeWriteAsNeeded(ctx, lockFlags); got != NixOk {
		t.Fatalf("FlakeLockFlagsSetModeWriteAsNeeded before override = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
	}
	overrideReference, _ := parseTestFlakeReference(t, ctx, fetchSettings, flakeSettings, parseFlags, "./c")
	if got := FlakeLockFlagsAddInputOverride(ctx, lockFlags, "b", overrideReference); got != NixOk {
		t.Fatalf("FlakeLockFlagsAddInputOverride(b) = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
	}
	lockedFlake = FlakeLock(ctx, fetchSettings, flakeSettings, state, lockFlags, reference)
	if lockedFlake == nil {
		t.Fatalf("FlakeLock with input override returned nil: err=%v msg=%q", ErrCode(ctx), errMsgString(t, ctx))
	}
	if got := lockedFlakeHello(t, ctx, flakeSettings, state, lockedFlake); got != "Claire" {
		t.Fatalf("override lock hello = %q, want Claire", got)
	}
	if got := lockedFlakeFingerprint(t, ctx, store, fetchSettings, lockedFlake); got == writtenFingerprint {
		t.Fatalf("override fingerprint = %q, want different from original lock", got)
	}
	LockedFlakeFree(lockedFlake)
}

func TestNixFlakeReferenceLockJSON(t *testing.T) {
	ctx, _, fetchSettings, flakeSettings, state := newTestFlakeState(t)

	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("EvalSymlinks(flake root): %v", err)
	}
	writeTestFlake(t, filepath.Join(root, "dep"), `
{
  outputs = { ... }: {
    hello = "locked";
  };
}
`)
	writeTestFlake(t, filepath.Join(root, "root"), `
{
  inputs.dep.url = "`+filepath.ToSlash(filepath.Join(root, "dep"))+`";
  outputs = { dep, ... }: {
    hello = dep.hello;
  };
}
`)

	parseFlags := newFlakeParseFlags(t, ctx, flakeSettings, root)
	reference, _ := parseTestFlakeReference(t, ctx, fetchSettings, flakeSettings, parseFlags, "./root")

	lockFlags := FlakeLockFlagsNew(ctx, flakeSettings)
	if lockFlags == nil {
		t.Fatalf("FlakeLockFlagsNew returned nil: err=%v msg=%q", ErrCode(ctx), errMsgString(t, ctx))
	}
	t.Cleanup(func() {
		FlakeLockFlagsFree(lockFlags)
	})

	if got := FlakeLockFlagsSetModeVirtual(ctx, lockFlags); got != NixOk {
		t.Fatalf("FlakeLockFlagsSetModeVirtual = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
	}
	lockedFlake := FlakeLock(ctx, fetchSettings, flakeSettings, state, lockFlags, reference)
	if lockedFlake == nil {
		t.Fatalf("FlakeLock in virtual mode returned nil: err=%v msg=%q", ErrCode(ctx), errMsgString(t, ctx))
	}
	lockJSON := ownedCString(t, LockedFlakeGetLockJson(ctx, lockedFlake))
	LockedFlakeFree(lockedFlake)

	if _, err := os.Stat(filepath.Join(root, "root", "flake.lock")); !os.IsNotExist(err) {
		t.Fatalf("virtual lock flake.lock stat err = %v, want not exist", err)
	}

	if got := FlakeLockFlagsSetModeCheck(ctx, lockFlags); got != NixOk {
		t.Fatalf("FlakeLockFlagsSetModeCheck = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
	}
	if got := FlakeLockFlagsSetReferenceLockJson(ctx, lockFlags, lockJSON, uint64(len(lockJSON))); got != NixOk {
		t.Fatalf("FlakeLockFlagsSetReferenceLockJson = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
	}

	lockedFlake = FlakeLock(ctx, fetchSettings, flakeSettings, state, lockFlags, reference)
	if lockedFlake == nil {
		t.Fatalf("FlakeLock with reference lock JSON returned nil: err=%v msg=%q", ErrCode(ctx), errMsgString(t, ctx))
	}
	if got := lockedFlakeHello(t, ctx, flakeSettings, state, lockedFlake); got != "locked" {
		t.Fatalf("reference lock hello = %q, want locked", got)
	}
	LockedFlakeFree(lockedFlake)
}

func TestNixFlakeLockFlagsAddInputOverrideEmptyPath(t *testing.T) {
	ctx, _, fetchSettings, flakeSettings, _ := newTestFlakeState(t)

	root := t.TempDir()
	writeTestFlake(t, root, `
{
  outputs = { ... }: { };
}
`)

	parseFlags := newFlakeParseFlags(t, ctx, flakeSettings, root)
	reference, _ := parseTestFlakeReference(t, ctx, fetchSettings, flakeSettings, parseFlags, ".")

	lockFlags := FlakeLockFlagsNew(ctx, flakeSettings)
	if lockFlags == nil {
		t.Fatalf("FlakeLockFlagsNew returned nil: err=%v msg=%q", ErrCode(ctx), errMsgString(t, ctx))
	}
	t.Cleanup(func() {
		FlakeLockFlagsFree(lockFlags)
	})

	if got := FlakeLockFlagsAddInputOverride(ctx, lockFlags, "", reference); got != NixErrNixError {
		t.Fatalf("FlakeLockFlagsAddInputOverride(empty) = %v, want %v", got, NixErrNixError)
	}
	if got := ErrCode(ctx); got != NixErrNixError {
		t.Fatalf("ErrCode after empty override = %v, want %v", got, NixErrNixError)
	}
	if msg := errMsgString(t, ctx); !strings.Contains(msg, "input override path cannot be zero-length") {
		t.Fatalf("ErrMsg after empty override = %q, want zero-length message", msg)
	}
}
