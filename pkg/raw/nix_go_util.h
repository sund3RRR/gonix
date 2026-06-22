#ifndef NIX_GO_UTIL_H
#define NIX_GO_UTIL_H

#include <stdbool.h>

#include "nix_api_util.h"

#ifdef __cplusplus
extern "C" {
#endif

nix_c_context *go_nix_c_context_create(void);
void go_nix_c_context_free(nix_c_context *ctx);

nix_err go_nix_libutil_init(nix_c_context *ctx);

char *go_nix_setting_get(nix_c_context *ctx, const char *key);
nix_err go_nix_setting_set(nix_c_context *ctx, const char *key, const char *value);

char *go_nix_version_get(void);

char *go_nix_err_msg(nix_c_context *ctx, const nix_c_context *read_ctx);
char *go_nix_err_info_msg(nix_c_context *ctx, const nix_c_context *read_ctx);
char *go_nix_err_name(nix_c_context *ctx, const nix_c_context *read_ctx);
nix_err go_nix_err_code(const nix_c_context *ctx);
nix_err go_nix_set_err_msg(nix_c_context *ctx, nix_err err, const char *msg);
void go_nix_clear_err(nix_c_context *ctx);

nix_err go_nix_set_verbosity(nix_c_context *ctx, nix_verbosity level);

/*
 * Read and update Nix's process-global logical interrupt state.
 *
 * Requesting interruption does not install signal handlers or raise signals,
 * but it does wake Nix subsystems registered for interrupt callbacks. Callers
 * must serialize cancellable operations across the process. The active native
 * call must return before the flag is cleared or another operation begins.
 */
void go_nix_interrupt_request(void);
void go_nix_interrupt_clear(void);
bool go_nix_interrupt_requested(void);

void go_nix_string_free(char *s);

#ifdef __cplusplus
}
#endif

#endif
