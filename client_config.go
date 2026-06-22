package gonix

import (
	"maps"
	"sort"
	"strconv"
	"strings"

	"github.com/sund3RRR/gonix/eval"
	"github.com/sund3RRR/gonix/nixcontext"
	"github.com/sund3RRR/gonix/store"
)

const (
	settingAllowImportFromDerivation = "allow-import-from-derivation"
	settingAcceptFlakeConfig         = "accept-flake-config"
	settingPureEval                  = "pure-eval"
	settingCores                     = "cores"
	settingMaxJobs                   = "max-jobs"
	settingSystem                    = "system"
	settingEvalSystem                = "eval-system"
	settingExperimentalFeatures      = "experimental-features"
	settingSubstituters              = "substituters"
	settingTrustedPublicKeys         = "trusted-public-keys"
)

// Verbosity is a Go-native Nix verbosity level.
type Verbosity = nixcontext.Verbosity

const (
	// VerbosityDefault leaves the current Nix verbosity unchanged.
	VerbosityDefault = nixcontext.VerbosityDefault
	// VerbosityError shows only errors.
	VerbosityError = nixcontext.VerbosityError
	// VerbosityWarn shows warnings and errors.
	VerbosityWarn = nixcontext.VerbosityWarn
	// VerbosityNotice shows notices, warnings, and errors.
	VerbosityNotice = nixcontext.VerbosityNotice
	// VerbosityInfo shows informational messages.
	VerbosityInfo = nixcontext.VerbosityInfo
	// VerbosityTalkative shows talkative Nix logs.
	VerbosityTalkative = nixcontext.VerbosityTalkative
	// VerbosityChatty shows chatty Nix logs.
	VerbosityChatty = nixcontext.VerbosityChatty
	// VerbosityDebug shows debug Nix logs.
	VerbosityDebug = nixcontext.VerbosityDebug
	// VerbosityVomit shows the most verbose Nix logs.
	VerbosityVomit = nixcontext.VerbosityVomit
)

// LogFormat is a Go-native Nix log format.
type LogFormat = nixcontext.LogFormat

const (
	// LogFormatRaw writes raw log messages.
	LogFormatRaw = nixcontext.LogFormatRaw
	// LogFormatRawWithLogs writes raw log messages including log records.
	LogFormatRawWithLogs = nixcontext.LogFormatRawWithLogs
	// LogFormatInternalJSON writes Nix's internal JSON log format.
	LogFormatInternalJSON = nixcontext.LogFormatInternalJSON
	// LogFormatBar writes progress-bar formatted logs.
	LogFormatBar = nixcontext.LogFormatBar
	// LogFormatBarWithLogs writes progress-bar formatted logs including log records.
	LogFormatBarWithLogs = nixcontext.LogFormatBarWithLogs
)

// ExperimentalFeature is a Go-native Nix experimental feature name.
//
// The type is open: use any newer Nix feature name before gonix adds a named
// constant.
type ExperimentalFeature = string

const (
	// ExperimentalFeatureNixCommand enables the modern nix command surface.
	ExperimentalFeatureNixCommand ExperimentalFeature = "nix-command"
	// ExperimentalFeatureFlakes enables flake evaluation support.
	ExperimentalFeatureFlakes ExperimentalFeature = "flakes"
	// ExperimentalFeatureFetchTree enables the fetch-tree experimental feature.
	ExperimentalFeatureFetchTree ExperimentalFeature = "fetch-tree"
	// ExperimentalFeatureCADerivations enables content-addressed derivations.
	ExperimentalFeatureCADerivations ExperimentalFeature = "ca-derivations"
)

