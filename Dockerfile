# syntax=docker/dockerfile:1.7

ARG GO_VERSION=1.25.0
ARG DISTROLESS_BASE_IMAGE=gcr.io/distroless/static-debian12:nonroot

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-bookworm AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT
ARG VERSION=dev
ARG COMMIT=none
ARG DATE=unknown

RUN --mount=type=cache,target=/root/.cache/go-build \
    mkdir -p /out && \
    CGO_ENABLED=0 \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH} \
    GOARM=${TARGETVARIANT#v} \
    go build -trimpath -ldflags="-s -w \
      -X github.com/galax-io/galaxio-cli/internal/buildinfo.Version=${VERSION} \
      -X github.com/galax-io/galaxio-cli/internal/buildinfo.Commit=${COMMIT} \
      -X github.com/galax-io/galaxio-cli/internal/buildinfo.Date=${DATE}" \
    -o /out/galaxio ./cmd/galaxio

FROM ${DISTROLESS_BASE_IMAGE}

LABEL org.opencontainers.image.title="galaxio-cli"
LABEL org.opencontainers.image.description="CLI for Gatling performance testing workflows."
LABEL org.opencontainers.image.source="https://github.com/galax-io/galaxio-cli"
LABEL org.opencontainers.image.licenses="GPL-2.0-only"

COPY --from=build /out/galaxio /usr/local/bin/galaxio

ENTRYPOINT ["/usr/local/bin/galaxio"]
