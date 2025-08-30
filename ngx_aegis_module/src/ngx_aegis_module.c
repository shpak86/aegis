/*

* nginx-antibot - Module via preaccess phase for correct proxy_pass operation

* Fixed asynchronous body reading issue in access phase

* Fixed duplicate headers issue by handling special headers properly

* Fixed JSON escape sequences in response body

*/

#include <ngx_config.h>
#include <ngx_core.h>
#include <ngx_http.h>
#include <errno.h>
#include <sys/socket.h>

typedef struct {
    ngx_flag_t enable;
    ngx_str_t endpoint;
} ngx_aegis_loc_conf_t;

/* Structure for response headers */
typedef struct {
    ngx_str_t name;
    ngx_str_t value;
} ngx_aegis_header_t;

typedef struct {
    ngx_aegis_header_t *headers;
    ngx_uint_t count;
} ngx_aegis_headers_t;

/* Request context for asynchronous processing */
typedef struct {
    ngx_http_request_t *r;
    ngx_uint_t processing;
    ngx_uint_t done;
} ngx_aegis_ctx_t;

static ngx_int_t ngx_aegis_preaccess_handler(ngx_http_request_t *r);
static void ngx_aegis_body_handler(ngx_http_request_t *r);
static ngx_int_t ngx_aegis_process(ngx_http_request_t *r);
static ngx_int_t ngx_aegis_init(ngx_conf_t *cf);
static void* ngx_aegis_create_conf(ngx_conf_t *cf);
static char* ngx_aegis_merge_conf(ngx_conf_t *cf, void *parent, void *child);
static char* ngx_aegis_enable(ngx_conf_t *cf, ngx_command_t *cmd, void *conf);

