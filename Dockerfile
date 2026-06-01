# Multi-stage production image for Bugbot / Repository Detective.
#
# Normal networks:
#   docker-compose -f docker-compose.public.yml up -d --build
#
# DNS-filtered networks (storage.googleapis.com blocked):
#   ./scripts/vendor-deps.sh
#   docker-compose -f docker-compose.public.yml up -d --build
#
# Air-gapped hosts: load a CI-built image (see docker-compose.offline.yml).

FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

ARG GOPROXY=https://proxy.golang.org,https://goproxy.io,direct
ENV GOPROXY=${GOPROXY}
ENV GOSUMDB=sum.golang.org

COPY go.mod go.sum ./
COPY vendor/ vendor/

# Use vendored modules when present; otherwise download (normal networks only).
RUN if [ -d vendor/modules.txt ]; then \
      echo "building with vendored modules (offline-friendly)"; \
    else \
      echo "vendor/ missing — downloading modules (requires module proxy)"; \
      go mod download; \
    fi

COPY . .

RUN if [ -d vendor/modules.txt ]; then \
      CGO_ENABLED=0 GOOS=linux go build -mod=vendor -ldflags="-s -w" -o gitea-bugbot .; \
    else \
      CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o gitea-bugbot .; \
    fi

FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata wget curl bash su-exec \
    && TRIVY_VERSION=0.57.1 \
    && curl -sfL "https://github.com/aquasecurity/trivy/releases/download/v${TRIVY_VERSION}/trivy_${TRIVY_VERSION}_Linux-64bit.tar.gz" \
       | tar xz -C /usr/local/bin trivy \
    && GRYPE_VERSION=0.84.0 \
    && curl -sSfL "https://github.com/anchore/grype/releases/download/v${GRYPE_VERSION}/grype_${GRYPE_VERSION}_linux_amd64.tar.gz" \
       | tar xz -C /usr/local/bin grype \
    && curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b /usr/local/bin v1.55.2 \
    && RUFF_VERSION=0.8.4 \
    && curl -sSL "https://github.com/astral-sh/ruff/releases/download/${RUFF_VERSION}/ruff-x86_64-unknown-linux-musl.tar.gz" \
       | tar xz -C /usr/local/bin --strip-components=1 "ruff-x86_64-unknown-linux-musl/ruff"

RUN addgroup -g 1001 -S bugbot && \
    adduser -u 1001 -S bugbot -G bugbot

WORKDIR /app

COPY --from=builder /app/gitea-bugbot .
COPY --from=builder /app/config ./config
COPY scripts/docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh

RUN chmod +x gitea-bugbot /usr/local/bin/docker-entrypoint.sh && \
    chown -R bugbot:bugbot /app

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=10s --start-period=20s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://127.0.0.1:8080/health || exit 1

ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
CMD ["/app/gitea-bugbot"]
