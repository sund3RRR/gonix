package flake_test

import (
	"encoding/json"
	"testing"

	"github.com/sund3RRR/gonix/flake"
)

func TestLockInfoJSON(t *testing.T) {
	const data = `{
  "version": 7,
  "root": "root",
  "nodes": {
    "root": {
      "inputs": {
        "direct": "dependency",
        "follows": ["dependency"],
        "emptyFollows": []
      }
    },
    "dependency": {
      "flake": false,
      "parent": [],
      "original": {
        "type": "path",
        "path": "./dependency",
        "count": 1,
        "enabled": true
      },
      "locked": {
        "type": "path",
        "narHash": "sha256-example"
      }
    }
  }
}`

	var info flake.LockInfo
	if err := json.Unmarshal([]byte(data), &info); err != nil {
		t.Fatalf("json.Unmarshal(LockInfo) error = %v", err)
	}
	if info.Version != 7 || info.Root != "root" {
		t.Fatalf("LockInfo = %#v, want version 7 and root", info)
	}

	root := info.Nodes["root"]
	if !root.Flake {
		t.Fatal("root.Flake = false, want omitted field to default to true")
	}
	if got, ok := root.Inputs["direct"].GetNode(); !ok || got != "dependency" {
		t.Fatalf("direct input = %q, %t, want dependency, true", got, ok)
	}
	if got, ok := root.Inputs["follows"].GetFollows(); !ok || len(got) != 1 || got[0] != "dependency" {
		t.Fatalf("follows input = %#v, %t, want [dependency], true", got, ok)
	}
	if got, ok := root.Inputs["emptyFollows"].GetFollows(); !ok || got == nil || len(got) != 0 {
		t.Fatalf("empty follows input = %#v, %t, want non-nil empty path, true", got, ok)
	}

	dependency := info.Nodes["dependency"]
	if dependency.Flake {
		t.Fatal("dependency.Flake = true, want explicit false")
	}
	if dependency.Parent == nil || len(dependency.Parent) != 0 {
		t.Fatalf("dependency.Parent = %#v, want present empty path", dependency.Parent)
	}

	var count uint64
	if err := json.Unmarshal(dependency.Original["count"], &count); err != nil {
		t.Fatalf("decode original count: %v", err)
	}
	if count != 1 {
		t.Fatalf("original count = %d, want 1", count)
	}
	var enabled bool
	if err := json.Unmarshal(dependency.Original["enabled"], &enabled); err != nil {
		t.Fatalf("decode original enabled: %v", err)
	}
	if !enabled {
		t.Fatal("original enabled = false, want true")
	}
}

func TestLockInputJSON(t *testing.T) {
	var direct flake.LockInput
	if err := json.Unmarshal([]byte(`"dependency"`), &direct); err != nil {
		t.Fatalf("json.Unmarshal(direct LockInput) error = %v", err)
	}
	if got, ok := direct.GetNode(); !ok || got != "dependency" {
		t.Fatalf("direct.GetNode() = %q, %t, want dependency, true", got, ok)
	}
	if got, ok := direct.GetFollows(); ok || got != nil {
		t.Fatalf("direct.GetFollows() = %#v, %t, want nil, false", got, ok)
	}

	var follows flake.LockInput
	if err := json.Unmarshal([]byte(`["dependency","nested"]`), &follows); err != nil {
		t.Fatalf("json.Unmarshal(follows LockInput) error = %v", err)
	}
	path, ok := follows.GetFollows()
	if !ok || len(path) != 2 || path[0] != "dependency" || path[1] != "nested" {
		t.Fatalf("follows.GetFollows() = %#v, %t", path, ok)
	}
	path[0] = "mutated"
	if fresh, _ := follows.GetFollows(); fresh[0] != "dependency" {
		t.Fatalf("GetFollows() exposed internal path: %#v", fresh)
	}

	for _, input := range []flake.LockInput{direct, follows} {
		encoded, err := json.Marshal(input)
		if err != nil {
			t.Fatalf("json.Marshal(LockInput) error = %v", err)
		}
		var decoded flake.LockInput
		if err := json.Unmarshal(encoded, &decoded); err != nil {
			t.Fatalf("round-trip LockInput error = %v", err)
		}
	}

	var invalid flake.LockInput
	if err := json.Unmarshal([]byte(`42`), &invalid); err == nil {
		t.Fatal("json.Unmarshal(number LockInput) error = nil")
	}
}
