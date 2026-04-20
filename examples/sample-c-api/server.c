/*
 * server.c — HTTP server for the CortexDocs sample API.
 * Built on mongoose 7.14 (single-file embed). Listens on :7890.
 *
 * Routes
 *   GET    /users          list all users
 *   GET    /users/{id}     get one user
 *   POST   /users          create a user  (JSON body: name, email, role)
 *   DELETE /users/{id}     delete a user
 */

#include "mongoose.h"
#include "users.h"

#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#define PORT      "7890"
#define MAX_USERS 256

/* ── data store ─────────────────────────────────────────────────────────── */

typedef struct {
    int  id;
    char name[128];
    char email[128];
    char role[32];
} Record;

static Record store[MAX_USERS];
static int    store_len = 0;
static int    next_id   = 1;

static void seed(void) {
    store[store_len++] = (Record){ next_id++, "Ada Lovelace", "ada@example.com",   "USER_ROLE_ADMIN"  };
    store[store_len++] = (Record){ next_id++, "Grace Hopper", "grace@example.com", "USER_ROLE_VIEWER" };
    store[store_len++] = (Record){ next_id++, "Alan Turing",  "alan@example.com",  "USER_ROLE_VIEWER" };
}

static int find_idx(int id) {
    for (int i = 0; i < store_len; i++)
        if (store[i].id == id) return i;
    return -1;
}

/* ── helpers ────────────────────────────────────────────────────────────── */

/* mg_str is not null-terminated; copy to int safely. */
static int mgstr_to_int(struct mg_str s) {
    char buf[16] = {0};
    size_t n = s.len < sizeof(buf) - 1 ? s.len : sizeof(buf) - 1;
    memcpy(buf, s.buf, n);
    return atoi(buf);
}

/* Copy a malloc'd mg_json_get_str result into dst, then free it. */
static void json_str_copy(struct mg_str body, const char *path,
                          char *dst, size_t dlen) {
    char *val = mg_json_get_str(body, path);
    if (val) {
        strncpy(dst, val, dlen - 1);
        dst[dlen - 1] = '\0';
        free(val);
    }
}

static void record_json(char *buf, size_t len, const Record *r) {
    snprintf(buf, len,
        "{\"id\":%d,\"name\":\"%s\",\"email\":\"%s\",\"role\":\"%s\"}",
        r->id, r->name, r->email, r->role);
}

static void store_json(char *buf, size_t len) {
    size_t pos = 0;
    pos += (size_t)snprintf(buf + pos, len - pos, "[");
    for (int i = 0; i < store_len; i++) {
        char item[512];
        record_json(item, sizeof(item), &store[i]);
        pos += (size_t)snprintf(buf + pos, len - pos,
                                "%s%s", item, i < store_len - 1 ? "," : "");
    }
    snprintf(buf + pos, len - pos, "]");
}

/* ── response helpers ───────────────────────────────────────────────────── */

#define CORS_JSON \
    "Content-Type: application/json\r\n" \
    "Access-Control-Allow-Origin: *\r\n"

#define CORS_ONLY \
    "Access-Control-Allow-Origin: *\r\n"

static void reply_json(struct mg_connection *c, int status, const char *body) {
    mg_http_reply(c, status, CORS_JSON, "%s\n", body);
}

/* ── route handlers ─────────────────────────────────────────────────────── */

static void handle_get_users(struct mg_connection *c) {
    char buf[65536];
    store_json(buf, sizeof(buf));
    reply_json(c, 200, buf);
}

static void handle_get_user(struct mg_connection *c, int id) {
    int idx = find_idx(id);
    if (idx < 0) { reply_json(c, 404, "{\"error\":\"not found\"}"); return; }
    char buf[512];
    record_json(buf, sizeof(buf), &store[idx]);
    reply_json(c, 200, buf);
}

