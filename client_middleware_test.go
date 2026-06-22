package gonix

import (
	"context"
	"errors"
	"testing"

	"github.com/sund3RRR/gonix/eval"
	"github.com/sund3RRR/gonix/store"
)

func TestClientBorrowedResourceHandlers(t *testing.T) {
	client := newTestClient(t)
	ctx := context.Background()

	if err := client.WithStore(ctx, func(s *store.Store) error {
		if s == nil {
			t.Fatal("Client.WithStore() received nil Store")
		}
		return nil
	}); err != nil {
		t.Fatalf("Client.WithStore() error = %v", err)
	}

	if err := client.WithEvaluator(ctx, func(e *eval.Evaluator) error {
		if e == nil {
			t.Fatal("Client.WithEvaluator() received nil Evaluator")
		}
		return nil
	}); err != nil {
		t.Fatalf("Client.WithEvaluator() error = %v", err)
	}

	if err := client.WithStoreAndEvaluator(ctx, func(s *store.Store, e *eval.Evaluator) error {
		if s == nil || e == nil {
			t.Fatalf("Client.WithStoreAndEvaluator() received %v, %v", s, e)
		}
		return nil
	}); err != nil {
		t.Fatalf("Client.WithStoreAndEvaluator() error = %v", err)
	}

	wantErr := errors.New("handler failure")
	if err := client.WithStore(ctx, func(*store.Store) error {
		return wantErr
	}); !errors.Is(err, wantErr) {
		t.Fatalf("Client.WithStore(handler error) = %v, want %v", err, wantErr)
	}
}

func TestClientRejectsConcurrentUse(t *testing.T) {
	client := newTestClient(t)

	started := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- client.WithStore(context.Background(), func(*store.Store) error {
			close(started)
			<-release
			return nil
		})
	}()
	<-started

	if err := client.WithStore(context.Background(), func(*store.Store) error {
		return nil
	}); !errors.Is(err, ErrConcurrentUse) {
		t.Fatalf("overlapping Client.WithStore() error = %v, want ErrConcurrentUse", err)
	}
	if err := client.Close(); !errors.Is(err, ErrConcurrentUse) {
		t.Fatalf("Client.Close() during operation error = %v, want ErrConcurrentUse", err)
	}

	close(release)
	if err := <-done; err != nil {
		t.Fatalf("first Client.WithStore() error = %v", err)
	}
}

func TestClientCancellationClosesClient(t *testing.T) {
	client, err := NewClient(ClientConfig{
		Store: StoreConfig{URI: store.Dummy},
	})
	if err != nil {
		t.Fatalf("NewClient(dummy) error = %v", err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("Client.Close() error = %v", err)
		}
	})

	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{})
	go func() {
		<-started
		cancel()
	}()

	err = client.WithStore(ctx, func(*store.Store) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled Client.WithStore() error = %v, want context.Canceled", err)
	}

	if err := client.WithStore(context.Background(), func(*store.Store) error {
		return nil
	}); !errors.Is(err, ErrClosed) {
		t.Fatalf("Client.WithStore() after cancellation error = %v, want ErrClosed", err)
	}
}
