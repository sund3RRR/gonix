#include "nix_go_store.h"

#include <stdint.h>
#include <stdlib.h>
#include <string.h>

typedef struct go_nix_string_capture {
    char *value;
    int failed;
} go_nix_string_capture;

typedef struct go_nix_store_realise_result {
    char *outname;
    StorePath *path;
} go_nix_store_realise_result;

struct go_nix_store_realise_results {
    go_nix_store_realise_result *items;
    size_t len;
    size_t cap;
    int failed;
};

struct go_nix_store_path_array {
    StorePath **items;
    size_t len;
    size_t cap;
    int failed;
};

void go_nix_capture_string(const char *s, unsigned int n, void *userdata);

static nix_err go_nix_set_error(nix_c_context *ctx, const char *msg)
{
    if (ctx != NULL) {
        nix_set_err_msg(ctx, NIX_ERR_UNKNOWN, msg);
    }
    return NIX_ERR_UNKNOWN;
}

static char *go_nix_copy_bytes(const char *s, size_t n)
{
    char *out = NULL;

    if (n == SIZE_MAX) {
        return NULL;
    }

    if (s == NULL && n != 0) {
        return NULL;
    }

    out = (char *)malloc(n + 1);
    if (out == NULL) {
        return NULL;
    }

    if (n != 0) {
        memcpy(out, s, n);
    }
    out[n] = '\0';
    return out;
}

static nix_err go_nix_params_pack(
    nix_c_context *ctx,
    go_nix_store_params params,
    const char ****packed
)
{
    const char ***out = NULL;

    *packed = NULL;

    if (params.len == 0) {
        return NIX_OK;
    }

    if (params.items == NULL) {
        return go_nix_set_error(ctx, "store parameters have non-zero length but no items");
    }

    out = (const char ***)calloc(params.len + 1, sizeof(char **));
    if (out == NULL) {
        return go_nix_set_error(ctx, "failed to allocate store parameter array");
    }

    for (size_t i = 0; i < params.len; i++) {
        const char **pair = (const char **)calloc(3, sizeof(char *));
        if (pair == NULL) {
            for (size_t j = 0; j < i; j++) {
                free((void *)out[j]);
            }
            free(out);
            return go_nix_set_error(ctx, "failed to allocate store parameter pair");
        }

        pair[0] = go_nix_copy_bytes(params.items[i].key, params.items[i].key_len);
        pair[1] = go_nix_copy_bytes(params.items[i].value, params.items[i].value_len);
        if (pair[0] == NULL || pair[1] == NULL) {
            free((void *)pair[0]);
            free((void *)pair[1]);
            free((void *)pair);
            for (size_t j = 0; j < i; j++) {
                free((void *)out[j][0]);
                free((void *)out[j][1]);
                free((void *)out[j]);
            }
            free(out);
            return go_nix_set_error(ctx, "failed to allocate store parameter string");
        }
        pair[2] = NULL;

        out[i] = pair;
    }

    out[params.len] = NULL;
    *packed = out;
    return NIX_OK;
}

static void go_nix_params_free(const char ***params)
{
    if (params == NULL) {
        return;
    }

    for (size_t i = 0; params[i] != NULL; i++) {
        free((void *)params[i][0]);
        free((void *)params[i][1]);
        free((void *)params[i]);
    }

    free((void *)params);
}

nix_err go_nix_libstore_init(nix_c_context *ctx)
{
    return nix_libstore_init(ctx);
}

nix_err go_nix_libstore_init_no_load_config(nix_c_context *ctx)
{
    return nix_libstore_init_no_load_config(ctx);
}

Store *go_nix_store_open(
    nix_c_context *ctx,
    const char *uri,
    go_nix_store_params params
)
{
    const char ***packed = NULL;
    nix_err err = go_nix_params_pack(ctx, params, &packed);
    if (err != NIX_OK) {
        return NULL;
    }

    Store *store = nix_store_open(ctx, uri, packed);
    go_nix_params_free(packed);
    return store;
}

void go_nix_store_free(Store *store)
{
    nix_store_free(store);
}

static char *go_nix_get_string(
    nix_c_context *ctx,
    nix_err (*fn)(nix_c_context *, Store *, nix_get_string_callback, void *),
    Store *store
)
{
    go_nix_string_capture capture = {0};

    nix_err err = fn(ctx, store, go_nix_capture_string, &capture);
    if (err != NIX_OK) {
        free(capture.value);
        return NULL;
    }
    if (capture.failed) {
        go_nix_set_error(ctx, "failed to allocate store string");
        return NULL;
    }

    return capture.value;
}

char *go_nix_store_get_uri(nix_c_context *ctx, Store *store)
{
    return go_nix_get_string(ctx, nix_store_get_uri, store);
}

char *go_nix_store_get_storedir(nix_c_context *ctx, Store *store)
{
    return go_nix_get_string(ctx, nix_store_get_storedir, store);
}

char *go_nix_store_get_version(nix_c_context *ctx, Store *store)
{
    return go_nix_get_string(ctx, nix_store_get_version, store);
}

