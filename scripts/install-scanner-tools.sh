#!/bin/sh
# Install pinned external scanner binaries (Alpine Linux / musl amd64).
# Used by Dockerfile scanner-tools stage. Versions documented in docs/DOCKER.md.
set -eu

TRIVY_VERSION="${TRIVY_VERSION:-0.57.1}"
GRYPE_VERSION="${GRYPE_VERSION:-0.84.0}"
GITLEAKS_VERSION="${GITLEAKS_VERSION:-8.21.2}"
SEMGREP_VERSION="${SEMGREP_VERSION:-1.76.0}"
HADOLINT_VERSION="${HADOLINT_VERSION:-2.12.0}"
CHECKOV_VERSION="${CHECKOV_VERSION:-3.2.254}"
GOLANGCI_VERSION="${GOLANGCI_VERSION:-1.55.2}"
SYFT_VERSION="${SYFT_VERSION:-1.18.1}"

install_trivy() {
  if [ -x /tmp/deploy-bin/trivy ]; then
    install -m 0755 /tmp/deploy-bin/trivy /usr/local/bin/trivy
    return
  fi
  curl -sfL "https://github.com/aquasecurity/trivy/releases/download/v${TRIVY_VERSION}/trivy_${TRIVY_VERSION}_Linux-64bit.tar.gz" \
    | tar xz -C /usr/local/bin trivy
}

install_grype() {
  if [ -x /tmp/deploy-bin/grype ]; then
    install -m 0755 /tmp/deploy-bin/grype /usr/local/bin/grype
    return
  fi
  curl -sSfL https://raw.githubusercontent.com/anchore/grype/main/install.sh \
    | sh -s -- -b /usr/local/bin "v${GRYPE_VERSION}"
}

install_syft() {
  if [ -x /tmp/deploy-bin/syft ]; then
    install -m 0755 /tmp/deploy-bin/syft /usr/local/bin/syft
    return
  fi
  curl -sSfL https://raw.githubusercontent.com/anchore/syft/main/install.sh \
    | sh -s -- -b /usr/local/bin "v${SYFT_VERSION}"
}

install_gitleaks() {
  if [ -x /tmp/deploy-bin/gitleaks ]; then
    install -m 0755 /tmp/deploy-bin/gitleaks /usr/local/bin/gitleaks
    return
  fi
  curl -sSfL "https://github.com/gitleaks/gitleaks/releases/download/v${GITLEAKS_VERSION}/gitleaks_${GITLEAKS_VERSION}_linux_x64.tar.gz" \
    | tar xz -C /usr/local/bin gitleaks
}

install_hadolint() {
  if [ -x /tmp/deploy-bin/hadolint ]; then
    install -m 0755 /tmp/deploy-bin/hadolint /usr/local/bin/hadolint
    return
  fi
  curl -sSfL "https://github.com/hadolint/hadolint/releases/download/v${HADOLINT_VERSION}/hadolint-Linux-x86_64" \
    -o /usr/local/bin/hadolint
  chmod +x /usr/local/bin/hadolint
}

install_semgrep() {
  pip3 install --no-cache-dir --break-system-packages "semgrep==${SEMGREP_VERSION}"
}

install_checkov() {
  pip3 install --no-cache-dir --break-system-packages "checkov==${CHECKOV_VERSION}"
}

install_golangci() {
  curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh \
    | sh -s -- -b /usr/local/bin "v${GOLANGCI_VERSION}"
}

install_shellcheck() {
  . /tmp/apk-retry.sh
  apk_retry shellcheck || echo "shellcheck install skipped"
}

install_ruff() {
  pip3 install --no-cache-dir --break-system-packages "ruff==0.8.4" || echo "ruff install skipped"
}

refresh_grype_db() {
  if command -v grype >/dev/null 2>&1; then
    grype db update || echo "grype db update skipped"
  fi
}

. /tmp/apk-retry.sh
apk_retry curl bash tar python3 py3-pip git ca-certificates

install_trivy
install_grype
refresh_grype_db
install_syft
install_gitleaks || echo "gitleaks install skipped"
install_hadolint || echo "hadolint install skipped"
install_semgrep || echo "semgrep install skipped"
install_checkov || echo "checkov install skipped"
install_golangci || echo "golangci-lint install skipped (optional)"
install_shellcheck || echo "shellcheck install skipped"
install_ruff || echo "ruff install skipped"

# Required SBOM toolchain — fail the image build if these are absent.
for required in trivy grype syft; do
  if ! command -v "$required" >/dev/null 2>&1; then
    echo "ERROR: required scanner binary missing after install: $required" >&2
    exit 1
  fi
done

for bin in trivy grype syft gitleaks semgrep govulncheck gosec staticcheck hadolint checkov shellcheck ruff cyclonedx-gomod; do
  if command -v "$bin" >/dev/null 2>&1; then
    echo "installed: $bin -> $($bin --version 2>/dev/null | head -1 || echo ok)"
  fi
done
