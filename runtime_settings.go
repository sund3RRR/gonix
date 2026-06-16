package gonix

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/sund3RRR/gonix/internal/status"
	"github.com/sund3RRR/gonix/internal/utils"
	nix "github.com/sund3RRR/nix-go-bindings"
)

// Setting returns the current value of a Nix setting.
func (r *Runtime) Setting(key string) (string, error) {
	if r.ctx == nil {
		return "", status.ErrClosed
	}

	ptr := nix.SettingGet(r.ctx, key)
	if ptr == nil {
		return "", fmt.Errorf("runtime: get setting %q: %w", key, status.FromContext(r.ctx))
	}

	return utils.TakeCString(ptr), nil
}

// SetSetting sets a Nix setting.
func (r *Runtime) SetSetting(key, value string) error {
	if r.ctx == nil {
		return status.ErrClosed
	}

	if code := nix.SettingSet(r.ctx, key, value); status.ErrorCode(code) != status.ErrorCodeOK {
		return fmt.Errorf("runtime: set setting %q: %w", key, status.FromContext(r.ctx))
	}

	return nil
}

// SetVerbosity sets the Nix verbosity level.
func (r *Runtime) SetVerbosity(level Verbosity) error {
	if r.ctx == nil {
		return status.ErrClosed
	}

	if code := nix.SetVerbosity(r.ctx, nix.NixVerbosity(level)); status.ErrorCode(code) != status.ErrorCodeOK {
		return fmt.Errorf("runtime: set verbosity: %w", status.FromContext(r.ctx))
	}

	return nil
}

// SetLogFormat sets the Nix log format.
func (r *Runtime) SetLogFormat(format LogFormat) error {
	if r.ctx == nil {
		return status.ErrClosed
	}

	if code := nix.SetLogFormat(r.ctx, string(format)); status.ErrorCode(code) != status.ErrorCodeOK {
		return fmt.Errorf("runtime: set log format: %w", status.FromContext(r.ctx))
	}

	return nil
}

// SetExperimentalFeatures sets Nix experimental features.
func (r *Runtime) SetExperimentalFeatures(features ...ExperimentalFeature) error {
	return r.SetSetting(settingExperimentalFeatures, formatExperimentalFeatures(features))
}

// SetCores sets the number of cores exposed to builders through NIX_BUILD_CORES.
func (r *Runtime) SetCores(cores int) error {
	return r.SetSetting(settingCores, strconv.Itoa(cores))
}

// SetMaxJobs sets the maximum number of local build jobs.
func (r *Runtime) SetMaxJobs(maxJobs int) error {
	return r.SetSetting(settingMaxJobs, strconv.Itoa(maxJobs))
}

// SetMaxJobsAuto lets Nix choose the maximum number of local build jobs.
func (r *Runtime) SetMaxJobsAuto() error {
	return r.SetSetting(settingMaxJobs, "auto")
}

// SetSystem sets the Nix build system.
func (r *Runtime) SetSystem(system System) error {
	return r.SetSetting(settingSystem, string(system))
}

// SetEvalSystem sets the Nix evaluation system.
func (r *Runtime) SetEvalSystem(system System) error {
	return r.SetSetting(settingEvalSystem, string(system))
}

// SetSubstituters sets the substituter store URLs.
func (r *Runtime) SetSubstituters(urls ...string) error {
	return r.SetSetting(settingSubstituters, strings.Join(urls, " "))
}

// SetTrustedPublicKeys sets trusted binary cache public keys.
func (r *Runtime) SetTrustedPublicKeys(keys ...string) error {
	return r.SetSetting(settingTrustedPublicKeys, strings.Join(keys, " "))
}

// SetPureEval controls pure evaluation mode.
func (r *Runtime) SetPureEval(pure bool) error {
	return r.SetSetting(settingPureEval, strconv.FormatBool(pure))
}

// SetImportFromDerivation controls whether evaluation may import from derivations.
func (r *Runtime) SetImportFromDerivation(allow bool) error {
	return r.SetSetting(settingAllowImportFromDerivation, strconv.FormatBool(allow))
}

// SetAcceptFlakeConfig controls whether flake-provided Nix settings are accepted.
func (r *Runtime) SetAcceptFlakeConfig(accept bool) error {
	return r.SetSetting(settingAcceptFlakeConfig, strconv.FormatBool(accept))
}

func (r *Runtime) applySettings(settings map[string]string) error {
	if len(settings) == 0 {
		return nil
	}

	if value, ok := settings[settingExperimentalFeatures]; ok {
		if err := r.SetSetting(settingExperimentalFeatures, value); err != nil {
			return err
		}
	}

	keys := make([]string, 0, len(settings))
	for key := range settings {
		if key == settingExperimentalFeatures {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		if err := r.SetSetting(key, settings[key]); err != nil {
			return err
		}
	}

	return nil
}
