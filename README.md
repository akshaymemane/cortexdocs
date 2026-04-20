# CortexDocs

CortexDocs is a C-first API documentation generator. Point it at a folder containing `.c` and `.h` files, and it turns Clang AST output plus doc comments into a normalized API model and a browsable docs experience.

## Install

**Requirements:** Go 1.21+ and `clang`.

```bash
go install github.com/akshaymemane/cortexdocs/cmd/cortexdocs@latest
```

`clang` ships with Xcode Command Line Tools on macOS. On Linux:

```bash
apt install clang   # Debian / Ubuntu
dnf install clang   # Fedora / RHEL
```

## Quick start

```bash
cortexdocs generate ./your-c-project
cortexdocs serve
```

Then open `http://localhost:8080`.

## What it does

- Parses C source and header files with `clang -Xclang -ast-dump=json`
- Extracts functions, structs, enums, and doc comments
- Converts comment annotations like `@route`, `@desc`, `@param`, `@response`, `@example`, and `@deprecated` into a normalized JSON IR
- Heuristically discovers endpoints from existing C code — method checks, URI comparisons, handler registration patterns
- Detects H2O-style route registration and YAML config `paths:` blocks
- Writes docs data to `output/api.json`
- Serves a React/Vite frontend from `web/dist`, or a built-in zero-dependency fallback viewer when the frontend is not built

## Comment format

```c
/**
 * @route GET /users
 * @desc Fetch the user collection.
 * @param [in] limit Maximum number of rows to return.
 * @example {"limit": 10}
 * @response 200 User[] Successful lookup.
 */
int get_users(int limit);
```

Single-line `///` doc comments are also supported:

```c
/// @route GET /users/{id}
/// @desc Get a single user by ID.
/// @param [in] user_id The numeric user ID.
/// @response 200 User Found user.
int get_user(int user_id);
```

Mark endpoints as deprecated:

```c
/**
 * @route DELETE /users/{id}
 * @deprecated Use PATCH /users/{id}/archive instead.
 */
int delete_user(int user_id);
```

### Supported tags

| Tag | Description |
|---|---|
| `@route METHOD /path` | Marks the function as an HTTP endpoint |
| `@desc text` | Description shown in the UI |
| `@param [in\|out\|in,out] name text` | Parameter with optional direction |
| `@response STATUS TYPE text` | Response variant |
| `@example json` | Pre-fills the Try API request body |
| `@deprecated` | Marks the endpoint or function as deprecated |

## CLI options

```bash
# Generate with a custom API name and output path
cortexdocs generate --name "My API" --output ./docs/api.json ./src

# Serve on a different port
cortexdocs serve --port 9090

# Pre-configure the Try API panel against a live server
CORTEXDOCS_TARGET_BASE_URL=http://localhost:7890 cortexdocs serve
```

## Repository layout

```text
cortexdocs/
├── cmd/
│   └── cortexdocs/     # CLI entry point (go install lands here)
├── internal/
│   ├── generator/      # Converts ParseResult → model.Spec
│   ├── model/          # Shared JSON-serialisable types
│   ├── parser/         # Clang AST walker + heuristic inference
│   └── server/         # HTTP server, /api/try proxy
├── examples/
│   └── sample-c-api/   # Runnable C API (mongoose, :7890)
├── output/             # Generated api.json lives here
└── web/                # React/Vite frontend
```

## Running the sample API

The `examples/sample-c-api` directory contains a fully runnable HTTP server (built on [mongoose](https://github.com/cesanta/mongoose)) so you can try the live Try API panel end-to-end.

```bash
# Terminal 1 — build and start the C API on :7890
cd examples/sample-c-api && make run

# Terminal 2 — start CortexDocs
cortexdocs generate ./examples/sample-c-api
cortexdocs serve
```

Open `http://localhost:8080`, set the Base URL to `http://localhost:7890`, and run live requests directly from the docs UI.

## Frontend

The React app lives in `web/` and fetches `/api.json` from the Go server.

```bash
cd web && npm install && npm run build
```

If `web/dist` does not exist, `cortexdocs serve` falls back to an inline viewer so docs are always accessible with zero frontend build steps.

## Heuristic extraction

For codebases that do not use CortexDocs doc comments, the parser infers endpoints from:

- `strcmp(req->method, "GET")` style method checks
- `strcmp(req->uri, "/users")` or `mg_match(...)` URI comparisons
- Handler registration calls like `register_handler(hostconf, "/foo", handler)`
- H2O `h2o_config_register_path(...)` calls
- H2O YAML `paths:` blocks in `.conf`, `.yaml`, and `.yml` files
- REST-shaped function names like `get_users`, `create_user`, `delete_user`

Heuristically inferred endpoints are labelled **Inferred** in the UI; doc-comment endpoints are labelled **Documented**.

## Notes

- `output/api.json` is the contract between the generator and the UI — the Go model types in `internal/model` and the TypeScript types in `web/src/types.ts` must stay in sync.
- The parser requires `clang` on `PATH`. If a file fails to parse, a warning is recorded in `api.json` and shown in the UI rather than aborting the whole run.
