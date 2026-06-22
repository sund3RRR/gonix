#include "nix_go_store.h"
#include "nix_go_util.h"

#include "nix_api_store_internal.h"
#include "nix_api_util_internal.h"
#ifndef _WIN32
#include "nix/store/remote-store.hh"
#endif
#include "nix/util/signals.hh"

static void requestInterrupt()
{
#ifndef _WIN32
    nix::unix::triggerInterrupt();
#else
    nix::setInterrupted(true);
#endif
}

extern "C" void go_nix_interrupt_request(void)
{
    requestInterrupt();
}

extern "C" void go_nix_interrupt_clear(void)
{
    nix::setInterrupted(false);
}

extern "C" bool go_nix_interrupt_requested(void)
{
    return nix::getInterrupted();
}

extern "C" nix_err go_nix_store_interrupt(
    nix_c_context *context,
    Store *store
)
{
    if (context == nullptr) {
        return NIX_ERR_UNKNOWN;
    }

    nix_clear_err(context);
    try {
        if (store == nullptr) {
            throw nix::UsageError("store must not be null");
        }

        requestInterrupt();

#ifndef _WIN32
        if (auto remote_store = store->ptr.dynamic_pointer_cast<nix::RemoteStore>()) {
            remote_store->shutdownConnections();
        }
#endif
    }
    NIXC_CATCH_ERRS
}
