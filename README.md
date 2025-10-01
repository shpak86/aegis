# Aegis Bot Protection System

**⚠️ Important: The project is in active development and should not be used in production!**

## Overview

### What is Aegis?

Aegis is a bot detection and protection system that adds a layer of security to web or proxy servers. It works as standalone service and allows to webservers like Nginx and Apache to check for automated script requests.

The system operates in real time, blocking bots 1–2 seconds after detection. Configuration is managed via directives in plain text config files. Aegis provides Prometheus metrics and can be easily integrated with monitoring systems. Aegis works as a standalone service on the same server as nginx or as a remote service, making it easy to integrate into existing infrastructure.

### Playground

You can try Aegis protection on http://project-aegis.ru playground. 

There are following protections confgured:
- `GET` requests on `/` are protecting
- `POST` requests on `/articles/` are protecting and limited with 2 RPS 

## Key Features

- ✅ Real-time bot detection
- ✅ Rate limiting
- ✅ Token protection
- ✅ Basic logging and monitoring capabilities
- ✅ JavaScript challenge mechanism
- ✅ Captcha challenge mechanism
- ✅ Permanent tokens

## System Requirements

- **OS:** Linux system
- **Nginx:** version 1.24+
- **CPU:** 1+ core
- **RAM:** Minimum 200MB
- **Disk space:** 1GB available disk space for logs and cache

## Architecture


## Installation

### Manual Installation

Build the package using build script. You can specify Aegis version using -a flag.

```bash
cd deployment
./build.sh -a 0.4.0
```

Script will print path to the created package, for example: `Build completed: aegis-0.4.0.tar.gz`. Extract this package as a privileged user. 

```bash
sudo tar -C / -vxf build/aegis-0.4.0.tar.gz 
```

Enable and start aegis.

```bash
systemctl daemon-reload
systemctl enable aegis
systemctl start aegis
```

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

## Monitoring

Aegis serves `http://localhost:2048/metrics` endpoint to provide Prometheus metrics.

Available metrics:
- `antibot_response`
- `revoke_token`
- `token_request`
- `challenge_request`

## Configuration

### Aegis Configuration

Aegis should be configured in `/etc/aegis/config.json`.


#### Main Parameters

- **`address`** - allows you to change the address which antibot is serving. Bu default address is **localhost:2048**
- **`logger.level`** - configures verbosity of the logger. Possible values: `DEBUG`, `INFO`, `WARNING`, `ERROR`
- **`verification.type`** - verification method. Possible values: `js-challenge` or `captcha`
- **`verification.complexity`** - complexity of the challenge. For JS-challenge complexity determines the time required for the solution. For captcha it determines the number of images.
  - `easy` - easy
  - `medium` - optimal
  - `hard` - hard
- **`permanent_tokens`** - list of permanint tokens which can be used for trusted clients. Permanent token is a plain string which is somehow should be sent to the clients.

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
  ],
  "permanent_tokens": ["0faa199f-935e-411f-b9a8-939ff655bf8a", "1478a524-933b-40f4-b3d4-1d13a18afb1e"]
}
```

In this example:
- All **index.html** pages are protected, but the RPS limit is set too high to be reached
- This protection grants access to **index.html** only for clients with a valid token
- Any client is allowed to **GET** no more than 10 comments per second, and **POST** only 2 comments per second
- A client is also allowed to make up to 100 **GET** requests to all `/api` endpoints
- There are two permanent tokens are defined

### Nginx Configuration

Aegis works as an auth service so configure nginx for such kind of interaction.

1. Configure `/aegis/` locations
  ```nginx
  # This location is used for token issuing and verification
  location /aegis/token {
    proxy_pass http://localhost:6996/aegis/token;
    
    proxy_set_header X-Original-Url $request_uri;
    proxy_set_header X-Original-Method $request_method;
    proxy_set_header X-Original-Addr $remote_addr;
  }

  # This is internal location for requests analisys
  location = /aegis/auth {
    internal;

    proxy_pass http://127.0.0.1:6996/aegis/handlers/http;
    
    # proxy headers
    proxy_set_header X-Original-Url $request_uri;
    proxy_set_header X-Original-Method $request_method;
    proxy_set_header X-Original-Addr $remote_addr;
    # Perfomance tuning
    proxy_buffering off;
    proxy_connect_timeout 1s;
    proxy_read_timeout 3s;
  }

  # Redirect response      
  location @handle_redirect {
    return 302 $auth_redirect;
  }
  
  # Error respose
  location @handle_error {
    return 502 "Bad Gateway";
  }
  ```
2. Configure proxy for all protected locations
  ```nginx
  # Authorize all / and /index.html requests using Aegis service (/aegis/auth)
  location ~ ^/(index.html)?$ {
    auth_request /aegis/auth;

    proxy_pass http://localhost:8081;
    
    auth_request_set $auth_redirect $upstream_http_location;
    auth_request_set $auth_status $upstream_status;
  
    # Redirect to the Aegis challenge if response code is 4xx
    error_page 401 402 403 404 405 406 407 408 409 410 411 412 413 414 415 416 417 = @handle_redirect;
    error_page 500 502 503 504 = @handle_error;
    
    proxy_set_header X-Original-Url $request_uri;
    proxy_set_header X-Original-Method $request_method;
    proxy_set_header X-Original-Addr $remote_addr;
  }

  # Authorize all requests on /articles/ using Aegis service (/aegis/auth)
  location /articles/ {
    auth_request /aegis/auth;

    proxy_pass http://localhost:8081/articles/;
    
    auth_request_set $auth_redirect $upstream_http_location;
    auth_request_set $auth_status $upstream_status;

    # Redirect to the Aegis challenge if response code is 4xx
    error_page 401 402 403 404 405 406 407 408 409 410 411 412 413 414 415 416 417 = @handle_redirect;
    error_page 500 502 503 504 = @handle_error;
    
    proxy_set_header X-Original-Url $request_uri;
    proxy_set_header X-Original-Method $request_method;
    proxy_set_header X-Original-Addr $remote_addr;
  }
```

## Aegis Token
There are two challenges available - chaptcha and js-challenge. After passing the test client receives a unique token `AEGIS_TOKEN`. Token is associated with client's fingerprint and can be used only by the client which passed the original challnge. This mechanism makes impossible to share token between bots or pass the cahllenge by the solver host and send tokens to the crawlers.

### Captcha
To obtain a token, the client should select images by the text description.

### JS-challenge
To obtain a token, the client must perform a hash computation with a specified prefix as proof of work. The calculation and assignment of the token are fully automated and do not require any manual action.

## Support

- **Supported OS:** Any Linux distribution
- **Deployment modes:** Sidecar on the same server with nginx or standalone service
