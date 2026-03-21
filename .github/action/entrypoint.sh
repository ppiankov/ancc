#!/usr/bin/env bash
set -euo pipefail

REPO="ppiankov/ancc"

install_ancc() {
  local version="${1:-latest}"
  local os
  local arch

  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  arch="$(uname -m)"
  case "$arch" in
    x86_64) arch="amd64" ;;
    aarch64|arm64) arch="arm64" ;;
  esac

  if [[ "$version" == "latest" ]]; then
    local curl_args=(-fsSL)
    if [[ -n "${GITHUB_TOKEN:-}" ]]; then
      curl_args+=(-H "Authorization: token ${GITHUB_TOKEN}")
    fi
    version="$(curl "${curl_args[@]}" "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"v?([^"]+)".*/\1/')"
  fi
  version="${version#v}"

  local url="https://github.com/${REPO}/releases/download/v${version}/ancc-${version}-${os}-${arch}.tar.gz"
  echo "Installing ancc v${version} from ${url}"

  local tmpdir
  tmpdir="$(mktemp -d)"
  curl -fsSL "$url" | tar xz -C "$tmpdir"
  sudo install "$tmpdir/ancc" /usr/local/bin/ancc
  rm -rf "$tmpdir"

  ancc --version
}

run_checks() {
  local checks="${1:-validate}"
  local fail_on_warn="${2:-false}"
  local exit_code=0

  IFS=',' read -ra check_list <<< "$checks"
  for check in "${check_list[@]}"; do
    check="$(echo "$check" | xargs)"
    echo "::group::ancc ${check}"

    local result
    result="$(ancc "$check" . --format json 2>&1)" || true
    echo "$result"

    local status
    status="$(echo "$result" | grep -o '"status"[[:space:]]*:[[:space:]]*"[^"]*"' | head -1 | sed 's/.*"\([^"]*\)"/\1/')" || status="unknown"

    if [[ "$status" == "fail" ]]; then
      exit_code=1
    elif [[ "$status" == "partial" && "$fail_on_warn" == "true" ]]; then
      exit_code=1
    elif [[ "$status" == "partial" && $exit_code -eq 0 ]]; then
      exit_code=2
    fi

    echo "::endgroup::"
  done

  if [[ -n "${GITHUB_OUTPUT:-}" ]]; then
    local final_status="pass"
    if [[ $exit_code -eq 1 ]]; then
      final_status="fail"
    elif [[ $exit_code -eq 2 ]]; then
      final_status="partial"
    fi
    echo "status=${final_status}" >> "$GITHUB_OUTPUT"
    echo "summary=ancc ${checks}: ${final_status}" >> "$GITHUB_OUTPUT"
  fi

  exit "$exit_code"
}

case "${1:-}" in
  install) install_ancc "${2:-latest}" ;;
  run)     run_checks "${2:-validate}" "${3:-false}" ;;
  *)       echo "Usage: $0 {install|run} [args...]" >&2; exit 1 ;;
esac
