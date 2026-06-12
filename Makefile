NIX_CONFIG ?= extra-experimental-features = nix-command flakes
export NIX_CONFIG

NIX_DEV_SHELL ?= github:sund3RRR/nix-go-bindings
NIX_DEVELOP = nix develop $(NIX_DEV_SHELL) --command

.PHONY: test

deps:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest

test:
	$(NIX_DEVELOP) go test -v -cover -race -count=1 ./...

lint:
	$(NIX_DEVELOP) golangci-lint run
