package raw

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const testDerivationJSON = `{
  "name": "go-bindings-test",
  "version": 4,
  "outputs": {
    "out": {
      "path": "d1nf171c83f8aczqcn20r20r1bisij3i-go-bindings-test"
    }
  },
  "inputs": {
    "srcs": [],
    "drvs": {}
  },
  "system": "x86_64-linux",
  "builder": "/bin/sh",
  "args": [],
  "env": {
    "builder": "/bin/sh",
    "name": "go-bindings-test",
    "out": "/nix/store/d1nf171c83f8aczqcn20r20r1bisij3i-go-bindings-test",
    "system": "x86_64-linux"
  }
}`

const testCADerivationJSON = `{
  "name": "go-bindings-gc-test",
  "version": 4,
  "outputs": {
    "out": {
      "method": "nar",
      "hashAlgo": "sha256"
    }
  },
  "inputs": {
    "srcs": [],
    "drvs": {}
  },
  "system": "x86_64-linux",
  "builder": "/bin/sh",
  "args": [],
  "env": {
    "builder": "/bin/sh",
    "name": "go-bindings-gc-test",
    "out": "/unused",
    "system": "x86_64-linux"
  }
}`

func errMsgString(t *testing.T, ctx *NixCContext) string {
	t.Helper()

	msg := ErrMsg(nil, ctx)
	if msg == nil {
		return ""
	}
	return ownedCString(t, msg)
}

func newTestStore(t *testing.T, ctx *NixCContext) *Store {
	t.Helper()

	if got := LibstoreInitNoLoadConfig(ctx); got != NixOk {
		t.Fatalf("LibstoreInitNoLoadConfig = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
	}

	store := StoreOpen(ctx, "dummy://", StoreParams{})
	if store == nil {
		t.Fatalf("StoreOpen(dummy://) returned nil: err=%v msg=%q", ErrCode(ctx), errMsgString(t, ctx))
	}
	t.Cleanup(func() {
		StoreFree(store)
	})

	return store
}

func mustParseJSON(t *testing.T, raw string) map[string]any {
	t.Helper()

	var parsed map[string]any
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		t.Fatalf("json.Unmarshal exported derivation: %v\n%s", err, raw)
	}

	return parsed
}

func storePathNameString(t *testing.T, path *StorePath) string {
	t.Helper()

	return ownedCString(t, StorePathName(path))
}

func storePathHashBytes(t *testing.T, ctx *NixCContext, path *StorePath) [20]byte {
	t.Helper()

	var hash StorePathHashPart
	if got := StorePathHash(ctx, path, &hash); got != NixOk {
		hash.Free()
		t.Fatalf("StorePathHash = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
	}
	hash.Deref()
	hash.Free()

	return hash.Bytes
}

func addTestCADerivation(t *testing.T, ctx *NixCContext, store *Store, name string) *StorePath {
	t.Helper()

	raw := strings.ReplaceAll(testCADerivationJSON, "go-bindings-gc-test", name)
	derivation := DerivationFromJson(ctx, store, raw)
	if derivation == nil {
		t.Fatalf("DerivationFromJson(%q) returned nil: err=%v msg=%q", name, ErrCode(ctx), errMsgString(t, ctx))
	}
	defer DerivationFree(derivation)

	path := AddDerivation(ctx, store, derivation)
	if path == nil {
		t.Fatalf("AddDerivation(%q) returned nil: err=%v msg=%q", name, ErrCode(ctx), errMsgString(t, ctx))
	}
	t.Cleanup(func() {
		StorePathFree(path)
	})
	return path
}

func newGCStoreRoot(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q): %v", root, err)
	}
	t.Cleanup(func() {
		chmodTreeWritable(t, resolvedRoot)
	})
	return resolvedRoot
}

func openTestLocalStoreAt(t *testing.T, ctx *NixCContext, root string) *Store {
	t.Helper()

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
	return store
}

