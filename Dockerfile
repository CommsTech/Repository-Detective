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
COPY . .

# Use vendored modules when vendor/ is in the build context; otherwise download.
RUN if [ -d vendor/modules.txt ]; then \
      echo "building with vendored modules (offline-friendly)"; \
    else \
      echo "downloading modules (requires module proxy)"; \
      go mod download; \
    fi

RUN if [ -d vendor/modules.txt ]; then \
      CGO_ENABLED=0 GOOS=linux go build -mod=vendor -ldflags="-s -w" -o repository-detective .; \
    else \
      CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o repository-detective .; \
    fi

FROM alpine:3.20

# External scanner binaries are optional because some networks block GitHub/CDNs.
# Enable at build time with: --build-arg INSTALL_EXTERNAL_TOOLS=true
ARG INSTALL_EXTERNAL_TOOLS=false

RUN apk --no-cache add ca-certificates tzdata wget su-exec \
    && if [ "$INSTALL_EXTERNAL_TOOLS" = "true" ]; then \
         apk --no-cache add curl bash tar; \
         TRIVY_VERSION=0.57.1; \
         curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh \
           | sh -s -- -b /usr/local/bin "v${TRIVY_VERSION}"; \
         GRYPE_VERSION=0.84.0; \
         curl -sSfL https://raw.githubusercontent.com/anchore/grype/main/install.sh \
           | sh -s -- -b /usr/local/bin "v${GRYPE_VERSION}"; \
         curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh \
           | sh -s -- -b /usr/local/bin v1.55.2; \
         RUFF_VERSION=0.8.4; \
         curl -sSL "https://github.com/astral-sh/ruff/releases/download/${RUFF_VERSION}/ruff-x86_64-unknown-linux-musl.tar.gz" \
           | tar xz -C /usr/local/bin --strip-components=1 "ruff-x86_64-unknown-linux-musl/ruff"; \
       fi

RUN addgroup -g 1001 -S repositorydetective && \
    adduser -u 1001 -S repositorydetective -G repositorydetective

WORKDIR /app

COPY --from=builder /app/repository-detective .
COPY --from=builder /app/config ./config
COPY scripts/docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh

RUN chmod +x repository-detective /usr/local/bin/docker-entrypoint.sh && \
    chown -R repositorydetective:repositorydetective /app

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=10s --start-period=20s --retries=3 \
    CMD wget -q -O /dev/null http://127.0.0.1:8081/health || exit 1

ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
CMD ["/app/repository-detective"]
