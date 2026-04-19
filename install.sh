#!/usr/bin/env bash

set -euo pipefail

readonly REPO="${LLMBUDGET_GITHUB_REPO:-Klassikcat/llm-budget-app}"
readonly BINARY_NAME="${LLMBUDGET_BINARY_NAME:-llm-budget-tracker}"

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    printf 'Error: required command not found: %s\n' "$1" >&2
    exit 1
  fi
}

detect_os() {
  local uname_os
  uname_os="$(uname -s)"
  case "$uname_os" in
    Linux)
      printf 'linux\n'
      ;;
    Darwin)
      printf 'darwin\n'
      ;;
    *)
      printf 'Error: unsupported operating system: %s\n' "$uname_os" >&2
      exit 1
      ;;
  esac
}

detect_arch() {
  local uname_arch
  uname_arch="$(uname -m)"
  case "$uname_arch" in
    x86_64|amd64)
      printf 'amd64\n'
      ;;
    arm64|aarch64)
      printf 'arm64\n'
      ;;
    *)
      printf 'Error: unsupported architecture: %s\n' "$uname_arch" >&2
      exit 1
      ;;
  esac
}

resolve_version() {
  local requested_version
  requested_version="$1"

  if [[ "$requested_version" != "latest" ]]; then
    printf '%s\n' "$requested_version"
    return 0
  fi

  local latest_url
  latest_url="$(curl -fsSLI -o /dev/null -w '%{url_effective}' "https://github.com/${REPO}/releases/latest")"
  if [[ -z "$latest_url" ]]; then
    printf 'Error: failed to resolve latest release for %s\n' "$REPO" >&2
    exit 1
  fi

  basename "$latest_url"
}

resolve_install_dir() {
  if [[ -n "${LLMBUDGET_INSTALL_DIR:-}" ]]; then
    mkdir -p "$LLMBUDGET_INSTALL_DIR"
    printf '%s\n' "$LLMBUDGET_INSTALL_DIR"
    return 0
  fi

  local path_dir
  IFS=':' read -r -a path_dirs <<<"${PATH:-}"
  for path_dir in "${path_dirs[@]}"; do
    if [[ -n "$path_dir" && -d "$path_dir" && -w "$path_dir" ]]; then
      printf '%s\n' "$path_dir"
      return 0
    fi
  done

  if [[ -z "${HOME:-}" ]]; then
    printf 'Error: could not determine an install directory from PATH and HOME is not set\n' >&2
    exit 1
  fi

  local fallback_dir
  fallback_dir="${HOME}/.local/bin"
  mkdir -p "$fallback_dir"
  printf '%s\n' "$fallback_dir"
}

resolve_default_db_path() {
  local os_name
  os_name="$1"

  case "$os_name" in
    linux)
      if [[ -n "${XDG_DATA_HOME:-}" ]]; then
        printf '%s/llmbudget/llmbudget.sqlite3\n' "$XDG_DATA_HOME"
      else
        if [[ -z "${HOME:-}" ]]; then
          printf 'Error: HOME must be set when XDG_DATA_HOME is not available\n' >&2
          exit 1
        fi
        printf '%s/.local/share/llmbudget/llmbudget.sqlite3\n' "$HOME"
      fi
      ;;
    darwin)
      if [[ -z "${HOME:-}" ]]; then
        printf 'Error: HOME must be set when LLMBUDGET_DB_PATH is not provided\n' >&2
        exit 1
      fi
      printf '%s/Library/Application Support/llmbudget/llmbudget.sqlite3\n' "$HOME"
      ;;
  esac
}

main() {
  require_command curl
  require_command chmod
  require_command mktemp
  require_command mv
  require_command uname
  require_command basename

  local requested_version="${1:-${LLMBUDGET_VERSION:-latest}}"
  local os_name arch_name version asset_name install_dir target_path download_url temp_file db_path
  local -a bootstrap_args

  os_name="$(detect_os)"
  arch_name="$(detect_arch)"
  version="$(resolve_version "$requested_version")"
  asset_name="${os_name}-${arch_name}"
  install_dir="$(resolve_install_dir)"
  target_path="${install_dir}/${BINARY_NAME}"
  download_url="https://github.com/${REPO}/releases/download/${version}/${asset_name}"
  temp_file="$(mktemp)"

  printf 'Downloading %s from %s\n' "$asset_name" "$download_url"
  curl -fL "$download_url" -o "$temp_file"

  chmod 0755 "$temp_file"
  mv "$temp_file" "$target_path"

  if [[ -n "${LLMBUDGET_DB_PATH:-}" ]]; then
    db_path="$LLMBUDGET_DB_PATH"
  else
    db_path="$(resolve_default_db_path "$os_name")"
  fi

  bootstrap_args=(--bootstrap-only --db "$db_path")

  "$target_path" "${bootstrap_args[@]}"

  printf '\nInstallation complete.\n'
  printf '  Version: %s\n' "$version"
  printf '  Binary:  %s\n' "$target_path"
  printf '  SQLite:  %s\n' "$db_path"

  if [[ ":${PATH}:" != *":${install_dir}:"* ]]; then
    printf '  Note:    %s is not currently on PATH. Add it to run %s directly.\n' "$install_dir" "$BINARY_NAME"
  fi
}

main "$@"
