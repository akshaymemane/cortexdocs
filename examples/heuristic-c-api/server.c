#include <string.h>

struct request {
  const char *method;
  const char *uri;
};

static int send_json(const char *payload) {
  return payload != 0;
}

int api_handler(struct request *req) {
  if (strcmp(req->method, "GET") == 0 && strcmp(req->uri, "/users") == 0) {
    return send_json("{\"items\":[]}");
  }

  if (strcmp(req->method, "POST") == 0 && strcmp(req->uri, "/users") == 0) {
    return send_json("{\"created\":true}");
  }

  if (strcmp(req->uri, "/health") == 0) {
    return send_json("{\"status\":\"ok\"}");
  }

  return 0;
}

