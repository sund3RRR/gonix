#include "nix_go_daemon.h"

#include "nix_api_store_internal.h"
#include "nix_api_util_internal.h"
#include "nix/store/daemon.hh"
#include "nix/util/serialise.hh"

static void go_nix_daemon_validate_fds(int from_fd, int to_fd)
{
    if (from_fd < 0 || to_fd < 0) {
        throw nix::UsageError("daemon connection file descriptors must be non-negative");
    }
}

extern "C" nix_err go_nix_daemon_process_connection_store(
    nix_c_context *context,
    Store *store,
    int from_fd,
    int to_fd,
    bool trusted,
    bool recursive
)
{
    nix_clear_err(context);
    try {
        if (store == nullptr) {
            throw nix::UsageError("store must not be null");
        }
        go_nix_daemon_validate_fds(from_fd, to_fd);

        nix::daemon::processConnection(
            store->ptr,
            nix::FdSource(from_fd),
            nix::FdSink(to_fd),
            trusted ? nix::Trusted : nix::NotTrusted,
            recursive ? nix::daemon::Recursive : nix::daemon::NotRecursive
        );
    }
    NIXC_CATCH_ERRS
}
