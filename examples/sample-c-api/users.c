#include "users.h"

int get_users(int limit) {
  return limit >= 0 ? limit : 0;
}

int get_user(int user_id) {
  return user_id > 0 ? user_id : -1;
}

int create_user(const char *name, const char *email, enum UserRole role) {
  (void)name;
  (void)email;
  (void)role;
  return 1;
}

int delete_user(int user_id) {
  (void)user_id;
  return 0;
}

int user_exists(int user_id) {
  return user_id > 0;
}
