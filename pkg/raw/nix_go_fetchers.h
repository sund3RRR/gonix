#ifndef NIX_GO_FETCHERS_H
#define NIX_GO_FETCHERS_H

#include "nix_go_util.h"
#include "nix_api_fetchers.h"

#ifdef __cplusplus
extern "C" {
#endif

nix_fetchers_settings *go_nix_fetchers_settings_new(nix_c_context *ctx);
void go_nix_fetchers_settings_free(nix_fetchers_settings *settings);

#ifdef __cplusplus
}
#endif

#endif
