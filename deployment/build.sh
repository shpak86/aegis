#!/bin/bash

set -euo pipefail
IFS=$'\n\t'

# Default versions can be set via environment variables
AEGIS_VERSION="0.4.1"
BUILD_DIR="$(pwd)/build"

# List of required utilities
REQUIRED_CMDS=(go)

function usage() {
  cat <<EOF
Usage: $0 [options]
  -a VERSION   AEGIS version
  -h           Show this message and exit
EOF
  exit 1
}

# Parse options
while getopts "n:a:h" opt; do
  case "$opt" in
    a) AEGIS_VERSION="$OPTARG" ;;
    h) usage ;;
    *) usage ;;
  esac
done

# Check for required utilities
for cmd in "${REQUIRED_CMDS[@]}"; do
  if ! command -v "$cmd" &>/dev/null; then
    echo "Error: utility '$cmd' not found. Please install it and try again." >&2
    exit 2
  fi
done

# Build aegis package
RELEASE_DIR_NAME="aegis-${AEGIS_VERSION}"

echo "Building..."
rm -rf $RELEASE_DIR_NAME
mkdir -p $RELEASE_DIR_NAME $RELEASE_DIR_NAME/usr/bin
cp -r assets/* $RELEASE_DIR_NAME/
go build -o $RELEASE_DIR_NAME/usr/bin/aegis ../cmd/main.go
cd $RELEASE_DIR_NAME
tar -czf "../${RELEASE_DIR_NAME}.tar.gz" .

echo "Build completed: ${RELEASE_DIR_NAME}.tar.gz"