#!/bin/bash -x

set -euo pipefail
IFS=$'\n\t'

# Default versions can be set via environment variables
NGINX_VERSION="1.28.0"
AEGIS_VERSION="0.2.3"
MODULE_DIR_NAME="ngx_http_aegis_module"
BUILD_DIR="$(pwd)/build"
TEMP_DIR="$(pwd)/temp"

# List of required utilities
REQUIRED_CMDS=(curl tar gcc make go)

function usage() {
  cat <<EOF
Usage: $0 [options]
  -n VERSION   NGINX version
  -a VERSION   AEGIS version
  -h           Show this message and exit
EOF
  exit 1
}

# Parse options
while getopts "n:a:h" opt; do
  case "$opt" in
    n) NGINX_VERSION="$OPTARG" ;;
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

RELEASE_DIR_NAME="aegis_nginx_${NGINX_VERSION}-${AEGIS_VERSION}"

echo "Building ngx_http_aegis_module (NGINX $NGINX_VERSION) and aegis ($AEGIS_VERSION)"

# Prepare directories
rm -rf "$TEMP_DIR" "$BUILD_DIR"
mkdir -p "$TEMP_DIR" "$BUILD_DIR/${MODULE_DIR_NAME}-$NGINX_VERSION" "$BUILD_DIR/aegis" "$TEMP_DIR/nginx-${NGINX_VERSION}/${MODULE_DIR_NAME}"

# Copy module files
echo "Copying $MODULE_DIR_NAME module"
cp -r "../${MODULE_DIR_NAME}" "$TEMP_DIR/nginx-${NGINX_VERSION}"
pushd "$TEMP_DIR/nginx-${NGINX_VERSION}/${MODULE_DIR_NAME}" >/dev/null
NGINX_VERSION=${NGINX_VERSION} make build
popd >/dev/null
cp $TEMP_DIR/nginx-${NGINX_VERSION}/${MODULE_DIR_NAME}/build/nginx-${NGINX_VERSION}/objs/ngx_http_aegis_module.so "$BUILD_DIR/${MODULE_DIR_NAME}-$NGINX_VERSION/"

# Build aegis binary
echo "Building aegis"
pushd .. >/dev/null
go build -o "${BUILD_DIR}/aegis/aegis" cmd/main.go
popd >/dev/null

# Prepare release archive
echo "Creating release package $RELEASE_DIR_NAME"
RELEASE_ROOT="$BUILD_DIR/$RELEASE_DIR_NAME"
mkdir -p "$RELEASE_ROOT/usr/bin" "$RELEASE_ROOT/usr/share/nginx/modules"
cp -r assets/* "$RELEASE_ROOT/"
cp "$BUILD_DIR/aegis/aegis" "$RELEASE_ROOT/usr/bin/"
cp "$BUILD_DIR/${MODULE_DIR_NAME}-$NGINX_VERSION/${MODULE_DIR_NAME}.so" "$RELEASE_ROOT/usr/share/nginx/modules/"
pushd "$RELEASE_ROOT" >/dev/null
tar -czf "../${RELEASE_DIR_NAME}.tar.gz" .
popd >/dev/null

echo "Build completed: $BUILD_DIR/${RELEASE_DIR_NAME}.tar.gz"