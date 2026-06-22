#include "nix_go_fetchers.h"

nix_fetchers_settings *go_nix_fetchers_settings_new(nix_c_context *ctx)
{
    return nix_fetchers_settings_new(ctx);
}

void go_nix_fetchers_settings_free(nix_fetchers_settings *settings)
{
    nix_fetchers_settings_free(settings);
}
