#!/usr/bin/env bash
#
# ctc installer — container to compose
#
#   curl https://ctc.dothis.online | bash
#
# Installs Go if missing, builds the ctc binary from source, and places it
# on your PATH.

set -euo pipefail

MODULE="github.com/chaiops/ctc"
VERSION="${CTC_VERSION:-latest}"
BIN_NAME="ctc"

# --- pretty output -----------------------------------------------------------
if [ -t 1 ]; then
  BOLD=$(printf '\033[1m'); DIM=$(printf '\033[2m')
  GREEN=$(printf '\033[32m'); RED=$(printf '\033[31m')
  YELLOW=$(printf '\033[33m'); RESET=$(printf '\033[0m')
else
  BOLD=""; DIM=""; GREEN=""; RED=""; YELLOW=""; RESET=""
fi

info()  { printf '%s==>%s %s\n' "$GREEN" "$RESET" "$1"; }
warn()  { printf '%s==>%s %s\n' "$YELLOW" "$RESET" "$1"; }
err()   { printf '%serror:%s %s\n' "$RED" "$RESET" "$1" >&2; }
step()  { printf '%s  •%s %s\n' "$DIM" "$RESET" "$1"; }

fail() { err "$1"; exit 1; }

# --- detect platform ---------------------------------------------------------
detect_platform() {
  local os arch
  os="$(uname -s)"
  arch="$(uname -m)"
  case "$os" in
    Linux)  GOOS="linux" ;;
    Darwin) GOOS="darwin" ;;
    *)      fail "unsupported OS: $os (Linux and macOS only)" ;;
  esac
  case "$arch" in
    x86_64|amd64)  GOARCH="amd64" ;;
    arm64|aarch64) GOARCH="arm64" ;;
    *)             fail "unsupported architecture: $arch" ;;
  esac
}

# --- ensure Go is available --------------------------------------------------
GO_MIN_MAJOR=1
GO_MIN_MINOR=21

go_version_ok() {
  command -v go >/dev/null 2>&1 || return 1
  local v major minor
  v="$(go env GOVERSION 2>/dev/null | sed 's/^go//')" || return 1
  major="${v%%.*}"
  minor="${v#*.}"; minor="${minor%%.*}"
  [ -z "$major" ] && return 1
  if [ "$major" -gt "$GO_MIN_MAJOR" ]; then return 0; fi
  if [ "$major" -eq "$GO_MIN_MAJOR" ] && [ "$minor" -ge "$GO_MIN_MINOR" ]; then return 0; fi
  return 1
}

# --- install ctc -------------------------------------------------------------
resolve_gobin() {
  GOBIN="$(go env GOBIN 2>/dev/null || true)"
  [ -z "$GOBIN" ] && GOBIN="$(go env GOPATH 2>/dev/null)/bin"
}

install_ctc() {
  step "building ${BIN_NAME} from ${MODULE}@${VERSION}"
  GOBIN="$GOBIN" go install "${MODULE}@${VERSION}" \
    || fail "go install failed"
}

on_path() {
  case ":${PATH}:" in *":${GOBIN}:"*) return 0 ;; *) return 1 ;; esac
}

# --- main --------------------------------------------------------------------
main() {
  printf '%s%s ctc installer %s\n\n' "$BOLD" "$GREEN" "$RESET"

  detect_platform
  info "platform: ${GOOS}/${GOARCH}"

  if ! command -v go >/dev/null 2>&1; then
    fail "Go is not installed or not on your PATH. Install Go ${GO_MIN_MAJOR}.${GO_MIN_MINOR}+ from https://go.dev/dl/ and try again."
  fi
  if ! go_version_ok; then
    fail "Go $(go env GOVERSION 2>/dev/null || echo '?') is too old. ctc needs Go ${GO_MIN_MAJOR}.${GO_MIN_MINOR}+ — upgrade from https://go.dev/dl/ and try again."
  fi
  info "Go found: $(go env GOVERSION)"

  resolve_gobin
  mkdir -p "$GOBIN"
  install_ctc

  local target="${GOBIN}/${BIN_NAME}"
  [ -x "$target" ] || fail "expected binary at ${target} but it is missing"

  printf '\n'
  info "${BIN_NAME} installed to ${BOLD}${target}${RESET}"

  if ! on_path; then
    printf '\n%s%s is not on your PATH.%s To run it later, add this to your shell profile:\n\n' \
      "$YELLOW" "$GOBIN" "$RESET"
    printf '    export PATH="%s:$PATH"\n' "$GOBIN"
  fi

  # Launch immediately. When invoked as `curl … | bash`, this script owns
  # bash's stdin, so the binary's stdin is the pipe, not the terminal. A TUI
  # needs the real tty — redirect stdin/stdout to /dev/tty when one exists.
  if [ -e /dev/tty ] && [ -r /dev/tty ]; then
    printf '\n'
    info "launching ${BIN_NAME}…"
    exec "$target" </dev/tty >/dev/tty
  else
    printf '\nNo terminal available. Run %s%s%s to get started.\n' \
      "$BOLD" "$BIN_NAME" "$RESET"
  fi
}

main "$@"
