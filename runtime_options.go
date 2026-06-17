package gonix

import (
	"maps"
	"strconv"
	"strings"
)

// Verbosity is a Go-native Nix verbosity level.
type Verbosity int

const (
	// VerbosityError shows only errors.
	VerbosityError Verbosity = iota

	// VerbosityWarn shows warnings and errors.
	VerbosityWarn

	// VerbosityNotice shows notices, warnings, and errors.
	VerbosityNotice

	// VerbosityInfo shows informational messages.
	VerbosityInfo

	// VerbosityTalkative shows talkative Nix logs.
	VerbosityTalkative

	// VerbosityChatty shows chatty Nix logs.
	VerbosityChatty

	// VerbosityDebug shows debug Nix logs.
	VerbosityDebug

	// VerbosityVomit shows the most verbose Nix logs.
	VerbosityVomit
)

// LogFormat is a Go-native Nix log format.
type LogFormat string

const (
	// LogFormatRaw writes raw log messages.
	LogFormatRaw LogFormat = "raw"

	// LogFormatRawWithLogs writes raw log messages including log records.
	LogFormatRawWithLogs LogFormat = "raw-with-logs"

	// LogFormatInternalJSON writes Nix's internal JSON log format.
	LogFormatInternalJSON LogFormat = "internal-json"

	// LogFormatBar writes progress-bar formatted logs.
	LogFormatBar LogFormat = "bar"

	// LogFormatBarWithLogs writes progress-bar formatted logs including log records.
	LogFormatBarWithLogs LogFormat = "bar-with-logs"
)

// ExperimentalFeature is a Go-native Nix experimental feature name.
//
// The type is open: cast a string to ExperimentalFeature to use newer Nix
// feature names before gonix adds a named constant.
type ExperimentalFeature string

const (
	// ExperimentalFeatureNixCommand enables the modern nix command surface.
	ExperimentalFeatureNixCommand ExperimentalFeature = "nix-command"

	// ExperimentalFeatureFlakes enables flake commands and flake evaluation support.
	ExperimentalFeatureFlakes ExperimentalFeature = "flakes"

	// ExperimentalFeatureFetchTree enables the fetch-tree experimental feature.
	ExperimentalFeatureFetchTree ExperimentalFeature = "fetch-tree"

	// ExperimentalFeatureCADerivations enables content-addressed derivations.
	ExperimentalFeatureCADerivations ExperimentalFeature = "ca-derivations"
)

// RuntimeOption configures Runtime creation.
type RuntimeOption func(*runtimeConfig)

// WithLoadConfig makes NewRuntime load the user's Nix configuration.
//
// Without this option, Runtime uses Nix store initialization without loading
// user configuration for reproducible SDK behavior.
func WithLoadConfig() RuntimeOption {
	return func(c *runtimeConfig) {
		c.loadConfig = true
	}
}

// WithSetting applies a Nix setting during Runtime creation.
//
// If the same scalar setting is configured more than once, the last value wins.
// List-like settings are split on whitespace, accumulated, and deduplicated.
func WithSetting(key, value string) RuntimeOption {
	return func(c *runtimeConfig) {
		c.setSetting(key, value)
	}
}

// WithSettings applies Nix settings during Runtime creation.
//
// The provided map is copied. Scalar settings use last-value-wins semantics;
// list-like settings are split on whitespace, accumulated, and deduplicated.
func WithSettings(settings map[string]string) RuntimeOption {
	copied := maps.Clone(settings)
	return func(c *runtimeConfig) {
		for key, value := range copied {
			c.setSetting(key, value)
		}
	}
}

// WithVerbosity sets the Nix verbosity level during Runtime creation.
func WithVerbosity(level Verbosity) RuntimeOption {
	return func(c *runtimeConfig) {
		c.verbosity = &level
	}
}

// WithLogFormat sets the Nix log format during Runtime creation.
func WithLogFormat(format LogFormat) RuntimeOption {
	return func(c *runtimeConfig) {
		c.logFormat = &format
	}
}

// WithExperimentalFeatures enables Nix experimental features during Runtime creation.
//
// Repeated calls accumulate features and remove duplicates.
func WithExperimentalFeatures(features ...ExperimentalFeature) RuntimeOption {
	return func(c *runtimeConfig) {
		values := make([]string, 0, len(features))
		for _, feature := range features {
			values = append(values, string(feature))
		}
		addListSettingValues(c.experimentalFeatures, values...)
	}
}

// WithSubstituters sets the substituter store URLs.
//
// Repeated calls accumulate URLs and remove duplicates.
func WithSubstituters(urls ...string) RuntimeOption {
	return func(c *runtimeConfig) {
		addListSettingValues(c.substituters, urls...)
	}
}

// WithTrustedPublicKeys sets trusted binary cache public keys.
//
// Repeated calls accumulate keys and remove duplicates.
func WithTrustedPublicKeys(keys ...string) RuntimeOption {
	return func(c *runtimeConfig) {
		addListSettingValues(c.trustedPublicKeys, keys...)
	}
}

// WithCores sets the number of cores exposed to builders through NIX_BUILD_CORES.
func WithCores(cores int) RuntimeOption {
	return func(c *runtimeConfig) {
		c.setSetting(settingCores, strconv.Itoa(cores))
	}
}

// WithMaxJobs sets the maximum number of local build jobs.
func WithMaxJobs(maxJobs int) RuntimeOption {
	return func(c *runtimeConfig) {
		c.setSetting(settingMaxJobs, strconv.Itoa(maxJobs))
	}
}

// WithMaxJobsAuto lets Nix choose the maximum number of local build jobs.
func WithMaxJobsAuto() RuntimeOption {
	return func(c *runtimeConfig) {
		c.setSetting(settingMaxJobs, "auto")
	}
}

// WithSystem sets the Nix build system.
func WithSystem(system string) RuntimeOption {
	return func(c *runtimeConfig) {
		c.setSetting(settingSystem, system)
	}
}

// WithEvalSystem sets the Nix evaluation system.
func WithEvalSystem(system string) RuntimeOption {
	return func(c *runtimeConfig) {
		c.setSetting(settingEvalSystem, system)
	}
}

// WithPureEval controls pure evaluation mode.
func WithPureEval(pure bool) RuntimeOption {
	return func(c *runtimeConfig) {
		c.setSetting(settingPureEval, strconv.FormatBool(pure))
	}
}

// WithImportFromDerivation controls whether evaluation may import from derivations.
func WithImportFromDerivation(allow bool) RuntimeOption {
	return func(c *runtimeConfig) {
		c.setSetting(settingAllowImportFromDerivation, strconv.FormatBool(allow))
	}
}

// WithAcceptFlakeConfig controls whether flake-provided Nix settings are accepted.
func WithAcceptFlakeConfig(accept bool) RuntimeOption {
	return func(c *runtimeConfig) {
		c.setSetting(settingAcceptFlakeConfig, strconv.FormatBool(accept))
	}
}

func formatExperimentalFeatures(features []ExperimentalFeature) string {
	parts := make([]string, 0, len(features))
	for _, feature := range features {
		parts = append(parts, string(feature))
	}

	return strings.Join(parts, " ")
}
