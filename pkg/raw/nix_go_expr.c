#include "nix_api_expr.h"
#include "nix_api_value.h"
#include "nix_api_external.h"
#include "nix_go_expr.h"

#include <stdint.h>
#include <stdlib.h>
#include <string.h>

typedef struct go_nix_expr_string_capture {
    char *value;
    int failed;
} go_nix_expr_string_capture;

static nix_err go_nix_expr_set_error(nix_c_context *ctx, const char *msg)
{
    if (ctx != NULL) {
        nix_set_err_msg(ctx, NIX_ERR_UNKNOWN, msg);
    }
    return NIX_ERR_UNKNOWN;
}

static char *go_nix_expr_copy_bytes(const char *s, size_t n)
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

static char *go_nix_expr_copy_string(const char *s, size_t n)
{
    if (s == NULL) {
        return NULL;
    }

    return go_nix_expr_copy_bytes(s, n);
}

static void go_nix_expr_capture_string(const char *s, unsigned int n, void *userdata)
{
    go_nix_expr_string_capture *capture = (go_nix_expr_string_capture *)userdata;
    capture->value = go_nix_expr_copy_string(s, (size_t)n);
    if (capture->value == NULL) {
        capture->failed = 1;
    }
}

static nix_err go_nix_string_array_pack(
    nix_c_context *ctx,
    go_nix_string_array array,
    const char ***packed
)
{
    const char **out = NULL;

    *packed = NULL;

    if (array.len == 0) {
        return NIX_OK;
    }

    if (array.items == NULL) {
        return go_nix_expr_set_error(ctx, "string array has non-zero length but no items");
    }

    out = (const char **)calloc(array.len + 1, sizeof(char *));
    if (out == NULL) {
        return go_nix_expr_set_error(ctx, "failed to allocate string array");
    }

    for (size_t i = 0; i < array.len; i++) {
        out[i] = go_nix_expr_copy_bytes(array.items[i].value, array.items[i].len);
        if (out[i] == NULL) {
            for (size_t j = 0; j < i; j++) {
                free((void *)out[j]);
            }
            free((void *)out);
            return go_nix_expr_set_error(ctx, "failed to allocate string array item");
        }
    }
    out[array.len] = NULL;

    *packed = out;
    return NIX_OK;
}

static void go_nix_string_array_free(const char **array)
{
    if (array == NULL) {
        return;
    }

    for (size_t i = 0; array[i] != NULL; i++) {
        free((void *)array[i]);
    }
    free((void *)array);
}

static nix_err go_nix_value_array_pack(
    nix_c_context *ctx,
    go_nix_value_array array,
    nix_value ***packed
)
{
    nix_value **out = NULL;

    *packed = NULL;

    if (array.len == 0) {
        return NIX_OK;
    }

    if (array.items == NULL) {
        return go_nix_expr_set_error(ctx, "value array has non-zero length but no items");
    }

    out = (nix_value **)calloc(array.len, sizeof(nix_value *));
    if (out == NULL) {
        return go_nix_expr_set_error(ctx, "failed to allocate value array");
    }

    for (size_t i = 0; i < array.len; i++) {
        if (array.items[i].value == NULL) {
            free(out);
            return go_nix_expr_set_error(ctx, "value array item is null");
        }
        out[i] = array.items[i].value;
    }

    *packed = out;
    return NIX_OK;
}

static void go_nix_value_array_free(nix_value **array)
{
    free(array);
}

nix_err go_nix_libexpr_init(nix_c_context *ctx)
{
    return nix_libexpr_init(ctx);
}

nix_err go_nix_expr_eval_from_string(
    nix_c_context *ctx,
    EvalState *state,
    const char *expr,
    const char *path,
    nix_value *value
)
{
    return nix_expr_eval_from_string(ctx, state, expr, path, value);
}

nix_err go_nix_value_call(
    nix_c_context *ctx,
    EvalState *state,
    nix_value *fn,
    nix_value *arg,
    nix_value *value
)
{
    return nix_value_call(ctx, state, fn, arg, value);
}

