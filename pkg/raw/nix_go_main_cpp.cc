#include "nix_go_main.h"

#include "nix_api_util_internal.h"
#include "nix/util/logging.hh"

#include <cstdlib>
#include <filesystem>
#include <memory>
#include <vector>

extern "C" nix_err go_nix_log_sink_install(
    nix_c_context *context,
    const char *destination
)
{
    nix_clear_err(context);
    try {
        if (destination == nullptr) {
            throw nix::UsageError("JSON log sink destination must not be null");
        }

        std::vector<std::unique_ptr<nix::Logger>> sinks;
        sinks.push_back(nix::makeJSONLogger(
            std::filesystem::path(destination),
            false
        ));

        try {
            nix::logger = nix::makeTeeLogger(
                std::move(nix::logger),
                std::move(sinks)
            );
        } catch (...) {
            /*
             * makeTeeLogger has consumed the process-global logger, so there
             * is no valid logger to restore or use for reporting the failure.
             * This matches Nix's applyJSONLogger failure handling.
             */
            std::abort();
        }
    }
    NIXC_CATCH_ERRS
}
