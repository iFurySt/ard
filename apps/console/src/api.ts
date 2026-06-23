import type {
  AdminAuditResponse,
  AdminAuditVerification,
  AdminStatusResponse,
  Catalog,
  CatalogEntry,
  ConsoleConfig,
  ExploreResponse,
  HealthResponse,
  ListResponse,
  SearchResponse
} from "./types";

const apiBaseFromEnv = import.meta.env.VITE_ARD_API_BASE ?? "";

export class APIError extends Error {
  status: number;
  body: string;

  constructor(status: number, statusText: string, body: string) {
    super(body ? `${status} ${statusText}: ${body}` : `${status} ${statusText}`);
    this.status = status;
    this.body = body;
  }
}

function endpoint(path: string, config: ConsoleConfig) {
  const configuredBase = config.apiBase.trim() || apiBaseFromEnv;
  if (!configuredBase) return path;
  return `${configuredBase.replace(/\/$/, "")}${path}`;
}

async function request<T>(path: string, config: ConsoleConfig, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers);
  headers.set("Accept", path === "/metrics" ? "text/plain" : "application/json");
  if (init.body && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }
  if (config.adminToken.trim()) {
    headers.set("Authorization", `Bearer ${config.adminToken.trim()}`);
  }

  const response = await fetch(endpoint(path, config), { ...init, headers });
  const text = await response.text();
  if (!response.ok) {
    throw new APIError(response.status, response.statusText, text);
  }
  if (path === "/metrics") {
    return text as T;
  }
  if (!text) {
    return undefined as T;
  }
  return JSON.parse(text) as T;
}

function queryString(params: Record<string, string | number | undefined>) {
  const query = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value !== undefined && `${value}`.trim() !== "") {
      query.set(key, `${value}`);
    }
  }
  const encoded = query.toString();
  return encoded ? `?${encoded}` : "";
}

export function getHealth(config: ConsoleConfig) {
  return request<HealthResponse>("/health", config);
}

export function getMetrics(config: ConsoleConfig) {
  return request<string>("/metrics", config);
}

export function browseEntries(config: ConsoleConfig, params: { pageSize?: number; pageToken?: string; filter?: string; orderBy?: string } = {}) {
  return request<ListResponse>(`/agents${queryString(params)}`, config);
}

export function searchEntries(config: ConsoleConfig, params: { text: string; federation: string; pageSize?: number }) {
  return request<SearchResponse>("/search", config, {
    method: "POST",
    body: JSON.stringify({
      query: { text: params.text },
      federation: params.federation,
      pageSize: params.pageSize ?? 20
    })
  });
}

export function exploreCatalog(config: ConsoleConfig) {
  return request<ExploreResponse>("/explore", config, {
    method: "POST",
    body: JSON.stringify({
      resultType: {
        facets: [
          { field: "type", limit: 12 },
          { field: "publisherId", limit: 12 },
          { field: "tags", limit: 12 }
        ]
      }
    })
  });
}

export function adminListEntries(config: ConsoleConfig, params: { status?: string; kind?: string; type?: string; pageSize?: number; pageToken?: string } = {}) {
  return request<ListResponse>(`/admin/entries${queryString(params)}`, config);
}

export function adminReviews(config: ConsoleConfig, params: { pageSize?: number; pageToken?: string } = {}) {
  return request<ListResponse>(`/admin/reviews${queryString(params)}`, config);
}

export function adminAudit(config: ConsoleConfig, params: { pageSize?: number; pageToken?: string } = {}) {
  return request<AdminAuditResponse>(`/admin/audit${queryString(params)}`, config);
}

export function adminVerifyAudit(config: ConsoleConfig) {
  return request<AdminAuditVerification>("/admin/audit/verify", config);
}

export function adminSetStatus(config: ConsoleConfig, identifier: string, status: string) {
  return request<AdminStatusResponse>(`/admin/entries/${encodeURIComponent(identifier)}/status`, config, {
    method: "PATCH",
    body: JSON.stringify({ status })
  });
}

export function adminApproveReview(config: ConsoleConfig, identifier: string, reason: string) {
  return request<AdminStatusResponse>(`/admin/reviews/${encodeURIComponent(identifier)}/approve`, config, {
    method: "POST",
    body: JSON.stringify({ reason })
  });
}

export function adminRejectReview(config: ConsoleConfig, identifier: string, reason: string) {
  return request<AdminStatusResponse>(`/admin/reviews/${encodeURIComponent(identifier)}/reject`, config, {
    method: "POST",
    body: JSON.stringify({ reason })
  });
}

export function adminDeleteEntry(config: ConsoleConfig, identifier: string) {
  return request<void>(`/admin/entries/${encodeURIComponent(identifier)}`, config, {
    method: "DELETE"
  });
}

export function adminImportCatalog(config: ConsoleConfig, catalog: Catalog) {
  return request<{ entries: number }>("/admin/catalogs", config, {
    method: "POST",
    body: JSON.stringify(catalog)
  });
}

export function adminImportEntry(config: ConsoleConfig, entry: CatalogEntry) {
  return request<CatalogEntry>("/admin/entries", config, {
    method: "POST",
    body: JSON.stringify(entry)
  });
}

export function adminExportCatalog(config: ConsoleConfig) {
  return request<Catalog>("/admin/catalog", config);
}