nix_err go_nix_value_call_multi(
    nix_c_context *ctx,
    EvalState *state,
    nix_value *fn,
    go_nix_value_array args,
    nix_value *value
)
{
    nix_value **packed = NULL;
    nix_err err = go_nix_value_array_pack(ctx, args, &packed);
    if (err != NIX_OK) {
        return err;
    }

    err = nix_value_call_multi(ctx, state, fn, args.len, packed, value);
    go_nix_value_array_free(packed);
    return err;
}

nix_err go_nix_value_force(nix_c_context *ctx, EvalState *state, nix_value *value)
{
    return nix_value_force(ctx, state, value);
}

nix_err go_nix_value_force_deep(nix_c_context *ctx, EvalState *state, nix_value *value)
{
    return nix_value_force_deep(ctx, state, value);
}

nix_eval_state_builder *go_nix_eval_state_builder_new(nix_c_context *ctx, Store *store)
{
    return nix_eval_state_builder_new(ctx, store);
}

nix_err go_nix_eval_state_builder_load(nix_c_context *ctx, nix_eval_state_builder *builder)
{
    return nix_eval_state_builder_load(ctx, builder);
}

nix_err go_nix_eval_state_builder_set_lookup_path(
    nix_c_context *ctx,
    nix_eval_state_builder *builder,
    go_nix_string_array lookup_path
)
{
    const char **packed = NULL;
    nix_err err = go_nix_string_array_pack(ctx, lookup_path, &packed);
    if (err != NIX_OK) {
        return err;
    }

    err = nix_eval_state_builder_set_lookup_path(ctx, builder, packed);
    go_nix_string_array_free(packed);
    return err;
}

EvalState *go_nix_eval_state_build(nix_c_context *ctx, nix_eval_state_builder *builder)
{
    return nix_eval_state_build(ctx, builder);
}

void go_nix_eval_state_builder_free(nix_eval_state_builder *builder)
{
    nix_eval_state_builder_free(builder);
}

EvalState *go_nix_state_create(nix_c_context *ctx, go_nix_string_array lookup_path, Store *store)
{
    const char **packed = NULL;
    nix_err err = go_nix_string_array_pack(ctx, lookup_path, &packed);
    if (err != NIX_OK) {
        return NULL;
    }

    EvalState *state = nix_state_create(ctx, packed, store);
    go_nix_string_array_free(packed);
    return state;
}

void go_nix_state_free(EvalState *state)
{
    nix_state_free(state);
}

nix_err go_nix_gc_incref(nix_c_context *ctx, const void *object)
{
    return nix_gc_incref(ctx, object);
}

nix_err go_nix_gc_decref(nix_c_context *ctx, const void *object)
{
    return nix_gc_decref(ctx, object);
}

void go_nix_gc_now(void)
{
    nix_gc_now();
}

void *go_nix_get_external_value_content(nix_c_context *ctx, ExternalValue *external)
{
    return nix_get_external_value_content(ctx, external);
}

nix_value *go_nix_alloc_value(nix_c_context *ctx, EvalState *state)
{
    return nix_alloc_value(ctx, state);
}

nix_err go_nix_value_incref(nix_c_context *ctx, nix_value *value)
{
    return nix_value_incref(ctx, value);
}

nix_err go_nix_value_decref(nix_c_context *ctx, nix_value *value)
{
    return nix_value_decref(ctx, value);
}

go_nix_value_type go_nix_get_type(nix_c_context *ctx, const nix_value *value)
{
    return (go_nix_value_type)nix_get_type(ctx, value);
}

char *go_nix_get_typename(nix_c_context *ctx, const nix_value *value)
{
    const char *type_name = nix_get_typename(ctx, value);
    char *copy = NULL;
    if (type_name == NULL) {
        return NULL;
    }

    copy = go_nix_expr_copy_string(type_name, strlen(type_name));
    free((void *)type_name);
    if (copy == NULL) {
        go_nix_expr_set_error(ctx, "failed to allocate type name");
    }
    return copy;
}

