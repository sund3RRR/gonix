package store

import (
	"reflect"
	"testing"
)

func TestConfig_Params(t *testing.T) {
	type fields struct {
		StoreDir          string
		PathInfoCacheSize int
		Trusted           bool
		Priority          int
		WantMassQuery     bool
		SystemFeatures    []string
		ReadOnly          bool
		BuildDir          string
		LogDir            string
		RealStoreDir      string
		RequireSignatures bool
		RootDir           string
		StateDir          string
	}
	tests := []struct {
		name    string
		fields  fields
		want    map[string]string
		wantErr bool
	}{
		{
			name: "empty_config",
			want: map[string]string{},
		},
		{
			name: "string_fields",
			fields: fields{
				StoreDir:     "/nix/store",
				BuildDir:     "/build",
				LogDir:       "/var/log/nix",
				RealStoreDir: "/real/store",
				RootDir:      "/mnt/root",
				StateDir:     "/var/lib/nix",
			},
			want: map[string]string{
				"store":     "/nix/store",
				"build-dir": "/build",
				"log":       "/var/log/nix",
				"real":      "/real/store",
				"root":      "/mnt/root",
				"state":     "/var/lib/nix",
			},
		},
		{
			name: "numeric_fields",
			fields: fields{
				PathInfoCacheSize: 4096,
				Priority:          -10,
			},
			want: map[string]string{
				"path-info-cache-size": "4096",
				"priority":             "-10",
			},
		},
		{
			name: "boolean_fields",
			fields: fields{
				Trusted:           true,
				WantMassQuery:     true,
				ReadOnly:          true,
				RequireSignatures: true,
			},
			want: map[string]string{
				"trusted":         "true",
				"want-mass-query": "true",
				"read-only":       "true",
				"require-sigs":    "true",
			},
		},
		{
			name: "system_features",
			fields: fields{
				SystemFeatures: []string{"kvm", "big-parallel", "nixos-test"},
			},
			want: map[string]string{
				"system-features": "kvm big-parallel nixos-test",
			},
		},
		{
			name: "zero_values_are_omitted",
			fields: fields{
				StoreDir:          "",
				PathInfoCacheSize: 0,
				Trusted:           false,
				Priority:          0,
				WantMassQuery:     false,
				SystemFeatures:    []string{},
				ReadOnly:          false,
				BuildDir:          "",
				LogDir:            "",
				RealStoreDir:      "",
				RequireSignatures: false,
				RootDir:           "",
				StateDir:          "",
			},
			want: map[string]string{},
		},
		{
			name: "all_fields",
			fields: fields{
				StoreDir:          "/nix/store",
				PathInfoCacheSize: 1024,
				Trusted:           true,
				Priority:          30,
				WantMassQuery:     true,
				SystemFeatures:    []string{"kvm", "benchmark"},
				ReadOnly:          true,
				BuildDir:          "/tmp/nix-build",
				LogDir:            "/var/log/nix",
				RealStoreDir:      "/nix/store",
				RequireSignatures: true,
				RootDir:           "/",
				StateDir:          "/nix/var/nix",
			},
			want: map[string]string{
				"store":                "/nix/store",
				"path-info-cache-size": "1024",
				"trusted":              "true",
				"priority":             "30",
				"want-mass-query":      "true",
				"system-features":      "kvm benchmark",
				"read-only":            "true",
				"build-dir":            "/tmp/nix-build",
				"log":                  "/var/log/nix",
				"real":                 "/nix/store",
				"require-sigs":         "true",
				"root":                 "/",
				"state":                "/nix/var/nix",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := Config{
				StoreDir:          tt.fields.StoreDir,
				PathInfoCacheSize: tt.fields.PathInfoCacheSize,
				Trusted:           tt.fields.Trusted,
				Priority:          tt.fields.Priority,
				WantMassQuery:     tt.fields.WantMassQuery,
				SystemFeatures:    tt.fields.SystemFeatures,
				ReadOnly:          tt.fields.ReadOnly,
				BuildDir:          tt.fields.BuildDir,
				LogDir:            tt.fields.LogDir,
				RealStoreDir:      tt.fields.RealStoreDir,
				RequireSignatures: tt.fields.RequireSignatures,
				RootDir:           tt.fields.RootDir,
				StateDir:          tt.fields.StateDir,
			}
			got, err := c.Params()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Params() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Config.Params() = %v, want %v", got, tt.want)
			}
		})
	}
}
