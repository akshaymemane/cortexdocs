# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Generate docs JSON from C source
go run ./cmd/cortexdocs generate ./examples/sample-c-api

# Serve docs at http://localhost:8080
go run ./cmd/cortexdocs serve

# With flags
go run ./cmd/cortexdocs generate --name "My API" --output ./output/api.json ./src
go run ./cmd/cortexdocs serve --port 9090

# Pre-configure Try API target
CORTEXDOCS_TARGET_BASE_URL=http://localhost:7890 go run ./cmd/cortexdocs serve

# Build and install the binary locally
go install ./cmd/cortexdocs

# Build the React frontend
cd web && npm install && npm run build

# Frontend dev server
cd web && npm run dev

# Build and run the sample C API server (listens on :7890)
cd examples/sample-c-api && make run
```

`clang` must be installed and on PATH — the parser shells out to `clang -Xclang -ast-dump=json`.

## Architecture

### Pipeline

`ParsePath` → `BuildSpec` → `WriteJSON` → `server.Start`

1. **`internal/parser`** — drives everything:
   - Shells out to `clang -Xclang -ast-dump=json -fsyntax-only` per `.c`/`.h` file, unmarshals the AST JSON, then walks nodes for `FunctionDecl`, `RecordDecl`, `EnumDecl`.
   - `comments.go` reads raw comment blocks from source text (`/** */` and `///`); `nearestComment` correlates a block to the AST node by line proximity. Supports `@route`, `@desc`, `@param [in/out]`, `@response`, `@example`, `@deprecated`.
   - `heuristics.go` runs _after_ AST walk: `inferEndpoints` scans source lines for route patterns (strcmp method/URI checks, H2O registration calls) and produces `EndpointDoc` candidates ranked by confidence. `attachHeuristicRoutes` back-fills route info onto `FunctionDoc` entries.
   - `config.go` handles H2O YAML config files — parses `paths:` blocks into additional endpoint candidates.
   - `types.go` holds the internal IR (`ParseResult`, `FunctionDoc`, `EndpointDoc`, `StructDoc`, `EnumDoc`).

2. **`internal/generator`** — `BuildSpec` converts `ParseResult` into `model.Spec` and `WriteJSON` writes `output/api.json`. Accepts a `name` parameter for the API title.

3. **`internal/model`** — shared JSON-serialisable types that are the contract between the generator and the frontend.

4. **`internal/server`** — stdlib `net/http` server. Serves `output/api.json` at `/api.json`, proxies Try API calls at `POST /api/try` (handles CORS and self-signed TLS). Falls back to inline `fallbackHTML` if `web/dist` doesn't exist.

### Frontend (`web/`)

React 18 + Vite + TypeScript. Fetches `/api.json` on load. Sidebar has three tabs (Endpoints / Functions / Types) with keyboard navigation. Try API panel supports path parameter substitution, custom headers, and `@example` pre-fill.

### Key invariant

`output/api.json` is the sole handoff between generator and UI. The Go model types in `internal/model/model.go` and the TypeScript types in `web/src/types.ts` must stay in sync manually.

### Endpoint source field

Each endpoint in the model carries `source`: `"docblock"` (explicit `@route` annotation), `"heuristic"` (inferred from C code patterns), or `"config"` (from H2O YAML). The UI renders these as Documented / Inferred / Config badges.
