package raw

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func newTestMainContext(t *testing.T) *NixCContext {
	t.Helper()

	ctx := newTestContext(t)
	if got := LibutilInit(ctx); got != NixOk {
		t.Fatalf("LibutilInit = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
	}

	return ctx
}

func TestNixMainSetLogFormat(t *testing.T) {
	ctx := newTestMainContext(t)

	formats := []string{
		"raw",
		"raw-with-logs",
		"internal-json",
		"bar",
		"bar-with-logs",
	}
	for _, format := range formats {
		if got := SetLogFormat(ctx, format); got != NixOk {
			t.Fatalf("SetLogFormat(%q) = %v, want %v: %s", format, got, NixOk, errMsgString(t, ctx))
		}
	}
}

func TestNixMainSetLogFormatInvalidSetsContextError(t *testing.T) {
	ctx := newTestMainContext(t)

	got := SetLogFormat(ctx, "go-bindings-test-invalid-log-format")
	if got == NixOk {
		t.Fatalf("SetLogFormat(invalid) = %v, want non-OK", got)
	}
	if errCode := ErrCode(ctx); errCode == NixOk {
		t.Fatalf("ErrCode after invalid SetLogFormat = %v, want non-OK", errCode)
	}
	if msg := errMsgString(t, ctx); !strings.Contains(msg, "invalid value") {
		t.Fatalf("ErrMsg after invalid SetLogFormat = %q, want invalid value message", msg)
	}
}

func TestNixMainInitPluginsDefaultEmptyConfig(t *testing.T) {
	ctx := newTestMainContext(t)

	if got := InitPlugins(ctx); got != NixOk {
		t.Fatalf("InitPlugins = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
	}
}

func emitTestTrace(t *testing.T, marker string) {
	t.Helper()

	ctx, _, state := newTestExprState(t)
	value := evalTestExpr(t, ctx, state, "builtins.trace "+strconv.Quote(marker)+" 42")
	if got := GetInt(ctx, value); got != 42 {
		t.Fatalf("GetInt(trace result) = %d, want 42", got)
	}
}

func requireJSONLogMarker(t *testing.T, data []byte, marker string) {
	t.Helper()

	found := false
	lines := 0
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		lines++
		if bytes.HasPrefix(line, []byte("@nix ")) {
			t.Fatalf("JSON log line has internal-json prefix: %q", line)
		}

		var event map[string]any
		if err := json.Unmarshal(line, &event); err != nil {
			t.Fatalf("json.Unmarshal log line: %v\n%s", err, line)
		}
		if message, ok := event["msg"].(string); ok && strings.Contains(message, marker) {
			found = true
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan JSON log: %v", err)
	}
	if lines == 0 {
		t.Fatal("JSON log sink produced no events")
	}
	if !found {
		t.Fatalf("JSON log sink did not contain trace marker %q:\n%s", marker, data)
	}
}

const (
	logSinkHelperScenarioEnv    = "NIX_GO_BINDINGS_LOG_SINK_HELPER_SCENARIO"
	logSinkHelperDestinationEnv = "NIX_GO_BINDINGS_LOG_SINK_HELPER_DESTINATION"
	logSinkHelperMarkerEnv      = "NIX_GO_BINDINGS_LOG_SINK_HELPER_MARKER"
)

func TestNixMainLogSinkHelper(t *testing.T) {
	scenario := os.Getenv(logSinkHelperScenarioEnv)
	if scenario == "" {
		return
	}

	destination := os.Getenv(logSinkHelperDestinationEnv)
	marker := os.Getenv(logSinkHelperMarkerEnv)

	ctx := newTestMainContext(t)
	if got := SetLogFormat(ctx, "raw"); got != NixOk {
		t.Fatalf("SetLogFormat(raw) = %v, want %v: %s", got, NixOk, errMsgString(t, ctx))
	}

	got := LogSinkInstall(ctx, destination)
	if scenario == "invalid" {
		if got == NixOk {
			t.Fatal("LogSinkInstall(invalid destination) = NixOk, want non-OK")
		}
		if ErrCode(ctx) == NixOk {
			t.Fatal("ErrCode after invalid LogSinkInstall = NixOk, want non-OK")
		}
		return
	}
	if got != NixOk {
		t.Fatalf("LogSinkInstall(%q) = %v, want %v: %s", destination, got, NixOk, errMsgString(t, ctx))
	}

	emitTestTrace(t, marker)
}

func logSinkHelperEnv(scenario, destination, marker string) []string {
	replacements := map[string]string{
		logSinkHelperScenarioEnv:    scenario,
		logSinkHelperDestinationEnv: destination,
		logSinkHelperMarkerEnv:      marker,
	}

	env := make([]string, 0, len(os.Environ())+len(replacements))
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if _, replaced := replacements[key]; !replaced {
			env = append(env, entry)
		}
	}
	for key, value := range replacements {
		env = append(env, key+"="+value)
	}
	return env
}

func runLogSinkHelper(t *testing.T, scenario, destination, marker string) []byte {
	t.Helper()

	cmd := exec.Command(
		os.Args[0],
		"-test.run=^TestNixMainLogSinkHelper$",
		"-test.v",
	)
	cmd.Env = logSinkHelperEnv(scenario, destination, marker)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("log sink helper failed: %v\n%s", err, output)
	}
	return output
}