func prepareGCStore(t *testing.T, ctx *NixCContext, names ...string) (*Store, string, []*StorePath) {
	t.Helper()

	if got := LibutilInit(ctx); got != NixOk {
		t.Fatalf("LibutilInit = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
	}
	if got := SettingSet(ctx, "experimental-features", "ca-derivations"); got != NixOk {
		t.Fatalf("SettingSet(ca-derivations) = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
	}
	if got := LibstoreInit(ctx); got != NixOk {
		t.Fatalf("LibstoreInit = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
	}

	root := newGCStoreRoot(t)
	seedStore := openTestLocalStoreAt(t, ctx, root)
	paths := make([]*StorePath, 0, len(names))
	for _, name := range names {
		paths = append(paths, addTestCADerivation(t, ctx, seedStore, name))
	}
	StoreFree(seedStore)

	store := openTestLocalStoreAt(t, ctx, root)
	t.Cleanup(func() {
		StoreFree(store)
	})
	return store, root, paths
}

func storeRealPathString(t *testing.T, ctx *NixCContext, store *Store, path *StorePath) string {
	t.Helper()

	value := StoreRealPath(ctx, store, path)
	if value == nil {
		t.Fatalf("StoreRealPath returned nil: err=%v msg=%q", ErrCode(ctx), errMsgString(t, ctx))
	}
	return ownedCString(t, value)
}

func gcResultPathSet(t *testing.T, results *StoreGCResults) map[string]struct{} {
	t.Helper()

	paths := make(map[string]struct{}, StoreGCResultsCount(results))
	for i := uint64(0); i < StoreGCResultsCount(results); i++ {
		path := StoreGCResultsPath(results, i)
		if path == nil {
			t.Fatalf("StoreGCResultsPath(%d) returned nil", i)
		}
		paths[ownedCString(t, path)] = struct{}{}
	}
	return paths
}

func collectGarbage(
	t *testing.T,
	ctx *NixCContext,
	store *Store,
	options StoreGCOptions,
) (*StoreGCResults, map[string]struct{}) {
	t.Helper()

	results := StoreCollectGarbage(ctx, store, options)
	if results == nil {
		t.Fatalf("StoreCollectGarbage(%v) returned nil: err=%v msg=%q", options.Action, ErrCode(ctx), errMsgString(t, ctx))
	}
	paths := gcResultPathSet(t, results)
	return results, paths
}

func TestNixStoreOpenAndStrings(t *testing.T) {
	ctx := newTestContext(t)
	store := newTestStore(t, ctx)

	storeDir := ownedCString(t, StoreGetStoredir(ctx, store))
	if storeDir != "/nix/store" {
		t.Fatalf("StoreGetStoredir = %q, want /nix/store", storeDir)
	}

	uri := ownedCString(t, StoreGetUri(ctx, store))
	if strings.TrimSpace(uri) == "" {
		t.Fatal("StoreGetUri returned an empty string")
	}

	version := StoreGetVersion(ctx, store)
	if version != nil {
		StringFree(version)
	}
}

func TestNixStoreInterruptDummy(t *testing.T) {
	InterruptClear()
	t.Cleanup(InterruptClear)

	ctx := newTestContext(t)
	store := newTestStore(t, ctx)

	if got := StoreInterrupt(ctx, store); got != NixOk {
		t.Fatalf("StoreInterrupt(dummy) = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
	}
	if !InterruptRequested() {
		t.Fatal("InterruptRequested after StoreInterrupt(dummy) = false, want true")
	}
}

func TestNixStoreInterruptNilSetsContextError(t *testing.T) {
	InterruptClear()
	t.Cleanup(InterruptClear)

	ctx := newTestContext(t)
	got := StoreInterrupt(ctx, nil)
	if got == NixOk {
		t.Fatal("StoreInterrupt(nil) = NixOk, want non-OK")
	}
	if ErrCode(ctx) == NixOk {
		t.Fatal("ErrCode after StoreInterrupt(nil) = NixOk, want non-OK")
	}
	if msg := errMsgString(t, ctx); !strings.Contains(msg, "store must not be null") {
		t.Fatalf("ErrMsg after StoreInterrupt(nil) = %q, want null-store message", msg)
	}
	if InterruptRequested() {
		t.Fatal("InterruptRequested after StoreInterrupt(nil) = true, want false")
	}
}

type blockingProxyWriter struct {
	dst       io.Writer
	block     *atomic.Bool
	blocked   chan<- struct{}
	release   <-chan struct{}
	blockOnce sync.Once
}

func (w *blockingProxyWriter) Write(p []byte) (int, error) {
	if w.block.Load() {
		w.blockOnce.Do(func() {
			close(w.blocked)
		})
		<-w.release
	}
	return w.dst.Write(p)
}

func TestNixStoreInterruptRemoteWakesBlockingIO(t *testing.T) {
	InterruptClear()
	t.Cleanup(InterruptClear)

	root, err := os.MkdirTemp("/tmp", "gonix-raw-interrupt-")
	if err != nil {
		t.Fatalf("os.MkdirTemp(/tmp): %v", err)
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		_ = os.RemoveAll(root)
		t.Fatalf("filepath.EvalSymlinks(%q): %v", root, err)
	}
	t.Cleanup(func() {
		chmodTreeWritable(t, resolvedRoot)
		if err := os.RemoveAll(resolvedRoot); err != nil {
			t.Errorf("os.RemoveAll(%q): %v", resolvedRoot, err)
		}
	})

	storeDir := filepath.Join(resolvedRoot, "store")
	stateDir := filepath.Join(resolvedRoot, "state")
	logDir := filepath.Join(resolvedRoot, "log")
	for _, dir := range []string{storeDir, stateDir, logDir} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatalf("os.MkdirAll(%q): %v", dir, err)
		}
	}

	socketPath := filepath.Join(resolvedRoot, "daemon.sock")
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: socketPath, Net: "unix"})
	if err != nil {
		t.Fatalf("net.ListenUnix(%q): %v", socketPath, err)
	}
	if err := listener.SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
		_ = listener.Close()
		t.Fatalf("listener.SetDeadline: %v", err)
	}

	storeURI := "local?store=" + storeDir + "&state=" + stateDir + "&log=" + logDir
	cmd := exec.Command(
		"nix",
		"--extra-experimental-features", "nix-command",
		"--option", "sandbox", "false",
		"daemon",
		"--stdio",
		"--store", storeURI,
	)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		_ = listener.Close()
		t.Fatalf("cmd.StdinPipe: %v", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = listener.Close()
		t.Fatalf("cmd.StdoutPipe: %v", err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		_ = listener.Close()
		t.Fatalf("start nix daemon --stdio: %v", err)
	}

	var conn *net.UnixConn
	releaseResponses := make(chan struct{})
	var releaseOnce sync.Once
	var stopOnce sync.Once
	stopDaemon := func() string {
		stopOnce.Do(func() {
			releaseOnce.Do(func() {
				close(releaseResponses)
			})
			_ = listener.Close()
			if conn != nil {
				_ = conn.Close()
			}
			_ = stdin.Close()
			_ = stdout.Close()
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
			_ = cmd.Wait()
		})
		return stderr.String()
	}

	var (
		operationCtx *NixCContext
		interruptCtx *NixCContext
		remoteStore  *Store
		path         *StorePath
	)
	defer func() {
		stopDaemon()
		if path != nil {
			StorePathFree(path)
		}
		if remoteStore != nil {
			StoreFree(remoteStore)
		}
		if interruptCtx != nil {
			CContextFree(interruptCtx)
		}
		if operationCtx != nil {
			CContextFree(operationCtx)
		}
	}()

	operationCtx = CContextCreate()
	if operationCtx == nil {
		t.Fatal("CContextCreate(operation) returned nil")
	}
	if got := LibutilInit(operationCtx); got != NixOk {
		t.Fatalf("LibutilInit(operation) = %v, want %v: %s", got, NixOk, errMsgString(t, operationCtx))
	}
	if got := LibstoreInitNoLoadConfig(operationCtx); got != NixOk {
		t.Fatalf("LibstoreInitNoLoadConfig(operation) = %v, want %v: %s", got, NixOk, errMsgString(t, operationCtx))
	}

	remoteStore = StoreOpen(operationCtx, "unix://"+socketPath, StoreParams{})
	if remoteStore == nil {
		t.Fatalf("StoreOpen(unix) returned nil: err=%v msg=%q", ErrCode(operationCtx), errMsgString(t, operationCtx))
	}

	type versionResult struct {
		value *byte
	}
	versionCh := make(chan versionResult, 1)
	go func() {
		versionCh <- versionResult{value: StoreGetVersion(operationCtx, remoteStore)}
	}()

	conn, err = listener.AcceptUnix()
	if err != nil {
		log := stopDaemon()
		t.Fatalf("listener.AcceptUnix: %v\ndaemon stderr:\n%s", err, log)
	}

	var blockResponses atomic.Bool
	responseBlocked := make(chan struct{})
	go func() {
		_, _ = io.Copy(stdin, conn)
		_ = stdin.Close()
	}()
	go func() {
		writer := &blockingProxyWriter{
			dst:     conn,
			block:   &blockResponses,
			blocked: responseBlocked,
			release: releaseResponses,
		}
		_, _ = io.Copy(writer, stdout)
		_ = conn.CloseWrite()
	}()

	select {
	case result := <-versionCh:
		if result.value == nil {
			log := stopDaemon()
			t.Fatalf(
				"StoreGetVersion(remote) returned nil: err=%v msg=%q\ndaemon stderr:\n%s",
				ErrCode(operationCtx),
				errMsgString(t, operationCtx),
				log,
			)
		}
		StringFree(result.value)
	case <-time.After(10 * time.Second):
		log := stopDaemon()
		t.Fatalf("timed out establishing remote store connection\ndaemon stderr:\n%s", log)
	}

	const rawPath = "/nix/store/00000000000000000000000000000000-interrupt-test"
	path = StoreParsePath(operationCtx, remoteStore, rawPath)
	if path == nil {
		log := stopDaemon()
		t.Fatalf(
			"StoreParsePath(remote) returned nil: err=%v msg=%q\ndaemon stderr:\n%s",
			ErrCode(operationCtx),
			errMsgString(t, operationCtx),
			log,
		)
	}

	type validityResult struct {
		valid bool
		code  NixErr
	}
	blockResponses.Store(true)
	validityCh := make(chan validityResult, 1)
	go func() {
		valid := StoreIsValidPath(operationCtx, remoteStore, path)
		validityCh <- validityResult{valid: valid, code: ErrCode(operationCtx)}
	}()

	select {
	case <-responseBlocked:
	case <-time.After(10 * time.Second):
		log := stopDaemon()
		t.Fatalf("daemon response was not blocked\ndaemon stderr:\n%s", log)
	}

	interruptCtx = CContextCreate()
	if interruptCtx == nil {
		t.Fatal("CContextCreate(interrupt) returned nil")
	}
	if got := StoreInterrupt(interruptCtx, remoteStore); got != NixOk {
		log := stopDaemon()
		t.Fatalf(
			"StoreInterrupt(remote) = %v, want %v: %s\ndaemon stderr:\n%s",
			got,
			NixOk,
			errMsgString(t, interruptCtx),
			log,
		)
	}

	select {
	case result := <-validityCh:
		if result.valid {
			t.Fatal("StoreIsValidPath after interruption = true, want false")
		}
		if result.code == NixOk {
			t.Fatal("ErrCode after interrupted StoreIsValidPath = NixOk, want non-OK")
		}
	case <-time.After(5 * time.Second):
		log := stopDaemon()
		t.Fatalf("StoreInterrupt did not wake blocked remote operation\ndaemon stderr:\n%s", log)
	}

	if !InterruptRequested() {
		t.Fatal("InterruptRequested after StoreInterrupt(remote) = false, want true")
	}
	InterruptClear()
	releaseOnce.Do(func() {
		close(releaseResponses)
	})
}

