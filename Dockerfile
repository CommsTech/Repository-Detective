# syntax=docker/dockerfile:1
# Repository Detective multi-target image: core | runner | all-in-one
# See README and scripts/docker-build-verify.sh for build examples.

ARG GO_VERSION=1.25

FROM golang:${GO_VERSION}-alpine AS builder

COPY scripts/apk-retry.sh /tmp/apk-retry.sh
RUN chmod +x /tmp/apk-retry.sh && . /tmp/apk-retry.sh && apk_retry git ca-certificates tzdata

WORKDIR /app

ARG GOPROXY=https://proxy.golang.org,direct
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

ENV GOPROXY=${GOPROXY}
ENV GOSUMDB=sum.golang.org

COPY go.mod go.sum ./
COPY . .

RUN if [ -f vendor/modules.txt ]; then \
      echo "building with vendored modules"; \
    else \
      go mod download; \
    fi

RUN if [ -f vendor/modules.txt ]; then \
      CGO_ENABLED=0 GOOS=linux go build -mod=vendor -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildDate=${BUILD_DATE}" -o repository-detective . && \
      CGO_ENABLED=0 GOOS=linux go build -mod=vendor -ldflags="-s -w" -o repository-detective-runner ./cmd/repository-detective-runner; \
    else \
      CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildDate=${BUILD_DATE}" -o repository-detective . && \
      CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o repository-detective-runner ./cmd/repository-detective-runner; \
    fi

# Install analysis CLIs required by the all-in-one/runner images. Fail the build
# if any binary is missing (older loop + `|| sleep` could mask install failures).
RUN set -eu; \
    go install golang.org/x/vuln/cmd/govulncheck@latest; \
    go install github.com/securego/gosec/v2/cmd/gosec@latest; \
    go install honnef.co/go/tools/cmd/staticcheck@v0.6.1; \
    go install github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@v1.9.0; \
    test -x /go/bin/govulncheck; \
    test -x /go/bin/gosec; \
    test -x /go/bin/staticcheck; \
    test -x /go/bin/cyclonedx-gomod

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
COPY scripts/apk-retry.sh /usr/local/lib/rd/apk-retry.sh
COPY scripts/install-scanner-tools.sh /tmp/install-scanner-tools.sh

RUN cp /usr/local/lib/rd/apk-retry.sh /tmp/apk-retry.sh && \
    chmod +x /tmp/apk-retry.sh /usr/local/lib/rd/apk-retry.sh /tmp/install-scanner-tools.sh && \
    if [ "$INSTALL_EXTERNAL_TOOLS" = "true" ]; then \
      /tmp/install-scanner-tools.sh; \
    else \
      . /tmp/apk-retry.sh && apk_retry git ca-certificates ;\
    fi

COPY --from=builder /go/bin/govulncheck /usr/local/bin/govulncheck
COPY --from=builder /go/bin/gosec /usr/local/bin/gosec
COPY --from=builder /go/bin/staticcheck /usr/local/bin/staticcheck
COPY --from=builder /go/bin/cyclonedx-gomod /usr/local/bin/cyclonedx-gomod
COPY --from=builder /usr/local/go /usr/local/go
ENV PATH="/usr/local/go/bin:${PATH}"

# Fail the build if SBOM / Go analysis CLIs did not land on PATH.
RUN set -eu; \
    test -x /usr/local/bin/govulncheck; \
    test -x /usr/local/bin/gosec; \
    test -x /usr/local/bin/staticcheck; \
    test -x /usr/local/bin/cyclonedx-gomod; \
    if [ "$INSTALL_EXTERNAL_TOOLS" = "true" ]; then \
      test -x /usr/local/bin/syft; \
      /usr/local/bin/syft version >/dev/null; \
      /usr/local/bin/cyclonedx-gomod version >/dev/null || /usr/local/bin/cyclonedx-gomod -h >/dev/null; \
    fi

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

COPY scripts/apk-retry.sh /usr/local/lib/rd/apk-retry.sh
COPY scripts/docker-alpine-runtime-setup.sh /usr/local/lib/rd/docker-alpine-runtime-setup.sh

RUN chmod +x /usr/local/lib/rd/apk-retry.sh /usr/local/lib/rd/docker-alpine-runtime-setup.sh && \
    /usr/local/lib/rd/docker-alpine-runtime-setup.sh ca-certificates tzdata wget su-exec git

WORKDIR /app

COPY --from=builder /app/repository-detective .
COPY config/config.yaml.example \
     config/gitleaks.toml \
     config/private-beta.example.yaml \
     config/runner.example.yaml \
     config/runner-delegation-test.yaml.example \
     ./config/
COPY scripts/docker-entrypoint.sh scripts/docker-healthcheck.sh /usr/local/bin/

RUN chmod +x repository-detective /usr/local/bin/docker-entrypoint.sh /usr/local/bin/docker-healthcheck.sh && \
    /usr/local/lib/rd/docker-alpine-runtime-setup.sh && \
    mkdir -p /app/data && \
    chown -R repositorydetective:repositorydetective /app

VOLUME ["/app/data"]
EXPOSE 8081

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

COPY scripts/docker-alpine-runtime-setup.sh /usr/local/lib/rd/docker-alpine-runtime-setup.sh

RUN chmod +x /usr/local/lib/rd/docker-alpine-runtime-setup.sh && \
    /usr/local/lib/rd/docker-alpine-runtime-setup.sh wget su-exec

WORKDIR /app

COPY --from=builder /app/repository-detective-runner /usr/local/bin/repository-detective-runner

RUN chmod +x /usr/local/bin/repository-detective-runner && \
    /usr/local/lib/rd/docker-alpine-runtime-setup.sh && \
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

COPY scripts/docker-alpine-runtime-setup.sh /usr/local/lib/rd/docker-alpine-runtime-setup.sh

RUN chmod +x /usr/local/lib/rd/docker-alpine-runtime-setup.sh && \
    /usr/local/lib/rd/docker-alpine-runtime-setup.sh wget su-exec git && \
    git --version

WORKDIR /app

COPY --from=builder /app/repository-detective .
COPY --from=builder /app/repository-detective-runner /usr/local/bin/repository-detective-runner
# Examples + allowlists only — never live config.yaml (see .dockerignore)
COPY config/config.yaml.example \
     config/gitleaks.toml \
     config/private-beta.example.yaml \
     config/runner.example.yaml \
     config/runner-delegation-test.yaml.example \
     ./config/
COPY scripts/docker-entrypoint.sh scripts/docker-healthcheck.sh /usr/local/bin/

RUN chmod +x repository-detective /usr/local/bin/repository-detective-runner \
    /usr/local/bin/docker-entrypoint.sh /usr/local/bin/docker-healthcheck.sh && \
    /usr/local/lib/rd/docker-alpine-runtime-setup.sh && \
    mkdir -p /app/data && chown -R repositorydetective:repositorydetective /app

VOLUME ["/app/data"]
EXPOSE 8081

HEALTHCHECK --interval=30s --timeout=10s --start-period=45s --retries=3 \
    CMD ["/usr/local/bin/docker-healthcheck.sh"]

USER repositorydetective

ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
CMD ["/app/repository-detective"]
