export type CatalogEntry = {
  identifier: string;
  displayName: string;
  type: string;
  url?: string;
  data?: Record<string, unknown>;
  description?: string;
  tags?: string[];
  capabilities?: string[];
  representativeQueries?: string[];
  version?: string;
  updatedAt?: string;
  metadata?: Record<string, unknown>;
  trustManifest?: Record<string, unknown>;
};

export type Catalog = {
  specVersion: string;
  host?: {
    displayName: string;
    identifier?: string;
    documentationUrl?: string;
    logoUrl?: string;
    trustManifest?: Record<string, unknown>;
  };
  entries: CatalogEntry[];
};

export type ListResponse = {
  items: CatalogEntry[];
  total?: number;
  pageToken?: string;
};

export type SearchResult = CatalogEntry & {
  score: number;
  source: string;
};

export type SearchResponse = {
  results: SearchResult[];
  referrals?: CatalogEntry[];
  pageToken?: string;
};

export type ExploreResponse = {
  resultType: string;
  facets: Record<string, { buckets: { value: string; count: number }[]; otherCount?: number }>;
};

export type HealthResponse = {
  status: string;
  entries: number;
  version?: string;
  commit?: string;
  buildDate?: string;
};

export type AdminAuditEvent = {
  id: string;
  action: string;
  identifier?: string;
  status?: string;
  reason?: string;
  requestId?: string;
  source: string;
  remoteAddr?: string;
  previousHash?: string;
  hash?: string;
  createdAt: string;
};

export type AdminAuditResponse = {
  items: AdminAuditEvent[];
  total: number;
  pageToken?: string;
};

export type AdminAuditVerification = {
  valid: boolean;
  total: number;
  lastHash?: string;
  firstInvalidEventId?: string;
  message?: string;
};

export type AdminStatusResponse = {
  identifier: string;
  status: string;
  reason?: string;
  approvals?: number;
  requiredApprovals?: number;
};

export type ConsoleConfig = {
  apiBase: string;
  adminToken: string;
};