char *go_nix_store_real_path(nix_c_context *ctx, Store *store, StorePath *path)
{
    go_nix_string_capture capture = {0};

    nix_err err = nix_store_real_path(ctx, store, path, go_nix_capture_string, &capture);
    if (err != NIX_OK) {
        free(capture.value);
        return NULL;
    }
    if (capture.failed) {
        go_nix_set_error(ctx, "failed to allocate store real path");
        return NULL;
    }

    return capture.value;
}

StorePath *go_nix_store_parse_path(nix_c_context *ctx, Store *store, const char *path)
{
    return nix_store_parse_path(ctx, store, path);
}

StorePath *go_nix_store_path_clone(const StorePath *path)
{
    return nix_store_path_clone(path);
}

void go_nix_store_path_free(StorePath *path)
{
    nix_store_path_free(path);
}

char *go_nix_store_path_name(const StorePath *path)
{
    go_nix_string_capture capture = {0};

    nix_store_path_name(path, go_nix_capture_string, &capture);
    if (capture.failed) {
        return NULL;
    }
    return capture.value;
}

nix_err go_nix_store_path_hash(
    nix_c_context *ctx,
    const StorePath *path,
    go_nix_store_path_hash_part *hash
)
{
    return nix_store_path_hash(ctx, path, (nix_store_path_hash_part *)hash);
}

StorePath *go_nix_store_create_from_parts(
    nix_c_context *ctx,
    const go_nix_store_path_hash_part *hash,
    const char *name,
    size_t name_len
)
{
    return nix_store_create_from_parts(ctx, (const nix_store_path_hash_part *)hash, name, name_len);
}

bool go_nix_store_is_valid_path(nix_c_context *ctx, Store *store, const StorePath *path)
{
    return nix_store_is_valid_path(ctx, store, path);
}

static int go_nix_store_realise_results_append(
    go_nix_store_realise_results *results,
    const char *outname,
    const StorePath *path
)
{
    go_nix_store_realise_result *items = NULL;
    char *outname_copy = NULL;
    StorePath *path_copy = NULL;
    size_t new_cap = 0;

    if (results->failed) {
        return 0;
    }

    if (results->len == results->cap) {
        new_cap = results->cap == 0 ? 4 : results->cap * 2;
        if (new_cap < results->cap) {
            results->failed = 1;
            return 0;
        }

        items = (go_nix_store_realise_result *)realloc(
            results->items,
            new_cap * sizeof(go_nix_store_realise_result)
        );
        if (items == NULL) {
            results->failed = 1;
            return 0;
        }

        results->items = items;
        results->cap = new_cap;
    }

    outname_copy = go_nix_copy_bytes(outname, outname == NULL ? 0 : strlen(outname));
    path_copy = nix_store_path_clone(path);
    if (outname_copy == NULL || path_copy == NULL) {
        free(outname_copy);
        nix_store_path_free(path_copy);
        results->failed = 1;
        return 0;
    }

    results->items[results->len].outname = outname_copy;
    results->items[results->len].path = path_copy;
    results->len++;
    return 1;
}

static void go_nix_store_realise_collect(
    void *userdata,
    const char *outname,
    const StorePath *out
)
{
    go_nix_store_realise_results *results = (go_nix_store_realise_results *)userdata;
    go_nix_store_realise_results_append(results, outname, out);
}

go_nix_store_realise_results *go_nix_store_realise_to_array(
    nix_c_context *ctx,
    Store *store,
    StorePath *path
)
{
    go_nix_store_realise_results *results =
        (go_nix_store_realise_results *)calloc(1, sizeof(go_nix_store_realise_results));
    if (results == NULL) {
        go_nix_set_error(ctx, "failed to allocate store realisation results");
        return NULL;
    }

    nix_err err = nix_store_realise(ctx, store, path, results, go_nix_store_realise_collect);
    if (err != NIX_OK) {
        go_nix_store_realise_results_free(results);
        return NULL;
    }

    if (results->failed) {
        go_nix_store_realise_results_free(results);
        go_nix_set_error(ctx, "failed to allocate store realisation callback result");
        return NULL;
    }

    return results;
}

size_t go_nix_store_realise_results_count(const go_nix_store_realise_results *results)
{
    return results == NULL ? 0 : results->len;
}

char *go_nix_store_realise_results_outname(
    const go_nix_store_realise_results *results,
    size_t index
)
{
    if (results == NULL || index >= results->len) {
        return NULL;
    }

    return go_nix_copy_bytes(results->items[index].outname, strlen(results->items[index].outname));
}

StorePath *go_nix_store_realise_results_path_clone(
    const go_nix_store_realise_results *results,
    size_t index
)
{
    if (results == NULL || index >= results->len) {
        return NULL;
    }

    return nix_store_path_clone(results->items[index].path);
}

void go_nix_store_realise_results_free(go_nix_store_realise_results *results)
{
    if (results == NULL) {
        return;
    }

    for (size_t i = 0; i < results->len; i++) {
        free(results->items[i].outname);
        nix_store_path_free(results->items[i].path);
    }

    free(results->items);
    free(results);
}

