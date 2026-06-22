#include "nix_go_flake.h"

#include <stdlib.h>
#include <string.h>

struct go_nix_flake_reference_result {
    nix_flake_reference *reference;
    char *fragment;
};

typedef struct go_nix_flake_fragment_capture {
    char *value;
    int failed;
} go_nix_flake_fragment_capture;

static char *go_nix_flake_copy_string(const char *s, unsigned int n)
{
    char *out = NULL;

    if (s == NULL) {
        return NULL;
    }

    out = (char *)malloc((size_t)n + 1);
    if (out == NULL) {
        return NULL;
    }

    memcpy(out, s, (size_t)n);
    out[n] = '\0';
    return out;
}

static void go_nix_flake_capture_fragment(const char *s, unsigned int n, void *userdata)
{
    go_nix_flake_fragment_capture *capture = (go_nix_flake_fragment_capture *)userdata;

    capture->value = go_nix_flake_copy_string(s, n);
    if (capture->value == NULL) {
        capture->failed = 1;
    }
}

nix_flake_settings *go_nix_flake_settings_new(nix_c_context *ctx)
{
    return nix_flake_settings_new(ctx);
}

void go_nix_flake_settings_free(nix_flake_settings *settings)
{
    nix_flake_settings_free(settings);
}

nix_err go_nix_flake_settings_add_to_eval_state_builder(
    nix_c_context *ctx,
    nix_flake_settings *settings,
    nix_eval_state_builder *builder
)
{
    return nix_flake_settings_add_to_eval_state_builder(ctx, settings, builder);
}

nix_flake_reference_parse_flags *go_nix_flake_reference_parse_flags_new(
    nix_c_context *ctx,
    nix_flake_settings *settings
)
{
    return nix_flake_reference_parse_flags_new(ctx, settings);
}

void go_nix_flake_reference_parse_flags_free(nix_flake_reference_parse_flags *flags)
{
    nix_flake_reference_parse_flags_free(flags);
}

nix_err go_nix_flake_reference_parse_flags_set_base_directory(
    nix_c_context *ctx,
    nix_flake_reference_parse_flags *flags,
    const char *base_directory,
    size_t base_directory_len
)
{
    return nix_flake_reference_parse_flags_set_base_directory(
        ctx,
        flags,
        base_directory,
        base_directory_len
    );
}

go_nix_flake_reference_result *go_nix_flake_reference_and_fragment_from_string(
    nix_c_context *ctx,
    nix_fetchers_settings *fetch_settings,
    nix_flake_settings *flake_settings,
    nix_flake_reference_parse_flags *parse_flags,
    const char *str,
    size_t str_len
)
{
    nix_flake_reference *reference = NULL;
    go_nix_flake_fragment_capture capture = {0};
    go_nix_flake_reference_result *result = NULL;
    nix_err err = NIX_OK;

    err = nix_flake_reference_and_fragment_from_string(
        ctx,
        fetch_settings,
        flake_settings,
        parse_flags,
        str,
        str_len,
        &reference,
        go_nix_flake_capture_fragment,
        &capture
    );
    if (err != NIX_OK) {
        free(capture.value);
        return NULL;
    }
    if (capture.failed) {
        nix_flake_reference_free(reference);
        free(capture.value);
        nix_set_err_msg(ctx, NIX_ERR_UNKNOWN, "failed to allocate flake reference fragment");
        return NULL;
    }

    result = (go_nix_flake_reference_result *)calloc(1, sizeof(go_nix_flake_reference_result));
    if (result == NULL) {
        nix_flake_reference_free(reference);
        free(capture.value);
        nix_set_err_msg(ctx, NIX_ERR_UNKNOWN, "failed to allocate flake reference result");
        return NULL;
    }

    result->reference = reference;
    result->fragment = capture.value;
    return result;
}

nix_flake_reference *go_nix_flake_reference_result_take_reference(
    go_nix_flake_reference_result *result
)
{
    nix_flake_reference *reference = NULL;

    if (result == NULL) {
        return NULL;
    }

    reference = result->reference;
    result->reference = NULL;
    return reference;
}

char *go_nix_flake_reference_result_take_fragment(go_nix_flake_reference_result *result)
{
    char *fragment = NULL;

    if (result == NULL) {
        return NULL;
    }

    fragment = result->fragment;
    result->fragment = NULL;
    return fragment;
}

void go_nix_flake_reference_result_free(go_nix_flake_reference_result *result)
{
    if (result == NULL) {
        return;
    }

    nix_flake_reference_free(result->reference);
    free(result->fragment);
    free(result);
}

void go_nix_flake_reference_free(nix_flake_reference *reference)
{
    nix_flake_reference_free(reference);
}

nix_flake_lock_flags *go_nix_flake_lock_flags_new(
    nix_c_context *ctx,
    nix_flake_settings *settings
)
{
    return nix_flake_lock_flags_new(ctx, settings);
}

void go_nix_flake_lock_flags_free(nix_flake_lock_flags *flags)
{
    nix_flake_lock_flags_free(flags);
}

nix_err go_nix_flake_lock_flags_set_mode_check(
    nix_c_context *ctx,
    nix_flake_lock_flags *flags
)
{
    return nix_flake_lock_flags_set_mode_check(ctx, flags);
}

nix_err go_nix_flake_lock_flags_set_mode_virtual(
    nix_c_context *ctx,
    nix_flake_lock_flags *flags
)
{
    return nix_flake_lock_flags_set_mode_virtual(ctx, flags);
}

nix_err go_nix_flake_lock_flags_set_mode_write_as_needed(
    nix_c_context *ctx,
    nix_flake_lock_flags *flags
)
{
    return nix_flake_lock_flags_set_mode_write_as_needed(ctx, flags);
}

nix_err go_nix_flake_lock_flags_add_input_override(
    nix_c_context *ctx,
    nix_flake_lock_flags *flags,
    const char *input_path,
    nix_flake_reference *flake_reference
)
{
    return nix_flake_lock_flags_add_input_override(ctx, flags, input_path, flake_reference);
}

nix_locked_flake *go_nix_flake_lock(
    nix_c_context *ctx,
    nix_fetchers_settings *fetch_settings,
    nix_flake_settings *flake_settings,
    EvalState *eval_state,
    nix_flake_lock_flags *flags,
    nix_flake_reference *flake_reference
)
{
    return nix_flake_lock(
        ctx,
        fetch_settings,
        flake_settings,
        eval_state,
        flags,
        flake_reference
    );
}

void go_nix_locked_flake_free(nix_locked_flake *locked_flake)
{
    nix_locked_flake_free(locked_flake);
}

nix_value *go_nix_locked_flake_get_output_attrs(
    nix_c_context *ctx,
    nix_flake_settings *settings,
    EvalState *eval_state,
    nix_locked_flake *locked_flake
)
{
    return nix_locked_flake_get_output_attrs(ctx, settings, eval_state, locked_flake);
}
