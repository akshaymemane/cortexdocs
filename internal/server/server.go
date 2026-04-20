package server

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func Start(addr string, root string) error {
	outputJSON := filepath.Join(root, "output", "api.json")
	distDir := filepath.Join(root, "web", "dist")
	defaultTargetBaseURL := strings.TrimSpace(os.Getenv("CORTEXDOCS_TARGET_BASE_URL"))

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("/api.json", func(writer http.ResponseWriter, request *http.Request) {
		http.ServeFile(writer, request, outputJSON)
	})
	mux.HandleFunc("/output/api.json", func(writer http.ResponseWriter, request *http.Request) {
		http.ServeFile(writer, request, outputJSON)
	})
	mux.HandleFunc("/api/runtime", func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(runtimeConfig{
			DefaultTargetBaseURL: defaultTargetBaseURL,
		})
	})
	mux.HandleFunc("/api/try", func(writer http.ResponseWriter, request *http.Request) {
		handleTryAPI(writer, request, defaultTargetBaseURL)
	})

	if isDir(distDir) {
		fileServer := http.FileServer(http.Dir(distDir))
		mux.Handle("/", http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			path := filepath.Join(distDir, request.URL.Path)
			if request.URL.Path != "/" && !isFile(path) {
				http.ServeFile(writer, request, filepath.Join(distDir, "index.html"))
				return
			}
			fileServer.ServeHTTP(writer, request)
		}))
	} else {
		mux.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
			writer.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = writer.Write([]byte(fallbackHTML))
		})
	}

	fmt.Printf("Serving CortexDocs at http://localhost%s\n", addr)
	fmt.Printf("API JSON available at http://localhost%s/api.json\n", addr)
	if defaultTargetBaseURL != "" {
		fmt.Printf("Default Try API target: %s\n", defaultTargetBaseURL)
	}
	return http.ListenAndServe(addr, mux)
}

type runtimeConfig struct {
	DefaultTargetBaseURL string `json:"defaultTargetBaseUrl"`
}

type tryAPIRequest struct {
	BaseURL  string            `json:"baseUrl"`
	Method   string            `json:"method"`
	Path     string            `json:"path"`
	Body     string            `json:"body"`
	Headers  map[string]string `json:"headers"`
	Insecure bool              `json:"insecure"`
}

type tryAPIResponse struct {
	RequestedURL string              `json:"requestedUrl"`
	Method       string              `json:"method"`
	Status       int                 `json:"status"`
	StatusText   string              `json:"statusText"`
	Headers      map[string][]string `json:"headers"`
	Body         string              `json:"body"`
}

type tryAPIError struct {
	Error string `json:"error"`
}

func handleTryAPI(writer http.ResponseWriter, request *http.Request, defaultTargetBaseURL string) {
	if request.Method != http.MethodPost {
		writer.Header().Set("Allow", http.MethodPost)
		http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload tryAPIRequest
	if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
		writeTryAPIError(writer, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}

	baseURL := strings.TrimSpace(payload.BaseURL)
	if baseURL == "" {
		baseURL = defaultTargetBaseURL
	}
	if baseURL == "" {
		writeTryAPIError(writer, http.StatusBadRequest, "missing base URL")
		return
	}
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		writeTryAPIError(writer, http.StatusBadRequest, "base URL must start with http:// or https://")
		return
	}

	method := strings.ToUpper(strings.TrimSpace(payload.Method))
	if method == "" || method == "ANY" {
		method = http.MethodGet
	}
	targetURL := joinURL(baseURL, payload.Path)

	httpRequest, err := http.NewRequest(method, targetURL, bytes.NewBufferString(payload.Body))
	if err != nil {
		writeTryAPIError(writer, http.StatusBadRequest, fmt.Sprintf("build request: %v", err))
		return
	}

	for name, value := range payload.Headers {
		if strings.TrimSpace(name) == "" || strings.TrimSpace(value) == "" {
			continue
		}
		httpRequest.Header.Set(name, value)
	}
	if payload.Body != "" && httpRequest.Header.Get("Content-Type") == "" {
		httpRequest.Header.Set("Content-Type", "text/plain; charset=utf-8")
	}

	client := &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: payload.Insecure || isLocalHTTPS(baseURL),
			},
		},
	}

	response, err := client.Do(httpRequest)
	if err != nil {
		writeTryAPIError(writer, http.StatusBadGateway, fmt.Sprintf("request failed: %v", err))
		return
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		writeTryAPIError(writer, http.StatusBadGateway, fmt.Sprintf("read response: %v", err))
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(writer).Encode(tryAPIResponse{
		RequestedURL: targetURL,
		Method:       method,
		Status:       response.StatusCode,
		StatusText:   response.Status,
		Headers:      response.Header,
		Body:         string(body),
	})
}

func writeTryAPIError(writer http.ResponseWriter, status int, message string) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(tryAPIError{Error: message})
}

func joinURL(baseURL string, routePath string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	routePath = strings.TrimSpace(routePath)
	if routePath == "" {
		return baseURL
	}
	if strings.HasPrefix(routePath, "http://") || strings.HasPrefix(routePath, "https://") {
		return routePath
	}
	if !strings.HasPrefix(routePath, "/") {
		routePath = "/" + routePath
	}
	return baseURL + routePath
}