func TestNixStoreParsePathRealPathAndValidity(t *testing.T) {
	ctx := newTestContext(t)
	store := newTestStore(t, ctx)

	const rawPath = "/nix/store/00000000000000000000000000000000-demo"
	path := StoreParsePath(ctx, store, rawPath)
	if path == nil {
		t.Fatalf("StoreParsePath returned nil: err=%v msg=%q", ErrCode(ctx), errMsgString(t, ctx))
	}
	t.Cleanup(func() {
		StorePathFree(path)
	})

	if name := storePathNameString(t, path); name != "demo" {
		t.Fatalf("StorePathName = %q, want demo", name)
	}

	zeroHash := storePathHashBytes(t, ctx, path)
	if zeroHash != ([20]byte{}) {
		t.Fatalf("StorePathHash for zero path = %v, want all zero bytes", zeroHash)
	}

	realPath := ownedCString(t, StoreRealPath(ctx, store, path))
	if realPath != rawPath {
		t.Fatalf("StoreRealPath = %q, want %q", realPath, rawPath)
	}

	if StoreIsValidPath(ctx, store, path) {
		t.Fatal("dummy store unexpectedly reports arbitrary path valid")
	}

	clone := StorePathClone(path)
	if clone == nil {
		t.Fatalf("StorePathClone returned nil: err=%v msg=%q", ErrCode(ctx), errMsgString(t, ctx))
	}
	t.Cleanup(func() {
		StorePathFree(clone)
	})
	if name := storePathNameString(t, clone); name != "demo" {
		t.Fatalf("StorePathName(clone) = %q, want demo", name)
	}
	if cloneHash := storePathHashBytes(t, ctx, clone); cloneHash != zeroHash {
		t.Fatalf("StorePathHash(clone) = %v, want %v", cloneHash, zeroHash)
	}

	const nonZeroRawPath = "/nix/store/11111111111111111111111111111111-demo"
	nonZeroPath := StoreParsePath(ctx, store, nonZeroRawPath)
	if nonZeroPath == nil {
		t.Fatalf("StoreParsePath(non-zero) returned nil: err=%v msg=%q", ErrCode(ctx), errMsgString(t, ctx))
	}
	t.Cleanup(func() {
		StorePathFree(nonZeroPath)
	})

	nonZeroHash := StorePathHashPart{
		Bytes: storePathHashBytes(t, ctx, nonZeroPath),
	}
	if nonZeroHash.Bytes == ([20]byte{}) {
		t.Fatal("StorePathHash for non-zero path returned all zero bytes")
	}

	const createdName = "created-from-parts"
	created := StoreCreateFromParts(ctx, &nonZeroHash, createdName, uint64(len(createdName)))
	nonZeroHash.Free()
	if created == nil {
		t.Fatalf("StoreCreateFromParts returned nil: err=%v msg=%q", ErrCode(ctx), errMsgString(t, ctx))
	}
	t.Cleanup(func() {
		StorePathFree(created)
	})
	if name := storePathNameString(t, created); name != createdName {
		t.Fatalf("StorePathName(created) = %q, want %q", name, createdName)
	}
	if createdHash := storePathHashBytes(t, ctx, created); createdHash != nonZeroHash.Bytes {
		t.Fatalf("StorePathHash(created) = %v, want %v", createdHash, nonZeroHash.Bytes)
	}
}

