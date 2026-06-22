#include "nix_go_util.h"

#include <stdlib.h>
#include <string.h>

typedef struct go_nix_string_capture {
    char *value;
    int failed;
} go_nix_string_capture;

static char *go_nix_copy_string(const char *s, unsigned int n)
{
    char *out = (char *)malloc((size_t)n + 1);
    if (out == NULL) {
        return NULL;
    }

    memcpy(out, s, (size_t)n);
    out[n] = '\0';
    return out;
}

void go_nix_capture_string(const char *s, unsigned int n, void *userdata)
{
    go_nix_string_capture *capture = (go_nix_string_capture *)userdata;

    capture->value = go_nix_copy_string(s, n);
    if (capture->value == NULL) {
        capture->failed = 1;
    }
}

nix_c_context *go_nix_c_context_create(void)
{
    return nix_c_context_create();
}

void go_nix_c_context_free(nix_c_context *ctx)
{
    nix_c_context_free(ctx);
}

nix_err go_nix_libutil_init(nix_c_context *ctx)
{
    return nix_libutil_init(ctx);
}

char *go_nix_setting_get(nix_c_context *ctx, const char *key)
{
    go_nix_string_capture capture = {0};

    nix_err err = nix_setting_get(ctx, key, go_nix_capture_string, &capture);
    if (err != NIX_OK) {
        free(capture.value);
        return NULL;
    }
    if (capture.failed) {
        nix_set_err_msg(ctx, NIX_ERR_UNKNOWN, "failed to allocate setting value");
        return NULL;
    }

    return capture.value;
}

nix_err go_nix_setting_set(nix_c_context *ctx, const char *key, const char *value)
{
    return nix_setting_set(ctx, key, value);
}

char *go_nix_version_get(void)
{
    const char *version = nix_version_get();
    if (version == NULL) {
        return NULL;
    }

    return go_nix_copy_string(version, (unsigned int)strlen(version));
}

char *go_nix_err_msg(nix_c_context *ctx, const nix_c_context *read_ctx)
{
    unsigned int n = 0;
    const char *msg = nix_err_msg(ctx, read_ctx, &n);
    if (msg == NULL) {
        return NULL;
    }

    return go_nix_copy_string(msg, n);
}

char *go_nix_err_info_msg(nix_c_context *ctx, const nix_c_context *read_ctx)
{
    go_nix_string_capture capture = {0};

    nix_err err = nix_err_info_msg(ctx, read_ctx, go_nix_capture_string, &capture);
    if (err != NIX_OK) {
        free(capture.value);
        return NULL;
    }
    if (capture.failed) {
        nix_set_err_msg(ctx, NIX_ERR_UNKNOWN, "failed to allocate error info message");
        return NULL;
    }

    return capture.value;
}

char *go_nix_err_name(nix_c_context *ctx, const nix_c_context *read_ctx)
{
    go_nix_string_capture capture = {0};

    nix_err err = nix_err_name(ctx, read_ctx, go_nix_capture_string, &capture);
    if (err != NIX_OK) {
        free(capture.value);
        return NULL;
    }
    if (capture.failed) {
        nix_set_err_msg(ctx, NIX_ERR_UNKNOWN, "failed to allocate error name");
        return NULL;
    }

    return capture.value;
}

nix_err go_nix_err_code(const nix_c_context *ctx)
{
    return nix_err_code(ctx);
}

nix_err go_nix_set_err_msg(nix_c_context *ctx, nix_err err, const char *msg)
{
    return nix_set_err_msg(ctx, err, msg);
}

void go_nix_clear_err(nix_c_context *ctx)
{
    nix_clear_err(ctx);
}

nix_err go_nix_set_verbosity(nix_c_context *ctx, nix_verbosity level)
{
    return nix_set_verbosity(ctx, level);
}

void go_nix_string_free(char *s)
{
    free(s);
}
