# syntax=docker/dockerfile:1
# Repository Detective — multi-target image build.
#
# Targets:
#   core       — control plane only (smallest; git for clones; no scanner binaries)
#   runner     — repository-detective-runner + scanner toolchain (Gitea Actions workers)
#   all-in-one — core + runner + scanners (homelab default)
#
# Examples:
#   docker build --target core -t repository-detective:core .
#   docker build --target runner -t repository-detective:runner .
#   docker build --target all-in-one -t repository-detective:all-in-one .
#
# Offline / DNS-filtered builds:
#   ./scripts/vendor-deps.sh
#   cp ~/.local/bin/trivy deploy/bin/trivy   # optional
#   docker build --target all-in-one --build-arg INSTALL_EXTERNAL_TOOLS=true .

ARG GO_VERSION=1.23

FROM golang:${GO_VERSION}-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

ARG GOPROXY=https://proxy.golang.org,direct
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

ENV GOPROXY=${GOPROXY}
ENV GOSUMDB=sum.golang.org

COPY go.mod go.sum ./
COPY . .

RUN if [ -d vendor/modules.txt ]; then \
      echo "building with vendored modules"; \
    else \
      go mod download; \
    fi

RUN if [ -d vendor/modules.txt ]; then \
      CGO_ENABLED=0 GOOS=linux go build -mod=vendor -ldflags="-s -w -X main.version=${VERSION}" -o repository-detective . && \
      CGO_ENABLED=0 GOOS=linux go build -mod=vendor -ldflags="-s -w" -o repository-detective-runner ./cmd/repository-detective-runner; \
    else \
      CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w -X main.version=${VERSION}" -o repository-detective . && \
      CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o repository-detective-runner ./cmd/repository-detective-runner; \
    fi

RUN go install golang.org/x/vuln/cmd/govulncheck@v1.1.3 && \
    go install github.com/securego/gosec/v2/cmd/gosec@v2.21.4 && \
    go install honnef.co/go/tools/cmd/staticcheck@v0.5.1

# Satisfy image scanners (build artifacts copied out before this stage is discarded).
USER nobody

# ---------------------------------------------------------------------------
# Scanner toolchain layer (shared by runner + all-in-one)
# ---------------------------------------------------------------------------
FROM alpine:3.20 AS scanner-tools

ARG INSTALL_EXTERNAL_TOOLS=true
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

COPY deploy/bin /tmp/deploy-bin
COPY scripts/install-scanner-tools.sh /tmp/install-scanner-tools.sh

RUN chmod +x /tmp/install-scanner-tools.sh && \
    if [ "$INSTALL_EXTERNAL_TOOLS" = "true" ]; then \
      /tmp/install-scanner-tools.sh; \
    else \
      apk add --no-cache git ca-certificates ;\
    fi

COPY --from=builder /go/bin/govulncheck /go/bin/gosec /go/bin/staticcheck /usr/local/bin/
COPY --from=builder /usr/local/go /usr/local/go
ENV PATH="/usr/local/go/bin:${PATH}"

RUN adduser -D -u 65532 scanner
USER scanner

# ---------------------------------------------------------------------------
# core — web/API/UI, migrations, scheduler; no external scanner binaries
# ---------------------------------------------------------------------------
FROM alpine:3.20 AS core

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

LABEL org.opencontainers.image.title="Repository Detective (core)" \
      org.opencontainers.image.description="Control plane: API, UI, DB, policy — no bundled scanners" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${COMMIT}" \
      org.opencontainers.image.created="${BUILD_DATE}" \
      com.commsnet.repository-detective.variant="core"

RUN apk add --no-cache ca-certificates tzdata wget su-exec git && \
    addgroup -g 1001 -S repositorydetective && \
    adduser -u 1001 -S repositorydetective -G repositorydetective

WORKDIR /app

COPY --from=builder /app/repository-detective .
COPY --from=builder /app/config ./config
COPY scripts/docker-entrypoint.sh scripts/docker-healthcheck.sh /usr/local/bin/

RUN chmod +x repository-detective /usr/local/bin/docker-entrypoint.sh /usr/local/bin/docker-healthcheck.sh && \
    mkdir -p /app/data && chown -R repositorydetective:repositorydetective /app

VOLUME ["/app/data"]
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=10s --start-period=25s --retries=3 \
    CMD ["/usr/local/bin/docker-healthcheck.sh"]

USER repositorydetective

ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
CMD ["/app/repository-detective"]

# ---------------------------------------------------------------------------
# runner — delegated scan worker image (Gitea Actions / CI)
# ---------------------------------------------------------------------------
FROM scanner-tools AS runner

USER root

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

LABEL org.opencontainers.image.title="Repository Detective (runner)" \
      org.opencontainers.image.description="Runner worker with scanner toolchain" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${COMMIT}" \
      org.opencontainers.image.created="${BUILD_DATE}" \
      com.commsnet.repository-detective.variant="runner"

RUN apk add --no-cache wget su-exec && \
    addgroup -g 1001 -S repositorydetective && \
    adduser -u 1001 -S repositorydetective -G repositorydetective

WORKDIR /app

COPY --from=builder /app/repository-detective-runner /usr/local/bin/repository-detective-runner

RUN chmod +x /usr/local/bin/repository-detective-runner && \
    mkdir -p /workspace && chown -R repositorydetective:repositorydetective /app /workspace

USER repositorydetective

ENTRYPOINT ["/usr/local/bin/repository-detective-runner"]

# ---------------------------------------------------------------------------
# all-in-one — homelab / simple single-container deploy (default target)
# ---------------------------------------------------------------------------
FROM scanner-tools AS all-in-one

USER root

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

LABEL org.opencontainers.image.title="Repository Detective (all-in-one)" \
      org.opencontainers.image.description="Core service plus scanner toolchain in one image" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${COMMIT}" \
      org.opencontainers.image.created="${BUILD_DATE}" \
      com.commsnet.repository-detective.variant="all-in-one"

RUN apk add --no-cache wget su-exec && \
    addgroup -g 1001 -S repositorydetective && \
    adduser -u 1001 -S repositorydetective -G repositorydetective

WORKDIR /app

COPY --from=builder /app/repository-detective .
COPY --from=builder /app/repository-detective-runner /usr/local/bin/repository-detective-runner
COPY --from=builder /app/config ./config
COPY scripts/docker-entrypoint.sh scripts/docker-healthcheck.sh /usr/local/bin/

RUN chmod +x repository-detective /usr/local/bin/repository-detective-runner \
    /usr/local/bin/docker-entrypoint.sh /usr/local/bin/docker-healthcheck.sh && \
    mkdir -p /app/data && chown -R repositorydetective:repositorydetective /app

VOLUME ["/app/data"]
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=10s --start-period=45s --retries=3 \
    CMD ["/usr/local/bin/docker-healthcheck.sh"]

USER repositorydetective

ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
CMD ["/app/repository-detective"]
