# CortexDocs

Generate API docs directly from C code.  
No JSON. No manual schemas.

⚡ C-first &nbsp;•&nbsp; ⚡ Code-driven &nbsp;•&nbsp; ⚡ Zero boilerplate

---

<!-- Replace with a real demo GIF: record `make run` → browser opening → clicking through endpoints -->
![CortexDocs Demo](docs/demo.gif)

---

## Quick start

```bash
git clone https://github.com/akshaymemane/cortexdocs
cd cortexdocs
make run
```

Open `http://localhost:8080`. Done.

---

## How it works

Write standard C doc comments. CortexDocs does the rest.

**Input — C source with doc comments:**

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

**Output — browsable docs UI with live Try API panel:**

<!-- Replace with a real screenshot of the UI -->
![CortexDocs UI](docs/screenshot.png)

No schema files. No annotation processors. Just parse, extract, render.

---

## vs. every other API doc tool

Unlike tools that require manually writing JSON schemas or YAML specs, CortexDocs extracts API definitions directly from C source code.  
Point it at a folder. Get docs.

---

## Install

Requires Go 1.21+ and `clang`.

```bash
go install github.com/akshaymemane/cortexdocs/cmd/cortexdocs@latest
```

`clang` ships with Xcode Command Line Tools on macOS. On Linux:

```bash
apt install clang   # Debian / Ubuntu
dnf install clang   # Fedora / RHEL
```

---

## Usage

```bash
# Generate docs from any C project
cortexdocs generate ./your-c-project

# Serve the docs UI
cortexdocs serve

# With options
cortexdocs generate --name "My API" --output ./docs/api.json ./src
cortexdocs serve --port 9090

# Pre-configure the Try API panel
CORTEXDOCS_TARGET_BASE_URL=http://localhost:7890 cortexdocs serve
```

---

## Comment format

```c
/**
 * @route   GET /users
 * @desc    Fetch the user collection.
 * @param   [in] limit  Maximum rows to return.
 * @example {"limit": 10}
 * @response 200 User[]  Successful lookup.
 */
int get_users(int limit);
```

Single-line `///` comments also work:

```c
/// @route GET /users/{id}
/// @desc  Get a single user by ID.
/// @response 200 User  Found user.
/// @response 404 void  Not found.
int get_user(int user_id);
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

---

## No doc comments? No problem.

CortexDocs also infers endpoints from existing C codebases with no annotations at all:

```c
// No @route needed — CortexDocs detects this pattern
if (strcmp(req->method, "GET") == 0 && strcmp(req->uri, "/users") == 0) {
    return send_json(get_users());
}
```

It also picks up H2O route registration calls and YAML config `paths:` blocks.  
Inferred endpoints are labelled **Inferred** in the UI so you always know what came from code vs. comments.

---

## Running the sample API end-to-end

The `examples/sample-c-api` directory includes a real HTTP server (built on [mongoose](https://github.com/cesanta/mongoose)) so you can try the live Try API panel:

```bash
# Terminal 1 — C API on :7890
cd examples/sample-c-api && make run

# Terminal 2 — CortexDocs UI on :8080
make run
```

Open `http://localhost:8080`, set Base URL to `http://localhost:7890`, and fire live requests directly from the docs.

---

## Repository layout

```
cortexdocs/
├── cmd/cortexdocs/     CLI entry point  (go install lands here)
├── internal/
│   ├── parser/         Clang AST walker + heuristic inference
│   ├── generator/      Converts ParseResult → JSON spec
│   ├── model/          Shared types (Go ↔ TypeScript contract)
│   └── server/         HTTP server + /api/try proxy
├── examples/
│   ├── sample-c-api/   Runnable users API (mongoose, :7890)
│   ├── heuristic-c-api/  No-annotation inference demo
│   └── h2o-style-api/  H2O registration + YAML config demo
└── web/                React/Vite frontend
```

---

## License

MIT — see [LICENSE](LICENSE).
