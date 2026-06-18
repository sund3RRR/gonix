package gonix

import (
	"maps"
	"reflect"
	"testing"
)

func TestClientConfigSerialize(t *testing.T) {
	t.Run("zero config enables flakes only", func(t *testing.T) {
		got := (ClientConfig{}).Serialize()
		want := map[string]string{
			settingExperimentalFeatures: "flakes nix-command",
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("Serialize() = %#v, want %#v", got, want)
		}
	})

	t.Run("lists are copied sorted and deduplicated", func(t *testing.T) {
		features := []string{"flakes", "nix-command", "flakes", "fetch-tree nix-command"}
		before := append([]string(nil), features...)
		got := (ClientConfig{
			ExperimentalFeatures: features,
			Substituters:         []string{"https://z.example", "https://a.example", "https://z.example"},
			TrustedPublicKeys:    []string{"z:key", "a:key", "z:key"},
		}).Serialize()

		if got[settingExperimentalFeatures] != "fetch-tree flakes nix-command" {
			t.Fatalf("experimental-features = %q", got[settingExperimentalFeatures])
		}
		if got[settingSubstituters] != "https://a.example https://z.example" {
			t.Fatalf("substituters = %q", got[settingSubstituters])
		}
		if got[settingTrustedPublicKeys] != "a:key z:key" {
			t.Fatalf("trusted-public-keys = %q", got[settingTrustedPublicKeys])
		}
		if !reflect.DeepEqual(features, before) {
			t.Fatalf("Serialize mutated input: got %v, want %v", features, before)
		}
	})

	t.Run("zero scalars omitted and raw settings win", func(t *testing.T) {
		raw := map[string]string{
			settingCores:                "0",
			settingMaxJobs:              "auto",
			settingPureEval:             "false",
			settingExperimentalFeatures: "",
		}
		before := maps.Clone(raw)
		got := (ClientConfig{
			PureEval:    true,
			Cores:       4,
			MaxJobs:     2,
			RawSettings: raw,
		}).Serialize()

		for key, want := range raw {
			if got[key] != want {
				t.Fatalf("Serialize()[%q] = %q, want %q", key, got[key], want)
			}
		}
		if !reflect.DeepEqual(raw, before) {
			t.Fatalf("Serialize mutated RawSettings: got %v, want %v", raw, before)
		}
	})
}
