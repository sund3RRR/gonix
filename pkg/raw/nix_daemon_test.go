package raw

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func newDaemonTestContext(t *testing.T) *NixCContext {
	t.Helper()

	ctx := newTestContext(t)
	if got := LibutilInit(ctx); got != NixOk {
		t.Fatalf("LibutilInit = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
	}
	if got := LibstoreInitNoLoadConfig(ctx); got != NixOk {
		t.Fatalf("LibstoreInitNoLoadConfig = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
	}

	return ctx
}

func newDaemonTestListener(t *testing.T) (*net.UnixListener, string) {
	t.Helper()

	socketPath := filepath.Join(t.TempDir(), "daemon.sock")
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: socketPath, Net: "unix"})
	if err != nil {
		t.Fatalf("net.ListenUnix(%q): %v", socketPath, err)
	}
	t.Cleanup(func() {
		if err := listener.Close(); err != nil && !strings.Contains(err.Error(), "use of closed network connection") {
			t.Errorf("listener.Close: %v", err)
		}
	})
	if err := listener.SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
		t.Fatalf("listener.SetDeadline: %v", err)
	}

	return listener, socketPath
}

func serveRawDaemonConnection(
	t *testing.T,
	listener *net.UnixListener,
	handler func(*os.File) NixErr,
) <-chan NixErr {
	t.Helper()

	resultCh := make(chan NixErr, 1)
	go func() {
		conn, err := listener.AcceptUnix()
		if err != nil {
			t.Errorf("listener.AcceptUnix: %v", err)
			resultCh <- NixErrUnknown
			return
		}
		defer conn.Close() //nolint:errcheck

		file, err := conn.File()
		if err != nil {
			t.Errorf("conn.File: %v", err)
			resultCh <- NixErrUnknown
			return
		}
		defer file.Close() //nolint:errcheck

		resultCh <- handler(file)
	}()

	return resultCh
}

func exerciseRawDaemonClient(t *testing.T, socketPath string) {
	t.Helper()

	ctx := newDaemonTestContext(t)
	client := StoreOpen(ctx, "unix://"+socketPath, StoreParams{})
	if client == nil {
		t.Fatalf("StoreOpen(unix) returned nil: err=%v msg=%q", ErrCode(ctx), errMsgString(t, ctx))
	}
	defer StoreFree(client)

	version := StoreGetVersion(ctx, client)
	if version == nil {
		t.Fatalf("StoreGetVersion(unix) returned nil: err=%v msg=%q", ErrCode(ctx), errMsgString(t, ctx))
	}
	StringFree(version)

	storeDir := StoreGetStoredir(ctx, client)
	if storeDir == nil {
		t.Fatalf("StoreGetStoredir(unix) returned nil: err=%v msg=%q", ErrCode(ctx), errMsgString(t, ctx))
	}
	if got := ownedCString(t, storeDir); strings.TrimSpace(got) == "" {
		t.Fatal("StoreGetStoredir(unix) returned empty store dir")
	}
}

func waitRawDaemonResult(t *testing.T, ctx *NixCContext, resultCh <-chan NixErr) {
	t.Helper()

	select {
	case got := <-resultCh:
		if got != NixOk {
			t.Fatalf("daemon process connection = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
		}
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for daemon connection to finish")
	}
}

func TestDaemonProcessConnectionStore(t *testing.T) {
	ctx := newDaemonTestContext(t)
	root := newGCStoreRoot(t)
	store := openTestLocalStoreAt(t, ctx, root)
	defer StoreFree(store)
	listener, socketPath := newDaemonTestListener(t)

	resultCh := serveRawDaemonConnection(t, listener, func(file *os.File) NixErr {
		fd := int32(file.Fd())
		return DaemonProcessConnectionStore(ctx, store, fd, fd, false, false)
	})

	exerciseRawDaemonClient(t, socketPath)
	waitRawDaemonResult(t, ctx, resultCh)
}

func TestDaemonProcessConnectionStoreInvalidFD(t *testing.T) {
	ctx := newDaemonTestContext(t)
	store := openTestLocalStoreAt(t, ctx, newGCStoreRoot(t))
	defer StoreFree(store)

	got := DaemonProcessConnectionStore(ctx, store, -1, -1, false, false)
	if got == NixOk {
		t.Fatalf("DaemonProcessConnectionStore(invalid fd) = %v, want non-OK", got)
	}
	if msg := errMsgString(t, ctx); !strings.Contains(msg, "file descriptors") {
		t.Fatalf("ErrMsg after invalid fd = %q, want file descriptor message", msg)
	}
}

func TestDaemonProcessConnectionStoreNilStore(t *testing.T) {
	ctx := newDaemonTestContext(t)

	got := DaemonProcessConnectionStore(ctx, nil, 0, 0, false, false)
	if got == NixOk {
		t.Fatalf("DaemonProcessConnectionStore(nil) = %v, want non-OK", got)
	}
	if msg := errMsgString(t, ctx); !strings.Contains(msg, "store must not be null") {
		t.Fatalf("ErrMsg after nil store = %q, want nil store message", msg)
	}
}
