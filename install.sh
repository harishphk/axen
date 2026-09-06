#!/bin/sh
set -e

# Axen installer script for macOS and Linux

GITHUB_REPO="harishphk/axen"
BINARY_NAME="axen"
TEMP_DIR=""

cleanup() {
  if [ -n "${TEMP_DIR}" ] && [ -d "${TEMP_DIR}" ]; then
    rm -rf "${TEMP_DIR}"
  fi
}
trap cleanup EXIT INT TERM

detect_platform() {
  OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
  case "${OS}" in
    darwin)  PLATFORM="darwin" ;;
    linux)   PLATFORM="linux" ;;
    *)       echo "Unsupported operating system: ${OS}"; exit 1 ;;
  esac

  ARCH="$(uname -m)"
  case "${ARCH}" in
    x86_64|amd64)  ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *)             echo "Unsupported architecture: ${ARCH}"; exit 1 ;;
  esac
}

fetch_latest_tag() {
  echo "Detecting latest release..."
  LATEST_TAG=$(curl -s "https://api.github.com/repos/${GITHUB_REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
  if [ -z "${LATEST_TAG}" ]; then
    # Fallback if GitHub API is rate-limited
    LATEST_TAG=$(curl -s -o /dev/null -w "%{redirect_url}" "https://github.com/${GITHUB_REPO}/releases/latest" | awk -F'/' '{print $NF}')
  fi

  if [ -z "${LATEST_TAG}" ]; then
    echo "Error: Could not retrieve latest release tag."
    exit 1
  fi
}

check_current_version() {
  if [ "${FORCE:-0}" != "1" ] && command -v "${BINARY_NAME}" >/dev/null 2>&1; then
    CURRENT_VERSION=$("${BINARY_NAME}" --version 2>/dev/null | awk '{print $NF}')
    if [ -n "${CURRENT_VERSION}" ] && { [ "${CURRENT_VERSION}" = "${LATEST_TAG}" ] || [ "${CURRENT_VERSION}" = "${LATEST_TAG#v}" ]; }; then
      echo "Axen is already up to date (${CURRENT_VERSION})."
      echo "To force reinstall, run: curl -fsSL https://raw.githubusercontent.com/${GITHUB_REPO}/main/install.sh | FORCE=1 sh"
      exit 0
    fi
  fi
}

download_and_verify() {
  VERSION_NUM="${LATEST_TAG#v}"
  FILENAME="axen_${VERSION_NUM}_${PLATFORM}_${ARCH}.tar.gz"
  DOWNLOAD_URL="https://github.com/${GITHUB_REPO}/releases/download/${LATEST_TAG}/${FILENAME}"
  CHECKSUM_URL="https://github.com/${GITHUB_REPO}/releases/download/${LATEST_TAG}/checksums.txt"

  TEMP_DIR=$(mktemp -d)
  cd "${TEMP_DIR}"

  echo "Downloading from ${DOWNLOAD_URL}..."
  if ! curl -fsSL -o "${FILENAME}" "${DOWNLOAD_URL}"; then
    echo "Error: Download failed. Release binary may not be available for this platform."
    exit 1
  fi

  echo "Downloading checksums..."
  if ! curl -fsSL -o "checksums.txt" "${CHECKSUM_URL}"; then
    echo "Error: Could not download checksums file."
    exit 1
  fi

  echo "Verifying checksum..."
  if command -v sha256sum > /dev/null 2>&1; then
    if ! sha256sum --check --ignore-missing checksums.txt; then
      echo "Error: Checksum verification failed. The downloaded file may be corrupted or tampered with."
      exit 1
    fi
  elif command -v shasum > /dev/null 2>&1; then
    if ! shasum -a 256 --check --ignore-missing checksums.txt; then
      echo "Error: Checksum verification failed. The downloaded file may be corrupted or tampered with."
      exit 1
    fi
  else
    echo "Warning: No checksum utility found (sha256sum or shasum). Skipping verification."
  fi

  echo "Extracting..."
  tar -xzf "${FILENAME}"
}

install_binary() {
  if [ -w "/usr/local/bin" ]; then
    INSTALL_DIR="/usr/local/bin"
  else
    INSTALL_DIR="${HOME}/.local/bin"
    mkdir -p "${INSTALL_DIR}"
  fi

  echo "Installing to ${INSTALL_DIR}/${BINARY_NAME}..."
  cp -f "${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
  chmod +x "${INSTALL_DIR}/${BINARY_NAME}"

  echo "Axen ${LATEST_TAG} has been installed successfully to ${INSTALL_DIR}/${BINARY_NAME}!"
  if [ "${INSTALL_DIR}" = "${HOME}/.local/bin" ]; then
    case :$PATH: in
      *:"${INSTALL_DIR}":*) ;;
      *) echo "Warning: ${INSTALL_DIR} is not in your PATH. You may need to add it to your shell configuration (e.g. ~/.bashrc or ~/.zshrc)." ;;
    esac
  fi
}

main() {
  detect_platform
  fetch_latest_tag
  check_current_version
  echo "Installing Axen ${LATEST_TAG} (${PLATFORM}/${ARCH})..."
  download_and_verify
  install_binary
}

main "$@"
