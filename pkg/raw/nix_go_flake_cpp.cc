#include "nix_go_flake.h"

#include "nix_api_fetchers_internal.hh"
#include "nix_api_flake_internal.hh"
#include "nix_api_store_internal.h"
#include "nix_api_util_internal.h"
#include "nix/util/hash.hh"
#include "nix/util/memory-source-accessor.hh"

#include <cstdlib>
#include <cstring>
#include <new>
#include <string>

static char *go_nix_flake_copy_cpp_string(const std::string &value)
{
    char *result = static_cast<char *>(std::malloc(value.size() + 1));
    if (result == nullptr) {
        throw std::bad_alloc();
    }

    std::memcpy(result, value.data(), value.size());
    result[value.size()] = '\0';
    return result;
}

extern "C" char *go_nix_locked_flake_get_lock_json(
    nix_c_context *context,
    nix_locked_flake *locked_flake
)
{
    nix_clear_err(context);
    try {
        auto lock_json = locked_flake->lockedFlake->lockFile.to_string().first;
        return go_nix_flake_copy_cpp_string(lock_json);
    }
    NIXC_CATCH_ERRS_NULL
}

extern "C" nix_err go_nix_flake_lock_flags_set_reference_lock_json(
    nix_c_context *context,
    nix_flake_lock_flags *flags,
    const char *lock_json,
    size_t lock_json_len
)
{
    nix_clear_err(context);
    try {
        if (flags == nullptr) {
            throw nix::UsageError("flake lock flags must not be null");
        }
        if (lock_json == nullptr) {
            throw nix::UsageError("reference lock JSON must not be null");
        }

        auto accessor = nix::make_ref<nix::MemorySourceAccessor>();
        auto path = accessor->addFile(
            nix::CanonPath("/flake.lock"),
            std::string(lock_json, lock_json_len)
        );
        flags->lockFlags->referenceLockFilePath = std::move(path);
    }
    NIXC_CATCH_ERRS
}

extern "C" char *go_nix_locked_flake_get_fingerprint(
    nix_c_context *context,
    Store *store,
    nix_fetchers_settings *fetch_settings,
    nix_locked_flake *locked_flake
)
{
    nix_clear_err(context);
    try {
        auto fingerprint = locked_flake->lockedFlake->getFingerprint(
            *store->ptr,
            *fetch_settings->settings
        );
        if (!fingerprint) {
            return nullptr;
        }

        return go_nix_flake_copy_cpp_string(
            fingerprint->to_string(nix::HashFormat::Base16, false)
        );
    }
    NIXC_CATCH_ERRS_NULL
}