func isLocalHTTPS(baseURL string) bool {
	return strings.HasPrefix(baseURL, "https://127.0.0.1") || strings.HasPrefix(baseURL, "https://localhost")
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func isFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

const fallbackHTML = `<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>CortexDocs</title>
    <style>
      :root {
        --bg: #f5f1e8;
        --card: rgba(255, 255, 255, 0.82);
        --ink: #17221c;
        --muted: #57635b;
        --border: rgba(23, 34, 28, 0.12);
        --accent: #db6c3f;
        --accent-2: #0c8f6d;
      }
      * { box-sizing: border-box; }
      body {
        margin: 0;
        font-family: "Avenir Next", "Segoe UI", sans-serif;
        color: var(--ink);
        background:
          radial-gradient(circle at top left, rgba(219, 108, 63, 0.20), transparent 24rem),
          radial-gradient(circle at top right, rgba(12, 143, 109, 0.18), transparent 26rem),
          var(--bg);
      }
      .layout {
        min-height: 100vh;
        display: grid;
        grid-template-columns: 320px 1fr;
      }
      .sidebar {
        padding: 2rem 1.25rem;
        border-right: 1px solid var(--border);
        backdrop-filter: blur(18px);
        background: rgba(255,255,255,0.48);
      }
      .content {
        padding: 2rem;
      }
      .eyebrow {
        color: var(--accent-2);
        text-transform: uppercase;
        letter-spacing: 0.12em;
        font-size: 0.74rem;
        font-weight: 700;
      }
      h1 {
        margin: 0.5rem 0 0.75rem;
        font-size: clamp(2rem, 4vw, 3.6rem);
        line-height: 0.96;
      }
      .muted { color: var(--muted); }
      .endpoint {
        padding: 0.9rem 1rem;
        border: 1px solid var(--border);
        border-radius: 18px;
        background: var(--card);
        margin-bottom: 0.75rem;
        cursor: pointer;
      }
      .endpoint.active {
        border-color: rgba(12, 143, 109, 0.4);
        box-shadow: 0 10px 30px rgba(23, 34, 28, 0.08);
      }
      .method {
        display: inline-block;
        padding: 0.2rem 0.55rem;
        border-radius: 999px;
        font-size: 0.72rem;
        font-weight: 700;
        background: rgba(12, 143, 109, 0.12);
        color: var(--accent-2);
      }
      .card {
        border: 1px solid var(--border);
        border-radius: 28px;
        background: var(--card);
        padding: 1.4rem;
        box-shadow: 0 18px 40px rgba(23, 34, 28, 0.08);
      }
      pre {
        overflow: auto;
        padding: 1rem;
        border-radius: 18px;
        background: #112019;
        color: #e4f9ef;
      }
      @media (max-width: 900px) {
        .layout { grid-template-columns: 1fr; }
        .sidebar { border-right: 0; border-bottom: 1px solid var(--border); }
      }
    </style>
  </head>
  <body>
    <div class="layout">
      <aside class="sidebar">
        <div class="eyebrow">C-first API docs</div>
        <h1>CortexDocs</h1>
        <p class="muted">Build the React app in <code>web/</code> for the full experience. This fallback viewer keeps <code>cortexdocs serve</code> useful with zero frontend build steps.</p>
        <div id="nav"></div>
      </aside>
      <main class="content">
        <div class="card" id="panel">Loading generated API model…</div>
      </main>
    </div>
    <script>
      const nav = document.getElementById("nav");
      const panel = document.getElementById("panel");

      fetch("/api.json")
        .then((response) => response.json())
        .then((data) => {
          if (!data.endpoints || data.endpoints.length === 0) {
            panel.innerHTML = "<h2>No endpoints found</h2><p class='muted'>Run <code>go run ./cli generate ./examples/sample-c-api</code> first.</p>";
            return;
          }

          let active = data.endpoints[0];
          const render = () => {
            panel.innerHTML = [
              "<div class='eyebrow'>Endpoint</div>",
              "<h2>" + active.name + "</h2>",
              "<p><span class='method'>" + active.method + "</span> <strong>" + active.path + "</strong></p>",
              "<p class='muted'>" + active.description + "</p>",
              "<h3>Signature</h3>",
              "<pre>" + active.signature + "</pre>",
              "<h3>Request Params</h3>",
              "<pre>" + JSON.stringify(active.params, null, 2) + "</pre>",
              "<h3>Responses</h3>",
              "<pre>" + JSON.stringify(active.responses, null, 2) + "</pre>"
            ].join("");

            Array.from(nav.querySelectorAll(".endpoint")).forEach((element) => {
              element.classList.toggle("active", element.dataset.id === active.id);
            });
          };

          nav.innerHTML = data.endpoints.map((endpoint) => [
            "<button class='endpoint' data-id='" + endpoint.id + "'>",
            "<div><span class='method'>" + endpoint.method + "</span></div>",
            "<div style='margin-top:0.55rem;font-weight:700'>" + endpoint.path + "</div>",
            "<div class='muted' style='margin-top:0.4rem'>" + endpoint.name + "</div>",
            "</button>"
          ].join("")).join("");

          Array.from(nav.querySelectorAll(".endpoint")).forEach((element) => {
            element.addEventListener("click", () => {
              active = data.endpoints.find((endpoint) => endpoint.id === element.dataset.id) || data.endpoints[0];
              render();
            });
          });

          render();
        })
        .catch(() => {
          panel.innerHTML = "<h2>API model missing</h2><p class='muted'>Generate docs first so <code>/output/api.json</code> exists.</p>";
        });
    </script>
  </body>
</html>
`
