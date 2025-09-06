# nginx-aegis Module - Quick Build

## Build Requirements

- gcc, make, wget
- nginx-compatible system (Linux, FreeBSD, macOS)
- Compatible with nginx 1.24+

## Quick Build

```bash
make
```

This will:
1. Download nginx source
2. Configure with aegis module
3. Build the dynamic module
4. Show path to .so file

## Output

The module will be built at:
```
./build/nginx-<version>/objs/ngx_http_aegis_module.so
```

## Installation

```bash
make install
```

This installs the module to standard nginx module paths and shows usage instructions.

## Custom nginx Version

```bash
NGINX_VERSION=1.26.0 make
```
