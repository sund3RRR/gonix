package gonix

import (
	"context"
	"errors"
	"net"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sund3RRR/gonix/nixcontext"
	"github.com/sund3RRR/gonix/store"
)

func newDaemonTestClient(t *testing.T) *Client {
	t.Helper()

	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("filepath.EvalSymlinks: %v", err)
	}

	client, err := NewClient(ClientConfig{
		Store: StoreConfig{
			URI: store.Local,
			Opts: []store.Option{
				store.WithStoreDir(filepath.Join(root, "store")),
				store.WithStateDir(filepath.Join(root, "state")),
				store.WithLogDir(filepath.Join(root, "log")),
				store.WithPathInfoCacheSize(0),
			},
		},
	})
	if err != nil {
		t.Fatalf("NewClient(local) error = %v", err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil && !errors.Is(err, ErrClosed) {
			t.Fatalf("Client.Close() error = %v", err)
		}
	})

	return client
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

func TestClientProcessDaemonConnection(t *testing.T) {
	client := newDaemonTestClient(t)
	listener, socketPath := newDaemonTestListener(t)

	serverErr := make(chan error, 1)
	go func() {
		conn, err := listener.AcceptUnix()
		if err != nil {
			serverErr <- err
			return
		}
		defer conn.Close() //nolint:errcheck

		file, err := conn.File()
		if err != nil {
			serverErr <- err
			return
		}
		defer file.Close() //nolint:errcheck

		fd := int(file.Fd())
		serverErr <- client.ProcessDaemonConnection(context.Background(), fd, fd, false, false)
	}()

	ctx, err := nixContextForDaemonClient()
	if err != nil {
		t.Fatalf("nixContextForDaemonClient() error = %v", err)
	}
	defer ctx.Close() //nolint:errcheck

	remote, err := store.New(ctx, "unix://"+socketPath)
	if err != nil {
		t.Fatalf("store.New(unix) error = %v", err)
	}
	if strings.TrimSpace(remote.Version()) == "" {
		t.Fatal("remote.Version() returned empty string")
	}
	if strings.TrimSpace(remote.StoreDir()) == "" {
		t.Fatal("remote.StoreDir() returned empty string")
	}
	if err := remote.Close(); err != nil {
		t.Fatalf("remote.Close() error = %v", err)
	}

	select {
	case err := <-serverErr:
		if err != nil {
			t.Fatalf("ProcessDaemonConnection() error = %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for daemon connection to finish")
	}
}

func TestClientProcessDaemonConnectionInvalidFD(t *testing.T) {
	client := newDaemonTestClient(t)

	err := client.ProcessDaemonConnection(context.Background(), -1, -1, false, false)
	if err == nil {
		t.Fatal("ProcessDaemonConnection(invalid fd) error = nil, want error")
	}
	if !strings.Contains(err.Error(), "fd must be non-negative") {
		t.Fatalf("ProcessDaemonConnection(invalid fd) error = %v, want fd error", err)
	}
}

func TestClientProcessDaemonConnectionClosedClient(t *testing.T) {
	client := newDaemonTestClient(t)
	if err := client.Close(); err != nil {
		t.Fatalf("Client.Close() error = %v", err)
	}

	if err := client.ProcessDaemonConnection(context.Background(), 0, 0, false, false); !errors.Is(err, ErrClosed) {
		t.Fatalf("ProcessDaemonConnection() after Close error = %v, want ErrClosed", err)
	}
}

func nixContextForDaemonClient() (*nixcontext.Context, error) {
	return nixcontext.New(nixcontext.Config{})
}
