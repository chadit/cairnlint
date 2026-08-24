# CI image: the toolchain the lint and test jobs need. Everything tracks
# latest so local and CI agree and new checks land the day they ship.
# Built and run by scripts/ci-container.sh.

# version-pin: skip gate tooling is always-latest by policy; the compile
# toolchain is governed by go.mod, not this tag
FROM golang:alpine

SHELL ["/bin/sh", "-o", "pipefail", "-c"]

ENV GOPATH=/cache/go \
    GOMODCACHE=/cache/go/pkg/mod \
    GOCACHE=/cache/go-build \
    GOFLAGS=-buildvcs=false

# build-base backs cgo, which `go test -race` needs on musl.
# hadolint ignore=DL3018
RUN apk add --no-cache bash build-base git curl

# hadolint ignore=DL3059,DL3062
RUN go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest \
    && install -m 0755 /cache/go/bin/golangci-lint /usr/local/bin/golangci-lint \
    && golangci-lint --version

# Bind-mounted at run time under whichever uid the runner maps in.
RUN mkdir -p /cache /work \
    && chmod 0777 /cache /work \
    && adduser -D -u 1001 ci

WORKDIR /work
USER 1001

CMD ["/bin/bash"]
