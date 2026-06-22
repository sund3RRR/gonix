#ifndef NIX_GO_MAIN_H
#define NIX_GO_MAIN_H

#include "nix_go_util.h"
#include "nix_api_main.h"

#ifdef __cplusplus
extern "C" {
#endif

nix_err go_nix_init_plugins(nix_c_context *ctx);
nix_err go_nix_set_log_format(nix_c_context *ctx, const char *format);

/*
 * Install a process-global logger that copies Nix events to an append-only
 * file or an existing Unix socket as unprefixed internal-json records.
 *
 * The installation lasts for the lifetime of the process. Additional calls
 * add additional sinks. Configure the main log format before installing sinks.
 */
nix_err go_nix_log_sink_install(
    nix_c_context *ctx,
    const char *destination
);

#ifdef __cplusplus
}
#endif

#endif
