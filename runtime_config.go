package gonix

import (
	"maps"
	"sort"
	"strings"
)

const (
	settingExperimentalFeatures      = "experimental-features"
	settingCores                     = "cores"
	settingMaxJobs                   = "max-jobs"
	settingSystem                    = "system"
	settingEvalSystem                = "eval-system"
	settingSubstituters              = "substituters"
	settingTrustedPublicKeys         = "trusted-public-keys"
	settingPureEval                  = "pure-eval"
	settingAllowImportFromDerivation = "allow-import-from-derivation"
	settingAcceptFlakeConfig         = "accept-flake-config"
)

type runtimeConfig struct {
	loadConfig           bool
	settings             map[string]string
	experimentalFeatures map[string]struct{}
	substituters         map[string]struct{}
	trustedPublicKeys    map[string]struct{}
	verbosity            *Verbosity
	logFormat            *LogFormat
}

func newRuntimeConfig() runtimeConfig {
	return runtimeConfig{
		settings:             make(map[string]string),
		experimentalFeatures: make(map[string]struct{}),
		substituters:         make(map[string]struct{}),
		trustedPublicKeys:    make(map[string]struct{}),
	}
}

func (c *runtimeConfig) setSetting(key, value string) {
	switch key {
	case settingExperimentalFeatures:
		addListSettingValues(c.experimentalFeatures, strings.Fields(value)...)
	case settingSubstituters:
		addListSettingValues(c.substituters, strings.Fields(value)...)
	case settingTrustedPublicKeys:
		addListSettingValues(c.trustedPublicKeys, strings.Fields(value)...)
	default:
		c.settings[key] = value
	}
}

func (c *runtimeConfig) serialize() map[string]string {
	union := make(map[string]string, len(c.settings)+3)
	maps.Copy(union, c.settings)

	union[settingExperimentalFeatures] = formatSettingSet(c.experimentalFeatures)
	union[settingSubstituters] = formatSettingSet(c.substituters)
	union[settingTrustedPublicKeys] = formatSettingSet(c.trustedPublicKeys)

	return union
}

func addListSettingValues(set map[string]struct{}, values ...string) {
	for _, value := range values {
		set[value] = struct{}{}
	}
}
func formatSettingSet(set map[string]struct{}) string {
	values := make([]string, 0, len(set))
	for value := range set {
		values = append(values, value)
	}
	sort.Strings(values)

	return strings.Join(values, " ")
}