// ClientConfig configures Client creation.
//
// The zero value preserves Nix defaults except that it enables nix-command and
// flakes, which are required by the Client flake workflow. Zero-value scalar
// fields are treated as unset. Use RawSettings to explicitly set false, zero,
// max-jobs=auto, or another exact Nix setting value.
type ClientConfig struct {
	// LoadConfig loads the user's Nix configuration during context bootstrap.
	LoadConfig bool
	// AllowImportFromDerivation enables import-from-derivation when true.
	AllowImportFromDerivation bool
	// AcceptFlakeConfig accepts settings supplied by flakes when true.
	AcceptFlakeConfig bool
	// PureEval enables pure evaluation when true.
	PureEval bool
	// Cores sets the number of cores exposed to builders when non-zero.
	Cores int
	// MaxJobs sets the maximum number of local build jobs when non-zero.
	MaxJobs int
	// System sets the Nix build system when non-empty.
	System string
	// EvalSystem sets the Nix evaluation system when non-empty.
	EvalSystem string
	// Verbosity sets Nix verbosity when not VerbosityDefault.
	Verbosity Verbosity
	// LogFormat sets the Nix log format when non-empty.
	LogFormat LogFormat
	// LogSinkPath sets the Nix log sink destination.
	LogSinkPath string
	// ExperimentalFeatures replaces the default feature list when non-empty.
	ExperimentalFeatures []string
	// Substituters sets substituter store URLs when non-empty.
	Substituters []string
	// TrustedPublicKeys sets trusted binary cache keys when non-empty.
	TrustedPublicKeys []string
	// Store configures the Client's owned default store.
	Store StoreConfig
	// Eval configures the Client's owned evaluator.
	Eval EvalConfig
	// RawSettings applies exact Nix setting values after typed fields and wins
	// on key conflicts.
	RawSettings map[string]string
}

// StoreConfig configures the store owned by Client.
type StoreConfig struct {
	// URI is the store URI. The empty value uses store.Auto.
	URI string
	// Opts are passed to store.New.
	Opts []store.Option
}

// EvalConfig configures the evaluator owned by Client.
type EvalConfig struct {
	// Opts are passed to eval.New before Client installs its required flake
	// settings integration.
	Opts []eval.Option
}

// Serialize returns the Nix settings represented by c.
//
// Returned settings are detached from c. List fields are deduplicated, sorted,
// and joined with spaces. RawSettings is applied last.
func (c ClientConfig) Serialize() map[string]string {
	settings := make(map[string]string)

	if c.AllowImportFromDerivation {
		settings[settingAllowImportFromDerivation] = strconv.FormatBool(true)
	}
	if c.AcceptFlakeConfig {
		settings[settingAcceptFlakeConfig] = strconv.FormatBool(true)
	}
	if c.PureEval {
		settings[settingPureEval] = strconv.FormatBool(true)
	}
	if c.Cores != 0 {
		settings[settingCores] = strconv.Itoa(c.Cores)
	}
	if c.MaxJobs != 0 {
		settings[settingMaxJobs] = strconv.Itoa(c.MaxJobs)
	}
	if c.System != "" {
		settings[settingSystem] = c.System
	}
	if c.EvalSystem != "" {
		settings[settingEvalSystem] = c.EvalSystem
	}
	if len(c.Substituters) != 0 {
		settings[settingSubstituters] = formatSettingList(c.Substituters)
	}
	if len(c.TrustedPublicKeys) != 0 {
		settings[settingTrustedPublicKeys] = formatSettingList(c.TrustedPublicKeys)
	}

	_, rawFeaturesSet := c.RawSettings[settingExperimentalFeatures]
	switch {
	case len(c.ExperimentalFeatures) != 0:
		settings[settingExperimentalFeatures] = formatSettingList(c.ExperimentalFeatures)
	case !rawFeaturesSet:
		settings[settingExperimentalFeatures] = formatSettingList([]string{
			ExperimentalFeatureNixCommand,
			ExperimentalFeatureFlakes,
		})
	}

	maps.Copy(settings, c.RawSettings)
	return settings
}

func formatSettingList(values []string) string {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		for _, field := range strings.Fields(value) {
			set[field] = struct{}{}
		}
	}

	sorted := make([]string, 0, len(set))
	for value := range set {
		sorted = append(sorted, value)
	}
	sort.Strings(sorted)

	return strings.Join(sorted, " ")
}
