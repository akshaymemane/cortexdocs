typedef struct st_h2o_hostconf_t h2o_hostconf_t;
typedef struct st_h2o_pathconf_t h2o_pathconf_t;
typedef struct st_h2o_handler_t h2o_handler_t;

typedef struct {
  const char *method;
} h2o_req_t;

int string_is(const char *value, const char *expected);
h2o_pathconf_t *h2o_config_register_path(h2o_hostconf_t *hostconf, const char *path, int flags);

static h2o_pathconf_t *register_handler(h2o_hostconf_t *hostconf, const char *path,
                                        int (*on_req)(h2o_handler_t *, h2o_req_t *)) {
  (void)hostconf;
  (void)path;
  (void)on_req;
  return 0;
}

static int chunked_test(h2o_handler_t *self, h2o_req_t *req) {
  (void)self;
  if (string_is(req->method, "GET")) {
    return 0;
  }
  return -1;
}

static int post_test(h2o_handler_t *self, h2o_req_t *req) {
  (void)self;
  if (string_is(req->method, "POST")) {
    return 0;
  }
  return -1;
}

int main(void) {
  h2o_hostconf_t *hostconf = 0;
  h2o_pathconf_t *pathconf;

  pathconf = register_handler(hostconf, "/post-test", post_test);
  (void)pathconf;
  pathconf = register_handler(hostconf, "/chunked-test", chunked_test);
  (void)pathconf;
  pathconf = h2o_config_register_path(hostconf, "/", 0);
  (void)pathconf;

  return 0;
}

