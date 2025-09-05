# Aegis Bot Protection System

**⚠️ Important: The project is in active development and should not be used in production!**

## Overview

### What is Aegis?

Aegis is a bot detection and protection system that adds a layer of security to an Nginx or Angie server. It consists of an nginx module along with the Aegis service. The module intercepts requests and controls their processing, while the Aegis service analyzes traffic and decides whether to block client requests based on the collected data.

The system operates in real time, blocking bots 1–2 seconds after detection. Configuration is managed via directives in plain text config files. Aegis provides Prometheus metrics and can be easily integrated with monitoring systems. Aegis can run as a sidecar on the same server as nginx or as a standalone service, making it easy to integrate into existing infrastructure.

## Key Features

- ✅ Real-time bot detection
- ✅ Rate limiting
- ✅ Token protection
- ✅ Basic logging and monitoring capabilities
- ✅ JavaScript challenge mechanism
- ✅ Captha

## System Requirements

- **OS:** Linux system
- **Nginx:** version 1.24+
- **CPU:** 1+ core
- **RAM:** Minimum 200MB
- **Disk space:** 1GB available disk space for logs and cache

## Architecture

```
┌─────────────┐ HTTP Request ┌──────────────────────┐
│   Client    │ ───────────► │      nginx           │
└─────────────┘              │ (ngx_aegis_module)   │
                             └──────────┬───────────┘
                                        │
                              Request + Client metadata
                                        ▼
                              ┌─────────────────┐
                              │                 │
                              │  Aegis Service  │
                              │                 │
                              └─────────┬───────┘
                                        │
                                    Verdict
                                        ▼
┌─────────────┐   Response    ┌───────────────────┐
│   Client    │ ◄──────────── │      nginx:       │
└─────────────┘               │  - Block (4xx)    │
                              │  - Redirect (3xx) │
                              │  - Allow (proxy)  │
                              └───────────────────┘
```

## Installation

### Manual Installation

Build the package using build script. You can specify Nginx version using -n flag. By default Nginx 1.28.0 is used.

```bash
cd deployment
./build.sh -n 1.24.0
```

Script will print path to the created package, for example: `Build completed: /home/user/repos/aegis/deployment/build/aegis_nginx_1.28.0-0.2.0.tar.gz`. Extract this package as a privileged user. 

```bash
sudo tar -C / -vxf build/aegis_nginx_1.28.0-0.2.0.tar.gz 
```

Enable and start aegis.

```bash
systemctl daemon-reload
systemctl enable aegis
systemctl start aegis
```

Nginx module is extracting to `/usr/share/nginx/modules/ngx_aegis_module.so`.

## Management

### Aegis Service

The service file is located at `/etc/systemd/system/aegis.service`. The service is managed using standard systemd commands.

```bash
# Start the service
systemctl start aegis

# Stop the service
systemctl stop aegis

# View service logs
journalctl -u aegis
```

### Nginx

The module writes messages into the standard Nginx log using the "aegis" tag. To filter these messages, execute the command:

```bash
tail -f /var/log/nginx/error.log | grep aegis
```

## Monitoring

Aegis serves `http://localhost:6996/metrics` endpoint to provide Prometheus metrics.

Available metrics:
- `antibot_response`
- `revoke_token`
- `token_request`
- `challenge_request`

## Configuration

### Aegis Configuration

Aegis should be configured in `/etc/aegis/config.json`.


#### Main Parameters

- **`address`** - allows you to change the address which antibot is serving. Bu default address is **localhost:6996**
- **`logger.level`** - configures verbosity of the logger. Possible values: `DEBUG`, `INFO`, `WARNING`, `ERROR`
- **`verification.type`** - verification method. Possible values: `js-challenge` or `captcha`
- **`verification.complexity`** - complexity of the JavaScript challenge:
  - `easy` - easy (less than 1 second to solve)
  - `medium` - optimal (1-5 seconds)
  - `hard` - hard (10-60 seconds to solve)

#### Protections

The list of protection definitions with fields:
- **`path`** - request path RegEx ⚠️ **Note:** Since the path is a regular expression, specifying `/user` will protect all paths containing this expression: `/user`, `/user/profile`, `/user/10042/profile`, `/some/other/user/profile`, `/username`, etc. Be careful and specify the most precise expressions possible.
- **`method`** - request method (`GET`, `POST`, etc.)
- **`rps`** - RPS limit for the client. If `rps` is not set or 0, protection will grant requests only from clients with valid cookie `AEGIS_TOKEN`.

#### Configuration Example

```json
{
  "logger": {
    "level": "info"
  },
  "verification": {
      "type": "js-challenge",
      "complexity": "medium"
  },
  "protections": [
    {
      "path": "/index.html$",
      "method": "GET"
    },
    {
      "path": "^/api/",
      "method": "GET",
      "rps": 100
    },
    {
      "path": "^/api/articles/\\d+/comments$",
      "method": "GET",
      "rps": 10
    },
    {
      "path": "^/api/articles/\\d+/comments$",
      "method": "POST",
      "rps": 2
    }
  ]
}
```

In this example:
- All **index.html** pages are protected, but the RPS limit is set too high to be reached
- This protection grants access to **index.html** only for clients with a valid token
- Any client is allowed to **GET** no more than 10 comments per second, and **POST** only 2 comments per second
- A client is also allowed to make up to 100 **GET** requests to all `/api` endpoints

### Nginx Configuration

Load aegis module with the `load_module` directive:

```nginx
load_module /usr/share/nginx/modules/ngx_aegis_module.so;
```

Aegis protects only endpoints with the `aegis_enable` directive, so you need to add this directive to all endpoints you want to protect.

**Important:** The `/aegis` endpoint is served by the antibot itself for service purposes, so it **must** be added to the configuration.

```nginx
# Antibot endpoint (required)
location /aegis/ {
    aegis_enable;
}

# Static files
location /downloads/ {
    aegis_enable;
}

# Reverse proxy
location /api/ {
    aegis_enable;
    proxy_pass http://backend-host/api/;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
}
```

## Aegis Token
There are two challenges available - chaptcha and js-challenge. 

### Captcha
To obtain a token, the client should select images by the text description.

### JS-challenge
To obtain a token, the client must perform a hash computation with a specified prefix as proof of work. The calculation and assignment of the token are fully automated and do not require any manual action.

## Changelog

### Version 0.2.0 (Septemper 5, 2025)

#### Added
- Captcha challenge

### Version 0.1.1 (August 4, 2025)

#### Changed
- Refactoring

### Version 0.1.0 (August 28, 2025)

#### Added
- Basic Aegis service
- Basic ngx_aegis_module
- Rate limiting
- JS challenge
- Client fingerprint
- Prometheus metrics

## Support

- **Supported Nginx versions:** 1.24+, 1.26+, 1.28+
- **Supported OS:** Any Linux distribution
- **Deployment modes:** Sidecar on the same server with nginx or standalone service

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

```
MIT License

Copyright (c) 2025 Aegis Bot Protection System

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```