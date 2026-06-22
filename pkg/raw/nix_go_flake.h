#ifndef NIX_GO_FLAKE_H
#define NIX_GO_FLAKE_H

#include <stddef.h>

#include "nix_go_expr.h"
#include "nix_go_fetchers.h"
#include "nix_api_flake.h"

#ifdef __cplusplus
extern "C" {
#endif

typedef struct nix_flake_settings nix_flake_settings;
typedef struct nix_flake_reference_parse_flags nix_flake_reference_parse_flags;
typedef struct nix_flake_reference nix_flake_reference;
typedef struct nix_flake_lock_flags nix_flake_lock_flags;
typedef struct nix_locked_flake nix_locked_flake;
typedef struct go_nix_flake_reference_result go_nix_flake_reference_result;

nix_flake_settings *go_nix_flake_settings_new(nix_c_context *ctx);
void go_nix_flake_settings_free(nix_flake_settings *settings);
nix_err go_nix_flake_settings_add_to_eval_state_builder(
    nix_c_context *ctx,
    nix_flake_settings *settings,
    nix_eval_state_builder *builder
);

nix_flake_reference_parse_flags *go_nix_flake_reference_parse_flags_new(
    nix_c_context *ctx,
    nix_flake_settings *settings
);
void go_nix_flake_reference_parse_flags_free(nix_flake_reference_parse_flags *flags);
nix_err go_nix_flake_reference_parse_flags_set_base_directory(
    nix_c_context *ctx,
    nix_flake_reference_parse_flags *flags,
    const char *base_directory,
    size_t base_directory_len
);

go_nix_flake_reference_result *go_nix_flake_reference_and_fragment_from_string(
    nix_c_context *ctx,
    nix_fetchers_settings *fetch_settings,
    nix_flake_settings *flake_settings,
    nix_flake_reference_parse_flags *parse_flags,
    const char *str,
    size_t str_len
);
nix_flake_reference *go_nix_flake_reference_result_take_reference(
    go_nix_flake_reference_result *result
);
char *go_nix_flake_reference_result_take_fragment(go_nix_flake_reference_result *result);
void go_nix_flake_reference_result_free(go_nix_flake_reference_result *result);
void go_nix_flake_reference_free(nix_flake_reference *reference);

nix_flake_lock_flags *go_nix_flake_lock_flags_new(
    nix_c_context *ctx,
    nix_flake_settings *settings
);
void go_nix_flake_lock_flags_free(nix_flake_lock_flags *flags);
nix_err go_nix_flake_lock_flags_set_mode_check(
    nix_c_context *ctx,
    nix_flake_lock_flags *flags
);
nix_err go_nix_flake_lock_flags_set_mode_virtual(
    nix_c_context *ctx,
    nix_flake_lock_flags *flags
);
nix_err go_nix_flake_lock_flags_set_mode_write_as_needed(
    nix_c_context *ctx,
    nix_flake_lock_flags *flags
);
nix_err go_nix_flake_lock_flags_add_input_override(
    nix_c_context *ctx,
    nix_flake_lock_flags *flags,
    const char *input_path,
    nix_flake_reference *flake_reference
);

nix_locked_flake *go_nix_flake_lock(
    nix_c_context *ctx,
    nix_fetchers_settings *fetch_settings,
    nix_flake_settings *flake_settings,
    EvalState *eval_state,
    nix_flake_lock_flags *flags,
    nix_flake_reference *flake_reference
);
void go_nix_locked_flake_free(nix_locked_flake *locked_flake);
char *go_nix_locked_flake_get_lock_json(
    nix_c_context *ctx,
    nix_locked_flake *locked_flake
);
char *go_nix_locked_flake_get_fingerprint(
    nix_c_context *ctx,
    Store *store,
    nix_fetchers_settings *fetch_settings,
    nix_locked_flake *locked_flake
);
nix_value *go_nix_locked_flake_get_output_attrs(
    nix_c_context *ctx,
    nix_flake_settings *settings,
    EvalState *eval_state,
    nix_locked_flake *locked_flake
);

#ifdef __cplusplus
}
#endif

#endif