func TestNixStoreParseInvalidPathSetsContextError(t *testing.T) {
	ctx := newTestContext(t)
	store := newTestStore(t, ctx)

	path := StoreParsePath(ctx, store, "/not-a-store-path")
	if path != nil {
		t.Fatal("StoreParsePath unexpectedly accepted invalid path")
	}
	if got := ErrCode(ctx); got == NixOk {
		t.Fatalf("ErrCode after invalid StoreParsePath = %v, want non-OK", got)
	}
	if msg := errMsgString(t, ctx); strings.TrimSpace(msg) == "" {
		t.Fatal("ErrMsg after invalid StoreParsePath returned an empty string")
	}

	ClearErr(ctx)
	if got := ErrCode(ctx); got != NixOk {
		t.Fatalf("ErrCode after ClearErr = %v, want %v", got, NixOk)
	}
}

func TestNixStoreOpenInvalidUriSetsContextError(t *testing.T) {
	ctx := newTestContext(t)
	if got := LibstoreInitNoLoadConfig(ctx); got != NixOk {
		t.Fatalf("LibstoreInitNoLoadConfig = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
	}

	store := StoreOpen(ctx, "go-bindings-test-invalid-store://", StoreParams{})
	if store != nil {
		StoreFree(store)
		t.Fatal("StoreOpen unexpectedly accepted invalid URI")
	}
	if got := ErrCode(ctx); got == NixOk {
		t.Fatalf("ErrCode after invalid StoreOpen = %v, want non-OK", got)
	}
	if msg := errMsgString(t, ctx); strings.TrimSpace(msg) == "" {
		t.Fatal("ErrMsg after invalid StoreOpen returned an empty string")
	}
}

func TestNixStoreDerivationLifecycleAndJSON(t *testing.T) {
	ctx := newTestContext(t)
	store := newTestStore(t, ctx)

	derivation := DerivationFromJson(ctx, store, testDerivationJSON)
	if derivation == nil {
		t.Fatalf("DerivationFromJson returned nil: err=%v msg=%q", ErrCode(ctx), errMsgString(t, ctx))
	}
	t.Cleanup(func() {
		DerivationFree(derivation)
	})

	exported := ownedCString(t, DerivationToJson(ctx, derivation))
	if strings.TrimSpace(exported) == "" {
		t.Fatal("DerivationToJson returned an empty string")
	}

	exportedJSON := mustParseJSON(t, exported)
	if exportedJSON["name"] != "go-bindings-test" {
		t.Fatalf("exported derivation name = %v, want go-bindings-test", exportedJSON["name"])
	}
	if exportedJSON["version"] != float64(4) {
		t.Fatalf("exported derivation version = %v, want 4", exportedJSON["version"])
	}
	outputs, ok := exportedJSON["outputs"].(map[string]any)
	if !ok {
		t.Fatalf("exported derivation outputs = %T, want JSON object", exportedJSON["outputs"])
	}
	if _, ok := outputs["out"]; !ok {
		t.Fatal("exported derivation outputs missing out")
	}

	clone := DerivationClone(derivation)
	if clone == nil {
		t.Fatalf("DerivationClone returned nil: err=%v msg=%q", ErrCode(ctx), errMsgString(t, ctx))
	}
	t.Cleanup(func() {
		DerivationFree(clone)
	})

	cloneExported := ownedCString(t, DerivationToJson(ctx, clone))
	cloneJSON := mustParseJSON(t, cloneExported)
	if !reflect.DeepEqual(cloneJSON, exportedJSON) {
		t.Fatalf("DerivationToJson(clone) = %#v, want %#v", cloneJSON, exportedJSON)
	}
}

func TestNixStoreInvalidDerivationJSONSetsContextError(t *testing.T) {
	ctx := newTestContext(t)
	store := newTestStore(t, ctx)

	derivation := DerivationFromJson(ctx, store, `{"version": 4}`)
	if derivation != nil {
		DerivationFree(derivation)
		t.Fatal("DerivationFromJson unexpectedly accepted invalid derivation JSON")
	}
	if got := ErrCode(ctx); got == NixOk {
		t.Fatalf("ErrCode after invalid DerivationFromJson = %v, want non-OK", got)
	}
	if msg := errMsgString(t, ctx); strings.TrimSpace(msg) == "" {
		t.Fatal("ErrMsg after invalid DerivationFromJson returned an empty string")
	}

	ClearErr(ctx)
	if got := ErrCode(ctx); got != NixOk {
		t.Fatalf("ErrCode after ClearErr = %v, want %v", got, NixOk)
	}
}

func TestNixStoreCopyPathRejectsNilArguments(t *testing.T) {
	ctx := newTestContext(t)
	store := newTestStore(t, ctx)

	if got := StoreCopyPath(ctx, nil, store, nil, false, false); got == NixOk {
		t.Fatalf("StoreCopyPath with nil arguments = %v, want non-OK", got)
	}
	if msg := errMsgString(t, ctx); strings.TrimSpace(msg) == "" {
		t.Fatal("ErrMsg after invalid StoreCopyPath returned an empty string")
	}
}

func TestNixStoreCallbackBackedAdapters(t *testing.T) {
	ctx := newTestContext(t)
	store := newTestStore(t, ctx)

	const rawPath = "/nix/store/00000000000000000000000000000000-demo"
	path := StoreParsePath(ctx, store, rawPath)
	if path == nil {
		t.Fatalf("StoreParsePath returned nil: err=%v msg=%q", ErrCode(ctx), errMsgString(t, ctx))
	}
	t.Cleanup(func() {
		StorePathFree(path)
	})

	if got := StoreRealiseResultsCount(nil); got != 0 {
		t.Fatalf("StoreRealiseResultsCount(nil) = %d, want 0", got)
	}
	if got := StoreRealiseResultsOutname(nil, 0); got != nil {
		StringFree(got)
		t.Fatal("StoreRealiseResultsOutname(nil, 0) returned non-nil")
	}
	if got := StoreRealiseResultsPathClone(nil, 0); got != nil {
		StorePathFree(got)
		t.Fatal("StoreRealiseResultsPathClone(nil, 0) returned non-nil")
	}
	StoreRealiseResultsFree(nil)

	if got := StoreRealiseOutput(ctx, nil, nil, "out"); got != nil {
		StorePathFree(got)
		t.Fatal("StoreRealiseOutput(nil, nil, out) returned non-nil")
	}
	if msg := errMsgString(t, ctx); !strings.Contains(msg, "must not be null") {
		t.Fatalf("ErrMsg after invalid StoreRealiseOutput = %q, want null argument message", msg)
	}

	results := StoreRealiseToArray(ctx, store, path)
	if results != nil {
		t.Cleanup(func() {
			StoreRealiseResultsFree(results)
		})
		for i := uint64(0); i < StoreRealiseResultsCount(results); i++ {
			outname := StoreRealiseResultsOutname(results, i)
			if outname == nil {
				t.Fatalf("StoreRealiseResultsOutname(%d) returned nil", i)
			}
			StringFree(outname)

			clone := StoreRealiseResultsPathClone(results, i)
			if clone == nil {
				t.Fatalf("StoreRealiseResultsPathClone(%d) returned nil", i)
			}
			StorePathFree(clone)
		}
	} else {
		ClearErr(ctx)
	}

	if got := StorePathArrayCount(nil); got != 0 {
		t.Fatalf("StorePathArrayCount(nil) = %d, want 0", got)
	}
	if got := StorePathArrayPathClone(nil, 0); got != nil {
		StorePathFree(got)
		t.Fatal("StorePathArrayPathClone(nil, 0) returned non-nil")
	}
	StorePathArrayFree(nil)

	paths := StoreGetFsClosureArray(ctx, store, path, false, false, false)
	if paths == nil {
		ClearErr(ctx)
		return
	}
	t.Cleanup(func() {
		StorePathArrayFree(paths)
	})
	for i := uint64(0); i < StorePathArrayCount(paths); i++ {
		clone := StorePathArrayPathClone(paths, i)
		if clone == nil {
			t.Fatalf("StorePathArrayPathClone(%d) returned nil", i)
		}
		StorePathFree(clone)
	}
}

func TestNixStoreGCRootsAndCollection(t *testing.T) {
	ctx := newTestContext(t)
	store, root, paths := prepareGCStore(
		t,
		ctx,
		"gc-temp-root",
		"gc-permanent-root",
		"gc-dead-specific",
		"gc-dead-all",
	)
	tempPath, permanentPath, deadPath, deadAllPath := paths[0], paths[1], paths[2], paths[3]

	tempPathString := storeRealPathString(t, ctx, store, tempPath)
	permanentPathString := storeRealPathString(t, ctx, store, permanentPath)
	deadPathString := storeRealPathString(t, ctx, store, deadPath)

	if got := StoreAddTempRoot(ctx, store, tempPath); got != NixOk {
		t.Fatalf("StoreAddTempRoot = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
	}

	rootPath := filepath.Join(root, "roots", "..", "roots", "permanent")
	canonicalRootPath := filepath.Clean(rootPath)
	returnedRoot := StoreAddPermanentRoot(ctx, store, permanentPath, rootPath)
	if returnedRoot == nil {
		t.Fatalf("StoreAddPermanentRoot returned nil: err=%v msg=%q", ErrCode(ctx), errMsgString(t, ctx))
	}
	if got := ownedCString(t, returnedRoot); got != canonicalRootPath {
		t.Fatalf("StoreAddPermanentRoot = %q, want %q", got, canonicalRootPath)
	}
	if target, err := os.Readlink(canonicalRootPath); err != nil {
		t.Fatalf("Readlink(%q): %v", canonicalRootPath, err)
	} else if target != permanentPathString {
		t.Fatalf("permanent root target = %q, want %q", target, permanentPathString)
	}

	for _, censor := range []bool{false, true} {
		roots := StoreFindRoots(ctx, store, censor)
		if roots == nil {
			t.Fatalf("StoreFindRoots(censor=%t) returned nil: err=%v msg=%q", censor, ErrCode(ctx), errMsgString(t, ctx))
		}

		found := false
		for i := uint64(0); i < StoreRootsCount(roots); i++ {
			path := StoreRootsPathClone(roots, i)
			link := StoreRootsLink(roots, i)
			if path == nil || link == nil {
				StorePathFree(path)
				StringFree(link)
				StoreRootsFree(roots)
				t.Fatalf("StoreFindRoots accessor %d returned nil", i)
			}

			pathString := storeRealPathString(t, ctx, store, path)
			linkString := ownedCString(t, link)
			StorePathFree(path)
			if pathString == permanentPathString && linkString == canonicalRootPath {
				found = true
			}
		}
		if got := StoreRootsPathClone(roots, StoreRootsCount(roots)); got != nil {
			StorePathFree(got)
			StoreRootsFree(roots)
			t.Fatal("StoreRootsPathClone(out of range) returned non-nil")
		}
		if got := StoreRootsLink(roots, StoreRootsCount(roots)); got != nil {
			StringFree(got)
			StoreRootsFree(roots)
			t.Fatal("StoreRootsLink(out of range) returned non-nil")
		}
		StoreRootsFree(roots)
		if !found {
			t.Fatalf("StoreFindRoots(censor=%t) did not include %q -> %q", censor, canonicalRootPath, permanentPathString)
		}
	}

	liveResults, livePaths := collectGarbage(t, ctx, store, StoreGCOptions{
		Action:   StoreGCReturnLive,
		MaxFreed: ^uint64(0),
	})
	if got := StoreGCResultsPath(liveResults, StoreGCResultsCount(liveResults)); got != nil {
		StringFree(got)
		StoreGCResultsFree(liveResults)
		t.Fatal("StoreGCResultsPath(out of range) returned non-nil")
	}
	StoreGCResultsFree(liveResults)
	for _, path := range []string{tempPathString, permanentPathString} {
		if _, ok := livePaths[path]; !ok {
			t.Fatalf("return-live results missing rooted path %q", path)
		}
	}

	deadResults, deadPaths := collectGarbage(t, ctx, store, StoreGCOptions{
		Action:   StoreGCReturnDead,
		MaxFreed: ^uint64(0),
	})
	StoreGCResultsFree(deadResults)
	if _, ok := deadPaths[deadPathString]; !ok {
		t.Fatalf("return-dead results missing %q", deadPathString)
	}
	for _, path := range []string{tempPathString, permanentPathString} {
		if _, ok := deadPaths[path]; ok {
			t.Fatalf("return-dead results unexpectedly include rooted path %q", path)
		}
	}

	limitedResults, limitedPaths := collectGarbage(t, ctx, store, StoreGCOptions{
		Action:   StoreGCDeleteDead,
		MaxFreed: 0,
	})
	if got := StoreGCResultsBytesFreed(limitedResults); got != 0 {
		StoreGCResultsFree(limitedResults)
		t.Fatalf("delete-dead with MaxFreed=0 freed %d bytes, want 0", got)
	}
	StoreGCResultsFree(limitedResults)
	if len(limitedPaths) != 0 {
		t.Fatalf("delete-dead with MaxFreed=0 returned %d paths, want 0", len(limitedPaths))
	}
	if _, err := os.Stat(deadPathString); err != nil {
		t.Fatalf("Stat(%q) after delete-dead with MaxFreed=0: %v", deadPathString, err)
	}

	specificResults, specificPaths := collectGarbage(t, ctx, store, StoreGCOptions{
		Action: StoreGCDeleteSpecific,
		PathsToDelete: StorePathList{
			Items: []StorePathItem{{Path: deadPath}},
			Len:   1,
		},
		MaxFreed: ^uint64(0),
	})
	if got := StoreGCResultsBytesFreed(specificResults); got == 0 {
		StoreGCResultsFree(specificResults)
		t.Fatal("delete-specific reported zero bytes freed")
	}
	StoreGCResultsFree(specificResults)
	if _, ok := specificPaths[deadPathString]; !ok {
		t.Fatalf("delete-specific results missing %q", deadPathString)
	}
	if _, err := os.Stat(deadPathString); !os.IsNotExist(err) {
		t.Fatalf("Stat(%q) after delete-specific = %v, want not-exist", deadPathString, err)
	}

	deadAllPathString := storeRealPathString(t, ctx, store, deadAllPath)
	allResults, allPaths := collectGarbage(t, ctx, store, StoreGCOptions{
		Action:   StoreGCDeleteDead,
		MaxFreed: ^uint64(0),
	})
	if got := StoreGCResultsBytesFreed(allResults); got == 0 {
		StoreGCResultsFree(allResults)
		t.Fatal("delete-dead reported zero bytes freed")
	}
	StoreGCResultsFree(allResults)
	if _, ok := allPaths[deadAllPathString]; !ok {
		t.Fatalf("delete-dead results missing %q", deadAllPathString)
	}
	if _, err := os.Stat(deadAllPathString); !os.IsNotExist(err) {
		t.Fatalf("Stat(%q) after delete-dead = %v, want not-exist", deadAllPathString, err)
	}
}

func TestNixStoreGCIgnoreLiveness(t *testing.T) {
	ctx := newTestContext(t)
	store, root, seededPaths := prepareGCStore(t, ctx, "gc-ignore-liveness")
	path := seededPaths[0]
	pathString := storeRealPathString(t, ctx, store, path)
	directRoot := filepath.Join(root, "state", "gcroots", "direct")
	if err := os.MkdirAll(filepath.Dir(directRoot), 0o700); err != nil {
		t.Fatalf("MkdirAll(%q): %v", filepath.Dir(directRoot), err)
	}
	if err := os.Symlink(pathString, directRoot); err != nil {
		t.Fatalf("Symlink(%q, %q): %v", pathString, directRoot, err)
	}

	options := StoreGCOptions{
		Action: StoreGCDeleteSpecific,
		PathsToDelete: StorePathList{
			Items: []StorePathItem{{Path: path}},
			Len:   1,
		},
		MaxFreed: ^uint64(0),
	}
	if results := StoreCollectGarbage(ctx, store, options); results != nil {
		StoreGCResultsFree(results)
		t.Fatal("delete-specific without IgnoreLiveness unexpectedly deleted a rooted path")
	}
	if got := ErrCode(ctx); got == NixOk {
		t.Fatal("delete-specific without IgnoreLiveness did not set a context error")
	}
	if _, err := os.Stat(pathString); err != nil {
		t.Fatalf("Stat(%q) after protected delete-specific: %v", pathString, err)
	}

	ClearErr(ctx)
	options.IgnoreLiveness = true
	results, paths := collectGarbage(t, ctx, store, options)
	StoreGCResultsFree(results)
	if _, ok := paths[pathString]; !ok {
		t.Fatalf("ignore-liveness delete results missing %q", pathString)
	}
	if _, err := os.Stat(pathString); !os.IsNotExist(err) {
		t.Fatalf("Stat(%q) after IgnoreLiveness delete = %v, want not-exist", pathString, err)
	}
}

func TestNixStoreGCAdapterErrorsAndOwnership(t *testing.T) {
	ctx := newTestContext(t)
	store := newTestStore(t, ctx)

	const rawPath = "/nix/store/00000000000000000000000000000000-gc-errors"
	path := StoreParsePath(ctx, store, rawPath)
	if path == nil {
		t.Fatalf("StoreParsePath returned nil: err=%v msg=%q", ErrCode(ctx), errMsgString(t, ctx))
	}
	t.Cleanup(func() {
		StorePathFree(path)
	})

	if got := StoreAddTempRoot(ctx, store, path); got != NixOk {
		t.Fatalf("StoreAddTempRoot(dummy) = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
	}

	if root := StoreAddPermanentRoot(ctx, store, path, filepath.Join(t.TempDir(), "root")); root != nil {
		StringFree(root)
		t.Fatal("StoreAddPermanentRoot(dummy) unexpectedly succeeded")
	}
	if got := ErrCode(ctx); got == NixOk {
		t.Fatal("StoreAddPermanentRoot(dummy) did not set a context error")
	}

	ClearErr(ctx)
	if roots := StoreFindRoots(ctx, store, false); roots != nil {
		StoreRootsFree(roots)
		t.Fatal("StoreFindRoots(dummy) unexpectedly succeeded")
	}
	if got := ErrCode(ctx); got == NixOk {
		t.Fatal("StoreFindRoots(dummy) did not set a context error")
	}

	ClearErr(ctx)
	if results := StoreCollectGarbage(ctx, store, StoreGCOptions{
		Action:   StoreGCReturnDead,
		MaxFreed: ^uint64(0),
	}); results != nil {
		StoreGCResultsFree(results)
		t.Fatal("StoreCollectGarbage(dummy) unexpectedly succeeded")
	}
	if got := ErrCode(ctx); got == NixOk {
		t.Fatal("StoreCollectGarbage(dummy) did not set a context error")
	}

	ClearErr(ctx)
	localStore, _ := newTestLocalStore(t, ctx)
	if results := StoreCollectGarbage(ctx, localStore, StoreGCOptions{
		Action:   StoreGCAction(99),
		MaxFreed: ^uint64(0),
	}); results != nil {
		StoreGCResultsFree(results)
		t.Fatal("StoreCollectGarbage(invalid action) unexpectedly succeeded")
	}
	if got := ErrCode(ctx); got == NixOk {
		t.Fatal("StoreCollectGarbage(invalid action) did not set a context error")
	}

	ClearErr(ctx)
	if results := StoreCollectGarbage(ctx, localStore, StoreGCOptions{
		Action: StoreGCDeleteSpecific,
		PathsToDelete: StorePathList{
			Len: 1,
		},
		MaxFreed: ^uint64(0),
	}); results != nil {
		StoreGCResultsFree(results)
		t.Fatal("StoreCollectGarbage(invalid path list) unexpectedly succeeded")
	}
	if got := ErrCode(ctx); got == NixOk {
		t.Fatal("StoreCollectGarbage(invalid path list) did not set a context error")
	}

	ClearErr(ctx)
	if results := StoreCollectGarbage(ctx, localStore, StoreGCOptions{
		Action: StoreGCDeleteSpecific,
		PathsToDelete: StorePathList{
			Items: []StorePathItem{{}},
			Len:   1,
		},
		MaxFreed: ^uint64(0),
	}); results != nil {
		StoreGCResultsFree(results)
		t.Fatal("StoreCollectGarbage(nil path item) unexpectedly succeeded")
	}
	if got := ErrCode(ctx); got == NixOk {
		t.Fatal("StoreCollectGarbage(nil path item) did not set a context error")
	}

	if got := StoreRootsCount(nil); got != 0 {
		t.Fatalf("StoreRootsCount(nil) = %d, want 0", got)
	}
	if got := StoreRootsPathClone(nil, 0); got != nil {
		StorePathFree(got)
		t.Fatal("StoreRootsPathClone(nil, 0) returned non-nil")
	}
	if got := StoreRootsLink(nil, 0); got != nil {
		StringFree(got)
		t.Fatal("StoreRootsLink(nil, 0) returned non-nil")
	}
	StoreRootsFree(nil)

	if got := StoreGCResultsCount(nil); got != 0 {
		t.Fatalf("StoreGCResultsCount(nil) = %d, want 0", got)
	}
	if got := StoreGCResultsPath(nil, 0); got != nil {
		StringFree(got)
		t.Fatal("StoreGCResultsPath(nil, 0) returned non-nil")
	}
	if got := StoreGCResultsBytesFreed(nil); got != 0 {
		t.Fatalf("StoreGCResultsBytesFreed(nil) = %d, want 0", got)
	}
	StoreGCResultsFree(nil)
}
