# Nix settings

This table lists Nix 2.34.7 settings relevant to `gonix.Runtime`.

Source: local `nix config show --json` from `nix (Nix) 2.34.7`, cross-checked
with the Nix 2.34.7 `nix.conf` manual.

`Runtime.SetSetting` and raw `WithSetting` pass values to Nix as strings. The
`Type` column describes the expected string shape. List-like values are
whitespace-separated unless noted otherwise. Integer settings accept Nix integer
suffixes `K`, `M`, `G`, and `T`.

Defaults that depend on the current machine, platform, Nix installation, or
system capabilities are written as `system host`.

## Full settings table

| Key | Type | Default | Experimental | Aliases | Description |
| --- | --- | --- | --- | --- | --- |
| abort-on-warn | bool | false |  |  | If true, `builtins.warn` throws an error when logging a warning. |
| accept-flake-config | bool | false | flakes |  | Whether to accept Nix configuration settings from a flake without prompting. |
| access-tokens | space list host=token | empty |  |  | Access tokens used for protected GitHub, GitLab, or other token-authenticated locations. |
| allow-dirty | bool | true |  |  | Whether to allow dirty Git or Mercurial trees. |
| allow-dirty-locks | bool | false | flakes |  | Whether to allow dirty inputs, such as dirty Git workdirs, to be locked via their NAR hash. |
| allow-import-from-derivation | bool | true |  |  | Whether evaluation may import from derivations. |
| allow-symlinked-store | bool | false |  |  | If true, Nix stops complaining when the store directory contains symlink components. |
| allow-unsafe-native-code-during-evaluation | bool | false |  |  | Enable builtins that allow executing native code during evaluation. |
| allowed-impure-host-deps | list<string> | empty |  |  | Prefixes derivations may ask to access, primarily on Darwin. |
| allowed-uris | list<string> | empty |  |  | URI prefixes allowed in restricted evaluation mode. |
| allowed-users | list<string> | * |  |  | Users allowed to connect to the Nix daemon. |
| always-allow-substitutes | bool | false |  |  | Ignore derivation `allowSubstitutes` and always try available substituters. |
| auto-allocate-uids | bool | false | auto-allocate-uids |  | Whether to select UIDs for builds automatically instead of using `build-users-group`. |
| auto-optimise-store | bool | false |  |  | Automatically hard-link identical store files to save disk space. |
| bash-prompt | string | empty |  |  | Bash prompt `PS1` in `nix develop` shells. |
| bash-prompt-prefix | string | empty |  |  | Prefix prepended to `PS1` in `nix develop` shells. |
| bash-prompt-suffix | string | empty |  |  | Suffix appended to `PS1` in `nix develop` shells. |
| build-dir | path | null |  |  | Override the `build-dir` store setting for stores that support it. |
| build-hook | list<string> | nix __build-remote |  |  | Path and arguments for the helper program that executes remote builds. |
| build-poll-interval | int seconds | 5 |  |  | How often to poll for build locks. |
| build-users-group | string | empty |  |  | Unix group containing Nix build user accounts. |
| builders | builder specs string | @/etc/nix/machines |  |  | Semicolon- or newline-separated list of remote build machines. |
| builders-use-substitutes | bool | false |  |  | Tell remote build machines to use their own substituters if available. |
| commit-lock-file-summary | string | empty | flakes | commit-lockfile-summary | Commit summary to use when committing changed flake lock files. |
| compress-build-log | bool | true |  | build-compress-log | Compress build logs written under `/nix/var/log/nix/drvs`. |
| connect-timeout | int seconds | 15 |  |  | Timeout for establishing binary cache or substituter connections. |
| cores | int | 0 |  | build-cores | Value of `NIX_BUILD_CORES` passed to builders. `0` means auto-detect. |
| darwin-log-sandbox-violations | bool | false |  |  | Log Darwin sandbox access violations to the system log. |
| debugger-on-trace | bool | false |  |  | Enter debugger on trace-like functions when debugger is enabled. |
| debugger-on-warn | bool | false |  |  | Enter debugger on `builtins.warn` when debugger is enabled. |
| diff-hook | path | null |  |  | Executable used to diff non-identical build results. |
| download-attempts | int | 5 |  |  | Number of times Nix attempts to download a file. |
| download-buffer-size | int bytes | 1048576 |  |  | Internal curl download buffer size. |
| download-speed | int KiB/s | 0 |  |  | Maximum download transfer rate. `0` means no limit. |
| eval-attrset-update-layer-rhs-threshold | int | 16 |  |  | Threshold for using a compact attrset-update representation. |
| eval-cache | bool | true |  |  | Whether to use the flake evaluation cache. |
| eval-profile-file | path | nix.profile |  |  | File where evaluation profile output is saved. |
| eval-profiler | enum string | disabled |  |  | Enables evaluation profiling. |
| eval-profiler-frequency | int Hz | 99 |  |  | Sampling rate for evaluation profilers. |
| eval-system | system string | empty |  |  | Overrides `builtins.currentSystem` when non-empty. |
| experimental-features | list<string> | empty |  |  | Experimental features enabled for this Nix instance. |
| external-builders | list<string> | empty |  |  | Helper programs that execute derivations. |
| extra-platforms | list<string> | system host |  |  | Additional executable system types supported by this machine. |
| fallback | bool | false |  | build-fallback | Fall back to building from source if a binary substitute fails. |
| flake-registry | path\|URI | https://channels.nixos.org/flake-registry.json | flakes |  | Path or URI of the global flake registry. |
| fsync-metadata | bool | true |  |  | Synchronously flush Nix store metadata changes to disk. |
| fsync-store-paths | bool | false |  |  | Call `fsync()` on store paths before registering them. |
| gc-reserved-space | int bytes | 8388608 |  |  | Disk space reserved for the garbage collector. |
| hashed-mirrors | list<string> | empty |  |  | Web servers used by `builtins.fetchurl` to obtain files by hash. |
| http-connections | int | 25 |  | binary-caches-parallel-connections | Maximum parallel TCP connections for downloads and binary caches. |
| http2 | bool | true |  |  | Enable HTTP/2 support. |
| id-count | int | 128 |  |  | Number of UIDs/GIDs to use for dynamic ID allocation. |
| ignore-try | bool | false |  |  | Ignore exceptions inside `tryEval` in debug mode. |
| impersonate-linux-26 | bool | false |  | build-impersonate-linux-26 | Impersonate a Linux 2.6 machine on newer kernels. |
| impure-env | space list name=value | empty | configurable-impure-env |  | Environment variables allowed for impure derivations. |
| json-log-path | path\|socket | null |  |  | File or Unix socket receiving internal JSON log records. |
| keep-build-log | bool | true |  | build-keep-log | Write build logs for derivations. |
| keep-derivations | bool | true |  | gc-keep-derivations | Keep derivations for non-garbage store paths during GC. |
| keep-env-derivations | bool | false |  | env-keep-derivations | Store derivations in Nix user environments. |
| keep-failed | bool | false |  |  | Keep temporary directories of failed builds. |
| keep-going | bool | false |  |  | Keep building derivations when another build fails. |
| keep-outputs | bool | false |  | gc-keep-outputs | Keep outputs of non-garbage derivations during GC. |
| lint-absolute-path-literals | enum ignore\|warn\|error | ignore |  |  | Controls handling of absolute and home path literals. |
| lint-short-path-literals | enum ignore\|warn\|error | ignore |  |  | Controls handling of short relative path literals. |
| lint-url-literals | enum ignore\|warn\|error | ignore |  |  | Controls handling of unquoted URL literals. |
| log-lines | int | 25 |  |  | Number of log tail lines shown when a build fails. |
| max-build-log-size | int bytes | 0 |  | build-max-log-size | Maximum bytes a builder can write to stdout/stderr. `0` means no limit. |
| max-call-depth | int | 10000 |  |  | Maximum Nix function call depth. |
| max-free | int bytes | 9223372036854775807 |  |  | Stop GC after at least this much free space is available. |
| max-jobs | int\|auto | 1 |  | build-max-jobs | Maximum number of local build jobs. `0` disables local builds. |
| max-silent-time | int seconds | 0 |  | build-max-silent-time | Maximum seconds a builder can be silent. `0` means no timeout. |
| max-substitution-jobs | int | 16 |  | substitution-max-jobs | Maximum parallel substitution jobs. |
| min-free | int bytes | 0 |  |  | Trigger GC below this much free space. `0` disables. |
| min-free-check-interval | int seconds | 5 |  |  | Seconds between checking free disk space. |
| nar-buffer-size | int bytes | 33554432 |  |  | Maximum NAR size before spilling to disk. |
| narinfo-cache-meta-ttl | int seconds | 604800 |  |  | TTL for binary cache metadata. |
| narinfo-cache-negative-ttl | int seconds | 3600 |  |  | TTL for negative substituter lookups. |
| narinfo-cache-positive-ttl | int seconds | 2592000 |  |  | TTL for positive substituter lookups. |
| netrc-file | path | /etc/nix/netrc |  |  | Absolute path to a `netrc` file for HTTP authentication. |
| nix-path | list<string> | empty |  |  | Search paths for Nix lookup path resolution. |
| nix-shell-always-looks-for-shell-nix | bool | true |  |  | Make `nix-shell` look for `shell.nix` in the working directory. |
| nix-shell-shebang-arguments-relative-to-script | bool | true |  |  | Resolve relative `nix-shell` shebang arguments relative to the script. |
| plugin-files | list<string> | empty |  |  | Plugin files to load into Nix. |
| post-build-hook | path | empty |  |  | Program executed after each build. |
| pre-build-hook | path | empty |  |  | Program that can set extra derivation-specific settings. |
| preallocate-contents | bool | false |  |  | Preallocate files when writing objects of known size. |
| print-missing | bool | true |  |  | Print paths that need to be built or downloaded. |
| pure-eval | bool | false |  |  | Make evaluation depend only on explicit inputs. |
| require-drop-supplementary-groups | bool | false |  |  | Require dropping supplementary groups during sandboxed builds. |
| require-sigs | bool | true |  |  | Require trusted signatures for non-content-addressed paths added or copied to the store. |
| restrict-eval | bool | false |  |  | Restrict evaluator file and URI access. |
| run-diff-hook | bool | false |  |  | Enable execution of `diff-hook`. |
| sandbox | bool\|relaxed | false |  | build-use-chroot, build-use-sandbox | Enable sandboxed builds. `relaxed` skips sandboxing for fixed-output or no-chroot derivations. |
| sandbox-fallback | bool | true |  |  | Disable sandboxing when the kernel does not allow it. |
| sandbox-paths | space list host[=sandbox] | empty |  | build-chroot-dirs, build-sandbox-paths | Paths bind-mounted into sandbox environments. |
| secret-key-files | list<string> | empty |  |  | Files containing secret signing keys. |
| show-trace | bool | false |  |  | Print stack traces for Nix expression evaluation errors. |
| ssl-cert-file | path | system host |  |  | CA certificate file for HTTPS downloads. |
| stalled-download-timeout | int seconds | 300 |  |  | Timeout for receiving data during downloads. |
| start-id | int | 56930 |  |  | First UID/GID for dynamic ID allocation. |
| store | store URI | auto |  |  | Store URL used for most operations. |
| substitute | bool | true |  | build-use-substitutes | Use binary substitutes if available. |
| substituters | list<string> | https://cache.nixos.org/ |  | binary-caches | Nix store URLs used as substituters. |
| sync-before-registering | bool | false |  |  | Call `sync()` before registering a path as valid. |
| system | system string | system host |  |  | System type of the current Nix installation. |
| system-features | list<string> | system host |  |  | System features supported by this machine. |
| tarball-ttl | int seconds | 3600 |  |  | Seconds a downloaded tarball is considered fresh. |
| timeout | int seconds | 0 |  | build-timeout | Maximum seconds a builder can run. `0` means no timeout. |
| trace-function-calls | bool | false |  |  | Trace every evaluator function call. |
| trace-import-from-derivation | bool | false |  |  | Trace import-from-derivation use. |
| trace-verbose | bool | false |  |  | Enable `builtins.traceVerbose` output. |
| trust-tarballs-from-git-forges | bool | true |  |  | Treat Git forge tarballs with a Git revision as locked. |
| trusted-public-keys | list<string> | cache.nixos.org-1:6NCHdD59X431o0gWypbMrAURkbJ16ZPMQFGspcDShjY= |  | binary-cache-public-keys | Trusted public binary cache signing keys. |
| trusted-substituters | list<string> | empty |  | trusted-binary-caches | Nix store URLs trusted as substituters. |
| trusted-users | list<string> | root |  |  | Users trusted by the Nix daemon. |
| upgrade-nix-store-path-url | path\|URI | https://github.com/NixOS/nixpkgs/raw/master/nixos/modules/installer/tools/nix-fallback-paths.nix |  |  | URL or file containing store paths of the latest Nix release. |
| use-case-hack | bool | true |  |  | macOS hack for file name case collisions. |
| use-registries | bool | true | flakes |  | Use flake registries to resolve flake references. |
| use-sqlite-wal | bool | true |  |  | Make SQLite use WAL mode. |
| use-xdg-base-directories | bool | false |  |  | Use XDG base directory locations for Nix files under `$HOME`. |
| user-agent-suffix | string | empty |  |  | Suffix appended to HTTP user agent. |
| warn-dirty | bool | true |  |  | Warn about dirty Git or Mercurial trees. |
| warn-large-path-threshold | int bytes | 0 |  |  | Warn when copying a path larger than this many bytes to the Nix store. |
| warn-short-path-literals | bool | false |  |  | Deprecated. |
