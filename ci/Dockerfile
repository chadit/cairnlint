# CI image: the toolchain the lint and test jobs need. Everything tracks
# latest so local and CI agree and new checks land the day they ship.
# Built and run by scripts/ci-container.sh.

# version-pin: skip gate tooling is always-latest by policy; the compile
# toolchain is governed by go.mod, not this tag
FROM docker.io/library/golang:alpine

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

# ci-container.sh always bind-mounts both paths, and a mount takes the host
# directory's ownership, so these modes only apply when the image runs unmounted.
RUN adduser -D -u 1001 ci \
    && mkdir -p /cache /work \
    && chown ci:ci /cache /work

WORKDIR /work
USER 1001

CMD ["/bin/bash"]