static void handle_post_users(struct mg_connection *c, struct mg_str body) {
    if (store_len >= MAX_USERS) {
        reply_json(c, 503, "{\"error\":\"store full\"}");
        return;
    }
    Record r = {0};
    r.id = next_id++;

    json_str_copy(body, "$.name",  r.name,  sizeof(r.name));
    json_str_copy(body, "$.email", r.email, sizeof(r.email));
    json_str_copy(body, "$.role",  r.role,  sizeof(r.role));

    if (!r.name[0])  snprintf(r.name,  sizeof(r.name),  "User %d", r.id);
    if (!r.email[0]) snprintf(r.email, sizeof(r.email), "user%d@example.com", r.id);
    if (!r.role[0])  snprintf(r.role,  sizeof(r.role),  "USER_ROLE_VIEWER");

    store[store_len++] = r;

    char buf[512];
    record_json(buf, sizeof(buf), &r);
    reply_json(c, 201, buf);
}

static void handle_delete_user(struct mg_connection *c, int id) {
    int idx = find_idx(id);
    if (idx < 0) { reply_json(c, 404, "{\"error\":\"not found\"}"); return; }
    store[idx] = store[--store_len]; /* swap-remove */
    mg_http_reply(c, 204, CORS_ONLY, "");
}

/* ── event handler ──────────────────────────────────────────────────────── */

static void fn(struct mg_connection *c, int ev, void *ev_data) {
    if (ev != MG_EV_HTTP_MSG) return;
    struct mg_http_message *hm = (struct mg_http_message *)ev_data;

    bool is_get    = mg_strcasecmp(hm->method, mg_str("GET"))    == 0;
    bool is_post   = mg_strcasecmp(hm->method, mg_str("POST"))   == 0;
    bool is_delete = mg_strcasecmp(hm->method, mg_str("DELETE")) == 0;
    bool is_opts   = mg_strcasecmp(hm->method, mg_str("OPTIONS"))== 0;

    /* CORS pre-flight */
    if (is_opts) {
        mg_http_reply(c, 204,
            "Access-Control-Allow-Origin: *\r\n"
            "Access-Control-Allow-Methods: GET, POST, DELETE, OPTIONS\r\n"
            "Access-Control-Allow-Headers: Content-Type\r\n",
            "");
        return;
    }

    struct mg_str caps[2] = {{0, 0}, {0, 0}};

    /* Exact: /users */
    if (mg_match(hm->uri, mg_str("/users"), NULL)) {
        if      (is_get)  handle_get_users(c);
        else if (is_post) handle_post_users(c, hm->body);
        else              reply_json(c, 405, "{\"error\":\"method not allowed\"}");

    /* Wildcard: /users/{id} */
    } else if (mg_match(hm->uri, mg_str("/users/*"), caps)) {
        int id = mgstr_to_int(caps[0]);
        if      (is_get)    handle_get_user(c, id);
        else if (is_delete) handle_delete_user(c, id);
        else                reply_json(c, 405, "{\"error\":\"method not allowed\"}");

    } else {
        reply_json(c, 404, "{\"error\":\"not found\"}");
    }
}

/* ── main ───────────────────────────────────────────────────────────────── */

int main(void) {
    seed();

    struct mg_mgr mgr;
    mg_mgr_init(&mgr);

    if (!mg_http_listen(&mgr, "http://0.0.0.0:" PORT, fn, NULL)) {
        fprintf(stderr, "Cannot bind to port " PORT "\n");
        return 1;
    }

    printf("Sample Users API  http://localhost:" PORT "\n");
    printf("  GET    /users\n");
    printf("  GET    /users/{id}\n");
    printf("  POST   /users   {\"name\":\"...\",\"email\":\"...\",\"role\":\"...\"}\n");
    printf("  DELETE /users/{id}\n\n");

    for (;;) mg_mgr_poll(&mgr, 1000);

    mg_mgr_free(&mgr);
    return 0;
}
