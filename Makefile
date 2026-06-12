NIX_CONFIG ?= extra-experimental-features = nix-command flakes
export NIX_CONFIG

NIX_DEV_SHELL ?= github:sund3RRR/nix-go-bindings
NIX_DEVELOP = nix develop $(NIX_DEV_SHELL) --command

.PHONY: test

test:
	$(NIX_DEVELOP) go test -v -cover -race -count=1 ./...
