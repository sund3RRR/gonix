#include "nix_go_store.h"

#include "nix_api_store_internal.h"
#include "nix_api_util_internal.h"
#include "nix/store/gc-store.hh"
#include "nix/store/local-fs-store.hh"
#include "nix/store/store-cast.hh"

#include <cstdlib>
#include <cstring>
#include <filesystem>
#include <memory>
#include <new>
#include <string>
#include <vector>

struct go_nix_store_root {
    nix::StorePath path;
    std::string link;
};

struct go_nix_store_roots {
    std::vector<go_nix_store_root> items;
};

struct go_nix_store_gc_results {
    std::vector<std::string> paths;
    uint64_t bytes_freed;
};

static char *go_nix_store_copy_cpp_string(const std::string &value)
{
    char *result = static_cast<char *>(std::malloc(value.size() + 1));
    if (result == nullptr) {
        throw std::bad_alloc();
    }

    std::memcpy(result, value.data(), value.size());
    result[value.size()] = '\0';
    return result;
}

static nix::GCAction go_nix_store_gc_action_to_cpp(go_nix_store_gc_action action)
{
    switch (action) {
    case GO_NIX_STORE_GC_RETURN_LIVE:
        return nix::GCAction::gcReturnLive;
    case GO_NIX_STORE_GC_RETURN_DEAD:
        return nix::GCAction::gcReturnDead;
    case GO_NIX_STORE_GC_DELETE_DEAD:
        return nix::GCAction::gcDeleteDead;
    case GO_NIX_STORE_GC_DELETE_SPECIFIC:
        return nix::GCAction::gcDeleteSpecific;
    }

    throw nix::UsageError("invalid garbage collection action");
}

extern "C" nix_err go_nix_store_add_temp_root(
    nix_c_context *context,
    Store *store,
    const StorePath *path
)
{
    nix_clear_err(context);
    try {
        if (store == nullptr || path == nullptr) {
            throw nix::UsageError("store and store path must not be null");
        }

        store->ptr->addTempRoot(path->path);
    }
    NIXC_CATCH_ERRS
}

extern "C" char *go_nix_store_add_permanent_root(
    nix_c_context *context,
    Store *store,
    const StorePath *path,
    const char *gc_root
)
{
    nix_clear_err(context);
    try {
        if (store == nullptr || path == nullptr || gc_root == nullptr) {
            throw nix::UsageError("store, store path, and GC root must not be null");
        }

        auto &local_store = nix::require<nix::LocalFSStore>(*store->ptr);
        auto result = local_store.addPermRoot(path->path, std::filesystem::path(gc_root));
        return go_nix_store_copy_cpp_string(result.string());
    }
    NIXC_CATCH_ERRS_NULL
}

extern "C" go_nix_store_roots *go_nix_store_find_roots(
    nix_c_context *context,
    Store *store,
    bool censor
)
{
    nix_clear_err(context);
    try {
        if (store == nullptr) {
            throw nix::UsageError("store must not be null");
        }

        auto &gc_store = nix::require<nix::GcStore>(*store->ptr);
        auto found = gc_store.findRoots(censor);
        auto result = std::make_unique<go_nix_store_roots>();

        for (const auto &[path, links] : found) {
            for (const auto &link : links) {
                result->items.push_back(go_nix_store_root{path, link});
            }
        }

        return result.release();
    }
    NIXC_CATCH_ERRS_NULL
}

extern "C" size_t go_nix_store_roots_count(const go_nix_store_roots *roots)
{
    return roots == nullptr ? 0 : roots->items.size();
}

extern "C" StorePath *go_nix_store_roots_path_clone(
    const go_nix_store_roots *roots,
    size_t index
)
{
    try {
        if (roots == nullptr || index >= roots->items.size()) {
            return nullptr;
        }

        return new StorePath{roots->items[index].path};
    } catch (...) {
        return nullptr;
    }
}

extern "C" char *go_nix_store_roots_link(
    const go_nix_store_roots *roots,
    size_t index
)
{
    try {
        if (roots == nullptr || index >= roots->items.size()) {
            return nullptr;
        }

        return go_nix_store_copy_cpp_string(roots->items[index].link);
    } catch (...) {
        return nullptr;
    }
}

extern "C" void go_nix_store_roots_free(go_nix_store_roots *roots)
{
    delete roots;
}

extern "C" go_nix_store_gc_results *go_nix_store_collect_garbage(
    nix_c_context *context,
    Store *store,
    go_nix_store_gc_options options
)
{
    nix_clear_err(context);
    try {
        if (store == nullptr) {
            throw nix::UsageError("store must not be null");
        }
        if (options.paths_to_delete.len != 0 && options.paths_to_delete.items == nullptr) {
            throw nix::UsageError("GC paths have non-zero length but no items");
        }

        nix::GCOptions cpp_options;
        cpp_options.action = go_nix_store_gc_action_to_cpp(options.action);
        cpp_options.ignoreLiveness = options.ignore_liveness;
        cpp_options.maxFreed = options.max_freed;

        for (size_t i = 0; i < options.paths_to_delete.len; ++i) {
            auto path = options.paths_to_delete.items[i].path;
            if (path == nullptr) {
                throw nix::UsageError("GC path must not be null");
            }
            cpp_options.pathsToDelete.insert(path->path);
        }

        nix::GCResults cpp_results;
        auto &gc_store = nix::require<nix::GcStore>(*store->ptr);
        gc_store.collectGarbage(cpp_options, cpp_results);

        auto result = std::make_unique<go_nix_store_gc_results>();
        result->paths.assign(cpp_results.paths.begin(), cpp_results.paths.end());
        result->bytes_freed = cpp_results.bytesFreed;
        return result.release();
    }
    NIXC_CATCH_ERRS_NULL
}

extern "C" size_t go_nix_store_gc_results_count(
    const go_nix_store_gc_results *results
)
{
    return results == nullptr ? 0 : results->paths.size();
}

extern "C" char *go_nix_store_gc_results_path(
    const go_nix_store_gc_results *results,
    size_t index
)
{
    try {
        if (results == nullptr || index >= results->paths.size()) {
            return nullptr;
        }

        return go_nix_store_copy_cpp_string(results->paths[index]);
    } catch (...) {
        return nullptr;
    }
}

extern "C" uint64_t go_nix_store_gc_results_bytes_freed(
    const go_nix_store_gc_results *results
)
{
    return results == nullptr ? 0 : results->bytes_freed;
}

extern "C" void go_nix_store_gc_results_free(go_nix_store_gc_results *results)
{
    delete results;
}
