# syntax=docker/dockerfile:1.7

ARG GO_VERSION=1.27

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-trixie AS build

ARG TARGETOS=linux
ARG TARGETARCH
ARG VERSION=dev
ARG VCS_REF=unknown
ARG BUILD_DATE=unknown

WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY cmd/ cmd/
COPY internal/ internal/

RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build \
      -trimpath \
      -ldflags="-s -w -X github.com/netspeedy/blackhole-threats/internal/buildinfo.Version=${VERSION} -X github.com/netspeedy/blackhole-threats/internal/buildinfo.Commit=${VCS_REF} -X github.com/netspeedy/blackhole-threats/internal/buildinfo.BuildDate=${BUILD_DATE}" \
      -o /out/blackhole-threats \
      ./cmd/blackhole-threats

FROM debian:trixie-slim

ARG VERSION=dev
ARG VCS_REF=unknown
ARG BUILD_DATE=unknown

LABEL org.opencontainers.image.title="blackhole-threats" \
      org.opencontainers.image.description="BGP blackhole route server for threat feeds" \
      org.opencontainers.image.url="https://github.com/netspeedy/blackhole-threats" \
      org.opencontainers.image.source="https://github.com/netspeedy/blackhole-threats" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${VCS_REF}" \
      org.opencontainers.image.created="${BUILD_DATE}" \
      org.opencontainers.image.licenses="MIT"

# ca-certificates for HTTPS feeds, tzdata for TZ.
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
         ca-certificates \
         tzdata \
    && rm -rf /var/lib/apt/lists/*

COPY --from=build /out/blackhole-threats /usr/sbin/blackhole-threats
COPY packaging/container/rootfs/ /

RUN mkdir -p /config \
    && chmod +x /usr/local/bin/blackhole-threats-entrypoint

EXPOSE 179

# The entrypoint execs the daemon, so the daemon is PID 1 and gets the signals docker sends:
# SIGUSR1 from docker kill refreshes the feeds, and SIGTERM from docker stop shuts it down
# cleanly. Don't put an init or supervisor in front of it without keeping that true.
ENTRYPOINT ["/usr/local/bin/blackhole-threats-entrypoint"]
