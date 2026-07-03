#ifndef NIX_GO_DAEMON_H
#define NIX_GO_DAEMON_H

#include <stdbool.h>

#include "nix_go_store.h"
#include "nix_go_util.h"

#ifdef __cplusplus
extern "C" {
#endif

nix_err go_nix_daemon_process_connection_store(
    nix_c_context *ctx,
    Store *store,
    int from_fd,
    int to_fd,
    bool trusted,
    bool recursive
);

#ifdef __cplusplus
}
#endif

#endif
