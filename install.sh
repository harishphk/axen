#!/bin/sh
set -e

# Axen installer script for macOS and Linux

# Define repository info
GITHUB_REPO="axen/axen"
BINARY_NAME="axen"

# Detect OS
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "${OS}" in
  darwin)  PLATFORM="darwin" ;;
  linux)   PLATFORM="linux" ;;
  *)       echo "Unsupported operating system: ${OS}"; exit 1 ;;
esac

# Detect architecture
ARCH="$(uname -m)"
case "${ARCH}" in
  x86_64|amd64)  ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *)             echo "Unsupported architecture: ${ARCH}"; exit 1 ;;
esac

echo "Detecting latest release..."
# Get latest release from GitHub API
LATEST_TAG=$(curl -s https://api.github.com/repos/${GITHUB_REPO}/releases/latest | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
if [ -z "${LATEST_TAG}" ]; then
  # Fallback to fetching via redirect if GitHub API is rate-limited
  LATEST_TAG=$(curl -s -o /dev/null -w "%{redirect_url}" https://github.com/${GITHUB_REPO}/releases/latest | awk -F'/' '{print $NF}')
fi

if [ -z "${LATEST_TAG}" ]; then
  echo "Error: Could not retrieve latest release tag."
  exit 1
fi

echo "Installing Axen ${LATEST_TAG} (${PLATFORM}/${ARCH})..."

VERSION_NUM="${LATEST_TAG#v}"
FILENAME="axen_${VERSION_NUM}_${PLATFORM}_${ARCH}.tar.gz"
DOWNLOAD_URL="https://github.com/${GITHUB_REPO}/releases/download/${LATEST_TAG}/${FILENAME}"

# Temporary download directory
TEMP_DIR=$(mktemp -d)
cd "${TEMP_DIR}"

CHECKSUM_URL="https://github.com/${GITHUB_REPO}/releases/download/${LATEST_TAG}/checksums.txt"

echo "Downloading from ${DOWNLOAD_URL}..."
if ! curl -sSL -o "${FILENAME}" "${DOWNLOAD_URL}"; then
  echo "Error: Download failed. Release binary may not be available for this platform."
  exit 1
fi

echo "Downloading checksums..."
if ! curl -sSL -o "checksums.txt" "${CHECKSUM_URL}"; then
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
  # macOS fallback
  if ! shasum -a 256 --check --ignore-missing checksums.txt; then
    echo "Error: Checksum verification failed. The downloaded file may be corrupted or tampered with."
    exit 1
  fi
else
  echo "Warning: No checksum utility found (sha256sum or shasum). Skipping verification."
fi

echo "Extracting..."
tar -xzf "${FILENAME}"

# Determine target directory
if [ -w "/usr/local/bin" ]; then
  INSTALL_DIR="/usr/local/bin"
  SUDO=""
else
  INSTALL_DIR="${HOME}/.local/bin"
  SUDO=""
  mkdir -p "${INSTALL_DIR}"
fi

echo "Installing to ${INSTALL_DIR}/${BINARY_NAME}..."
${SUDO} cp "${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
${SUDO} chmod +x "${INSTALL_DIR}/${BINARY_NAME}"

# Clean up
cd - > /dev/null
rm -rf "${TEMP_DIR}"

echo "Axen ${LATEST_TAG} has been installed successfully to ${INSTALL_DIR}/${BINARY_NAME}!"
if [ "${INSTALL_DIR}" = "${HOME}/.local/bin" ]; then
  case :$PATH: in
    *:"${INSTALL_DIR}":*) ;;
    *) echo "Warning: ${INSTALL_DIR} is not in your PATH. You may need to add it to your shell configuration (e.g. ~/.bashrc or ~/.zshrc)." ;;
  esac
fi
