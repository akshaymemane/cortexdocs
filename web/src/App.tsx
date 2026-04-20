import {
  startTransition,
  useDeferredValue,
  useEffect,
  useRef,
  useState,
} from "react";
import type {
  ApiEnum,
  ApiFunction,
  ApiSpec,
  ApiStruct,
  Endpoint,
  Parameter,
  ResponseDoc,
  RuntimeConfig,
  TryApiResponse,
} from "./types";

// ─── Method colours ──────────────────────────────────────────────────────────

const METHOD_BG: Record<string, string> = {
  GET: "bg-sea/15 text-sea",
  POST: "bg-coral/15 text-coral",
  PUT: "bg-amber-500/15 text-amber-600",
  DELETE: "bg-red-500/15 text-red-600",
  PATCH: "bg-violet-500/15 text-violet-600",
  HEAD: "bg-sky-500/15 text-sky-600",
  OPTIONS: "bg-slate-400/15 text-slate-500",
  ANY: "bg-ink/8 text-ink/50",
};
const METHOD_BORDER: Record<string, string> = {
  GET: "border-sea/30",
  POST: "border-coral/30",
  PUT: "border-amber-500/30",
  DELETE: "border-red-500/30",
  PATCH: "border-violet-500/30",
  HEAD: "border-sky-500/30",
  OPTIONS: "border-slate-400/30",
  ANY: "border-ink/15",
};

function methodBg(m: string) {
  return METHOD_BG[m.toUpperCase()] ?? METHOD_BG.ANY;
}
function methodBorder(m: string) {
  return METHOD_BORDER[m.toUpperCase()] ?? METHOD_BORDER.ANY;
}

// ─── Source badge ─────────────────────────────────────────────────────────────

const SOURCE_META: Record<string, { label: string; cls: string }> = {
  docblock: { label: "Documented", cls: "bg-sea/10 text-sea" },
  heuristic: { label: "Inferred", cls: "bg-coral/10 text-coral" },
  config: { label: "Config", cls: "bg-sand text-ink/55" },
};

// ─── Utilities ────────────────────────────────────────────────────────────────

function shortFile(file: string, line: number) {
  const parts = file.split("/");
  return `${parts.slice(-2).join("/")}:${line}`;
}

function extractPathParams(path: string): string[] {
  return [...path.matchAll(/\{([^}]+)\}/g)].map((m) => m[1]);
}

function supportsBody(method?: string) {
  return method === "POST" || method === "PUT" || method === "PATCH";
}

function isInsecureLocal(url: string) {
  return (
    url.startsWith("https://127.0.0.1") ||
    url.startsWith("https://localhost")
  );
}

function statusClass(status: string) {
  const code = parseInt(status);
  if (code < 300) return "bg-sea/12 text-sea";
  if (code < 400) return "bg-amber-500/12 text-amber-600";
  return "bg-red-500/12 text-red-600";
}

// ─── Atomic components ────────────────────────────────────────────────────────

function MethodBadge({ method }: { method: string }) {
  return (
    <span
      className={`rounded-full px-2.5 py-0.5 text-[11px] font-bold uppercase tracking-widest ${methodBg(method)}`}
    >
      {method}
    </span>
  );
}

function SourceTag({ source }: { source?: string }) {
  const s = SOURCE_META[source ?? ""];
  if (!s) return null;
  return (
    <span
      className={`rounded-full px-2.5 py-0.5 text-[10px] font-semibold uppercase tracking-wider ${s.cls}`}
    >
      {s.label}
    </span>
  );
}

function DeprecatedBadge() {
  return (
    <span className="rounded-full bg-red-500/10 px-2.5 py-0.5 text-[10px] font-semibold uppercase tracking-wider text-red-500">
      Deprecated
    </span>
  );
}

function CopyButton({ text, small }: { text: string; small?: boolean }) {
  const [copied, setCopied] = useState(false);
  return (
    <button
      onClick={() => {
        void navigator.clipboard.writeText(text).then(() => {
          setCopied(true);
          setTimeout(() => setCopied(false), 1500);
        });
      }}
      className={`rounded-lg transition ${
        small
          ? "px-2 py-0.5 text-[11px] text-white/50 hover:text-white/90"
          : "px-3 py-1 text-xs font-medium text-ink/40 hover:text-ink"
      }`}
    >
      {copied ? "Copied!" : "Copy"}
    </button>
  );
}

function StatusBadge({ status }: { status: string }) {
  return (
    <span
      className={`rounded-full px-2 py-0.5 text-[11px] font-bold ${statusClass(status)}`}
    >
      {status}
    </span>
  );
}

// ─── Table components ────────────────────────────────────────────────────────

