/*
 * nginx-aegis module - COOKIE PARSING FIXED
 * Advanced antibot integration module for nginx 1.24+
 * 
 * Copyright (c) 2025 nginx-aegis
 * Licensed under MIT License
 */

#include <ngx_config.h>
#include <ngx_core.h>
#include <ngx_http.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <unistd.h>

/* Logging macro */
#define AEGIS_LOG(level, log, fmt, ...) \
    ngx_log_error(level, log, 0, "[aegis] " fmt, ##__VA_ARGS__)

/* Buffer limits for Aegis responses */
#define AEGIS_MIN_BUFFER_SIZE   (8 * 1024)      /* 8KB minimum */
#define AEGIS_MAX_BUFFER_SIZE   (500 * 1024)    /* 500KB maximum */
#define AEGIS_INITIAL_BUFFER    (64 * 1024)     /* 64KB initial */

/* Module configuration structure */
typedef struct {
    ngx_flag_t  enable;
    ngx_str_t   endpoint;
    ngx_uint_t  timeout;
    ngx_flag_t  log_blocked;
} ngx_http_aegis_loc_conf_t;

/* Request context structure */
typedef struct {
    ngx_http_request_t *r;
    ngx_uint_t          processing;
    ngx_uint_t          done;
    ngx_uint_t          result;
} ngx_http_aegis_ctx_t;

/* Header structure for Aegis response */
typedef struct {
    ngx_str_t name;
    ngx_str_t value;
} ngx_http_aegis_header_t;

/* Aegis response structure */
typedef struct {
    ngx_int_t                   code;
    ngx_str_t                   body;
    ngx_array_t                *headers;  /* array of ngx_http_aegis_header_t */
} ngx_http_aegis_response_t;

/* Forward declarations */
static ngx_int_t ngx_http_aegis_handler(ngx_http_request_t *r);
static void ngx_http_aegis_body_handler(ngx_http_request_t *r);
static ngx_int_t ngx_http_aegis_process(ngx_http_request_t *r);
static ngx_int_t ngx_http_aegis_send_request(ngx_http_request_t *r, ngx_str_t *payload, ngx_http_aegis_response_t *aegis_resp);
static ngx_int_t ngx_http_aegis_parse_response(ngx_http_request_t *r, u_char *data, size_t len, ngx_http_aegis_response_t *resp);
static ngx_int_t ngx_http_aegis_set_headers(ngx_http_request_t *r, ngx_http_aegis_response_t *resp);
static ngx_int_t ngx_http_aegis_build_json_payload(ngx_http_request_t *r, ngx_str_t *payload);
static u_char *ngx_http_aegis_escape_json_string(ngx_pool_t *pool, u_char *src, size_t len);
static ngx_int_t ngx_http_aegis_simple_json_get_int(u_char *json, size_t len, const char *key, ngx_int_t *value);
static ngx_int_t ngx_http_aegis_simple_json_get_str(u_char *json, size_t len, const char *key, ngx_str_t *value, ngx_pool_t *pool);
static ngx_int_t ngx_http_aegis_parse_headers_json(u_char *json, size_t len, ngx_array_t *headers, ngx_pool_t *pool);
static ngx_int_t ngx_http_aegis_html_decode(ngx_pool_t *pool, ngx_str_t *src, ngx_str_t *dst);
static ngx_int_t ngx_http_aegis_json_unescape(ngx_pool_t *pool, ngx_str_t *src, ngx_str_t *dst);

/* Configuration functions */
static void *ngx_http_aegis_create_loc_conf(ngx_conf_t *cf);
static char *ngx_http_aegis_merge_loc_conf(ngx_conf_t *cf, void *parent, void *child);
static ngx_int_t ngx_http_aegis_init(ngx_conf_t *cf);

/* Custom directive handlers */
static char *ngx_http_aegis_enable(ngx_conf_t *cf, ngx_command_t *cmd, void *conf);
static char *ngx_http_aegis_log_blocked(ngx_conf_t *cf, ngx_command_t *cmd, void *conf);

/* Module directives */
static ngx_command_t ngx_http_aegis_commands[] = {
    {
        ngx_string("aegis_enable"),
        NGX_HTTP_LOC_CONF|NGX_CONF_NOARGS,
        ngx_http_aegis_enable,
        NGX_HTTP_LOC_CONF_OFFSET,
        0,
        NULL
    },
    {
        ngx_string("aegis_endpoint"),
        NGX_HTTP_LOC_CONF|NGX_CONF_TAKE1,
        ngx_conf_set_str_slot,
        NGX_HTTP_LOC_CONF_OFFSET,
        offsetof(ngx_http_aegis_loc_conf_t, endpoint),
        NULL
    },
    {
        ngx_string("aegis_timeout"),
        NGX_HTTP_LOC_CONF|NGX_CONF_TAKE1,
        ngx_conf_set_num_slot,
        NGX_HTTP_LOC_CONF_OFFSET,
        offsetof(ngx_http_aegis_loc_conf_t, timeout),
        NULL
    },
    {
        ngx_string("aegis_log_blocked"),
        NGX_HTTP_LOC_CONF|NGX_CONF_NOARGS,
        ngx_http_aegis_log_blocked,
        NGX_HTTP_LOC_CONF_OFFSET,
        0,
        NULL
    },
    ngx_null_command
};

/* Module context */
static ngx_http_module_t ngx_http_aegis_module_ctx = {
    NULL,                          /* preconfiguration */
    ngx_http_aegis_init,           /* postconfiguration */
    NULL,                          /* create main configuration */
    NULL,                          /* init main configuration */
    NULL,                          /* create server configuration */
    NULL,                          /* merge server configuration */
    ngx_http_aegis_create_loc_conf, /* create location configuration */
    ngx_http_aegis_merge_loc_conf   /* merge location configuration */
};

/* Module definition */
ngx_module_t ngx_http_aegis_module = {
    NGX_MODULE_V1,
    &ngx_http_aegis_module_ctx,    /* module context */
    ngx_http_aegis_commands,       /* module directives */
    NGX_HTTP_MODULE,               /* module type */
    NULL,                          /* init master */
    NULL,                          /* init module */
    NULL,                          /* init process */
    NULL,                          /* init thread */
    NULL,                          /* exit thread */
    NULL,                          /* exit process */
    NULL,                          /* exit master */
    NGX_MODULE_V1_PADDING
};

/* Custom directive handler for aegis_enable */
static char *
ngx_http_aegis_enable(ngx_conf_t *cf, ngx_command_t *cmd, void *conf)
{
    ngx_http_aegis_loc_conf_t *alcf = conf;

    if (alcf->enable != NGX_CONF_UNSET) {
        return "is duplicate";
    }

    alcf->enable = 1;

    return NGX_CONF_OK;
}

/* Custom directive handler for aegis_log_blocked */
static char *
ngx_http_aegis_log_blocked(ngx_conf_t *cf, ngx_command_t *cmd, void *conf)
{
    ngx_http_aegis_loc_conf_t *alcf = conf;

    if (alcf->log_blocked != NGX_CONF_UNSET) {
        return "is duplicate";
    }

    alcf->log_blocked = 1;

    return NGX_CONF_OK;
}

/* Main handler - registered in NGX_HTTP_PREACCESS_PHASE */
static ngx_int_t
ngx_http_aegis_handler(ngx_http_request_t *r)
{
    ngx_http_aegis_loc_conf_t  *alcf;
    ngx_http_aegis_ctx_t       *ctx;
    ngx_int_t                   rc;

    AEGIS_LOG(NGX_LOG_DEBUG, r->connection->log, "handler started for %V %V", 
              &r->method_name, &r->uri);

    alcf = ngx_http_get_module_loc_conf(r, ngx_http_aegis_module);

    if (!alcf->enable) {
        AEGIS_LOG(NGX_LOG_DEBUG, r->connection->log, "module disabled");
        return NGX_DECLINED;
    }

    /* Skip subrequests */
    if (r != r->main) {
        return NGX_DECLINED;
    }

    ctx = ngx_http_get_module_ctx(r, ngx_http_aegis_module);

    if (ctx != NULL) {
        if (ctx->done) {
            return ctx->result;
        }
        /* Already processing */
        return NGX_DONE;
    }

    /* Create context */
    ctx = ngx_pcalloc(r->pool, sizeof(ngx_http_aegis_ctx_t));
    if (ctx == NULL) {
        AEGIS_LOG(NGX_LOG_ERR, r->connection->log, "failed to allocate context");
        return NGX_ERROR;
    }

    ctx->r = r;
    ctx->processing = 1;
    ctx->done = 0;
    ctx->result = NGX_DECLINED;

    ngx_http_set_ctx(r, ctx, ngx_http_aegis_module);

    /* Check if we need to read request body */
    if (r->method == NGX_HTTP_POST || r->method == NGX_HTTP_PUT || 
        r->method == NGX_HTTP_PATCH) {

        AEGIS_LOG(NGX_LOG_DEBUG, r->connection->log, "reading body for %V", &r->method_name);

        rc = ngx_http_read_client_request_body(r, ngx_http_aegis_body_handler);

        if (rc >= NGX_HTTP_SPECIAL_RESPONSE) {
            AEGIS_LOG(NGX_LOG_ERR, r->connection->log, "failed to read body: %i", rc);
            ctx->done = 1;
            ctx->result = rc;
            return rc;
        }

        if (rc == NGX_AGAIN) {
            return NGX_DONE;
        }

        /* Body read synchronously */
        return ngx_http_aegis_process(r);
    }

    /* No body needed, process immediately */
    return ngx_http_aegis_process(r);
}

/* Body reading completion handler */
static void
ngx_http_aegis_body_handler(ngx_http_request_t *r)
{
    ngx_http_aegis_ctx_t *ctx;
    ngx_int_t             rc;

    ctx = ngx_http_get_module_ctx(r, ngx_http_aegis_module);
    if (ctx == NULL) {
        AEGIS_LOG(NGX_LOG_ERR, r->connection->log, "context not found in body handler");
        ngx_http_finalize_request(r, NGX_HTTP_INTERNAL_SERVER_ERROR);
        return;
    }

    rc = ngx_http_aegis_process(r);

    ctx->done = 1;
    ctx->result = rc;

    if (rc == NGX_DECLINED) {
        /* Allow request to continue */
        ctx->processing = 0;
        r->count--;
        ngx_http_core_run_phases(r);
    } else {
        /* Block or error */
        ctx->processing = 0;
        ngx_http_finalize_request(r, rc);
    }
}

/* Main processing function - WITH FULL HEADERS SUPPORT */
static ngx_int_t
ngx_http_aegis_process(ngx_http_request_t *r)
{
    ngx_str_t                   payload;
    ngx_http_aegis_response_t   aegis_resp;
    ngx_http_aegis_loc_conf_t  *alcf;
    ngx_int_t                   rc;
    ngx_buf_t                  *b;
    ngx_chain_t                 out;

    AEGIS_LOG(NGX_LOG_DEBUG, r->connection->log, "processing request");

    alcf = ngx_http_get_module_loc_conf(r, ngx_http_aegis_module);

    /* Initialize response structure */
    ngx_memzero(&aegis_resp, sizeof(ngx_http_aegis_response_t));
    aegis_resp.code = 0; /* Default: allow */
    aegis_resp.headers = ngx_array_create(r->pool, 10, sizeof(ngx_http_aegis_header_t));
    if (aegis_resp.headers == NULL) {
        return NGX_ERROR;
    }

    /* Build JSON payload */
    rc = ngx_http_aegis_build_json_payload(r, &payload);
    if (rc != NGX_OK) {
        AEGIS_LOG(NGX_LOG_ERR, r->connection->log, "failed to build JSON payload");
        return NGX_DECLINED; /* fail-open */
    }

    /* Send request to aegis service and get response */
    rc = ngx_http_aegis_send_request(r, &payload, &aegis_resp);
    if (rc != NGX_OK) {
        AEGIS_LOG(NGX_LOG_WARN, r->connection->log, "aegis service unavailable, allowing request");
        return NGX_DECLINED; /* fail-open */
    }

    AEGIS_LOG(NGX_LOG_DEBUG, r->connection->log, "aegis response code: %i, headers: %ui, body_len: %uz", 
             aegis_resp.code, aegis_resp.headers->nelts, aegis_resp.body.len);

    /* Process Aegis decision */
    if (aegis_resp.code == 0) {
        /* Allow request */
        AEGIS_LOG(NGX_LOG_DEBUG, r->connection->log, "request allowed by aegis");
        return NGX_DECLINED;
    } else {
        /* Block request */
        if (alcf->log_blocked) {
            AEGIS_LOG(NGX_LOG_WARN, r->connection->log, "request blocked by aegis (code: %i) from %V, body_len: %uz", 
                     aegis_resp.code, &r->connection->addr_text, aegis_resp.body.len);
        }

        /* Set response status */
        r->headers_out.status = aegis_resp.code;

        /* Set headers from Aegis response */
        rc = ngx_http_aegis_set_headers(r, &aegis_resp);
        if (rc != NGX_OK) {
            AEGIS_LOG(NGX_LOG_ERR, r->connection->log, "failed to set response headers");
        }

        /* Prepare response body */
        if (aegis_resp.body.len > 0) {
            r->headers_out.content_length_n = aegis_resp.body.len;

            /* Create buffer for response body - SUPPORT LARGE BODIES */
            b = ngx_create_temp_buf(r->pool, aegis_resp.body.len);
            if (b == NULL) {
                AEGIS_LOG(NGX_LOG_ERR, r->connection->log, "failed to create response buffer for %uz bytes", 
                         aegis_resp.body.len);
                return NGX_HTTP_INTERNAL_SERVER_ERROR;
            }

            b->last = ngx_copy(b->pos, aegis_resp.body.data, aegis_resp.body.len);
            b->last_buf = 1;
            b->last_in_chain = 1;

            out.buf = b;
            out.next = NULL;

            AEGIS_LOG(NGX_LOG_DEBUG, r->connection->log, "prepared response body %uz bytes", aegis_resp.body.len);

            /* Send headers */
            rc = ngx_http_send_header(r);
            if (rc == NGX_ERROR || rc > NGX_OK || r->header_only) {
                return rc;
            }

            /* Send body */
            return ngx_http_output_filter(r, &out);
        } else {
            /* No body, just return status code */
            r->headers_out.content_length_n = 0;
            r->header_only = 1;

            return ngx_http_send_header(r);
        }
    }
}

/* Set response headers from Aegis response */
static ngx_int_t
ngx_http_aegis_set_headers(ngx_http_request_t *r, ngx_http_aegis_response_t *resp)
{
    ngx_http_aegis_header_t    *headers;
    ngx_table_elt_t            *h;
    ngx_uint_t                  i;

    if (resp->headers == NULL || resp->headers->nelts == 0) {
        /* Set default content type */
        ngx_str_set(&r->headers_out.content_type, "text/plain");
        r->headers_out.content_type_len = r->headers_out.content_type.len;
        AEGIS_LOG(NGX_LOG_DEBUG, r->connection->log, "no headers from aegis, using default content-type");
        return NGX_OK;
    }

    headers = resp->headers->elts;

    for (i = 0; i < resp->headers->nelts; i++) {
        AEGIS_LOG(NGX_LOG_DEBUG, r->connection->log, "setting header: %V: %V", 
                 &headers[i].name, &headers[i].value);

        /* Handle special headers */
        if (ngx_strcasecmp(headers[i].name.data, (u_char *) "content-type") == 0) {
            r->headers_out.content_type = headers[i].value;
            r->headers_out.content_type_len = headers[i].value.len;
            continue;
        }

        if (ngx_strcasecmp(headers[i].name.data, (u_char *) "location") == 0) {
            h = ngx_list_push(&r->headers_out.headers);
            if (h == NULL) {
                return NGX_ERROR;
            }
            h->hash = 1;
            ngx_str_set(&h->key, "Location");
            h->value = headers[i].value;
            AEGIS_LOG(NGX_LOG_DEBUG, r->connection->log, "set Location header: %V", &headers[i].value);
            continue;
        }

        if (ngx_strcasecmp(headers[i].name.data, (u_char *) "www-authenticate") == 0) {
            h = ngx_list_push(&r->headers_out.headers);
            if (h == NULL) {
                return NGX_ERROR;
            }
            h->hash = 1;
            ngx_str_set(&h->key, "WWW-Authenticate");
            h->value = headers[i].value;
            continue;
        }

        if (ngx_strcasecmp(headers[i].name.data, (u_char *) "cache-control") == 0) {
            h = ngx_list_push(&r->headers_out.headers);
            if (h == NULL) {
                return NGX_ERROR;
            }
            h->hash = 1;
            ngx_str_set(&h->key, "Cache-Control");
            h->value = headers[i].value;
            continue;
        }

        /* Handle general headers */
        h = ngx_list_push(&r->headers_out.headers);
        if (h == NULL) {
            return NGX_ERROR;
        }

        h->hash = 1;
        h->key = headers[i].name;
        h->value = headers[i].value;
    }

    /* Set default content type if not provided */
    if (r->headers_out.content_type.len == 0) {
        ngx_str_set(&r->headers_out.content_type, "text/plain");
        r->headers_out.content_type_len = r->headers_out.content_type.len;
    }

    return NGX_OK;
}

/* Build JSON payload for aegis service - WITH FIXED COOKIES PARSING */
static ngx_int_t
ngx_http_aegis_build_json_payload(ngx_http_request_t *r, ngx_str_t *payload)
{
    u_char              *p;
    size_t               len;
    ngx_str_t            body_str = ngx_null_string;
    ngx_list_part_t     *part;
    ngx_table_elt_t     *h;
    ngx_uint_t           i, first;
    u_char              *url_esc, *method_esc, *body_esc;

    /* Get request body if available */
    if (r->request_body && r->request_body->bufs) {
        ngx_chain_t  *cl;
        ngx_buf_t    *buf;
        size_t        body_len = 0;
        u_char       *body_p;

        /* Calculate body length */
        for (cl = r->request_body->bufs; cl; cl = cl->next) {
            buf = cl->buf;
            if (!buf->in_file) {
                body_len += buf->last - buf->pos;
            }
        }

        if (body_len > 0 && body_len < 64 * 1024) { /* Limit body size */
            body_str.data = ngx_pnalloc(r->pool, body_len);
            if (body_str.data) {
                body_p = body_str.data;
                for (cl = r->request_body->bufs; cl; cl = cl->next) {
                    buf = cl->buf;
                    if (!buf->in_file) {
                        body_p = ngx_copy(body_p, buf->pos, buf->last - buf->pos);
                    }
                }
                body_str.len = body_len;
            }
        }
    }

    /* Escape JSON strings */
    url_esc = ngx_http_aegis_escape_json_string(r->pool, r->uri.data, r->uri.len);
    method_esc = ngx_http_aegis_escape_json_string(r->pool, r->method_name.data, r->method_name.len);
    body_esc = body_str.len > 0 ? 
        ngx_http_aegis_escape_json_string(r->pool, body_str.data, body_str.len) : 
        (u_char *)"";

    if (!url_esc || !method_esc) {
        return NGX_ERROR;
    }

    /* Calculate payload size - generous estimation */
    len = 1024 + r->connection->addr_text.len + 
          ngx_strlen(url_esc) + ngx_strlen(method_esc) + ngx_strlen(body_esc);

    /* Add headers estimation */
    part = &r->headers_in.headers.part;
    h = part->elts;

    for (i = 0; ; i++) {
        if (i >= part->nelts) {
            if (part->next == NULL) {
                break;
            }
            part = part->next;
            h = part->elts;
            i = 0;
        }
        len += h[i].key.len + h[i].value.len + 20; /* JSON overhead */
    }

    /* Add cookie estimation */
    if (r->headers_in.cookie) {
        len += r->headers_in.cookie->value.len + 50;
    }

    /* Allocate payload buffer */
    payload->data = ngx_pnalloc(r->pool, len);
    if (payload->data == NULL) {
        return NGX_ERROR;
    }

    /* Build JSON - start with fixed structure */
    p = payload->data;
    p = ngx_sprintf(p, "{\"clientAddress\":\"%V\",", &r->connection->addr_text);
    p = ngx_sprintf(p, "\"url\":\"%s\",", url_esc);
    p = ngx_sprintf(p, "\"method\":\"%s\",", method_esc);
    p = ngx_sprintf(p, "\"body\":\"%s\",", body_esc);

    /* Add headers */
    p = ngx_sprintf(p, "\"headers\":{");

    part = &r->headers_in.headers.part;
    h = part->elts;
    first = 1;

    for (i = 0; ; i++) {
        if (i >= part->nelts) {
            if (part->next == NULL) {
                break;
            }
            part = part->next;
            h = part->elts;
            i = 0;
        }

        u_char *key_esc = ngx_http_aegis_escape_json_string(r->pool, h[i].key.data, h[i].key.len);
        u_char *val_esc = ngx_http_aegis_escape_json_string(r->pool, h[i].value.data, h[i].value.len);

        if (key_esc && val_esc) {
            if (!first) *p++ = ',';
            p = ngx_sprintf(p, "\"%s\":\"%s\"", key_esc, val_esc);
            first = 0;
        }
    }

    /* Close headers and add cookies - FIXED PARSING */
    p = ngx_sprintf(p, "},\"cookies\":{");

    if (r->headers_in.cookie) {
        /* Parse cookies properly: "name1=value1; name2=value2" */
        u_char *cookie_start = r->headers_in.cookie->value.data;
        u_char *cookie_end = cookie_start + r->headers_in.cookie->value.len;
        u_char *p_cookie = cookie_start;
        ngx_uint_t cookie_first = 1;

        AEGIS_LOG(NGX_LOG_DEBUG, r->connection->log, "parsing cookies: %V", &r->headers_in.cookie->value);

        while (p_cookie < cookie_end) {
            u_char *name_start, *name_end, *value_start, *value_end;
            u_char *name_esc, *value_esc;

            /* Skip whitespace */
            while (p_cookie < cookie_end && (*p_cookie == ' ' || *p_cookie == '\t')) {
                p_cookie++;
            }

            if (p_cookie >= cookie_end) break;

            /* Find cookie name */
            name_start = p_cookie;
            while (p_cookie < cookie_end && *p_cookie != '=' && *p_cookie != ';') {
                p_cookie++;
            }
            name_end = p_cookie;

            if (p_cookie >= cookie_end || *p_cookie != '=') {
                /* No value, skip to next cookie */
                while (p_cookie < cookie_end && *p_cookie != ';') {
                    p_cookie++;
                }
                if (p_cookie < cookie_end) p_cookie++; /* Skip ';' */
                continue;
            }

            p_cookie++; /* Skip '=' */

            /* Find cookie value */
            value_start = p_cookie;
            while (p_cookie < cookie_end && *p_cookie != ';') {
                p_cookie++;
            }
            value_end = p_cookie;

            if (p_cookie < cookie_end) p_cookie++; /* Skip ';' */

            /* Escape cookie name and value */
            ngx_str_t cookie_name = {name_end - name_start, name_start};
            ngx_str_t cookie_value = {value_end - value_start, value_start};

            name_esc = ngx_http_aegis_escape_json_string(r->pool, cookie_name.data, cookie_name.len);
            value_esc = ngx_http_aegis_escape_json_string(r->pool, cookie_value.data, cookie_value.len);

            if (name_esc && value_esc) {
                if (!cookie_first) *p++ = ',';
                p = ngx_sprintf(p, "\"%s\":\"%s\"", name_esc, value_esc);
                cookie_first = 0;

                AEGIS_LOG(NGX_LOG_DEBUG, r->connection->log, "parsed cookie: %s=%s", name_esc, value_esc);
            }
        }
    }

    p = ngx_sprintf(p, "}}");
    payload->len = p - payload->data;

    AEGIS_LOG(NGX_LOG_DEBUG, r->connection->log, "built JSON payload: %uz bytes", payload->len);

    return NGX_OK;
}

/* Send HTTP request to aegis service - COMPILATION FIXED */
static ngx_int_t
ngx_http_aegis_send_request(ngx_http_request_t *r, ngx_str_t *payload, ngx_http_aegis_response_t *aegis_resp)
{
    ngx_http_aegis_loc_conf_t *alcf;
    int                        sockfd;
    struct sockaddr_in         server_addr;
    u_char                    *http_request;
    size_t                     request_len;
    ssize_t                    sent, received;
    size_t                     total_received = 0;   /* FIXED: size_t */
    u_char                    *response_buf, *new_buf;
    u_char                    *body_start;
    struct timeval             timeout;
    ngx_int_t                  rc;
    size_t                     buffer_size, new_size;

    alcf = ngx_http_get_module_loc_conf(r, ngx_http_aegis_module);

    /* Start with initial buffer size */
    buffer_size = AEGIS_INITIAL_BUFFER;
    response_buf = ngx_pnalloc(r->pool, buffer_size);
    if (response_buf == NULL) {
        AEGIS_LOG(NGX_LOG_ERR, r->connection->log, "failed to allocate initial response buffer (%uz bytes)", buffer_size);
        return NGX_ERROR;
    }

    AEGIS_LOG(NGX_LOG_DEBUG, r->connection->log, "allocated initial response buffer: %uz bytes", buffer_size);

    /* Create socket */
    sockfd = socket(AF_INET, SOCK_STREAM, 0);
    if (sockfd < 0) {
        AEGIS_LOG(NGX_LOG_ERR, r->connection->log, "socket creation failed: %s", strerror(errno));
        return NGX_ERROR;
    }

    /* Set socket timeout */
    timeout.tv_sec = alcf->timeout / 1000;
    timeout.tv_usec = (alcf->timeout % 1000) * 1000;

    setsockopt(sockfd, SOL_SOCKET, SO_RCVTIMEO, &timeout, sizeof(timeout));
    setsockopt(sockfd, SOL_SOCKET, SO_SNDTIMEO, &timeout, sizeof(timeout));

    /* Setup server address */
    ngx_memzero(&server_addr, sizeof(server_addr));
    server_addr.sin_family = AF_INET;
    server_addr.sin_port = htons(6996);
    server_addr.sin_addr.s_addr = inet_addr("127.0.0.1");

    /* Connect */
    if (connect(sockfd, (struct sockaddr*)&server_addr, sizeof(server_addr)) < 0) {
        AEGIS_LOG(NGX_LOG_ERR, r->connection->log, "connect failed: %s", strerror(errno));
        close(sockfd);
        return NGX_ERROR;
    }

    /* Build HTTP request */
    request_len = 200 + payload->len; /* HTTP headers + JSON payload */
    http_request = ngx_pnalloc(r->pool, request_len);
    if (http_request == NULL) {
        close(sockfd);
        return NGX_ERROR;
    }

    request_len = ngx_sprintf(http_request, 
                              "POST /api/v1/check HTTP/1.1\r\n"
                              "Host: localhost:6996\r\n"
                              "Content-Type: application/json\r\n"
                              "Content-Length: %uz\r\n"
                              "Connection: close\r\n"
                              "\r\n"
                              "%V", payload->len, payload) - http_request;

    /* Send request */
    sent = send(sockfd, http_request, request_len, 0);
    if (sent != (ssize_t)request_len) {
        AEGIS_LOG(NGX_LOG_ERR, r->connection->log, "send incomplete: %z/%uz", sent, request_len);
        close(sockfd);
        return NGX_ERROR;
    }

    /* Receive response with dynamic buffer expansion */
    while (1) {
        /* Check if we need more space */
        if (total_received >= buffer_size - 1) {
            if (buffer_size >= AEGIS_MAX_BUFFER_SIZE) {
                AEGIS_LOG(NGX_LOG_ERR, r->connection->log, 
                         "response too large: %uz bytes (max: 500KB)", total_received);
                close(sockfd);
                return NGX_ERROR;
            }

            /* Expand buffer by doubling size */
            new_size = buffer_size * 2;
            if (new_size > AEGIS_MAX_BUFFER_SIZE) {
                new_size = AEGIS_MAX_BUFFER_SIZE;
            }

            new_buf = ngx_pnalloc(r->pool, new_size);
            if (new_buf == NULL) {
                AEGIS_LOG(NGX_LOG_ERR, r->connection->log, "failed to expand response buffer to %uz bytes", new_size);
                close(sockfd);
                return NGX_ERROR;
            }

            /* Copy existing data */
            ngx_memcpy(new_buf, response_buf, total_received);
            response_buf = new_buf;
            buffer_size = new_size;

            AEGIS_LOG(NGX_LOG_DEBUG, r->connection->log, "expanded response buffer to %uz bytes", buffer_size);
        }

        /* Receive more data */
        received = recv(sockfd, response_buf + total_received, buffer_size - total_received - 1, 0);
        if (received <= 0) {
            break;
        }
        total_received += (size_t)received;  /* FIXED: cast to size_t */
    }

    close(sockfd);

    if (total_received <= 0) {
        AEGIS_LOG(NGX_LOG_ERR, r->connection->log, "no response from aegis service");
        return NGX_ERROR;
    }

    response_buf[total_received] = '\0';

    AEGIS_LOG(NGX_LOG_DEBUG, r->connection->log, "received %uz bytes from aegis (buffer: %uz bytes)", 
             total_received, buffer_size);

    /* Find HTTP body (JSON) */
    body_start = (u_char*)strstr((char*)response_buf, "\r\n\r\n");
    if (body_start == NULL) {
        body_start = (u_char*)strstr((char*)response_buf, "\n\n");
        if (body_start == NULL) {
            AEGIS_LOG(NGX_LOG_ERR, r->connection->log, "invalid HTTP response format");
            return NGX_ERROR;
        }
        body_start += 2;
    } else {
        body_start += 4;
    }

    size_t json_len = total_received - (size_t)(body_start - response_buf);
    AEGIS_LOG(NGX_LOG_DEBUG, r->connection->log, "HTTP body size: %uz bytes", json_len);

    /* Parse JSON response */
    rc = ngx_http_aegis_parse_response(r, body_start, json_len, aegis_resp);
    if (rc != NGX_OK) {
        AEGIS_LOG(NGX_LOG_ERR, r->connection->log, "failed to parse aegis response");
        return NGX_ERROR;
    }

    return NGX_OK;
}

/* Unescape JSON string (handle \n, \t, \", \\, etc.) */
static ngx_int_t
ngx_http_aegis_json_unescape(ngx_pool_t *pool, ngx_str_t *src, ngx_str_t *dst)
{
    u_char *p, *d, *end;
    size_t  unescaped_len = 0;

    if (src == NULL || src->len == 0 || src->data == NULL) {
        dst->data = NULL;
        dst->len = 0;
        return NGX_OK;
    }

    end = src->data + src->len;

    /* First pass: calculate unescaped length */
    for (p = src->data; p < end; p++) {
        if (*p == '\\' && p + 1 < end) {
            switch (*(p + 1)) {
            case 'n': case 't': case 'r': case 'b': case 'f':
            case '"': case '\\': case '/':
                unescaped_len++; /* Two chars → one char */
                p++; /* Skip next char */
                break;
            case 'u':
                if (p + 5 < end) {
                    unescaped_len++; /* \uXXXX → one char (simplified) */
                    p += 5; /* Skip u and 4 hex digits */
                } else {
                    unescaped_len++; /* Keep as is if incomplete */
                }
                break;
            default:
                unescaped_len += 2; /* Keep both chars if unknown escape */
                p++;
                break;
            }
        } else {
            unescaped_len++;
        }
    }

    /* Allocate unescaped buffer */
    dst->data = ngx_pnalloc(pool, unescaped_len + 1);
    if (dst->data == NULL) {
        return NGX_ERROR;
    }

    /* Second pass: unescape */
    d = dst->data;
    for (p = src->data; p < end; p++) {
        if (*p == '\\' && p + 1 < end) {
            switch (*(p + 1)) {
            case 'n':
                *d++ = '\n';
                p++;
                break;
            case 't':
                *d++ = '\t';
                p++;
                break;
            case 'r':
                *d++ = '\r';
                p++;
                break;
            case 'b':
                *d++ = '\b';
                p++;
                break;
            case 'f':
                *d++ = '\f';
                p++;
                break;
            case '"':
                *d++ = '"';
                p++;
                break;
            case '\\':
                *d++ = '\\';
                p++;
                break;
            case '/':
                *d++ = '/';
                p++;
                break;
            case 'u':
                if (p + 5 < end) {
                    /* Simple Unicode handling - just use '?' for now */
                    *d++ = '?';
                    p += 5; /* Skip u and 4 hex digits */
                } else {
                    *d++ = *p; /* Keep backslash if incomplete sequence */
                }
                break;
            default:
                /* Unknown escape, keep both characters */
                *d++ = *p;
                p++;
                *d++ = *p;
                break;
            }
        } else {
            *d++ = *p;
        }
    }

    *d = '\0';
    dst->len = d - dst->data;

    AEGIS_LOG(NGX_LOG_DEBUG, pool->log, "JSON unescaped: %uz → %uz bytes", src->len, dst->len);

    return NGX_OK;
}

/* HTML decode function for Aegis body content */
static ngx_int_t
ngx_http_aegis_html_decode(ngx_pool_t *pool, ngx_str_t *src, ngx_str_t *dst)
{
    u_char *p, *d, *end;
    size_t  decoded_len = 0;

    if (src == NULL || src->len == 0 || src->data == NULL) {
        dst->data = NULL;
        dst->len = 0;
        return NGX_OK;
    }

    end = src->data + src->len;

    /* First pass: calculate decoded length */
    for (p = src->data; p < end; p++) {
        if (*p == '&') {
            if (p + 4 <= end && ngx_strncmp(p, "&lt;", 4) == 0) {
                decoded_len++; /* &lt; → < */
                p += 3; /* Skip remaining chars, loop will +1 */
            } else if (p + 4 <= end && ngx_strncmp(p, "&gt;", 4) == 0) {
                decoded_len++; /* &gt; → > */
                p += 3;
            } else if (p + 5 <= end && ngx_strncmp(p, "&amp;", 5) == 0) {
                decoded_len++; /* &amp; → & */
                p += 4;
            } else if (p + 6 <= end && ngx_strncmp(p, "&quot;", 6) == 0) {
                decoded_len++; /* &quot; → " */
                p += 5;
            } else if (p + 6 <= end && ngx_strncmp(p, "&#x27;", 6) == 0) {
                decoded_len++; /* &#x27; → ' */
                p += 5;
            } else {
                decoded_len++; /* Unknown entity, keep as is */
            }
        } else {
            decoded_len++;
        }
    }

    /* Allocate decoded buffer */
    dst->data = ngx_pnalloc(pool, decoded_len + 1);
    if (dst->data == NULL) {
        return NGX_ERROR;
    }

    /* Second pass: decode */
    d = dst->data;
    for (p = src->data; p < end; p++) {
        if (*p == '&') {
            if (p + 4 <= end && ngx_strncmp(p, "&lt;", 4) == 0) {
                *d++ = '<';
                p += 3;
            } else if (p + 4 <= end && ngx_strncmp(p, "&gt;", 4) == 0) {
                *d++ = '>';
                p += 3;
            } else if (p + 5 <= end && ngx_strncmp(p, "&amp;", 5) == 0) {
                *d++ = '&';
                p += 4;
            } else if (p + 6 <= end && ngx_strncmp(p, "&quot;", 6) == 0) {
                *d++ = '"';
                p += 5;
            } else if (p + 6 <= end && ngx_strncmp(p, "&#x27;", 6) == 0) {
                *d++ = '\'';
                p += 5;
            } else {
                *d++ = *p; /* Unknown entity, keep as is */
            }
        } else {
            *d++ = *p;
        }
    }

    *d = '\0';
    dst->len = d - dst->data;

    AEGIS_LOG(NGX_LOG_DEBUG, pool->log, "HTML decoded: %uz → %uz bytes", src->len, dst->len);

    return NGX_OK;
}

/* Parse JSON response from Aegis service - WITH JSON UNESCAPING AND HTML DECODING */
static ngx_int_t
ngx_http_aegis_parse_response(ngx_http_request_t *r, u_char *data, size_t len, ngx_http_aegis_response_t *resp)
{
    ngx_int_t   rc;
    ngx_str_t   raw_body, json_unescaped_body;

    AEGIS_LOG(NGX_LOG_DEBUG, r->connection->log, "parsing JSON response (%uz bytes): %.200s%s", 
             len, data, len > 200 ? "..." : "");

    /* Initialize response */
    resp->code = 0;
    resp->body.data = NULL;
    resp->body.len = 0;

    /* Parse "code" field */
    rc = ngx_http_aegis_simple_json_get_int(data, len, "code", &resp->code);
    if (rc != NGX_OK) {
        AEGIS_LOG(NGX_LOG_ERR, r->connection->log, "failed to parse 'code' field from JSON");
        return NGX_ERROR;
    }

    /* Parse "body" field if present - WITH JSON UNESCAPING AND HTML DECODING */
    rc = ngx_http_aegis_simple_json_get_str(data, len, "body", &raw_body, r->pool);
    if (rc == NGX_OK && raw_body.len > 0) {
        AEGIS_LOG(NGX_LOG_DEBUG, r->connection->log, "raw JSON body field: %uz bytes", raw_body.len);

        /* Step 1: JSON unescape (\n → \n, \" → ", \\ → \) */
        rc = ngx_http_aegis_json_unescape(r->pool, &raw_body, &json_unescaped_body);
        if (rc != NGX_OK) {
            AEGIS_LOG(NGX_LOG_ERR, r->connection->log, "failed to JSON unescape body content");
            json_unescaped_body = raw_body; /* Fall back to raw version */
        }

        /* Step 2: HTML decode (&lt; → <, &gt; → >, &amp; → &) */
        rc = ngx_http_aegis_html_decode(r->pool, &json_unescaped_body, &resp->body);
        if (rc != NGX_OK) {
            AEGIS_LOG(NGX_LOG_ERR, r->connection->log, "failed to HTML decode body content");
            resp->body = json_unescaped_body; /* Fall back to JSON unescaped version */
        }

        AEGIS_LOG(NGX_LOG_DEBUG, r->connection->log, "processed body: raw=%uz → json_unescaped=%uz → html_decoded=%uz bytes", 
                 raw_body.len, json_unescaped_body.len, resp->body.len);
    } else {
        AEGIS_LOG(NGX_LOG_DEBUG, r->connection->log, "'body' field not found in JSON");
    }

    /* Parse "headers" field if present */
    rc = ngx_http_aegis_parse_headers_json(data, len, resp->headers, r->pool);
    if (rc == NGX_OK) {
        AEGIS_LOG(NGX_LOG_DEBUG, r->connection->log, "successfully parsed %ui headers from JSON", resp->headers->nelts);
    } else {
        AEGIS_LOG(NGX_LOG_DEBUG, r->connection->log, "'headers' field not found or empty in JSON");
    }

    AEGIS_LOG(NGX_LOG_DEBUG, r->connection->log, "parsed JSON: code=%i, body_len=%uz, headers=%ui", 
             resp->code, resp->body.len, resp->headers->nelts);

    return NGX_OK;
}

/* Simple JSON string field extractor - IMPROVED FOR LARGE STRINGS */
static ngx_int_t
ngx_http_aegis_simple_json_get_str(u_char *json, size_t len, const char *key, ngx_str_t *value, ngx_pool_t *pool)
{
    u_char *p, *end, *start, *value_end;
    size_t  key_len = strlen(key);
    ngx_int_t quotes_level = 0;

    end = json + len;

    /* Search for "key": */
    for (p = json; p < end - key_len - 3; p++) {
        if (*p == '"' && 
            ngx_strncmp(p + 1, key, key_len) == 0 && 
            *(p + 1 + key_len) == '"' &&
            *(p + 2 + key_len) == ':') {

            p += 3 + key_len;

            /* Skip whitespace */
            while (p < end && (*p == ' ' || *p == '\t' || *p == '\n' || *p == '\r')) {
                p++;
            }

            if (p >= end || *p != '"') break;

            p++; /* Skip opening quote */
            start = p;

            /* Find closing quote - HANDLE LARGE STRINGS AND ESCAPES */
            quotes_level = 1;  /* We are inside quotes */
            while (p < end && quotes_level > 0) {
                if (*p == '\\') {
                    p += 2; /* Skip escaped character */
                    continue;
                }
                if (*p == '"') {
                    quotes_level--;
                    if (quotes_level == 0) {
                        value_end = p;
                        break;
                    }
                }
                p++;
            }

            if (quotes_level > 0 || p >= end) {
                /* Unclosed quote or end of data */
                break;
            }

            /* Allocate and copy string */
            value->len = value_end - start;
            if (value->len > 0) {
                value->data = ngx_pnalloc(pool, value->len + 1);
                if (value->data == NULL) {
                    return NGX_ERROR;
                }
                ngx_memcpy(value->data, start, value->len);
                value->data[value->len] = '\0';

                AEGIS_LOG(NGX_LOG_DEBUG, pool->log, "extracted JSON field '%s': %uz bytes", key, value->len);
            } else {
                value->data = NULL;
            }

            return NGX_OK;
        }
    }

    value->data = NULL;
    value->len = 0;
    return NGX_ERROR;
}

/* Simple JSON integer field extractor */
static ngx_int_t
ngx_http_aegis_simple_json_get_int(u_char *json, size_t len, const char *key, ngx_int_t *value)
{
    u_char *p, *end, *start;
    size_t  key_len = strlen(key);

    end = json + len;

    /* Search for "key": */
    for (p = json; p < end - key_len - 3; p++) {
        if (*p == '"' && 
            ngx_strncmp(p + 1, key, key_len) == 0 && 
            *(p + 1 + key_len) == '"' &&
            *(p + 2 + key_len) == ':') {

            p += 3 + key_len;

            /* Skip whitespace */
            while (p < end && (*p == ' ' || *p == '\t' || *p == '\n' || *p == '\r')) {
                p++;
            }

            if (p >= end) break;

            /* Parse integer */
            start = p;
            if (*p == '-') p++;

            while (p < end && *p >= '0' && *p <= '9') {
                p++;
            }

            if (p > start) {
                *value = ngx_atoi(start, p - start);
                return NGX_OK;
            }
            break;
        }
    }

    return NGX_ERROR;
}

/* Parse headers object from JSON */
static ngx_int_t
ngx_http_aegis_parse_headers_json(u_char *json, size_t len, ngx_array_t *headers, ngx_pool_t *pool)
{
    u_char                     *p, *end, *key_start, *key_end, *value_start, *value_end;
    ngx_http_aegis_header_t    *header;
    ngx_int_t                   brace_level = 0, in_headers = 0;

    end = json + len;

    AEGIS_LOG(NGX_LOG_DEBUG, pool->log, "searching for headers in JSON (%uz bytes)", len);

    /* Find "headers":{...} object */
    for (p = json; p < end - 10; p++) {
        if (*p == '"' && ngx_strncmp(p + 1, "headers", 7) == 0 && *(p + 8) == '"') {
            p += 9; /* Skip "headers" */

            /* Skip whitespace and colon */
            while (p < end && (*p == ' ' || *p == '\t' || *p == '\n' || *p == '\r' || *p == ':')) {
                p++;
            }

            if (p >= end || *p != '{') {
                AEGIS_LOG(NGX_LOG_DEBUG, pool->log, "headers field found but no opening brace");
                return NGX_ERROR;
            }

            p++; /* Skip opening { */
            in_headers = 1;
            brace_level = 1;

            AEGIS_LOG(NGX_LOG_DEBUG, pool->log, "found headers object, parsing contents");
            break;
        }
    }

    if (!in_headers) {
        AEGIS_LOG(NGX_LOG_DEBUG, pool->log, "headers field not found in JSON");
        return NGX_ERROR;
    }

    /* Parse key-value pairs inside headers object */
    while (p < end && brace_level > 0) {
        /* Skip whitespace and commas */
        while (p < end && (*p == ' ' || *p == '\t' || *p == '\n' || *p == '\r' || *p == ',')) {
            p++;
        }

        if (p >= end) break;

        if (*p == '}') {
            brace_level--;
            if (brace_level == 0) break;
            p++;
            continue;
        }

        if (*p == '{') {
            brace_level++;
            p++;
            continue;
        }

        /* Expect opening quote for key */
        if (*p != '"') {
            p++;
            continue;
        }

        p++; /* Skip opening quote */
        key_start = p;

        /* Find closing quote for key */
        while (p < end && *p != '"') {
            if (*p == '\\') p++; /* Skip escaped characters */
            p++;
        }

        if (p >= end) break;

        key_end = p;
        p++; /* Skip closing quote */

        /* Skip whitespace and colon */
        while (p < end && (*p == ' ' || *p == '\t' || *p == '\n' || *p == '\r' || *p == ':')) {
            p++;
        }

        if (p >= end || *p != '"') {
            /* Value is not a string, skip it */
            continue;
        }

        p++; /* Skip opening quote for value */
        value_start = p;

        /* Find closing quote for value */
        while (p < end && *p != '"') {
            if (*p == '\\') p++; /* Skip escaped characters */
            p++;
        }

        if (p >= end) break;

        value_end = p;
        p++; /* Skip closing quote */

        /* Create header entry */
        header = ngx_array_push(headers);
        if (header == NULL) {
            return NGX_ERROR;
        }

        /* Copy header name */
        header->name.len = key_end - key_start;
        header->name.data = ngx_pnalloc(pool, header->name.len + 1);
        if (header->name.data == NULL) {
            return NGX_ERROR;
        }
        ngx_memcpy(header->name.data, key_start, header->name.len);
        header->name.data[header->name.len] = '\0';

        /* Copy header value */  
        header->value.len = value_end - value_start;
        header->value.data = ngx_pnalloc(pool, header->value.len + 1);
        if (header->value.data == NULL) {
            return NGX_ERROR;
        }
        ngx_memcpy(header->value.data, value_start, header->value.len);
        header->value.data[header->value.len] = '\0';

        AEGIS_LOG(NGX_LOG_DEBUG, pool->log, "parsed header from JSON: %V: %V", &header->name, &header->value);
    }

    AEGIS_LOG(NGX_LOG_DEBUG, pool->log, "finished parsing headers, found %ui headers", headers->nelts);

    return headers->nelts > 0 ? NGX_OK : NGX_ERROR;
}

/* Escape JSON string */
static u_char *
ngx_http_aegis_escape_json_string(ngx_pool_t *pool, u_char *src, size_t len)
{
    u_char  *dst, *p;
    size_t   escaped_len = 0;
    size_t   i;

    if (src == NULL || len == 0) {
        dst = ngx_pnalloc(pool, 1);
        if (dst) *dst = '\0';
        return dst;
    }

    /* Calculate escaped length */
    for (i = 0; i < len; i++) {
        switch (src[i]) {
        case '"': case '\\': case '/': case '\b': 
        case '\f': case '\n': case '\r': case '\t':
            escaped_len += 2;
            break;
        default:
            if (src[i] < 0x20) {
                escaped_len += 6; /* \uXXXX */
            } else {
                escaped_len++;
            }
            break;
        }
    }

    dst = ngx_pnalloc(pool, escaped_len + 1);
    if (dst == NULL) {
        return NULL;
    }

    p = dst;
    for (i = 0; i < len; i++) {
        switch (src[i]) {
        case '"':  *p++ = '\\'; *p++ = '"'; break;
        case '\\': *p++ = '\\'; *p++ = '\\'; break;
        case '/':  *p++ = '\\'; *p++ = '/'; break;
        case '\b':  *p++ = '\\'; *p++ = 'b'; break;
        case '\f':  *p++ = '\\'; *p++ = 'f'; break;
        case '\n':  *p++ = '\\'; *p++ = 'n'; break;
        case '\r':  *p++ = '\\'; *p++ = 'r'; break;
        case '\t':  *p++ = '\\'; *p++ = 't'; break;
        default:
            if (src[i] < 0x20) {
                p = ngx_sprintf(p, "\\u%04X", (unsigned)src[i]);
            } else {
                *p++ = src[i];
            }
            break;
        }
    }
    *p = '\0';
    return dst;
}

/* Configuration functions */
static void *
ngx_http_aegis_create_loc_conf(ngx_conf_t *cf)
{
    ngx_http_aegis_loc_conf_t *conf;

    conf = ngx_pcalloc(cf->pool, sizeof(ngx_http_aegis_loc_conf_t));
    if (conf == NULL) {
        return NULL;
    }

    conf->enable = NGX_CONF_UNSET;
    conf->timeout = NGX_CONF_UNSET_UINT;
    conf->log_blocked = NGX_CONF_UNSET;

    return conf;
}

static char *
ngx_http_aegis_merge_loc_conf(ngx_conf_t *cf, void *parent, void *child)
{
    ngx_http_aegis_loc_conf_t *prev = parent;
    ngx_http_aegis_loc_conf_t *conf = child;

    ngx_conf_merge_value(conf->enable, prev->enable, 0);
    ngx_conf_merge_str_value(conf->endpoint, prev->endpoint, "http://localhost:6996/api/v1/check");
    ngx_conf_merge_uint_value(conf->timeout, prev->timeout, 5000);
    ngx_conf_merge_value(conf->log_blocked, prev->log_blocked, 1);

    return NGX_CONF_OK;
}

/* Module initialization */
static ngx_int_t
ngx_http_aegis_init(ngx_conf_t *cf)
{
    ngx_http_handler_pt        *h;
    ngx_http_core_main_conf_t  *cmcf;

    cmcf = ngx_http_conf_get_module_main_conf(cf, ngx_http_core_module);

    h = ngx_array_push(&cmcf->phases[NGX_HTTP_PREACCESS_PHASE].handlers);
    if (h == NULL) {
        return NGX_ERROR;
    }

    *h = ngx_http_aegis_handler;

    AEGIS_LOG(NGX_LOG_INFO, cf->log, "module initialized in preaccess phase");

    return NGX_OK;
}