nix_derivation *go_nix_derivation_from_json(
    nix_c_context *ctx,
    Store *store,
    const char *json
)
{
    return nix_derivation_from_json(ctx, store, json);
}

nix_derivation *go_nix_derivation_clone(const nix_derivation *derivation)
{
    return nix_derivation_clone(derivation);
}

void go_nix_derivation_free(nix_derivation *derivation)
{
    nix_derivation_free(derivation);
}

char *go_nix_derivation_to_json(nix_c_context *ctx, const nix_derivation *derivation)
{
    go_nix_string_capture capture = {0};

    nix_err err = nix_derivation_to_json(ctx, derivation, go_nix_capture_string, &capture);
    if (err != NIX_OK) {
        free(capture.value);
        return NULL;
    }
    if (capture.failed) {
        go_nix_set_error(ctx, "failed to allocate derivation JSON");
        return NULL;
    }

    return capture.value;
}

StorePath *go_nix_add_derivation(
    nix_c_context *ctx,
    Store *store,
    nix_derivation *derivation
)
{
    return nix_add_derivation(ctx, store, derivation);
}

nix_err go_nix_store_copy_closure(
    nix_c_context *ctx,
    Store *src_store,
    Store *dst_store,
    StorePath *path
)
{
    return nix_store_copy_closure(ctx, src_store, dst_store, path);
}

static int go_nix_store_path_array_append(
    go_nix_store_path_array *paths,
    const StorePath *path
)
{
    StorePath **items = NULL;
    StorePath *path_copy = NULL;
    size_t new_cap = 0;

    if (paths->failed) {
        return 0;
    }

    if (paths->len == paths->cap) {
        new_cap = paths->cap == 0 ? 8 : paths->cap * 2;
        if (new_cap < paths->cap) {
            paths->failed = 1;
            return 0;
        }

        items = (StorePath **)realloc(paths->items, new_cap * sizeof(StorePath *));
        if (items == NULL) {
            paths->failed = 1;
            return 0;
        }

        paths->items = items;
        paths->cap = new_cap;
    }

    path_copy = nix_store_path_clone(path);
    if (path_copy == NULL) {
        paths->failed = 1;
        return 0;
    }

    paths->items[paths->len] = path_copy;
    paths->len++;
    return 1;
}

static void go_nix_store_get_fs_closure_collect(
    nix_c_context *context,
    void *userdata,
    const StorePath *store_path
)
{
    go_nix_store_path_array *paths = (go_nix_store_path_array *)userdata;
    if (!go_nix_store_path_array_append(paths, store_path)) {
        go_nix_set_error(context, "failed to allocate store closure path");
    }
}

go_nix_store_path_array *go_nix_store_get_fs_closure_array(
    nix_c_context *ctx,
    Store *store,
    const StorePath *store_path,
    bool flip_direction,
    bool include_outputs,
    bool include_derivers
)
{
    go_nix_store_path_array *paths =
        (go_nix_store_path_array *)calloc(1, sizeof(go_nix_store_path_array));
    if (paths == NULL) {
        go_nix_set_error(ctx, "failed to allocate store closure result");
        return NULL;
    }

    nix_err err = nix_store_get_fs_closure(
        ctx,
        store,
        store_path,
        flip_direction,
        include_outputs,
        include_derivers,
        paths,
        go_nix_store_get_fs_closure_collect
    );
    if (err != NIX_OK) {
        go_nix_store_path_array_free(paths);
        return NULL;
    }

    if (paths->failed) {
        go_nix_store_path_array_free(paths);
        go_nix_set_error(ctx, "failed to allocate store closure result");
        return NULL;
    }

    return paths;
}

size_t go_nix_store_path_array_count(const go_nix_store_path_array *paths)
{
    return paths == NULL ? 0 : paths->len;
}

StorePath *go_nix_store_path_array_path_clone(
    const go_nix_store_path_array *paths,
    size_t index
)
{
    if (paths == NULL || index >= paths->len) {
        return NULL;
    }

    return nix_store_path_clone(paths->items[index]);
}

void go_nix_store_path_array_free(go_nix_store_path_array *paths)
{
    if (paths == NULL) {
        return;
    }

    for (size_t i = 0; i < paths->len; i++) {
        nix_store_path_free(paths->items[i]);
    }

    free(paths->items);
    free(paths);
}

nix_derivation *go_nix_store_drv_from_store_path(
    nix_c_context *ctx,
    Store *store,
    const StorePath *path
)
{
    return nix_store_drv_from_store_path(ctx, store, path);
}

StorePath *go_nix_store_query_path_from_hash_part(
    nix_c_context *ctx,
    Store *store,
    const char *hash
)
{
    return nix_store_query_path_from_hash_part(ctx, store, hash);
}

nix_err go_nix_store_copy_path(
    nix_c_context *ctx,
    Store *src_store,
    Store *dst_store,
    const StorePath *path,
    bool repair,
    bool check_sigs
)
{
    return nix_store_copy_path(ctx, src_store, dst_store, path, repair, check_sigs);
}