#define ANTIBOT_LOG(level, r, fmt, ...) \
    ngx_log_error(level, r->connection->log, 0, "[aegis] " fmt, ##__VA_ARGS__)

static ngx_command_t ngx_aegis_commands[] = {
    { ngx_string("aegis_enable"),
      NGX_HTTP_LOC_CONF|NGX_CONF_NOARGS,
      ngx_aegis_enable,
      NGX_HTTP_LOC_CONF_OFFSET,
      offsetof(ngx_aegis_loc_conf_t, enable),
      NULL },

    { ngx_string("aegis_endpoint"),
      NGX_HTTP_LOC_CONF|NGX_CONF_TAKE1,
      ngx_conf_set_str_slot,
      NGX_HTTP_LOC_CONF_OFFSET,
      offsetof(ngx_aegis_loc_conf_t, endpoint),
      NULL },

    ngx_null_command
};

static ngx_http_module_t ngx_aegis_ctx = {
    NULL,                                  /* preconfiguration */
    ngx_aegis_init,                /* postconfiguration */

    NULL,                                  /* create main configuration */
    NULL,                                  /* init main configuration */

    NULL,                                  /* create server configuration */
    NULL,                                  /* merge server configuration */

    ngx_aegis_create_conf,         /* create location configuration */
    ngx_aegis_merge_conf           /* merge location configuration */
};

ngx_module_t ngx_aegis_module = {
    NGX_MODULE_V1,
    &ngx_aegis_ctx,
    ngx_aegis_commands,
    NGX_HTTP_MODULE,
    NULL, NULL, NULL, NULL, NULL, NULL, NULL,
    NGX_MODULE_V1_PADDING
};

/* Decode JSON escape sequences in string - NEW FUNCTION */
static ngx_int_t
json_unescape_string(ngx_pool_t *pool, u_char *src, size_t src_len, ngx_str_t *dst)
{
    if (src_len == 0) {
        dst->data = (u_char*)"";
        dst->len = 0;
        return NGX_OK;
    }
    
    /* Allocate memory for destination (worst case: same size as source) */
    dst->data = ngx_pnalloc(pool, src_len + 1);
    if (!dst->data) {
        dst->len = 0;
        return NGX_ERROR;
    }
    
    u_char *d = dst->data;
    u_char *s = src;
    u_char *end = src + src_len;
    
    while (s < end) {
        if (*s == '\\' && s + 1 < end) {
            s++; /* skip backslash */
            switch (*s) {
                case 'n':
                    *d++ = '\n';
                    break;
                case 'r': 
                    *d++ = '\r';
                    break;
                case 't':
                    *d++ = '\t';
                    break;
                case '\\':
                    *d++ = '\\';
                    break;
                case '"':
                    *d++ = '"';
                    break;
                case '/':
                    *d++ = '/';
                    break;
                case 'b':
                    *d++ = '\b';
                    break;
                case 'f':
                    *d++ = '\f';
                    break;
                case 'u':
                    /* Handle Unicode escape \uXXXX */
                    if (s + 4 < end) {
                        /* For simplicity, just copy the unicode sequence as-is */
                        /* Full Unicode support would require more complex parsing */
                        *d++ = '\\';
                        *d++ = 'u';
                        s++;
                        *d++ = *s++;
                        *d++ = *s++;
                        *d++ = *s++;
                        *d++ = *s;
                    } else {
                        /* Invalid escape sequence, copy as-is */
                        *d++ = '\\';
                        *d++ = *s;
                    }
                    break;
                default:
                    /* Unknown escape sequence, copy both characters */
                    *d++ = '\\';
                    *d++ = *s;
                    break;
            }
            s++;
        } else {
            *d++ = *s++;
        }
    }
    
    *d = '\0';
    dst->len = d - dst->data;
    return NGX_OK;
}

/* JSON string escaping */
static u_char *
escape_json_string(ngx_pool_t *pool, u_char *src, size_t len)
{
    if (len == 0) {
        u_char *dst = ngx_pnalloc(pool, 1);
        if (dst) dst[0] = '\0';
        return dst;
    }

    size_t escaped = 0;
    for (size_t i = 0; i < len; i++) {
        if (src[i] == '"' || src[i] == '\\' || src[i] < 32) {
            escaped++;
        }
    }

    u_char *dst = ngx_pnalloc(pool, len + escaped + 1);
    if (!dst) return NULL;

    u_char *p = dst;
    for (size_t i = 0; i < len; i++) {
        switch (src[i]) {
            case '"': *p++ = '\\'; *p++ = '"'; break;
            case '\\': *p++ = '\\'; *p++ = '\\'; break;
            case '\n': *p++ = '\\'; *p++ = 'n'; break;
            case '\r': *p++ = '\\'; *p++ = 'r'; break;
            case '\t': *p++ = '\\'; *p++ = 't'; break;
            default: *p++ = src[i]; break;
        }
    }
    *p = '\0';
    return dst;
}

/* Get request body from buffer chain */
static ngx_str_t
get_request_body(ngx_http_request_t *r)
{
    ngx_str_t body;
    
    /* Initialize empty result */
    body.len = 0;
    body.data = NULL;

    if (!r->request_body || !r->request_body->bufs) {
        return body;
    }

    size_t total_len = 0;
    ngx_chain_t *cl;

    /* Calculate total length */
    for (cl = r->request_body->bufs; cl; cl = cl->next) {
        ngx_buf_t *b = cl->buf;
        if (b && b->pos && b->last && b->last > b->pos) {
            total_len += b->last - b->pos;
        }
    }

    if (total_len == 0) return body;

    body.data = ngx_pnalloc(r->pool, total_len + 1);
    if (!body.data) {
        body.len = 0;
        body.data = NULL;
        return body;
    }

    u_char *p = body.data;
    for (cl = r->request_body->bufs; cl; cl = cl->next) {
        ngx_buf_t *b = cl->buf;
        if (b && b->pos && b->last && b->last > b->pos) {
            size_t len = b->last - b->pos;
            ngx_memcpy(p, b->pos, len);
            p += len;
        }
    }

    *p = '\0';
    body.len = total_len;
    return body;
}

/* Build JSON payload for antibot service */
static ngx_int_t
antibot_build_json(ngx_http_request_t *r, ngx_str_t *out)
{
    u_char *buf = ngx_pnalloc(r->pool, 16384);
    if (!buf) return NGX_ERROR;

    u_char *p = buf;
    u_char *end = buf + 16384;

    /* Client IP address */
    u_char addr[NGX_INET6_ADDRSTRLEN];
    size_t addr_len = ngx_sock_ntop(r->connection->sockaddr, r->connection->socklen,
                                   addr, NGX_INET6_ADDRSTRLEN, 0);

    /* Escaped strings for JSON */
    u_char *escaped_addr = escape_json_string(r->pool, addr, addr_len);
    u_char *escaped_url = escape_json_string(r->pool, r->unparsed_uri.data, r->unparsed_uri.len);
    u_char *escaped_method = escape_json_string(r->pool, r->method_name.data, r->method_name.len);

    if (!escaped_addr || !escaped_url || !escaped_method) {
        ANTIBOT_LOG(NGX_LOG_ERR, r, "failed to escape JSON strings");
        return NGX_ERROR;
    }

    ngx_str_t body = get_request_body(r);
    u_char *escaped_body = escape_json_string(r->pool, body.data, body.len);
    if (!escaped_body) escaped_body = (u_char*)"";

    ANTIBOT_LOG(NGX_LOG_DEBUG, r, "request body length: %uz", body.len);

    /* Main JSON structure */
    p = ngx_snprintf(p, end - p,
                    "{\"clientAddress\":\"%s\",\"url\":\"%s\",\"method\":\"%s\",\"body\":\"%s\",",
                    escaped_addr, escaped_url, escaped_method, escaped_body);

    /* Request headers */
    p = ngx_snprintf(p, end - p, "\"headers\":{");
    
    ngx_list_part_t *part = &r->headers_in.headers.part;
    ngx_table_elt_t *h = part->elts;
    ngx_uint_t first = 1;

    for (ngx_uint_t i = 0; ; i++) {
        if (i >= part->nelts) {
            if (part->next == NULL) break;
            part = part->next;
            h = part->elts;
            i = 0;
        }

        u_char *escaped_key = escape_json_string(r->pool, h[i].key.data, h[i].key.len);
        u_char *escaped_val = escape_json_string(r->pool, h[i].value.data, h[i].value.len);

        if (escaped_key && escaped_val) {
            if (!first) p = ngx_snprintf(p, end - p, ",");
            first = 0;
            p = ngx_snprintf(p, end - p, "\"%s\":\"%s\"", escaped_key, escaped_val);
        }
    }

    p = ngx_snprintf(p, end - p, "},\"cookies\":{");

    /* Extract cookies from Cookie header */
    first = 1;
    part = &r->headers_in.headers.part;
    h = part->elts;

    for (ngx_uint_t i = 0; ; i++) {
        if (i >= part->nelts) {
            if (part->next == NULL) break;
            part = part->next;
            h = part->elts;
            i = 0;
        }

        /* Process Cookie header */
        if (h[i].key.len == 6 && ngx_strncasecmp(h[i].key.data, (u_char*)"cookie", 6) == 0) {
            u_char *cookie_start = h[i].value.data;
            u_char *cookie_end = h[i].value.data + h[i].value.len;

            /* Parse cookie value pairs */
            while (cookie_start < cookie_end) {
                /* Skip whitespace and semicolons */
                while (cookie_start < cookie_end && (*cookie_start == ' ' || *cookie_start == ';')) {
                    cookie_start++;
                }

                if (cookie_start >= cookie_end) break;

                /* Find '=' separator */
                u_char *eq = cookie_start;
                while (eq < cookie_end && *eq != '=' && *eq != ';') eq++;

                if (eq >= cookie_end || *eq != '=') {
                    cookie_start = eq + 1;
                    continue;
                }

                /* Extract value */
                u_char *val_start = eq + 1;
                u_char *val_end = val_start;
                while (val_end < cookie_end && *val_end != ';') val_end++;

                u_char *escaped_key = escape_json_string(r->pool, cookie_start, eq - cookie_start);
                u_char *escaped_val = escape_json_string(r->pool, val_start, val_end - val_start);

                if (escaped_key && escaped_val) {
                    if (!first) p = ngx_snprintf(p, end - p, ",");
                    first = 0;
                    p = ngx_snprintf(p, end - p, "\"%s\":\"%s\"", escaped_key, escaped_val);
                }

                cookie_start = val_end;
            }
        }
    }

    p = ngx_snprintf(p, end - p, "}}");

    out->data = buf;
    out->len = p - buf;
    return NGX_OK;
}

/* Parse response headers from antibot JSON response */
static ngx_int_t
parse_response_headers(ngx_http_request_t *r, u_char *json_body, ngx_aegis_headers_t *headers)
{
    headers->headers = NULL;
    headers->count = 0;

    /* Find "headers" field in JSON */
    u_char *headers_pos = (u_char*)strstr((char*)json_body, "\"headers\":");
    if (!headers_pos) {
        ANTIBOT_LOG(NGX_LOG_DEBUG, r, "no headers field in response");
        return NGX_OK;
    }

    /* Find opening brace of headers object */
    u_char *start = (u_char*)strchr((char*)headers_pos + 10, '{');
    if (!start) {
        ANTIBOT_LOG(NGX_LOG_DEBUG, r, "headers field is not an object");
        return NGX_OK;
    }

    start++; /* skip '{' */

    /* Find closing brace of headers object */
    u_char *end = start;
    int brace_count = 1;
    while (*end && brace_count > 0) {
        if (*end == '{') brace_count++;
        else if (*end == '}') brace_count--;
        end++;
    }

    if (brace_count != 0) {
        ANTIBOT_LOG(NGX_LOG_ERR, r, "malformed headers object in response");
        return NGX_ERROR;
    }

    end--; /* point to '}' */

    /* Empty object {} */
    if (end <= start) {
        ANTIBOT_LOG(NGX_LOG_DEBUG, r, "empty headers object");
        return NGX_OK;
    }

    /* Count headers (number of commas + 1) */
    ngx_uint_t count = 1;
    for (u_char *p = start; p < end; p++) {
        if (*p == ',' && *(p-1) != '\\') count++;
    }

    /* Allocate memory for headers */
    headers->headers = ngx_pnalloc(r->pool, count * sizeof(ngx_aegis_header_t));
    if (!headers->headers) {
        ANTIBOT_LOG(NGX_LOG_ERR, r, "failed to allocate memory for headers");
        return NGX_ERROR;
    }

    /* Parse headers */
    u_char *current = start;
    ngx_uint_t idx = 0;

    while (current < end && idx < count) {
        /* Skip whitespace and commas */
        while (current < end && (*current == ' ' || *current == ',' || *current == '\t' || *current == '\n')) {
            current++;
        }

        if (current >= end) break;

        /* Find key (in quotes) */
        if (*current != '"') {
            ANTIBOT_LOG(NGX_LOG_ERR, r, "expected quoted key in headers");
            return NGX_ERROR;
        }

        current++; /* skip '"' */
        u_char *key_start = current;
        while (current < end && *current != '"') {
            if (*current == '\\') current++; /* skip escaped character */
            current++;
        }
        u_char *key_end = current;

        if (current >= end) {
            ANTIBOT_LOG(NGX_LOG_ERR, r, "unterminated key in headers");
            return NGX_ERROR;
        }

        current++; /* skip '"' */

        /* Skip whitespace and find ':' */
        while (current < end && (*current == ' ' || *current == '\t')) current++;
        if (current >= end || *current != ':') {
            ANTIBOT_LOG(NGX_LOG_ERR, r, "expected ':' after key in headers");
            return NGX_ERROR;
        }

        current++; /* skip ':' */

        /* Skip whitespace before value */
        while (current < end && (*current == ' ' || *current == '\t')) current++;

        /* Find value (in quotes) */
        if (current >= end || *current != '"') {
            ANTIBOT_LOG(NGX_LOG_ERR, r, "expected quoted value in headers");
            return NGX_ERROR;
        }

        current++; /* skip '"' */
        u_char *val_start = current;
        while (current < end && *current != '"') {
            if (*current == '\\') current++; /* skip escaped character */
            current++;
        }
        u_char *val_end = current;

        if (current >= end) {
            ANTIBOT_LOG(NGX_LOG_ERR, r, "unterminated value in headers");
            return NGX_ERROR;
        }

        current++; /* skip '"' */

        /* Create strings for key and value */
        size_t key_len = key_end - key_start;
        size_t val_len = val_end - val_start;

        headers->headers[idx].name.data = ngx_pnalloc(r->pool, key_len + 1);
        headers->headers[idx].value.data = ngx_pnalloc(r->pool, val_len + 1);

        if (!headers->headers[idx].name.data || !headers->headers[idx].value.data) {
            ANTIBOT_LOG(NGX_LOG_ERR, r, "failed to allocate memory for header %ui", idx);
            return NGX_ERROR;
        }

        ngx_memcpy(headers->headers[idx].name.data, key_start, key_len);
        headers->headers[idx].name.data[key_len] = '\0';
        headers->headers[idx].name.len = key_len;

        ngx_memcpy(headers->headers[idx].value.data, val_start, val_len);
        headers->headers[idx].value.data[val_len] = '\0';
        headers->headers[idx].value.len = val_len;

        ANTIBOT_LOG(NGX_LOG_DEBUG, r, "parsed header: %V: %V",
                   &headers->headers[idx].name, &headers->headers[idx].value);

        idx++;
    }

    headers->count = idx;
    ANTIBOT_LOG(NGX_LOG_INFO, r, "parsed %ui headers from antibot response", headers->count);
    return NGX_OK;
}

/* HTTP request to antibot service - FIXED VERSION WITH PROPER RESPONSE READING */
static ngx_int_t
antibot_call_service(ngx_http_request_t *r, ngx_str_t *json_req, int *resp_code,
                     ngx_str_t *resp_body, ngx_aegis_headers_t *resp_headers)
{
    /* Create TCP socket */
    int sock = socket(AF_INET, SOCK_STREAM, 0);
    if (sock == -1) {
        ANTIBOT_LOG(NGX_LOG_ERR, r, "socket() failed: %s", strerror(errno));
        return NGX_ERROR;
    }

    /* Setup server address */
    struct sockaddr_in addr;
    ngx_memzero(&addr, sizeof(addr));
    addr.sin_family = AF_INET;
    addr.sin_port = htons(6996);
    addr.sin_addr.s_addr = htonl(0x7f000001);

    /* Connect to antibot service */
    if (connect(sock, (struct sockaddr*)&addr, sizeof(addr)) < 0) {
        ANTIBOT_LOG(NGX_LOG_ERR, r, "connect() failed: %s", strerror(errno));
        close(sock);
        return NGX_ERROR;
    }

    /* Build HTTP request headers */
    u_char header[512];
    int hl = ngx_snprintf(header, sizeof(header),
                         "POST /api/v1/check HTTP/1.1\r\n"
                         "Host: localhost:6996\r\n"
                         "Content-Type: application/json\r\n"
                         "Content-Length: %uz\r\n"
                         "Connection: close\r\n\r\n",
                         json_req->len) - header;

    /* Send HTTP headers */
    if (send(sock, header, hl, 0) < 0) {
        ANTIBOT_LOG(NGX_LOG_ERR, r, "send() header failed: %s", strerror(errno));
        close(sock);
        return NGX_ERROR;
    }

    /* Send JSON payload */
    if (send(sock, json_req->data, json_req->len, 0) < 0) {
        ANTIBOT_LOG(NGX_LOG_ERR, r, "send() body failed: %s", strerror(errno));
        close(sock);
        return NGX_ERROR;
    }

    /* Receive response - FIXED: Read all data in loop */
    size_t buf_size = 32768; /* Увеличен размер буфера */
    u_char *buf = ngx_pnalloc(r->pool, buf_size);
    if (!buf) {
        ANTIBOT_LOG(NGX_LOG_ERR, r, "failed to allocate response buffer");
        close(sock);
        return NGX_ERROR;
    }

    size_t total_received = 0;
    ssize_t n;
    
    /* Read response in loop until all data received or connection closed */
    while (total_received < buf_size - 1) {
        n = recv(sock, buf + total_received, buf_size - total_received - 1, 0);
        
        if (n < 0) {
            ANTIBOT_LOG(NGX_LOG_ERR, r, "recv() failed: %s", strerror(errno));
            close(sock);
            return NGX_ERROR;
        }
        
        if (n == 0) {
            /* Connection closed by server */
            ANTIBOT_LOG(NGX_LOG_DEBUG, r, "connection closed by antibot service, received %uz bytes", total_received);
            break;
        }
        
        total_received += n;
        ANTIBOT_LOG(NGX_LOG_DEBUG, r, "received %z bytes, total %uz bytes", n, total_received);
        
        /* Check if we have received complete HTTP response */
        buf[total_received] = '\0';
        
        /* Look for end of HTTP response (\r\n\r\n) and complete body */
        u_char *body_start = (u_char*)strstr((char*)buf, "\r\n\r\n");
        if (body_start) {
            body_start += 4; /* Skip \r\n\r\n */
            
            /* Try to parse Content-Length if available */
            u_char *content_length_pos = (u_char*)strstr((char*)buf, "Content-Length:");
            if (content_length_pos) {
                content_length_pos += 15; /* Skip "Content-Length:" */
                while (*content_length_pos == ' ') content_length_pos++; /* Skip spaces */
                
                int content_length = atoi((char*)content_length_pos);
                size_t headers_length = body_start - buf;
                size_t expected_total = headers_length + content_length;
                
                ANTIBOT_LOG(NGX_LOG_DEBUG, r, "expected response length: %uz, received: %uz", 
                           expected_total, total_received);
                
                if (total_received >= expected_total) {
                    ANTIBOT_LOG(NGX_LOG_DEBUG, r, "received complete HTTP response with Content-Length");
                    break;
                }
            } else {
                /* No Content-Length, check for Connection: close */
                u_char *connection_pos = (u_char*)strstr((char*)buf, "Connection:");
                if (connection_pos) {
                    u_char *close_pos = (u_char*)strstr((char*)connection_pos, "close");
                    if (close_pos) {
                        /* Connection: close - read until connection is closed */
                        continue;
                    }
                }
            }
        }
    }

    close(sock);

    if (total_received == 0) {
        ANTIBOT_LOG(NGX_LOG_ERR, r, "no data received from antibot service");
        return NGX_ERROR;
    }

    buf[total_received] = '\0';
    ANTIBOT_LOG(NGX_LOG_DEBUG, r, "total received from antibot: %uz bytes", total_received);

    /* Find HTTP response body */
    u_char *body = (u_char*)strstr((char*)buf, "\r\n\r\n");
    if (!body) {
        ANTIBOT_LOG(NGX_LOG_ERR, r, "no HTTP body separator found in response");
        return NGX_ERROR;
    }

    body += 4;
    ANTIBOT_LOG(NGX_LOG_DEBUG, r, "antibot response body: %s", body);

    /* Parse JSON response - extract "code" field */
    u_char *code_pos = (u_char*)strstr((char*)body, "\"code\":");
    if (!code_pos) {
        ANTIBOT_LOG(NGX_LOG_ERR, r, "code field not found in response");
        return NGX_ERROR;
    }

    *resp_code = atoi((char*)code_pos + 7);

    /* Initialize response body */
    resp_body->data = (u_char*)"";
    resp_body->len = 0;

    /* Extract "body" field if present - WITH JSON UNESCAPE */
    u_char *body_pos = (u_char*)strstr((char*)body, "\"body\":");
    if (body_pos) {
        u_char *start = (u_char*)strchr((char*)body_pos + 7, '"');
        if (start) {
            start++; /* skip opening quote */
            
            /* Find end quote - handle escaped quotes properly */
            u_char *end = start;
            while (end < buf + total_received) {
                if (*end == '"' && (end == start || *(end-1) != '\\')) {
                    break;
                }
                end++;
            }
            
            if (end < buf + total_received && *end == '"') {
                size_t raw_len = end - start;
                
                ANTIBOT_LOG(NGX_LOG_DEBUG, r, "raw body from antibot (len=%uz): %.*s", 
                           raw_len, (int)ngx_min(raw_len, 100), start);
                
                /* Decode JSON escape sequences in body */
                ngx_str_t unescaped_body;
                if (json_unescape_string(r->pool, start, raw_len, &unescaped_body) == NGX_OK) {
                    *resp_body = unescaped_body;
                    ANTIBOT_LOG(NGX_LOG_DEBUG, r, "unescaped body (len=%uz): %.*s", 
                               resp_body->len, (int)ngx_min(resp_body->len, 100), resp_body->data);
                } else {
                    ANTIBOT_LOG(NGX_LOG_ERR, r, "failed to unescape JSON body");
                    /* Fallback to raw body */
                    resp_body->data = ngx_pnalloc(r->pool, raw_len + 1);
                    if (resp_body->data) {
                        ngx_memcpy(resp_body->data, start, raw_len);
                        resp_body->data[raw_len] = '\0';
                        resp_body->len = raw_len;
                    }
                }
            } else {
                ANTIBOT_LOG(NGX_LOG_ERR, r, "unterminated body field in JSON response");
            }
        }
    }

    /* Parse headers from response */
    if (parse_response_headers(r, body, resp_headers) != NGX_OK) {
        ANTIBOT_LOG(NGX_LOG_ERR, r, "failed to parse response headers");
        return NGX_ERROR;
    }

    return NGX_OK;
}

// /* HTTP request to antibot service - UPDATED VERSION */

// static ngx_int_t
// antibot_call_service(ngx_http_request_t *r, ngx_str_t *json_req, int *resp_code,
//                      ngx_str_t *resp_body, ngx_aegis_headers_t *resp_headers)
// {
//     /* Create TCP socket */
//     int sock = socket(AF_INET, SOCK_STREAM, 0);
//     if (sock == -1) {
//         ANTIBOT_LOG(NGX_LOG_ERR, r, "socket() failed: %s", strerror(errno));
//         return NGX_ERROR;
//     }

//     /* Setup server address */
//     struct sockaddr_in addr;
//     ngx_memzero(&addr, sizeof(addr));
//     addr.sin_family = AF_INET;
//     addr.sin_port = htons(6996);
//     addr.sin_addr.s_addr = htonl(0x7f000001);

//     /* Connect to antibot service */
//     if (connect(sock, (struct sockaddr*)&addr, sizeof(addr)) < 0) {
//         ANTIBOT_LOG(NGX_LOG_ERR, r, "connect() failed: %s", strerror(errno));
//         close(sock);
//         return NGX_ERROR;
//     }

//     /* Build HTTP request headers */
//     u_char header[512];
//     int hl = ngx_snprintf(header, sizeof(header),
//                          "POST /api/v1/check HTTP/1.1\r\n"
//                          "Host: localhost:6996\r\n"
//                          "Content-Type: application/json\r\n"
//                          "Content-Length: %uz\r\n"
//                          "Connection: close\r\n\r\n",
//                          json_req->len) - header;

//     /* Send HTTP headers */
//     if (send(sock, header, hl, 0) < 0) {
//         ANTIBOT_LOG(NGX_LOG_ERR, r, "send() header failed: %s", strerror(errno));
//         close(sock);
//         return NGX_ERROR;
//     }

//     /* Send JSON payload */
//     if (send(sock, json_req->data, json_req->len, 0) < 0) {
//         ANTIBOT_LOG(NGX_LOG_ERR, r, "send() body failed: %s", strerror(errno));
//         close(sock);
//         return NGX_ERROR;
//     }

//     /* Receive response */
//     u_char buf[8192];
//     ssize_t n = recv(sock, buf, sizeof(buf)-1, 0);
//     close(sock);

//     if (n <= 0) {
//         ANTIBOT_LOG(NGX_LOG_ERR, r, "recv() failed or empty response");
//         return NGX_ERROR;
//     }

//     buf[n] = '\0';

//     /* Find HTTP response body */
//     u_char *body = (u_char*)strstr((char*)buf, "\r\n\r\n");
//     if (!body) {
//         ANTIBOT_LOG(NGX_LOG_ERR, r, "no HTTP body separator found");
//         return NGX_ERROR;
//     }

//     body += 4;
//     ANTIBOT_LOG(NGX_LOG_DEBUG, r, "antibot response body: %s", body);

//     /* Parse JSON response - extract "code" field */
//     u_char *code_pos = (u_char*)strstr((char*)body, "\"code\":");
//     if (!code_pos) {
//         ANTIBOT_LOG(NGX_LOG_ERR, r, "code field not found in response");
//         return NGX_ERROR;
//     }

//     *resp_code = atoi((char*)code_pos + 7);

//     /* Initialize response body */
//     resp_body->data = (u_char*)"";
//     resp_body->len = 0;

//     /* Extract "body" field if present - WITH JSON UNESCAPE */
//     u_char *body_pos = (u_char*)strstr((char*)body, "\"body\":");
//     if (body_pos) {
//         u_char *start = (u_char*)strchr((char*)body_pos + 7, '"');
//         if (start) {
//             start++; /* skip opening quote */
//             u_char *end = (u_char*)strchr((char*)start, '"');
//             if (end) {
//                 size_t raw_len = end - start;
                
//                 ANTIBOT_LOG(NGX_LOG_DEBUG, r, "raw body from antibot (len=%uz): %.*s", 
//                            raw_len, (int)raw_len, start);
                
//                 /* Decode JSON escape sequences in body */
//                 ngx_str_t unescaped_body;
//                 if (json_unescape_string(r->pool, start, raw_len, &unescaped_body) == NGX_OK) {
//                     *resp_body = unescaped_body;
//                     ANTIBOT_LOG(NGX_LOG_DEBUG, r, "unescaped body (len=%uz): %.*s", 
//                                resp_body->len, (int)resp_body->len, resp_body->data);
//                 } else {
//                     ANTIBOT_LOG(NGX_LOG_ERR, r, "failed to unescape JSON body");
//                     /* Fallback to raw body */
//                     resp_body->data = ngx_pnalloc(r->pool, raw_len);
//                     if (resp_body->data) {
//                         ngx_memcpy(resp_body->data, start, raw_len);
//                         resp_body->len = raw_len;
//                     }
//                 }
//             }
//         }
//     }

//     /* Parse headers from response */
//     if (parse_response_headers(r, body, resp_headers) != NGX_OK) {
//         ANTIBOT_LOG(NGX_LOG_ERR, r, "failed to parse response headers");
//         return NGX_ERROR;
//     }

//     return NGX_OK;
// }
// */

/* Add response headers to nginx response - FIXED VERSION */
static ngx_int_t
add_response_headers(ngx_http_request_t *r, ngx_aegis_headers_t *resp_headers)
{
    if (!resp_headers || resp_headers->count == 0) {
        return NGX_OK;
    }

    ANTIBOT_LOG(NGX_LOG_INFO, r, "adding %ui headers to response", resp_headers->count);

    for (ngx_uint_t i = 0; i < resp_headers->count; i++) {
        ngx_str_t *name = &resp_headers->headers[i].name;
        ngx_str_t *value = &resp_headers->headers[i].value;

        ANTIBOT_LOG(NGX_LOG_DEBUG, r, "processing header: %V: %V", name, value);

        /* Handle special headers that have dedicated fields in headers_out */
        if (name->len == 12 && ngx_strncasecmp(name->data, (u_char*)"content-type", 12) == 0) {
            /* Set Content-Type in dedicated field */
            r->headers_out.content_type.len = value->len;
            r->headers_out.content_type.data = value->data;
            r->headers_out.content_type_lowcase = NULL;
            ANTIBOT_LOG(NGX_LOG_DEBUG, r, "set content_type field: %V", value);
            
        } else if (name->len == 14 && ngx_strncasecmp(name->data, (u_char*)"content-length", 14) == 0) {
            /* Set Content-Length numeric value */
            r->headers_out.content_length_n = ngx_atoof(value->data, value->len);
            /* Also create header element for Content-Length */
            if (r->headers_out.content_length) {
                r->headers_out.content_length->hash = 0; /* Remove existing */
            }
            r->headers_out.content_length = ngx_list_push(&r->headers_out.headers);
            if (r->headers_out.content_length) {
                r->headers_out.content_length->hash = 1;
                r->headers_out.content_length->key.len = name->len;
                r->headers_out.content_length->key.data = name->data;
                r->headers_out.content_length->value.len = value->len;
                r->headers_out.content_length->value.data = value->data;
            }
            ANTIBOT_LOG(NGX_LOG_DEBUG, r, "set content_length field: %V", value);
            
        } else if (name->len == 8 && ngx_strncasecmp(name->data, (u_char*)"location", 8) == 0) {
            /* Set Location header */
            if (r->headers_out.location) {
                r->headers_out.location->hash = 0; /* Remove existing */
            }
            r->headers_out.location = ngx_list_push(&r->headers_out.headers);
            if (r->headers_out.location) {
                r->headers_out.location->hash = 1;
                r->headers_out.location->key.len = name->len;
                r->headers_out.location->key.data = name->data;
                r->headers_out.location->value.len = value->len;
                r->headers_out.location->value.data = value->data;
            }
            ANTIBOT_LOG(NGX_LOG_DEBUG, r, "set location field: %V", value);
            
        } else if (name->len == 13 && ngx_strncasecmp(name->data, (u_char*)"last-modified", 13) == 0) {
            /* Set Last-Modified header */
            if (r->headers_out.last_modified) {
                r->headers_out.last_modified->hash = 0; /* Remove existing */
            }
            r->headers_out.last_modified = ngx_list_push(&r->headers_out.headers);
            if (r->headers_out.last_modified) {
                r->headers_out.last_modified->hash = 1;
                r->headers_out.last_modified->key.len = name->len;
                r->headers_out.last_modified->key.data = name->data;
                r->headers_out.last_modified->value.len = value->len;
                r->headers_out.last_modified->value.data = value->data;
            }
            ANTIBOT_LOG(NGX_LOG_DEBUG, r, "set last_modified field: %V", value);
            
        } else if (name->len == 4 && ngx_strncasecmp(name->data, (u_char*)"etag", 4) == 0) {
            /* Set ETag header */
            if (r->headers_out.etag) {
                r->headers_out.etag->hash = 0; /* Remove existing */
            }
            r->headers_out.etag = ngx_list_push(&r->headers_out.headers);
            if (r->headers_out.etag) {
                r->headers_out.etag->hash = 1;
                r->headers_out.etag->key.len = name->len;
                r->headers_out.etag->key.data = name->data;
                r->headers_out.etag->value.len = value->len;
                r->headers_out.etag->value.data = value->data;
            }
            ANTIBOT_LOG(NGX_LOG_DEBUG, r, "set etag field: %V", value);
            
        } else if (name->len == 7 && ngx_strncasecmp(name->data, (u_char*)"expires", 7) == 0) {
            /* Set Expires header */
            if (r->headers_out.expires) {
                r->headers_out.expires->hash = 0; /* Remove existing */
            }
            r->headers_out.expires = ngx_list_push(&r->headers_out.headers);
            if (r->headers_out.expires) {
                r->headers_out.expires->hash = 1;
                r->headers_out.expires->key.len = name->len;
                r->headers_out.expires->key.data = name->data;
                r->headers_out.expires->value.len = value->len;
                r->headers_out.expires->value.data = value->data;
            }
            ANTIBOT_LOG(NGX_LOG_DEBUG, r, "set expires field: %V", value);
            
        } else if (name->len == 6 && ngx_strncasecmp(name->data, (u_char*)"server", 6) == 0) {
            /* Set Server header */
            if (r->headers_out.server) {
                r->headers_out.server->hash = 0; /* Remove existing */
            }
            r->headers_out.server = ngx_list_push(&r->headers_out.headers);
            if (r->headers_out.server) {
                r->headers_out.server->hash = 1;
                r->headers_out.server->key.len = name->len;
                r->headers_out.server->key.data = name->data;
                r->headers_out.server->value.len = value->len;
                r->headers_out.server->value.data = value->data;
            }
            ANTIBOT_LOG(NGX_LOG_DEBUG, r, "set server field: %V", value);
            
        } else {
            /* Regular header - add to headers list */
            ngx_table_elt_t *h = ngx_list_push(&r->headers_out.headers);
            if (!h) {
                ANTIBOT_LOG(NGX_LOG_ERR, r, "failed to allocate header %ui", i);
                return NGX_ERROR;
            }

            h->hash = 1;
            h->key.len = name->len;
            h->key.data = name->data;
            h->value.len = value->len;
            h->value.data = value->data;
            ANTIBOT_LOG(NGX_LOG_DEBUG, r, "added regular header: %V: %V", &h->key, &h->value);
        }
    }

    return NGX_OK;
}

/* Send response from antibot to client - FIXED VERSION */
static ngx_int_t
antibot_send_response(ngx_http_request_t *r, int service_code, ngx_str_t *service_body,
                      ngx_aegis_headers_t *service_headers)
{
    /* Set response status */
    r->headers_out.status = service_code;
    r->headers_out.content_length_n = service_body->len;

    /* Add additional headers from antibot response FIRST */
    /* This allows antibot to override Content-Type and other headers */
    if (add_response_headers(r, service_headers) != NGX_OK) {
        ANTIBOT_LOG(NGX_LOG_ERR, r, "failed to add response headers");
        return NGX_HTTP_INTERNAL_SERVER_ERROR;
    }

    /* Only set default content type if antibot didn't provide one */
    if (r->headers_out.content_type.len == 0) {
        ngx_str_set(&r->headers_out.content_type, "text/plain; charset=utf-8");
        ANTIBOT_LOG(NGX_LOG_DEBUG, r, "set default content_type");
    }

    /* Create response buffer */
    ngx_buf_t *b = ngx_pcalloc(r->pool, sizeof(ngx_buf_t));
    if (!b) {
        ANTIBOT_LOG(NGX_LOG_ERR, r, "failed to allocate buffer");
        return NGX_HTTP_INTERNAL_SERVER_ERROR;
    }

    b->pos = service_body->data;
    b->last = service_body->data + service_body->len;
    b->memory = 1;
    b->last_buf = 1;

    /* Create output chain */
    ngx_chain_t out;
    out.buf = b;
    out.next = NULL;

    /* Send headers */
    ngx_int_t rc = ngx_http_send_header(r);
    if (rc == NGX_ERROR || rc > NGX_OK) {
        return rc;
    }

    /* Send body */
    return ngx_http_output_filter(r, &out);
}

/* Main antibot processing logic */
static ngx_int_t
ngx_aegis_process(ngx_http_request_t *r)
{
    ANTIBOT_LOG(NGX_LOG_INFO, r, "processing request: %V %V", &r->method_name, &r->uri);

    /* Build JSON payload for antibot service */
    ngx_str_t json_req;
    if (antibot_build_json(r, &json_req) != NGX_OK) {
        ANTIBOT_LOG(NGX_LOG_ERR, r, "failed to build JSON");
        return NGX_HTTP_INTERNAL_SERVER_ERROR;
    }

    /* Initialize response variables */
    int service_code = 0;
    ngx_str_t service_body;
    ngx_aegis_headers_t service_headers;

    service_body.data = (u_char*)"";
    service_body.len = 0;
    service_headers.headers = NULL;
    service_headers.count = 0;

    /* Call antibot service */
    if (antibot_call_service(r, &json_req, &service_code, &service_body, &service_headers) != NGX_OK) {
        ANTIBOT_LOG(NGX_LOG_ERR, r, "failed to call antibot service");
        return NGX_HTTP_INTERNAL_SERVER_ERROR;
    }

    ANTIBOT_LOG(NGX_LOG_INFO, r, "antibot returned code=%d", service_code);

    /* Mark processing as complete */
    ngx_aegis_ctx_t *ctx = ngx_http_get_module_ctx(r, ngx_aegis_module);
    if (ctx) {
        ctx->done = 1;
    }

    if (service_code == 0) {
        ANTIBOT_LOG(NGX_LOG_DEBUG, r, "antibot allowed request, continuing");
        return NGX_DECLINED; /* Continue processing */
    }

    /* Block request */
    ANTIBOT_LOG(NGX_LOG_INFO, r, "antibot blocked request with code %d", service_code);
    return antibot_send_response(r, service_code, &service_body, &service_headers);
}

/* Callback after reading request body - DO NOT finalize request! */
static void
ngx_aegis_body_handler(ngx_http_request_t *r)
{
    ANTIBOT_LOG(NGX_LOG_DEBUG, r, "body read complete, processing request");

    ngx_int_t rc = ngx_aegis_process(r);

    /* DO NOT call ngx_http_finalize_request for NGX_DECLINED!
     * Allow nginx to continue phase processing */
    if (rc == NGX_DECLINED) {
        /* Unlock request processing */
        r->count--;
        /* Restart phase processing */
        ngx_http_core_run_phases(r);
    } else {
        /* If blocking or error - finalize */
        ngx_http_finalize_request(r, rc);
    }
}

/* Preaccess phase handler */
static ngx_int_t
ngx_aegis_preaccess_handler(ngx_http_request_t *r)
{
    ngx_aegis_loc_conf_t *conf =
        ngx_http_get_module_loc_conf(r, ngx_aegis_module);

    if (!conf->enable) {
        return NGX_DECLINED;
    }

    ngx_aegis_ctx_t *ctx = ngx_http_get_module_ctx(r, ngx_aegis_module);

    /* If already processed - skip */
    if (ctx && ctx->done) {
        ANTIBOT_LOG(NGX_LOG_DEBUG, r, "antibot check already completed");
        return NGX_DECLINED;
    }

    if (ctx && ctx->processing) {
        ANTIBOT_LOG(NGX_LOG_DEBUG, r, "request already being processed");
        return NGX_DECLINED;
    }

    /* Create context */
    if (!ctx) {
        ctx = ngx_pcalloc(r->pool, sizeof(ngx_aegis_ctx_t));
        if (!ctx) {
            return NGX_ERROR;
        }

        ctx->r = r;
        ngx_http_set_ctx(r, ctx, ngx_aegis_module);
    }

    ctx->processing = 1;

    ANTIBOT_LOG(NGX_LOG_DEBUG, r, "antibot preaccess handler called for: %V %V", &r->method_name, &r->uri);

    /* For POST/PUT/PATCH read body asynchronously */
    if (r->method & (NGX_HTTP_POST|NGX_HTTP_PUT|NGX_HTTP_PATCH)) {
        ANTIBOT_LOG(NGX_LOG_DEBUG, r, "reading request body asynchronously");
        ngx_int_t rc = ngx_http_read_client_request_body(r, ngx_aegis_body_handler);
        if (rc >= NGX_HTTP_SPECIAL_RESPONSE) {
            return rc;
        }
        return NGX_DONE;
    }

    /* For GET process immediately */
    return ngx_aegis_process(r);
}

/* Module initialization */
static ngx_int_t
ngx_aegis_init(ngx_conf_t *cf)
{
    ngx_http_handler_pt *h;
    ngx_http_core_main_conf_t *cmcf;

    cmcf = ngx_http_conf_get_module_main_conf(cf, ngx_http_core_module);

    /* Register in PREACCESS phase instead of ACCESS */
    h = ngx_array_push(&cmcf->phases[NGX_HTTP_PREACCESS_PHASE].handlers);
    if (h == NULL) {
        return NGX_ERROR;
    }

    *h = ngx_aegis_preaccess_handler;
    return NGX_OK;
}

/* Configuration functions */
static void*
ngx_aegis_create_conf(ngx_conf_t *cf)
{
    ngx_aegis_loc_conf_t *conf = ngx_pcalloc(cf->pool, sizeof(*conf));
    if (!conf) return NULL;

    conf->enable = NGX_CONF_UNSET;
    return conf;
}

static char*
ngx_aegis_merge_conf(ngx_conf_t *cf, void *parent, void *child)
{
    ngx_aegis_loc_conf_t *prev = parent;
    ngx_aegis_loc_conf_t *conf = child;

    ngx_conf_merge_value(conf->enable, prev->enable, 0);
    ngx_conf_merge_str_value(conf->endpoint, prev->endpoint, "http://localhost:6996/api/v1/check");

    return NGX_CONF_OK;
}

/* Enable directive - DO NOT set content handler */
static char*
ngx_aegis_enable(ngx_conf_t *cf, ngx_command_t *cmd, void *conf)
{
    ngx_aegis_loc_conf_t *alcf = conf;
    alcf->enable = 1;

    /* Work ONLY through preaccess phase */
    return NGX_CONF_OK;
}