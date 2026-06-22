package gonix

import (
	"context"
	"errors"
	"testing"

	"github.com/sund3RRR/gonix/eval"
)

func newEvalTestClient(t *testing.T) *Client {
	t.Helper()

	client, err := NewClient(ClientConfig{})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("Client.Close() error = %v", err)
		}
	})

	return client
}

func TestClientEvalPrimitives(t *testing.T) {
	client := newEvalTestClient(t)

	var integer int64
	if err := client.Eval(context.Background(), "1 + 2", &integer); err != nil {
		t.Fatalf("Client.Eval(int) error = %v", err)
	}
	if integer != 3 {
		t.Fatalf("Client.Eval(int) = %d, want 3", integer)
	}

	var text string
	if err := client.Eval(context.Background(), `"gonix"`, &text); err != nil {
		t.Fatalf("Client.Eval(string) error = %v", err)
	}
	if text != "gonix" {
		t.Fatalf("Client.Eval(string) = %q, want gonix", text)
	}
}

func TestClientEvalCompositeValues(t *testing.T) {
	client := newEvalTestClient(t)

	var record struct {
		Name   string         `nix:"name" validate:"required"`
		Values map[string]int `nix:"values" validate:"required"`
		Items  []string       `nix:"items" validate:"required"`
	}
	if err := client.Eval(
		context.Background(),
		`{ name = "demo"; values = { one = 1; two = 2; }; items = [ "a" "b" ]; }`,
		&record,
	); err != nil {
		t.Fatalf("Client.Eval(composite) error = %v", err)
	}

	if record.Name != "demo" {
		t.Fatalf("record.Name = %q, want demo", record.Name)
	}
	if record.Values["one"] != 1 || record.Values["two"] != 2 {
		t.Fatalf("record.Values = %#v, want one=1 and two=2", record.Values)
	}
	if len(record.Items) != 2 || record.Items[0] != "a" || record.Items[1] != "b" {
		t.Fatalf("record.Items = %#v, want [a b]", record.Items)
	}
}

func TestClientEvalErrors(t *testing.T) {
	client := newEvalTestClient(t)

	t.Run("invalid expression", func(t *testing.T) {
		var out int
		err := client.Eval(context.Background(), "let =", &out)
		var nixErr *Error
		if !errors.As(err, &nixErr) {
			t.Fatalf("Client.Eval(invalid expression) error = %v, want Nix Error", err)
		}
	})

	tests := []struct {
		name string
		out  any
		want error
	}{
		{
			name: "nil",
			out:  nil,
			want: &eval.InvalidUnmarshalError{},
		},
		{
			name: "non-pointer",
			out:  0,
			want: &eval.InvalidUnmarshalError{},
		},
		{
			name: "unsupported",
			out:  new(chan int),
			want: &eval.UnsupportedTypeError{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.Eval(context.Background(), "1", tt.out)
			switch tt.want.(type) {
			case *eval.InvalidUnmarshalError:
				var target *eval.InvalidUnmarshalError
				if !errors.As(err, &target) {
					t.Fatalf("Client.Eval() error = %v, want InvalidUnmarshalError", err)
				}
			case *eval.UnsupportedTypeError:
				var target *eval.UnsupportedTypeError
				if !errors.As(err, &target) {
					t.Fatalf("Client.Eval() error = %v, want UnsupportedTypeError", err)
				}
			}
		})
	}
}

func TestClientEvalLifecycle(t *testing.T) {
	client := newEvalTestClient(t)

	for i := range 100 {
		var out int
		if err := client.Eval(context.Background(), "40 + 2", &out); err != nil {
			t.Fatalf("Client.Eval() call %d error = %v", i, err)
		}
		if out != 42 {
			t.Fatalf("Client.Eval() call %d = %d, want 42", i, out)
		}
	}

	if err := client.Close(); err != nil {
		t.Fatalf("Client.Close() error = %v", err)
	}

	var out int
	if err := client.Eval(context.Background(), "1", &out); !errors.Is(err, ErrClosed) {
		t.Fatalf("Client.Eval() after Close error = %v, want ErrClosed", err)
	}
}
