#ifndef USERS_H
#define USERS_H

/**
 * @desc Represents a user returned by the API.
 */
typedef struct User {
  int id;
  const char *name;
  const char *email;
} User;

/**
 * @desc Supported roles for user creation.
 */
enum UserRole {
  USER_ROLE_VIEWER,
  USER_ROLE_ADMIN
};

/**
 * @route GET /users
 * @desc Fetch the current user collection.
 * @param [in] limit Maximum number of rows to return.
 * @example {"limit": 10}
 * @response 200 User[] Successful lookup.
 */
int get_users(int limit);

/// @route GET /users/{id}
/// @desc Get a single user by ID.
/// @param [in] user_id The numeric user ID to fetch.
/// @response 200 User Found user.
/// @response 404 void User not found.
int get_user(int user_id);

/**
 * @route POST /users
 * @desc Create a user in the in-memory sample backend.
 * @param [in] name Display name for the user.
 * @param [in] email Email address to persist.
 * @param [in] role One of the values from UserRole.
 * @example {"name": "Grace Hopper", "email": "grace@example.com", "role": "USER_ROLE_ADMIN"}
 * @response 201 User Newly created user.
 */
int create_user(const char *name, const char *email, enum UserRole role);

/**
 * @route DELETE /users/{id}
 * @desc Remove a user permanently.
 * @deprecated Use PATCH /users/{id}/archive instead.
 * @param [in] user_id The ID of the user to delete.
 * @response 204 void User deleted.
 */
int delete_user(int user_id);

/**
 * @desc Internal helper used by the sample backend.
 * @param [in] user_id User identifier to check.
 */
int user_exists(int user_id);

#endif