func requireOriginalLoggerMarker(t *testing.T, output []byte, marker string) {
	t.Helper()

	if !bytes.Contains(output, []byte("trace: "+marker)) {
		t.Fatalf("original logger did not receive trace marker %q:\n%s", marker, output)
	}
}

func TestNixMainLogSinkFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nix-events.jsonl")
	const marker = "go-bindings-file-log-sink"
	output := runLogSinkHelper(t, "file", path, marker)
	requireOriginalLoggerMarker(t, output, marker)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile(%q): %v", path, err)
	}
	requireJSONLogMarker(t, data, marker)
}

func TestNixMainLogSinkUnixSocket(t *testing.T) {
	socketDir, err := os.MkdirTemp("/tmp", "gonix-raw-")
	if err != nil {
		t.Fatalf("os.MkdirTemp(/tmp): %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(socketDir); err != nil {
			t.Errorf("os.RemoveAll(%q): %v", socketDir, err)
		}
	})

	socketPath := filepath.Join(socketDir, "log.sock")
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: socketPath, Net: "unix"})
	if err != nil {
		t.Fatalf("net.ListenUnix(%q): %v", socketPath, err)
	}
	t.Cleanup(func() {
		if err := listener.Close(); err != nil {
			t.Errorf("listener.Close: %v", err)
		}
	})
	if err := listener.SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
		t.Fatalf("listener.SetDeadline: %v", err)
	}

	type socketResult struct {
		data []byte
		err  error
	}
	resultCh := make(chan socketResult, 1)
	go func() {
		conn, err := listener.AcceptUnix()
		if err != nil {
			resultCh <- socketResult{err: err}
			return
		}
		defer func() {
			_ = conn.Close()
		}()
		if err := conn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
			resultCh <- socketResult{err: err}
			return
		}
		data, err := io.ReadAll(conn)
		resultCh <- socketResult{data: data, err: err}
	}()

	const marker = "go-bindings-socket-log-sink"
	output := runLogSinkHelper(t, "socket", socketPath, marker)
	requireOriginalLoggerMarker(t, output, marker)

	select {
	case result := <-resultCh:
		if result.err != nil {
			t.Fatalf("read Unix socket log: %v", result.err)
		}
		requireJSONLogMarker(t, result.data, marker)
	case <-time.After(12 * time.Second):
		t.Fatal("timed out waiting for Unix socket log EOF")
	}
}

func TestNixMainLogSinkInvalidDestination(t *testing.T) {
	runLogSinkHelper(t, "invalid", t.TempDir(), "")
}
