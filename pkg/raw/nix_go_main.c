#include "nix_go_main.h"

nix_err go_nix_init_plugins(nix_c_context *ctx)
{
    return nix_init_plugins(ctx);
}

nix_err go_nix_set_log_format(nix_c_context *ctx, const char *format)
{
    return nix_set_log_format(ctx, format);
}
