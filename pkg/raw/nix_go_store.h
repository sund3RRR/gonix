#ifndef NIX_GO_STORE_H
#define NIX_GO_STORE_H

#include <stddef.h>
#include <stdbool.h>
#include <stdint.h>

#include "nix_go_util.h"
#include "nix_api_store.h"
#include "nix_api_store/store_path.h"
#include "nix_api_store/derivation.h"

#ifdef __cplusplus
extern "C" {
#endif

typedef struct go_nix_store_param {
    const char *key;
    size_t key_len;
    const char *value;
    size_t value_len;
} go_nix_store_param;

typedef struct go_nix_store_params {
    const go_nix_store_param *items;
    size_t len;
} go_nix_store_params;

typedef struct go_nix_store_path_hash_part {
    unsigned char bytes[20];
} go_nix_store_path_hash_part;

typedef struct go_nix_store_realise_results go_nix_store_realise_results;
typedef struct go_nix_store_path_array go_nix_store_path_array;
typedef struct go_nix_store_roots go_nix_store_roots;
typedef struct go_nix_store_gc_results go_nix_store_gc_results;

typedef enum go_nix_store_gc_action {
    GO_NIX_STORE_GC_RETURN_LIVE = 0,
    GO_NIX_STORE_GC_RETURN_DEAD = 1,
    GO_NIX_STORE_GC_DELETE_DEAD = 2,
    GO_NIX_STORE_GC_DELETE_SPECIFIC = 3
} go_nix_store_gc_action;

typedef struct go_nix_store_path_item {
    const StorePath *path;
} go_nix_store_path_item;

typedef struct go_nix_store_path_list {
    const go_nix_store_path_item *items;
    size_t len;
} go_nix_store_path_list;

typedef struct go_nix_store_gc_options {
    go_nix_store_gc_action action;
    bool ignore_liveness;
    go_nix_store_path_list paths_to_delete;
    uint64_t max_freed;
} go_nix_store_gc_options;

nix_err go_nix_libstore_init(nix_c_context *ctx);
nix_err go_nix_libstore_init_no_load_config(nix_c_context *ctx);

/*
 * Request a process-global Nix interruption. For RemoteStore backends, also
 * shut down active connections so blocking daemon protocol I/O wakes up.
 *
 * ctx must be separate from the context used by the active operation. A
 * remotely interrupted store must be discarded after the active call returns.
 */
nix_err go_nix_store_interrupt(
    nix_c_context *ctx,
    Store *store
);

Store *go_nix_store_open(
    nix_c_context *ctx,
    const char *uri,
    go_nix_store_params params
);

void go_nix_store_free(Store *store);

char *go_nix_store_get_uri(nix_c_context *ctx, Store *store);
char *go_nix_store_get_storedir(nix_c_context *ctx, Store *store);
char *go_nix_store_get_version(nix_c_context *ctx, Store *store);
char *go_nix_store_real_path(nix_c_context *ctx, Store *store, StorePath *path);

StorePath *go_nix_store_parse_path(nix_c_context *ctx, Store *store, const char *path);

StorePath *go_nix_store_path_clone(const StorePath *path);
void go_nix_store_path_free(StorePath *path);
char *go_nix_store_path_name(const StorePath *path);
nix_err go_nix_store_path_hash(
    nix_c_context *ctx,
    const StorePath *path,
    go_nix_store_path_hash_part *hash
);
StorePath *go_nix_store_create_from_parts(
    nix_c_context *ctx,
    const go_nix_store_path_hash_part *hash,
    const char *name,
    size_t name_len
);

bool go_nix_store_is_valid_path(
    nix_c_context *ctx,
    Store *store,
    const StorePath *path
);

nix_err go_nix_store_add_temp_root(
    nix_c_context *ctx,
    Store *store,
    const StorePath *path
);

char *go_nix_store_add_permanent_root(
    nix_c_context *ctx,
    Store *store,
    const StorePath *path,
    const char *gc_root
);

go_nix_store_roots *go_nix_store_find_roots(
    nix_c_context *ctx,
    Store *store,
    bool censor
);
size_t go_nix_store_roots_count(const go_nix_store_roots *roots);
StorePath *go_nix_store_roots_path_clone(
    const go_nix_store_roots *roots,
    size_t index
);
char *go_nix_store_roots_link(
    const go_nix_store_roots *roots,
    size_t index
);
void go_nix_store_roots_free(go_nix_store_roots *roots);

go_nix_store_gc_results *go_nix_store_collect_garbage(
    nix_c_context *ctx,
    Store *store,
    go_nix_store_gc_options options
);
size_t go_nix_store_gc_results_count(const go_nix_store_gc_results *results);
char *go_nix_store_gc_results_path(
    const go_nix_store_gc_results *results,
    size_t index
);
uint64_t go_nix_store_gc_results_bytes_freed(
    const go_nix_store_gc_results *results
);
void go_nix_store_gc_results_free(go_nix_store_gc_results *results);

go_nix_store_realise_results *go_nix_store_realise_to_array(
    nix_c_context *ctx,
    Store *store,
    StorePath *path
);
size_t go_nix_store_realise_results_count(const go_nix_store_realise_results *results);
char *go_nix_store_realise_results_outname(
    const go_nix_store_realise_results *results,
    size_t index
);
StorePath *go_nix_store_realise_results_path_clone(
    const go_nix_store_realise_results *results,
    size_t index
);
void go_nix_store_realise_results_free(go_nix_store_realise_results *results);

nix_derivation *go_nix_derivation_from_json(
    nix_c_context *ctx,
    Store *store,
    const char *json
);

nix_derivation *go_nix_derivation_clone(const nix_derivation *derivation);
void go_nix_derivation_free(nix_derivation *derivation);
char *go_nix_derivation_to_json(nix_c_context *ctx, const nix_derivation *derivation);

StorePath *go_nix_add_derivation(
    nix_c_context *ctx,
    Store *store,
    nix_derivation *derivation
);

nix_err go_nix_store_copy_closure(
    nix_c_context *ctx,
    Store *src_store,
    Store *dst_store,
    StorePath *path
);

go_nix_store_path_array *go_nix_store_get_fs_closure_array(
    nix_c_context *ctx,
    Store *store,
    const StorePath *store_path,
    bool flip_direction,
    bool include_outputs,
    bool include_derivers
);
size_t go_nix_store_path_array_count(const go_nix_store_path_array *paths);
StorePath *go_nix_store_path_array_path_clone(
    const go_nix_store_path_array *paths,
    size_t index
);
void go_nix_store_path_array_free(go_nix_store_path_array *paths);

nix_derivation *go_nix_store_drv_from_store_path(
    nix_c_context *ctx,
    Store *store,
    const StorePath *path
);

StorePath *go_nix_store_query_path_from_hash_part(
    nix_c_context *ctx,
    Store *store,
    const char *hash
);

nix_err go_nix_store_copy_path(
    nix_c_context *ctx,
    Store *src_store,
    Store *dst_store,
    const StorePath *path,
    bool repair,
    bool check_sigs
);

#ifdef __cplusplus
}
#endif

#endif
