#ifndef NIX_GO_EXPR_H
#define NIX_GO_EXPR_H

#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>

#include "nix_go_store.h"

#ifdef __cplusplus
extern "C" {
#endif

typedef struct nix_eval_state_builder nix_eval_state_builder;
typedef struct EvalState EvalState;
typedef struct nix_value nix_value;
typedef struct BindingsBuilder BindingsBuilder;
typedef struct ListBuilder ListBuilder;
typedef struct PrimOp PrimOp;
typedef struct ExternalValue ExternalValue;
typedef struct nix_realised_string nix_realised_string;

typedef enum go_nix_value_type {
    GO_NIX_TYPE_THUNK = 0,
    GO_NIX_TYPE_INT = 1,
    GO_NIX_TYPE_FLOAT = 2,
    GO_NIX_TYPE_BOOL = 3,
    GO_NIX_TYPE_STRING = 4,
    GO_NIX_TYPE_PATH = 5,
    GO_NIX_TYPE_NULL = 6,
    GO_NIX_TYPE_ATTRS = 7,
    GO_NIX_TYPE_LIST = 8,
    GO_NIX_TYPE_FUNCTION = 9,
    GO_NIX_TYPE_EXTERNAL = 10,
    GO_NIX_TYPE_FAILED = 11
} go_nix_value_type;

typedef struct go_nix_string_item {
    const char *value;
    size_t len;
} go_nix_string_item;

typedef struct go_nix_string_array {
    const go_nix_string_item *items;
    size_t len;
} go_nix_string_array;

typedef struct go_nix_value_item {
    nix_value *value;
} go_nix_value_item;

typedef struct go_nix_value_array {
    const go_nix_value_item *items;
    size_t len;
} go_nix_value_array;

nix_err go_nix_libexpr_init(nix_c_context *ctx);

nix_err go_nix_expr_eval_from_string(
    nix_c_context *ctx,
    EvalState *state,
    const char *expr,
    const char *path,
    nix_value *value
);
nix_err go_nix_value_call(
    nix_c_context *ctx,
    EvalState *state,
    nix_value *fn,
    nix_value *arg,
    nix_value *value
);
nix_err go_nix_value_call_multi(
    nix_c_context *ctx,
    EvalState *state,
    nix_value *fn,
    go_nix_value_array args,
    nix_value *value
);
nix_err go_nix_value_force(nix_c_context *ctx, EvalState *state, nix_value *value);
nix_err go_nix_value_force_deep(nix_c_context *ctx, EvalState *state, nix_value *value);

nix_eval_state_builder *go_nix_eval_state_builder_new(nix_c_context *ctx, Store *store);
nix_err go_nix_eval_state_builder_load(nix_c_context *ctx, nix_eval_state_builder *builder);
nix_err go_nix_eval_state_builder_set_lookup_path(
    nix_c_context *ctx,
    nix_eval_state_builder *builder,
    go_nix_string_array lookup_path
);
EvalState *go_nix_eval_state_build(nix_c_context *ctx, nix_eval_state_builder *builder);
void go_nix_eval_state_builder_free(nix_eval_state_builder *builder);
EvalState *go_nix_state_create(nix_c_context *ctx, go_nix_string_array lookup_path, Store *store);
void go_nix_state_free(EvalState *state);

nix_err go_nix_gc_incref(nix_c_context *ctx, const void *object);
nix_err go_nix_gc_decref(nix_c_context *ctx, const void *object);
void go_nix_gc_now(void);
void *go_nix_get_external_value_content(nix_c_context *ctx, ExternalValue *external);

nix_value *go_nix_alloc_value(nix_c_context *ctx, EvalState *state);
nix_err go_nix_value_incref(nix_c_context *ctx, nix_value *value);
nix_err go_nix_value_decref(nix_c_context *ctx, nix_value *value);
go_nix_value_type go_nix_get_type(nix_c_context *ctx, const nix_value *value);
char *go_nix_get_typename(nix_c_context *ctx, const nix_value *value);
bool go_nix_get_bool(nix_c_context *ctx, const nix_value *value);
char *go_nix_get_string(nix_c_context *ctx, const nix_value *value);
char *go_nix_get_path_string(nix_c_context *ctx, const nix_value *value);
unsigned int go_nix_get_list_size(nix_c_context *ctx, const nix_value *value);
unsigned int go_nix_get_attrs_size(nix_c_context *ctx, const nix_value *value);
double go_nix_get_float(nix_c_context *ctx, const nix_value *value);
int64_t go_nix_get_int(nix_c_context *ctx, const nix_value *value);
ExternalValue *go_nix_get_external(nix_c_context *ctx, nix_value *value);
nix_value *go_nix_get_list_byidx(
    nix_c_context *ctx,
    const nix_value *value,
    EvalState *state,
    unsigned int ix
);
nix_value *go_nix_get_list_byidx_lazy(
    nix_c_context *ctx,
    const nix_value *value,
    EvalState *state,
    unsigned int ix
);
nix_value *go_nix_get_attr_byname(
    nix_c_context *ctx,
    const nix_value *value,
    EvalState *state,
    const char *name
);
nix_value *go_nix_get_attr_byname_lazy(
    nix_c_context *ctx,
    const nix_value *value,
    EvalState *state,
    const char *name
);
bool go_nix_has_attr_byname(
    nix_c_context *ctx,
    const nix_value *value,
    EvalState *state,
    const char *name
);
nix_value *go_nix_get_attr_byidx(
    nix_c_context *ctx,
    nix_value *value,
    EvalState *state,
    unsigned int i
);
nix_value *go_nix_get_attr_byidx_lazy(
    nix_c_context *ctx,
    nix_value *value,
    EvalState *state,
    unsigned int i
);
char *go_nix_get_attr_name_byidx(
    nix_c_context *ctx,
    nix_value *value,
    EvalState *state,
    unsigned int i
);

nix_err go_nix_init_bool(nix_c_context *ctx, nix_value *value, bool b);
nix_err go_nix_init_string(nix_c_context *ctx, nix_value *value, const char *str);
nix_err go_nix_init_path_string(nix_c_context *ctx, EvalState *state, nix_value *value, const char *str);
nix_err go_nix_init_float(nix_c_context *ctx, nix_value *value, double d);
nix_err go_nix_init_int(nix_c_context *ctx, nix_value *value, int64_t i);
nix_err go_nix_init_null(nix_c_context *ctx, nix_value *value);
nix_err go_nix_init_apply(nix_c_context *ctx, nix_value *value, nix_value *fn, nix_value *arg);
nix_err go_nix_copy_value(nix_c_context *ctx, nix_value *value, const nix_value *source);

ListBuilder *go_nix_make_list_builder(nix_c_context *ctx, EvalState *state, size_t capacity);
nix_err go_nix_list_builder_insert(
    nix_c_context *ctx,
    ListBuilder *list_builder,
    unsigned int index,
    nix_value *value
);
nix_err go_nix_make_list(nix_c_context *ctx, ListBuilder *list_builder, nix_value *value);
void go_nix_list_builder_free(ListBuilder *list_builder);

BindingsBuilder *go_nix_make_bindings_builder(nix_c_context *ctx, EvalState *state, size_t capacity);
nix_err go_nix_bindings_builder_insert(
    nix_c_context *ctx,
    BindingsBuilder *builder,
    const char *name,
    nix_value *value
);
nix_err go_nix_make_attrs(nix_c_context *ctx, nix_value *value, BindingsBuilder *builder);
void go_nix_bindings_builder_free(BindingsBuilder *builder);

nix_realised_string *go_nix_string_realise(
    nix_c_context *ctx,
    EvalState *state,
    nix_value *value,
    bool is_ifd
);
char *go_nix_realised_string_get_buffer(nix_realised_string *realised_string);
size_t go_nix_realised_string_get_buffer_size(nix_realised_string *realised_string);
size_t go_nix_realised_string_get_store_path_count(nix_realised_string *realised_string);
StorePath *go_nix_realised_string_get_store_path(nix_realised_string *realised_string, size_t index);
void go_nix_realised_string_free(nix_realised_string *realised_string);

#ifdef __cplusplus
}
#endif

#endif