bool go_nix_get_bool(nix_c_context *ctx, const nix_value *value)
{
    return nix_get_bool(ctx, value);
}

char *go_nix_get_string(nix_c_context *ctx, const nix_value *value)
{
    go_nix_expr_string_capture capture = {0};

    nix_err err = nix_get_string(ctx, value, go_nix_expr_capture_string, &capture);
    if (err != NIX_OK) {
        free(capture.value);
        return NULL;
    }
    if (capture.failed) {
        go_nix_expr_set_error(ctx, "failed to allocate value string");
        return NULL;
    }

    return capture.value;
}

char *go_nix_get_path_string(nix_c_context *ctx, const nix_value *value)
{
    const char *path = nix_get_path_string(ctx, value);
    if (path == NULL) {
        return NULL;
    }

    char *copy = go_nix_expr_copy_string(path, strlen(path));
    if (copy == NULL) {
        go_nix_expr_set_error(ctx, "failed to allocate path string");
    }
    return copy;
}

unsigned int go_nix_get_list_size(nix_c_context *ctx, const nix_value *value)
{
    return nix_get_list_size(ctx, value);
}

unsigned int go_nix_get_attrs_size(nix_c_context *ctx, const nix_value *value)
{
    return nix_get_attrs_size(ctx, value);
}

double go_nix_get_float(nix_c_context *ctx, const nix_value *value)
{
    return nix_get_float(ctx, value);
}

int64_t go_nix_get_int(nix_c_context *ctx, const nix_value *value)
{
    return nix_get_int(ctx, value);
}

ExternalValue *go_nix_get_external(nix_c_context *ctx, nix_value *value)
{
    return nix_get_external(ctx, value);
}

nix_value *go_nix_get_list_byidx(
    nix_c_context *ctx,
    const nix_value *value,
    EvalState *state,
    unsigned int ix
)
{
    return nix_get_list_byidx(ctx, value, state, ix);
}

nix_value *go_nix_get_list_byidx_lazy(
    nix_c_context *ctx,
    const nix_value *value,
    EvalState *state,
    unsigned int ix
)
{
    return nix_get_list_byidx_lazy(ctx, value, state, ix);
}

nix_value *go_nix_get_attr_byname(
    nix_c_context *ctx,
    const nix_value *value,
    EvalState *state,
    const char *name
)
{
    return nix_get_attr_byname(ctx, value, state, name);
}

nix_value *go_nix_get_attr_byname_lazy(
    nix_c_context *ctx,
    const nix_value *value,
    EvalState *state,
    const char *name
)
{
    return nix_get_attr_byname_lazy(ctx, value, state, name);
}

bool go_nix_has_attr_byname(
    nix_c_context *ctx,
    const nix_value *value,
    EvalState *state,
    const char *name
)
{
    return nix_has_attr_byname(ctx, value, state, name);
}

nix_value *go_nix_get_attr_byidx(
    nix_c_context *ctx,
    nix_value *value,
    EvalState *state,
    unsigned int i
)
{
    const char *name = NULL;
    return nix_get_attr_byidx(ctx, value, state, i, &name);
}

nix_value *go_nix_get_attr_byidx_lazy(
    nix_c_context *ctx,
    nix_value *value,
    EvalState *state,
    unsigned int i
)
{
    const char *name = NULL;
    return nix_get_attr_byidx_lazy(ctx, value, state, i, &name);
}

char *go_nix_get_attr_name_byidx(
    nix_c_context *ctx,
    nix_value *value,
    EvalState *state,
    unsigned int i
)
{
    const char *name = nix_get_attr_name_byidx(ctx, value, state, i);
    if (name == NULL) {
        return NULL;
    }

    char *copy = go_nix_expr_copy_string(name, strlen(name));
    if (copy == NULL) {
        go_nix_expr_set_error(ctx, "failed to allocate attribute name");
    }
    return copy;
}

