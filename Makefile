NIX_CONFIG ?= extra-experimental-features = nix-command flakes
export NIX_CONFIG

NIX_DEV_SHELL ?= path:.
NIX_DEVELOP = nix develop $(NIX_DEV_SHELL) --command

.PHONY: generate test lint check

generate:
	nix run path:.#generate-go-bindings

test:
	$(NIX_DEVELOP) go test -v -cover -race -count=1 ./...

lint:
	$(NIX_DEVELOP) golangci-lint run

check: test lint