function ParamsTable({ params }: { params: Parameter[] }) {
  if (!params.length)
    return <p className="text-sm italic text-ink/40">No parameters.</p>;
  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead>
          <tr className="text-left text-[11px] font-semibold uppercase tracking-widest text-ink/35">
            <th className="pb-3 pr-5">Name</th>
            <th className="pb-3 pr-5">Type</th>
            <th className="pb-3 pr-5">Dir</th>
            <th className="pb-3">Description</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-ink/6">
          {params.map((p) => (
            <tr key={p.name} className="group">
              <td className="py-2.5 pr-5 font-mono text-[13px] font-medium text-ink">
                {p.name}
              </td>
              <td className="py-2.5 pr-5 font-mono text-[13px] text-sea/80">
                {p.type}
              </td>
              <td className="py-2.5 pr-5">
                <span className="rounded-full bg-sand/70 px-2 py-0.5 text-[10px] font-medium text-ink/50">
                  {p.direction || "in"}
                </span>
              </td>
              <td className="py-2.5 text-ink/60">
                {p.description || <span className="text-ink/25">—</span>}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function ResponsesTable({ responses }: { responses: ResponseDoc[] }) {
  if (!responses.length)
    return <p className="text-sm italic text-ink/40">No response docs.</p>;
  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead>
          <tr className="text-left text-[11px] font-semibold uppercase tracking-widest text-ink/35">
            <th className="pb-3 pr-5">Status</th>
            <th className="pb-3 pr-5">Type</th>
            <th className="pb-3">Description</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-ink/6">
          {responses.map((r, i) => (
            <tr key={i}>
              <td className="py-2.5 pr-5">
                <StatusBadge status={r.status} />
              </td>
              <td className="py-2.5 pr-5 font-mono text-[13px] text-sea/80">
                {r.type}
              </td>
              <td className="py-2.5 text-ink/60">
                {r.description || <span className="text-ink/25">—</span>}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function FieldsTable({ fields }: { fields: ApiStruct["fields"] }) {
  if (!fields.length)
    return <p className="text-sm italic text-ink/40">No fields.</p>;
  return (
    <table className="w-full text-sm">
      <thead>
        <tr className="text-left text-[11px] font-semibold uppercase tracking-widest text-ink/35">
          <th className="pb-3 pr-5">Field</th>
          <th className="pb-3 pr-5">Type</th>
          <th className="pb-3">Description</th>
        </tr>
      </thead>
      <tbody className="divide-y divide-ink/6">
        {fields.map((f) => (
          <tr key={f.name}>
            <td className="py-2.5 pr-5 font-mono text-[13px] font-medium text-ink">
              {f.name}
            </td>
            <td className="py-2.5 pr-5 font-mono text-[13px] text-sea/80">
              {f.type}
            </td>
            <td className="py-2.5 text-ink/60">
              {f.description || <span className="text-ink/25">—</span>}
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

function EnumValuesTable({ values }: { values: ApiEnum["values"] }) {
  if (!values.length)
    return <p className="text-sm italic text-ink/40">No values.</p>;
  return (
    <table className="w-full text-sm">
      <thead>
        <tr className="text-left text-[11px] font-semibold uppercase tracking-widest text-ink/35">
          <th className="pb-3 pr-5">Constant</th>
          <th className="pb-3">Description</th>
        </tr>
      </thead>
      <tbody className="divide-y divide-ink/6">
        {values.map((v) => (
          <tr key={v.name}>
            <td className="py-2.5 pr-5 font-mono text-[13px] font-medium text-ink">
              {v.name}
            </td>
            <td className="py-2.5 text-ink/60">
              {v.description || <span className="text-ink/25">—</span>}
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

// ─── Panel wrapper ────────────────────────────────────────────────────────────

function Panel({
  title,
  badge,
  children,
}: {
  title: React.ReactNode;
  badge?: React.ReactNode;
  children: React.ReactNode;
}) {
  return (
    <section className="animate-fade-up rounded-[34px] border border-ink/10 bg-white/72 p-6 shadow-float backdrop-blur-xl">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <h2 className="font-display text-2xl leading-none">{title}</h2>
        {badge && <div className="flex flex-wrap items-center gap-2">{badge}</div>}
      </div>
      <div className="mt-5">{children}</div>
    </section>
  );
}

// ─── Signature block ──────────────────────────────────────────────────────────

function SignatureBlock({ signature }: { signature: string }) {
  return (
    <div className="relative mt-5 rounded-[24px] bg-ink px-5 py-4 shadow-float">
      <div className="flex items-center justify-between">
        <span className="text-[10px] font-semibold uppercase tracking-widest text-white/35">
          Signature
        </span>
        <CopyButton text={signature} small />
      </div>
      <pre className="mt-2 overflow-x-auto whitespace-pre-wrap font-mono text-[13px] leading-6 text-white">
        {signature}
      </pre>
    </div>
  );
}

// ─── Try API panel ────────────────────────────────────────────────────────────

function TryApiPanel({
  endpoint,
  targetBaseUrl,
  setTargetBaseUrl,
}: {
  endpoint: Endpoint;
  targetBaseUrl: string;
  setTargetBaseUrl: (v: string) => void;
}) {
  const [requestBody, setRequestBody] = useState(endpoint.example ?? "");
  const [pathParams, setPathParams] = useState<Record<string, string>>(() =>
    Object.fromEntries(extractPathParams(endpoint.path).map((n) => [n, ""])),
  );
  const [headers, setHeaders] = useState<{ key: string; val: string }[]>([]);
  const [showHeaders, setShowHeaders] = useState(false);
  const [tryResult, setTryResult] = useState("");
  const [isRunning, setIsRunning] = useState(false);

  const pathParamNames = extractPathParams(endpoint.path);

  const run = async () => {
    if (!targetBaseUrl) {
      const mocked =
        endpoint.method === "GET"
          ? [{ id: 1, name: "Ada Lovelace", email: "ada@example.dev" }]
          : endpoint.method === "POST"
            ? { id: 42, name: "Grace Hopper", email: "grace@example.dev" }
            : { ok: true };
      setTryResult(
        JSON.stringify(
          {
            mode: "mock",
            hint: "Set a base URL to send real requests.",
            method: endpoint.method,
            path: endpoint.path,
            status: endpoint.responses[0]?.status ?? "200",
            body: mocked,
          },
          null,
          2,
        ),
      );
      return;
    }

    let resolvedPath = endpoint.path;
    for (const [k, v] of Object.entries(pathParams)) {
      if (v) resolvedPath = resolvedPath.replace(`{${k}}`, v);
    }
    const headersMap: Record<string, string> = {};
    for (const h of headers) {
      if (h.key.trim() && h.val.trim()) headersMap[h.key.trim()] = h.val.trim();
    }

    setIsRunning(true);
    try {
      const res = await fetch("/api/try", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          baseUrl: targetBaseUrl,
          method: endpoint.method,
          path: resolvedPath,
          body: supportsBody(endpoint.method) ? requestBody : "",
          insecure: isInsecureLocal(targetBaseUrl),
          headers: headersMap,
        }),
      });
      const payload = (await res.json()) as TryApiResponse | { error: string };
      setTryResult(JSON.stringify(payload, null, 2));
    } catch (err) {
      setTryResult(
        JSON.stringify(
          { error: err instanceof Error ? err.message : "Unknown error" },
          null,
          2,
        ),
      );
    } finally {
      setIsRunning(false);
    }
  };

  return (
    <Panel
      title="Try API"
      badge={
        <span
          className={`rounded-full px-3 py-1 text-[10px] font-bold uppercase tracking-widest ${
            targetBaseUrl ? "bg-sea/12 text-sea" : "bg-sand text-ink/50"
          }`}
        >
          {targetBaseUrl ? "Live" : "Mock"}
        </span>
      }
    >
      {pathParamNames.length > 0 && (
        <div className="mb-4">
          <p className="mb-2 text-[11px] font-semibold uppercase tracking-widest text-ink/40">
            Path Parameters
          </p>
          <div className="grid gap-2">
            {pathParamNames.map((name) => (
              <div key={name} className="flex items-center gap-3">
                <span className="w-24 shrink-0 font-mono text-xs text-ink/55">
                  {"{"}
                  {name}
                  {"}"}
                </span>
                <input
                  className="flex-1 rounded-2xl border border-ink/10 bg-white/80 px-3 py-2 text-sm outline-none transition focus:border-sea"
                  placeholder={name}
                  value={pathParams[name] ?? ""}
                  onChange={(e) =>
                    setPathParams((prev) => ({ ...prev, [name]: e.target.value }))
                  }
                />
              </div>
            ))}
          </div>
        </div>
      )}

      <p className="mb-2 text-[11px] font-semibold uppercase tracking-widest text-ink/40">
        Base URL
      </p>
      <input
        className="w-full rounded-2xl border border-ink/10 bg-white/80 px-4 py-3 text-sm outline-none transition focus:border-sea"
        placeholder="http://localhost:7890"
        value={targetBaseUrl}
        onChange={(e) => setTargetBaseUrl(e.target.value)}
      />

      {supportsBody(endpoint.method) && (
        <>
          <p className="mb-2 mt-4 text-[11px] font-semibold uppercase tracking-widest text-ink/40">
            Request Body
          </p>
          <textarea
            className="min-h-[100px] w-full rounded-[22px] border border-ink/10 bg-white/80 px-4 py-3 font-mono text-sm outline-none transition focus:border-sea"
            placeholder="{}"
            value={requestBody}
            onChange={(e) => setRequestBody(e.target.value)}
          />
        </>
      )}

      <div className="mt-4">
        <button
          onClick={() => setShowHeaders((h) => !h)}
          className="flex items-center gap-1.5 text-xs font-semibold text-ink/40 transition hover:text-ink"
        >
          <span className="text-[10px]">{showHeaders ? "▾" : "▸"}</span>
          Headers ({headers.length})
        </button>
        {showHeaders && (
          <div className="mt-2 space-y-2">
            {headers.map((h, i) => (
              <div key={i} className="flex gap-2">
                <input
                  className="flex-1 rounded-xl border border-ink/10 bg-white/80 px-3 py-2 text-sm outline-none focus:border-sea"
                  placeholder="Header-Name"
                  value={h.key}
                  onChange={(e) =>
                    setHeaders((prev) =>
                      prev.map((x, j) =>
                        j === i ? { ...x, key: e.target.value } : x,
                      ),
                    )
                  }
                />
                <input
                  className="flex-1 rounded-xl border border-ink/10 bg-white/80 px-3 py-2 text-sm outline-none focus:border-sea"
                  placeholder="value"
                  value={h.val}
                  onChange={(e) =>
                    setHeaders((prev) =>
                      prev.map((x, j) =>
                        j === i ? { ...x, val: e.target.value } : x,
                      ),
                    )
                  }
                />
                <button
                  onClick={() =>
                    setHeaders((prev) => prev.filter((_, j) => j !== i))
                  }
                  className="px-2 text-lg text-ink/30 transition hover:text-ink"
                >
                  ×
                </button>
              </div>
            ))}
            <button
              onClick={() => setHeaders((prev) => [...prev, { key: "", val: "" }])}
              className="text-xs font-semibold text-sea transition hover:text-sea/70"
            >
              + Add header
            </button>
          </div>
        )}
      </div>

      <button
        className="mt-5 rounded-full bg-coral px-5 py-3 text-sm font-semibold text-white transition hover:bg-[#c85f36] disabled:cursor-not-allowed disabled:opacity-50"
        onClick={run}
        disabled={isRunning}
      >
        {isRunning ? "Running…" : `Run ${endpoint.method}`}
      </button>

      <div className="relative mt-5">
        <pre className="min-h-[200px] overflow-auto rounded-[24px] bg-ink px-5 py-4 text-[13px] leading-6 text-white">
          {tryResult ||
            `// ${endpoint.method} ${endpoint.path}\n// Set a base URL above and click Run.`}
        </pre>
        {tryResult && (
          <div className="absolute right-3 top-3">
            <CopyButton text={tryResult} small />
          </div>
        )}
      </div>
    </Panel>
  );
}

// ─── Detail panels ────────────────────────────────────────────────────────────

function EndpointDetail({
  endpoint,
  targetBaseUrl,
  setTargetBaseUrl,
}: {
  endpoint: Endpoint;
  targetBaseUrl: string;
  setTargetBaseUrl: (v: string) => void;
}) {
  return (
    <div className="space-y-5">
      <Panel
        title={
          <span className="flex flex-wrap items-center gap-2">
            <MethodBadge method={endpoint.method} />
            <span className="font-mono">{endpoint.path}</span>
          </span>
        }
        badge={
          <>
            <SourceTag source={endpoint.source} />
            {endpoint.deprecated && <DeprecatedBadge />}
          </>
        }
      >
        <p className="text-xs font-semibold uppercase tracking-widest text-ink/40">
          {endpoint.name}
        </p>
        <p className="mt-3 max-w-2xl text-base leading-7 text-ink/70">
          {endpoint.description}
        </p>
        <SignatureBlock signature={endpoint.signature} />
        <div className="mt-5 grid gap-3 sm:grid-cols-2">
          <InfoCard label="Return Type" value={endpoint.returnType} mono />
          <InfoCard
            label="Source"
            value={shortFile(endpoint.file, endpoint.line)}
            mono
          />
        </div>
      </Panel>

      <div className="grid gap-5 xl:grid-cols-2">
        <Panel
          title="Parameters"
          badge={
            <span className="rounded-full bg-sand/80 px-3 py-1 text-[11px] font-semibold text-ink/45">
              {endpoint.params.length}
            </span>
          }
        >
          <ParamsTable params={endpoint.params} />
        </Panel>
        <Panel
          title="Responses"
          badge={
            <span className="rounded-full bg-sand/80 px-3 py-1 text-[11px] font-semibold text-ink/45">
              {endpoint.responses.length}
            </span>
          }
        >
          <ResponsesTable responses={endpoint.responses} />
        </Panel>
      </div>

      <TryApiPanel
        key={endpoint.id}
        endpoint={endpoint}
        targetBaseUrl={targetBaseUrl}
        setTargetBaseUrl={setTargetBaseUrl}
      />
    </div>
  );
}

function FunctionDetail({ fn }: { fn: ApiFunction }) {
  return (
    <div className="space-y-5">
      <Panel
        title={<span className="font-mono">{fn.name}</span>}
        badge={
          <>
            <span className="rounded-full bg-sand/80 px-3 py-1 text-[11px] font-semibold uppercase tracking-wider text-ink/45">
              function
            </span>
            {fn.deprecated && <DeprecatedBadge />}
          </>
        }
      >
        <p className="mt-1 max-w-2xl text-base leading-7 text-ink/70">
          {fn.description}
        </p>
        <SignatureBlock signature={fn.signature} />
        <div className="mt-5 grid gap-3 sm:grid-cols-2">
          <InfoCard label="Return Type" value={fn.returnType} mono />
          <InfoCard label="Source" value={shortFile(fn.file, fn.line)} mono />
        </div>
      </Panel>

      <Panel
        title="Parameters"
        badge={
          <span className="rounded-full bg-sand/80 px-3 py-1 text-[11px] font-semibold text-ink/45">
            {fn.params.length}
          </span>
        }
      >
        <ParamsTable params={fn.params} />
      </Panel>
    </div>
  );
}

function StructDetail({ struct }: { struct: ApiStruct }) {
  return (
    <div className="space-y-5">
      <Panel
        title={<span className="font-mono">{struct.name}</span>}
        badge={
          <span className="rounded-full bg-sand/80 px-3 py-1 text-[11px] font-semibold uppercase tracking-wider text-ink/45">
            struct
          </span>
        }
      >
        <p className="mt-1 max-w-2xl text-base leading-7 text-ink/70">
          {struct.description}
        </p>
        <div className="mt-5">
          <InfoCard label="Source" value={shortFile(struct.file, struct.line)} mono />
        </div>
      </Panel>
      <Panel
        title="Fields"
        badge={
          <span className="rounded-full bg-sand/80 px-3 py-1 text-[11px] font-semibold text-ink/45">
            {struct.fields.length}
          </span>
        }
      >
        <FieldsTable fields={struct.fields} />
      </Panel>
    </div>
  );
}

function EnumDetail({ enumItem }: { enumItem: ApiEnum }) {
  return (
    <div className="space-y-5">
      <Panel
        title={<span className="font-mono">{enumItem.name}</span>}
        badge={
          <span className="rounded-full bg-sand/80 px-3 py-1 text-[11px] font-semibold uppercase tracking-wider text-ink/45">
            enum
          </span>
        }
      >
        <p className="mt-1 max-w-2xl text-base leading-7 text-ink/70">
          {enumItem.description}
        </p>
        <div className="mt-5">
          <InfoCard label="Source" value={shortFile(enumItem.file, enumItem.line)} mono />
        </div>
      </Panel>
      <Panel
        title="Values"
        badge={
          <span className="rounded-full bg-sand/80 px-3 py-1 text-[11px] font-semibold text-ink/45">
            {enumItem.values.length}
          </span>
        }
      >
        <EnumValuesTable values={enumItem.values} />
      </Panel>
    </div>
  );
}

function InfoCard({
  label,
  value,
  mono,
}: {
  label: string;
  value: string;
  mono?: boolean;
}) {
  return (
    <div className="rounded-[22px] border border-ink/8 bg-sand/30 px-4 py-3">
      <p className="text-[10px] font-semibold uppercase tracking-widest text-ink/40">
        {label}
      </p>
      <p className={`mt-1.5 text-sm text-ink ${mono ? "font-mono" : ""}`}>
        {value}
      </p>
    </div>
  );
}

// ─── Selection types ──────────────────────────────────────────────────────────

type Tab = "endpoints" | "functions" | "types";
type SelectionKind = "endpoint" | "function" | "struct" | "enum";
type Selection = { kind: SelectionKind; id: string } | null;
type TypeItem =
  | { itemKind: "struct"; item: ApiStruct }
  | { itemKind: "enum"; item: ApiEnum };

// ─── App ──────────────────────────────────────────────────────────────────────

function App() {
  const [data, setData] = useState<ApiSpec | null>(null);
  const [tab, setTab] = useState<Tab>("endpoints");
  const [selection, setSelection] = useState<Selection>(null);
  const [query, setQuery] = useState("");
  const deferredQuery = useDeferredValue(query);
  const [showWarnings, setShowWarnings] = useState(true);
  const [mobileShowDetail, setMobileShowDetail] = useState(false);
  const [targetBaseUrl, setTargetBaseUrl] = useState("");

  // Track active list for keyboard nav
  const activeListRef = useRef<{ id: string; kind: SelectionKind }[]>([]);

  useEffect(() => {
    fetch("/api.json")
      .then((r) => {
        if (!r.ok) throw new Error("failed");
        return r.json() as Promise<ApiSpec>;
      })
      .then((payload) => {
        startTransition(() => {
          setData(payload);
          if (payload.endpoints[0]) {
            setSelection({ kind: "endpoint", id: payload.endpoints[0].id });
          }
        });
      })
      .catch(() => setData(null));
  }, []);

  useEffect(() => {
    fetch("/api/runtime")
      .then((r) => r.json() as Promise<RuntimeConfig>)
      .then((p) => {
        if (p.defaultTargetBaseUrl) setTargetBaseUrl(p.defaultTargetBaseUrl);
      })
      .catch(() => undefined);
  }, []);

  const q = deferredQuery.toLowerCase();

  const filteredEndpoints =
    data?.endpoints.filter((e) =>
      `${e.method} ${e.path} ${e.name} ${e.description}`.toLowerCase().includes(q),
    ) ?? [];

  const filteredFunctions =
    data?.functions.filter((f) =>
      `${f.name} ${f.description}`.toLowerCase().includes(q),
    ) ?? [];

  const filteredTypes: TypeItem[] = [
    ...(data?.structs ?? []).map(
      (s): TypeItem => ({ itemKind: "struct", item: s }),
    ),
    ...(data?.enums ?? []).map((e): TypeItem => ({ itemKind: "enum", item: e })),
  ].filter((t) =>
    `${t.item.name} ${t.item.description}`.toLowerCase().includes(q),
  );

  // Update keyboard nav list
  if (tab === "endpoints") {
    activeListRef.current = filteredEndpoints.map((e) => ({
      id: e.id,
      kind: "endpoint",
    }));
  } else if (tab === "functions") {
    activeListRef.current = filteredFunctions.map((f) => ({
      id: f.id,
      kind: "function",
    }));
  } else {
    activeListRef.current = filteredTypes.map((t) => ({
      id: t.item.id,
      kind: t.itemKind,
    }));
  }

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key !== "ArrowUp" && e.key !== "ArrowDown") return;
      const target = e.target as HTMLElement;
      if (
        target.tagName === "INPUT" ||
        target.tagName === "TEXTAREA" ||
        target.tagName === "SELECT"
      )
        return;
      e.preventDefault();
      const list = activeListRef.current;
      if (!list.length) return;
      const idx = list.findIndex((item) => item.id === selection?.id);
      const next =
        e.key === "ArrowDown"
          ? Math.min(idx + 1, list.length - 1)
          : Math.max(idx - 1, 0);
      const item = list[next];
      if (item) {
        setSelection({ kind: item.kind, id: item.id });
        setMobileShowDetail(true);
      }
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [selection]);

  function handleTabChange(t: Tab) {
    setTab(t);
    setQuery("");
    if (!data) return;
    if (t === "endpoints" && data.endpoints.length > 0) {
      setSelection({ kind: "endpoint", id: data.endpoints[0].id });
    } else if (t === "functions" && data.functions.length > 0) {
      setSelection({ kind: "function", id: data.functions[0].id });
    } else if (t === "types") {
      if (data.structs.length > 0) {
        setSelection({ kind: "struct", id: data.structs[0].id });
      } else if (data.enums.length > 0) {
        setSelection({ kind: "enum", id: data.enums[0].id });
      }
    }
    setMobileShowDetail(false);
  }

  function select(kind: SelectionKind, id: string) {
    setSelection({ kind, id });
    setMobileShowDetail(true);
  }

  // Derive active item
  const activeEndpoint =
    selection?.kind === "endpoint"
      ? data?.endpoints.find((e) => e.id === selection.id)
      : undefined;
  const activeFunction =
    selection?.kind === "function"
      ? data?.functions.find((f) => f.id === selection.id)
      : undefined;
  const activeStruct =
    selection?.kind === "struct"
      ? data?.structs.find((s) => s.id === selection.id)
      : undefined;
  const activeEnum =
    selection?.kind === "enum"
      ? data?.enums.find((e) => e.id === selection.id)
      : undefined;

  const hasWarnings =
    showWarnings && data?.warnings && data.warnings.length > 0;

  return (
    <div className="min-h-screen bg-canvas bg-grain text-ink">
      {/* Warnings banner */}
      {hasWarnings && (
        <div className="border-b border-amber-500/20 bg-amber-50/60 px-5 py-3">
          <div className="mx-auto flex max-w-[1600px] items-start gap-3">
            <span className="mt-0.5 text-amber-500">⚠</span>
            <div className="flex-1">
              <p className="text-sm font-semibold text-amber-700">
                {data!.warnings!.length} parse warning
                {data!.warnings!.length !== 1 ? "s" : ""}
              </p>
              <ul className="mt-1 space-y-0.5">
                {data!.warnings!.map((w, i) => (
                  <li key={i} className="font-mono text-xs text-amber-600/80">
                    {w}
                  </li>
                ))}
              </ul>
            </div>
            <button
              onClick={() => setShowWarnings(false)}
              className="text-lg leading-none text-amber-500/50 transition hover:text-amber-600"
            >
              ×
            </button>
          </div>
        </div>
      )}

      <div className="mx-auto grid min-h-screen max-w-[1600px] grid-cols-1 lg:grid-cols-[360px_minmax(0,1fr)]">
        {/* ── Sidebar ─────────────────────────────────────────────────────── */}
        <aside
          className={`${mobileShowDetail ? "hidden" : "flex"} flex-col border-b border-ink/10 bg-white/55 px-5 py-8 backdrop-blur-xl lg:flex lg:border-b-0 lg:border-r`}
        >
          <div className="animate-fade-up">
            <p className="text-xs font-semibold uppercase tracking-[0.24em] text-sea">
              C-first API docs
            </p>
            <h1 className="mt-3 font-display text-5xl leading-none tracking-tight">
              {data?.name ?? "CortexDocs"}
            </h1>
            {data && (
              <p className="mt-2 font-mono text-xs text-ink/30">
                {data.sourcePath} ·{" "}
                {new Date(data.generatedAt).toLocaleDateString()}
              </p>
            )}
          </div>

          {/* Stats */}
          <div className="mt-6 rounded-[28px] border border-ink/10 bg-white/70 p-4 shadow-float">
            <div className="grid grid-cols-4 gap-2 text-center text-sm">
              <Metric label="Endpoints" value={data?.summary.endpointCount ?? 0} />
              <Metric label="Functions" value={data?.summary.functionCount ?? 0} />
              <Metric label="Structs" value={data?.summary.structCount ?? 0} />
              <Metric label="Enums" value={data?.summary.enumCount ?? 0} />
            </div>
          </div>

          {/* Tabs */}
          <div className="mt-6 flex gap-1 rounded-2xl bg-sand/60 p-1">
            {(["endpoints", "functions", "types"] as Tab[]).map((t) => (
              <button
                key={t}
                onClick={() => handleTabChange(t)}
                className={`flex-1 rounded-xl py-2 text-xs font-semibold capitalize tracking-wide transition ${
                  tab === t
                    ? "bg-white shadow-sm text-ink"
                    : "text-ink/40 hover:text-ink"
                }`}
              >
                {t}
              </button>
            ))}
          </div>

          {/* Search */}
          <div className="mt-4">
            <input
              className="w-full rounded-2xl border border-ink/10 bg-white/80 px-4 py-3 text-sm outline-none transition focus:border-sea"
              placeholder={
                tab === "endpoints"
                  ? "Search GET /users…"
                  : tab === "functions"
                    ? "Search function name…"
                    : "Search type name…"
              }
              value={query}
              onChange={(e) => setQuery(e.target.value)}
            />
          </div>

          {/* List */}
          <div className="mt-4 flex-1 space-y-2.5 overflow-y-auto pb-4">
            {tab === "endpoints" &&
              filteredEndpoints.map((ep, i) => {
                const active = selection?.id === ep.id && selection.kind === "endpoint";
                return (
                  <button
                    key={ep.id}
                    onClick={() => select("endpoint", ep.id)}
                    className={`w-full animate-fade-up rounded-[24px] border px-4 py-4 text-left transition ${
                      active
                        ? `border-transparent bg-white shadow-float ${methodBorder(ep.method)}`
                        : "border-ink/10 bg-white/60 hover:border-ink/20 hover:bg-white"
                    }`}
                    style={{ animationDelay: `${i * 60}ms` }}
                  >
                    <div className="flex flex-wrap items-center gap-2">
                      <MethodBadge method={ep.method} />
                      {ep.deprecated && <DeprecatedBadge />}
                    </div>
                    <p className="mt-2.5 font-mono text-sm font-semibold text-ink">
                      {ep.path}
                    </p>
                    <p className="mt-1 line-clamp-2 text-xs leading-5 text-ink/50">
                      {ep.description}
                    </p>
                  </button>
                );
              })}

            {tab === "functions" &&
              filteredFunctions.map((fn, i) => {
                const active = selection?.id === fn.id && selection.kind === "function";
                return (
                  <button
                    key={fn.id}
                    onClick={() => select("function", fn.id)}
                    className={`w-full animate-fade-up rounded-[24px] border px-4 py-4 text-left transition ${
                      active
                        ? "border-sea/30 bg-white shadow-float"
                        : "border-ink/10 bg-white/60 hover:border-ink/20 hover:bg-white"
                    }`}
                    style={{ animationDelay: `${i * 60}ms` }}
                  >
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="font-mono text-sm font-semibold text-ink">
                        {fn.name}
                      </span>
                      {fn.deprecated && <DeprecatedBadge />}
                    </div>
                    <p className="mt-1 font-mono text-xs text-sea/70">
                      {fn.returnType}
                    </p>
                    <p className="mt-1 line-clamp-2 text-xs leading-5 text-ink/50">
                      {fn.description}
                    </p>
                  </button>
                );
              })}

            {tab === "types" &&
              filteredTypes.map((t, i) => {
                const active =
                  selection?.id === t.item.id && selection.kind === t.itemKind;
                return (
                  <button
                    key={t.item.id}
                    onClick={() => select(t.itemKind, t.item.id)}
                    className={`w-full animate-fade-up rounded-[24px] border px-4 py-4 text-left transition ${
                      active
                        ? "border-sea/30 bg-white shadow-float"
                        : "border-ink/10 bg-white/60 hover:border-ink/20 hover:bg-white"
                    }`}
                    style={{ animationDelay: `${i * 60}ms` }}
                  >
                    <div className="flex items-center gap-2">
                      <span className="rounded-full bg-sand/80 px-2.5 py-0.5 text-[10px] font-bold uppercase tracking-wider text-ink/50">
                        {t.itemKind}
                      </span>
                      <span className="font-mono text-sm font-semibold text-ink">
                        {t.item.name}
                      </span>
                    </div>
                    <p className="mt-1.5 line-clamp-2 text-xs leading-5 text-ink/50">
                      {t.item.description}
                    </p>
                  </button>
                );
              })}

            {/* Empty states */}
            {tab === "endpoints" && filteredEndpoints.length === 0 && (
              <p className="py-4 text-center text-sm text-ink/35">
                {data ? "No endpoints match." : "Run generate first."}
              </p>
            )}
            {tab === "functions" && filteredFunctions.length === 0 && (
              <p className="py-4 text-center text-sm text-ink/35">
                {data ? "No functions match." : "Run generate first."}
              </p>
            )}
            {tab === "types" && filteredTypes.length === 0 && (
              <p className="py-4 text-center text-sm text-ink/35">
                {data ? "No types match." : "Run generate first."}
              </p>
            )}
          </div>
        </aside>

        {/* ── Main ────────────────────────────────────────────────────────── */}
        <main
          className={`${mobileShowDetail ? "block" : "hidden"} px-5 py-8 lg:block lg:px-8`}
        >
          {/* Mobile back */}
          <button
            className="mb-5 flex items-center gap-1.5 text-sm font-medium text-ink/50 transition hover:text-ink lg:hidden"
            onClick={() => setMobileShowDetail(false)}
          >
            ← Back
          </button>

          {!data && (
            <div className="flex h-[60vh] items-center justify-center">
              <div className="rounded-[34px] border border-ink/10 bg-white/72 p-10 text-center shadow-float backdrop-blur-xl">
                <p className="font-display text-3xl">No API model yet</p>
                <p className="mt-4 text-sm text-ink/60">
                  Run{" "}
                  <code className="rounded-lg bg-sand px-2 py-0.5 font-mono text-xs">
                    go run ./cli generate ./examples/sample-c-api
                  </code>{" "}
                  then refresh.
                </p>
              </div>
            </div>
          )}

          {data && activeEndpoint && (
            <EndpointDetail
              key={activeEndpoint.id}
              endpoint={activeEndpoint}
              targetBaseUrl={targetBaseUrl}
              setTargetBaseUrl={setTargetBaseUrl}
            />
          )}
          {data && activeFunction && (
            <FunctionDetail key={activeFunction.id} fn={activeFunction} />
          )}
          {data && activeStruct && (
            <StructDetail key={activeStruct.id} struct={activeStruct} />
          )}
          {data && activeEnum && (
            <EnumDetail key={activeEnum.id} enumItem={activeEnum} />
          )}

          {data && !activeEndpoint && !activeFunction && !activeStruct && !activeEnum && (
            <div className="flex h-[60vh] items-center justify-center">
              <p className="text-sm text-ink/35">Select an item from the sidebar.</p>
            </div>
          )}
        </main>
      </div>
    </div>
  );
}

function Metric({ label, value }: { label: string; value: number }) {
  return (
    <div>
      <div className="font-display text-2xl leading-none">{value}</div>
      <div className="mt-1 text-[10px] uppercase tracking-wider text-ink/40">{label}</div>
    </div>
  );
}

export default App;