nix_err go_nix_init_bool(nix_c_context *ctx, nix_value *value, bool b)
{
    return nix_init_bool(ctx, value, b);
}

nix_err go_nix_init_string(nix_c_context *ctx, nix_value *value, const char *str)
{
    return nix_init_string(ctx, value, str);
}

nix_err go_nix_init_path_string(nix_c_context *ctx, EvalState *state, nix_value *value, const char *str)
{
    return nix_init_path_string(ctx, state, value, str);
}

nix_err go_nix_init_float(nix_c_context *ctx, nix_value *value, double d)
{
    return nix_init_float(ctx, value, d);
}

nix_err go_nix_init_int(nix_c_context *ctx, nix_value *value, int64_t i)
{
    return nix_init_int(ctx, value, i);
}

nix_err go_nix_init_null(nix_c_context *ctx, nix_value *value)
{
    return nix_init_null(ctx, value);
}

nix_err go_nix_init_apply(nix_c_context *ctx, nix_value *value, nix_value *fn, nix_value *arg)
{
    return nix_init_apply(ctx, value, fn, arg);
}

nix_err go_nix_copy_value(nix_c_context *ctx, nix_value *value, const nix_value *source)
{
    return nix_copy_value(ctx, value, source);
}

ListBuilder *go_nix_make_list_builder(nix_c_context *ctx, EvalState *state, size_t capacity)
{
    return nix_make_list_builder(ctx, state, capacity);
}

nix_err go_nix_list_builder_insert(
    nix_c_context *ctx,
    ListBuilder *list_builder,
    unsigned int index,
    nix_value *value
)
{
    return nix_list_builder_insert(ctx, list_builder, index, value);
}

nix_err go_nix_make_list(nix_c_context *ctx, ListBuilder *list_builder, nix_value *value)
{
    return nix_make_list(ctx, list_builder, value);
}

void go_nix_list_builder_free(ListBuilder *list_builder)
{
    nix_list_builder_free(list_builder);
}

BindingsBuilder *go_nix_make_bindings_builder(nix_c_context *ctx, EvalState *state, size_t capacity)
{
    return nix_make_bindings_builder(ctx, state, capacity);
}

nix_err go_nix_bindings_builder_insert(
    nix_c_context *ctx,
    BindingsBuilder *builder,
    const char *name,
    nix_value *value
)
{
    return nix_bindings_builder_insert(ctx, builder, name, value);
}

nix_err go_nix_make_attrs(nix_c_context *ctx, nix_value *value, BindingsBuilder *builder)
{
    return nix_make_attrs(ctx, value, builder);
}

void go_nix_bindings_builder_free(BindingsBuilder *builder)
{
    nix_bindings_builder_free(builder);
}

nix_realised_string *go_nix_string_realise(
    nix_c_context *ctx,
    EvalState *state,
    nix_value *value,
    bool is_ifd
)
{
    return nix_string_realise(ctx, state, value, is_ifd);
}

char *go_nix_realised_string_get_buffer(nix_realised_string *realised_string)
{
    const char *start = nix_realised_string_get_buffer_start(realised_string);
    size_t size = nix_realised_string_get_buffer_size(realised_string);
    return go_nix_expr_copy_string(start, size);
}

size_t go_nix_realised_string_get_buffer_size(nix_realised_string *realised_string)
{
    return nix_realised_string_get_buffer_size(realised_string);
}

size_t go_nix_realised_string_get_store_path_count(nix_realised_string *realised_string)
{
    return nix_realised_string_get_store_path_count(realised_string);
}

StorePath *go_nix_realised_string_get_store_path(nix_realised_string *realised_string, size_t index)
{
    const StorePath *path = nix_realised_string_get_store_path(realised_string, index);
    if (path == NULL) {
        return NULL;
    }

    return nix_store_path_clone(path);
}

void go_nix_realised_string_free(nix_realised_string *realised_string)
{
    nix_realised_string_free(realised_string);
}
