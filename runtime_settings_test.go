package gonix

import (
	"strings"
	"testing"
)

func TestRuntimeSettings(t *testing.T) {
	tests := []struct {
		name    string
		opts    []Option
		run     func(t *testing.T, r *Runtime)
		wantErr bool
	}{
		{
			name: "with_setting_applies_setting",
			opts: []Option{
				WithSetting("experimental-features", "nix-command flakes"),
			},
			run: func(t *testing.T, r *Runtime) {
				t.Helper()

				got, err := r.Setting("experimental-features")
				if err != nil {
					t.Fatalf("Runtime.Setting() error = %v", err)
				}
				if strings.TrimSpace(got) == "" {
					t.Fatal("Runtime.Setting(experimental-features) returned empty value")
				}
			},
		},
		{
			name: "with_settings_copies_map_and_last_option_wins",
			opts: func() []Option {
				settings := map[string]string{"experimental-features": "nix-command"}
				opts := []Option{
					WithSettings(settings),
					WithSetting("experimental-features", "nix-command flakes"),
				}
				settings["experimental-features"] = "ca-derivations"
				return opts
			}(),
			run: func(t *testing.T, r *Runtime) {
				t.Helper()

				got, err := r.Setting("experimental-features")
				if err != nil {
					t.Fatalf("Runtime.Setting() error = %v", err)
				}
				if !strings.Contains(got, "nix-command") || !strings.Contains(got, "flakes") {
					t.Fatalf("Runtime.Setting(experimental-features) = %q, want nix-command and flakes", got)
				}
				if strings.Contains(got, "ca-derivations") {
					t.Fatalf("Runtime.Setting(experimental-features) = %q, want copied pre-mutation map value", got)
				}
			},
		},
		{
			name: "setters_after_creation",
			run: func(t *testing.T, r *Runtime) {
				t.Helper()

				if err := r.SetSetting("experimental-features", "nix-command flakes"); err != nil {
					t.Fatalf("Runtime.SetSetting() error = %v", err)
				}
				if err := r.SetLogFormat(LogFormatRaw); err != nil {
					t.Fatalf("Runtime.SetLogFormat() error = %v", err)
				}
				if err := r.SetVerbosity(VerbosityWarn); err != nil {
					t.Fatalf("Runtime.SetVerbosity() error = %v", err)
				}
			},
		},
		{
			name: "invalid_log_format",
			run: func(t *testing.T, r *Runtime) {
				t.Helper()

				if err := r.SetLogFormat(LogFormat("go-bindings-test-invalid-log-format")); err == nil {
					t.Fatal("Runtime.SetLogFormat(invalid) error = nil, want error")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := NewRuntime(tt.opts...)
			if tt.wantErr {
				if err == nil {
					t.Fatal("NewRuntime() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("NewRuntime() error = %v", err)
			}
			t.Cleanup(func() {
				if err := r.Close(); err != nil {
					t.Fatalf("Runtime.Close() error = %v", err)
				}
			})

			tt.run(t, r)
		})
	}
}

func TestRuntimeTypedSettingOptions(t *testing.T) {
	tests := []struct {
		name    string
		opts    []Option
		run     func(t *testing.T, r *Runtime)
		wantErr bool
	}{
		{
			name: "experimental_features",
			opts: []Option{
				WithExperimentalFeatures(ExperimentalFeatureNixCommand, ExperimentalFeatureFlakes),
			},
			run: func(t *testing.T, r *Runtime) {
				t.Helper()
				requireSettingContains(t, r, "experimental-features", "nix-command", "flakes")
			},
		},
		{
			name: "experimental_features_accumulate",
			opts: []Option{
				WithExperimentalFeatures(ExperimentalFeatureNixCommand),
				WithExperimentalFeatures(ExperimentalFeatureFlakes),
				WithExperimentalFeatures(ExperimentalFeatureFetchTree),
			},
			run: func(t *testing.T, r *Runtime) {
				t.Helper()
				requireSettingContains(t, r, "experimental-features", "nix-command", "flakes", "fetch-tree")
			},
		},
		{
			name: "numeric_and_system_settings",
			opts: []Option{
				WithCores(2),
				WithMaxJobs(1),
				WithSystem(SystemX8664Linux),
			},
			run: func(t *testing.T, r *Runtime) {
				t.Helper()
				requireSetting(t, r, "cores", "2")
				requireSetting(t, r, "max-jobs", "1")
				requireSetting(t, r, "system", "x86_64-linux")
			},
		},
		{
			name: "max_jobs_auto",
			opts: []Option{
				WithMaxJobsAuto(),
			},
			run: func(t *testing.T, r *Runtime) {
				t.Helper()
				requireSettingNotEmpty(t, r, "max-jobs")
			},
		},
		{
			name: "substituters_and_keys",
			opts: []Option{
				WithSubstituters("https://cache.nixos.org/", "https://example.com/cache"),
				WithTrustedPublicKeys("cache.nixos.org-1:6NCHdD59X431o0gWypbMrAURkbJ16ZPMQFGspcDShjY="),
			},
			run: func(t *testing.T, r *Runtime) {
				t.Helper()
				requireSetting(t, r, "substituters", "https://cache.nixos.org/ https://example.com/cache")
				requireSetting(t, r, "trusted-public-keys", "cache.nixos.org-1:6NCHdD59X431o0gWypbMrAURkbJ16ZPMQFGspcDShjY=")
			},
		},
		{
			name: "list_settings_accumulate_and_dedupe",
			opts: []Option{
				WithSubstituters("https://cache.nixos.org/"),
				WithSubstituters("https://example.com/cache", "https://cache.nixos.org/"),
				WithTrustedPublicKeys("cache.nixos.org-1:6NCHdD59X431o0gWypbMrAURkbJ16ZPMQFGspcDShjY="),
				WithTrustedPublicKeys("cache.nixos.org-1:6NCHdD59X431o0gWypbMrAURkbJ16ZPMQFGspcDShjY="),
			},
			run: func(t *testing.T, r *Runtime) {
				t.Helper()
				requireSetting(t, r, "substituters", "https://cache.nixos.org/ https://example.com/cache")
				requireSetting(t, r, "trusted-public-keys", "cache.nixos.org-1:6NCHdD59X431o0gWypbMrAURkbJ16ZPMQFGspcDShjY=")
			},
		},
		{
			name: "raw_list_setting_accumulates_with_typed_options",
			opts: []Option{
				WithExperimentalFeatures(ExperimentalFeatureNixCommand, ExperimentalFeatureFlakes),
				WithSetting("experimental-features", "ca-derivations"),
			},
			run: func(t *testing.T, r *Runtime) {
				t.Helper()
				requireSettingContains(t, r, "experimental-features", "nix-command", "flakes", "ca-derivations")
			},
		},
		{
			name: "eval_system_is_not_registered_by_current_runtime_init",
			opts: []Option{
				WithEvalSystem(SystemX8664Linux),
			},
			wantErr: true,
		},
		{
			name: "pure_eval_is_not_registered_by_current_runtime_init",
			opts: []Option{
				WithPureEval(true),
			},
			wantErr: true,
		},
		{
			name: "import_from_derivation_is_not_registered_by_current_runtime_init",
			opts: []Option{
				WithImportFromDerivation(false),
			},
			wantErr: true,
		},
		{
			name: "accept_flake_config_is_not_registered_by_current_runtime_init",
			opts: []Option{
				WithExperimentalFeatures(ExperimentalFeatureNixCommand, ExperimentalFeatureFlakes),
				WithAcceptFlakeConfig(true),
			},
			wantErr: true,
		},
		{
			name: "raw_option_overrides_typed",
			opts: []Option{
				WithCores(2),
				WithSetting("cores", "4"),
			},
			run: func(t *testing.T, r *Runtime) {
				t.Helper()
				requireSetting(t, r, "cores", "4")
			},
		},
		{
			name: "typed_option_overrides_raw",
			opts: []Option{
				WithSetting("cores", "4"),
				WithCores(2),
			},
			run: func(t *testing.T, r *Runtime) {
				t.Helper()
				requireSetting(t, r, "cores", "2")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := NewRuntime(tt.opts...)
			if tt.wantErr {
				if err == nil {
					t.Fatal("NewRuntime() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("NewRuntime() error = %v", err)
			}
			t.Cleanup(func() {
				if err := r.Close(); err != nil {
					t.Fatalf("Runtime.Close() error = %v", err)
				}
			})

			tt.run(t, r)
		})
	}
}

func TestRuntimeTypedSettingSetters(t *testing.T) {
	r, err := NewRuntime()
	if err != nil {
		t.Fatalf("NewRuntime() error = %v", err)
	}
	t.Cleanup(func() {
		if err := r.Close(); err != nil {
			t.Fatalf("Runtime.Close() error = %v", err)
		}
	})

	if err := r.SetExperimentalFeatures(ExperimentalFeatureNixCommand, ExperimentalFeatureFlakes); err != nil {
		t.Fatalf("Runtime.SetExperimentalFeatures() error = %v", err)
	}
	requireSettingContains(t, r, "experimental-features", "nix-command", "flakes")

	if err := r.SetCores(2); err != nil {
		t.Fatalf("Runtime.SetCores() error = %v", err)
	}
	requireSetting(t, r, "cores", "2")

	if err := r.SetMaxJobs(1); err != nil {
		t.Fatalf("Runtime.SetMaxJobs() error = %v", err)
	}
	requireSetting(t, r, "max-jobs", "1")

	if err := r.SetMaxJobsAuto(); err != nil {
		t.Fatalf("Runtime.SetMaxJobsAuto() error = %v", err)
	}
	requireSettingNotEmpty(t, r, "max-jobs")

	if err := r.SetSystem(SystemX8664Linux); err != nil {
		t.Fatalf("Runtime.SetSystem() error = %v", err)
	}
	requireSetting(t, r, "system", "x86_64-linux")

	if err := r.SetEvalSystem(SystemX8664Linux); err == nil {
		t.Fatal("Runtime.SetEvalSystem() error = nil, want error")
	}

	if err := r.SetSubstituters("https://cache.nixos.org/", "https://example.com/cache"); err != nil {
		t.Fatalf("Runtime.SetSubstituters() error = %v", err)
	}
	requireSetting(t, r, "substituters", "https://cache.nixos.org/ https://example.com/cache")

	if err := r.SetTrustedPublicKeys("cache.nixos.org-1:6NCHdD59X431o0gWypbMrAURkbJ16ZPMQFGspcDShjY="); err != nil {
		t.Fatalf("Runtime.SetTrustedPublicKeys() error = %v", err)
	}
	requireSetting(t, r, "trusted-public-keys", "cache.nixos.org-1:6NCHdD59X431o0gWypbMrAURkbJ16ZPMQFGspcDShjY=")

	if err := r.SetPureEval(true); err == nil {
		t.Fatal("Runtime.SetPureEval() error = nil, want error")
	}

	if err := r.SetImportFromDerivation(false); err == nil {
		t.Fatal("Runtime.SetImportFromDerivation() error = nil, want error")
	}

	if err := r.SetAcceptFlakeConfig(true); err == nil {
		t.Fatal("Runtime.SetAcceptFlakeConfig() error = nil, want error")
	}
}

func TestRuntimeSettingsMethodsAfterClose(t *testing.T) {
	r, err := NewRuntime()
	if err != nil {
		t.Fatalf("NewRuntime() error = %v", err)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("Runtime.Close() error = %v", err)
	}

	_, err = r.Setting("experimental-features")
	requireRuntimeClosedError(t, err)
	requireRuntimeClosedError(t, r.SetSetting("experimental-features", "nix-command flakes"))
	requireRuntimeClosedError(t, r.SetVerbosity(VerbosityWarn))
	requireRuntimeClosedError(t, r.SetLogFormat(LogFormatRaw))
	requireRuntimeClosedError(t, r.SetExperimentalFeatures(ExperimentalFeatureNixCommand))
	requireRuntimeClosedError(t, r.SetCores(2))
	requireRuntimeClosedError(t, r.SetMaxJobs(1))
	requireRuntimeClosedError(t, r.SetMaxJobsAuto())
	requireRuntimeClosedError(t, r.SetSystem(SystemX8664Linux))
	requireRuntimeClosedError(t, r.SetEvalSystem(SystemX8664Linux))
	requireRuntimeClosedError(t, r.SetSubstituters("https://cache.nixos.org/"))
	requireRuntimeClosedError(t, r.SetTrustedPublicKeys("cache.nixos.org-1:6NCHdD59X431o0gWypbMrAURkbJ16ZPMQFGspcDShjY="))
	requireRuntimeClosedError(t, r.SetPureEval(true))
	requireRuntimeClosedError(t, r.SetImportFromDerivation(false))
	requireRuntimeClosedError(t, r.SetAcceptFlakeConfig(true))
}

func requireSetting(t *testing.T, r *Runtime, key string, want string) {
	t.Helper()

	got, err := r.Setting(key)
	if err != nil {
		t.Fatalf("Runtime.Setting(%q) error = %v", key, err)
	}
	if got != want {
		t.Fatalf("Runtime.Setting(%q) = %q, want %q", key, got, want)
	}
}

func requireSettingContains(t *testing.T, r *Runtime, key string, wants ...string) {
	t.Helper()

	got, err := r.Setting(key)
	if err != nil {
		t.Fatalf("Runtime.Setting(%q) error = %v", key, err)
	}
	for _, want := range wants {
		if !strings.Contains(got, want) {
			t.Fatalf("Runtime.Setting(%q) = %q, want to contain %q", key, got, want)
		}
	}
}

func requireSettingNotEmpty(t *testing.T, r *Runtime, key string) {
	t.Helper()

	got, err := r.Setting(key)
	if err != nil {
		t.Fatalf("Runtime.Setting(%q) error = %v", key, err)
	}
	if strings.TrimSpace(got) == "" {
		t.Fatalf("Runtime.Setting(%q) = %q, want non-empty value", key, got)
	}
}
