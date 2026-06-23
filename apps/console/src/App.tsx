import {
  Activity,
  Archive,
  Boxes,
  Check,
  ClipboardCheck,
  Download,
  FileJson,
  Gauge,
  Home,
  KeyRound,
  ListFilter,
  Plus,
  RefreshCcw,
  Search,
  Settings,
  ShieldCheck,
  Trash2,
  X
} from "lucide-react";
import { useEffect, useMemo, useState, type FormEvent, type ReactNode } from "react";
import { Link, Navigate, Route, Routes, useLocation } from "react-router-dom";
import {
  adminApproveReview,
  adminAudit,
  adminDeleteEntry,
  adminExportCatalog,
  adminImportCatalog,
  adminImportEntry,
  adminListEntries,
  adminRejectReview,
  adminReviews,
  adminSetStatus,
  adminVerifyAudit,
  browseEntries,
  exploreCatalog,
  getHealth,
  getMetrics,
  searchEntries
} from "./api";
import { Badge, Button, DataTable, FieldSelect, SidebarItem, TextInput } from "./components/cds";
import type { AdminAuditEvent, AdminAuditVerification, Catalog, CatalogEntry, ConsoleConfig, ExploreResponse, HealthResponse, ListResponse, SearchResult } from "./types";

const configStorageKey = "openard.console.config";
const pageTitles: Record<string, string> = {
  overview: "Overview",
  discover: "Discover",
  catalog: "Catalog",
  add: "Add resource",
  reviews: "Reviews",
  audit: "Audit log",
  operations: "Operations",
  settings: "Settings"
};

const resourceKinds = ["All", "mcp", "a2a", "skill", "openapi", "catalog", "registry"];
const lifecycleStatuses = ["All", "active", "pending", "disabled"];
const federationModes = ["auto", "none", "referrals"];

function readConfig(): ConsoleConfig {
  try {
    const stored = window.localStorage.getItem(configStorageKey);
    if (stored) {
      const parsed = JSON.parse(stored) as Partial<ConsoleConfig>;
      return {
        apiBase: parsed.apiBase ?? "",
        adminToken: parsed.adminToken ?? ""
      };
    }
  } catch {
    // Ignore unreadable local storage; the console remains usable for this session.
  }
  return { apiBase: "", adminToken: "" };
}

function saveConfig(config: ConsoleConfig) {
  try {
    window.localStorage.setItem(configStorageKey, JSON.stringify(config));
  } catch {
    // Ignore unavailable storage.
  }
}

function titleForPath(pathname: string) {
  const [segment] = pathname.split("/").filter(Boolean);
  const title = pageTitles[segment || "overview"] ?? "OpenARD Console";
  return title === "OpenARD Console" ? title : `${title} | OpenARD Console`;
}

export default function App() {
  const location = useLocation();
  const [config, setConfigState] = useState(readConfig);

  function setConfig(next: ConsoleConfig) {
    setConfigState(next);
    saveConfig(next);
  }

  useEffect(() => {
    document.title = titleForPath(location.pathname);
  }, [location.pathname]);

  return (
    <div className="min-h-screen bg-canvas text-ink">
      <div className="flex min-h-screen">
        <Sidebar />
        <main className="min-w-0 flex-1">
          <div className="mx-auto w-full max-w-7xl overflow-hidden px-6 pb-8 pt-6 md:px-8">
            <TopBar config={config} />
            <Routes>
              <Route path="/" element={<Navigate to="/overview" replace />} />
              <Route path="/overview" element={<OverviewPage config={config} />} />
              <Route path="/discover" element={<DiscoverPage config={config} />} />
              <Route path="/catalog" element={<CatalogPage config={config} />} />
              <Route path="/add" element={<AddResourcePage config={config} />} />
              <Route path="/reviews" element={<ReviewsPage config={config} />} />
              <Route path="/audit" element={<AuditPage config={config} />} />
              <Route path="/operations" element={<OperationsPage config={config} />} />
              <Route path="/settings" element={<SettingsPage config={config} onSave={setConfig} />} />
            </Routes>
          </div>
        </main>
      </div>
    </div>
  );
}

function Sidebar() {
  return (
    <aside aria-label="Main navigation" className="sticky top-0 flex h-screen w-[240px] shrink-0 flex-col border-r-[0.5px] border-line bg-[#f9f9f7] px-3 py-3 shadow-[inset_-4px_0px_6px_-4px_rgba(0,0,0,0.04)]">
      <Link to="/overview" className="mb-4 flex h-10 items-center gap-2 rounded-lg px-2 text-ink">
        <div className="flex h-7 w-7 items-center justify-center rounded-[7px] bg-ink text-white">
          <Boxes className="h-4 w-4" />
        </div>
        <div className="min-w-0">
          <div className="truncate text-sm [font-weight:650]">OpenARD Console</div>
          <div className="truncate text-xs text-muted">Registry administration</div>
        </div>
      </Link>
      <nav className="flex min-h-0 flex-1 flex-col gap-1 overflow-y-auto">
        <SidebarRow to="/overview" icon={<Home className="h-4 w-4" />}>Overview</SidebarRow>
        <SidebarRow to="/discover" icon={<Search className="h-4 w-4" />}>Discover</SidebarRow>
        <SidebarRow to="/catalog" icon={<ListFilter className="h-4 w-4" />}>Catalog</SidebarRow>
        <SidebarRow to="/add" icon={<Plus className="h-4 w-4" />}>Add resource</SidebarRow>
        <SidebarRow to="/reviews" icon={<ClipboardCheck className="h-4 w-4" />}>Reviews</SidebarRow>
        <SidebarRow to="/audit" icon={<ShieldCheck className="h-4 w-4" />}>Audit log</SidebarRow>
        <SidebarRow to="/operations" icon={<Gauge className="h-4 w-4" />}>Operations</SidebarRow>
      </nav>
      <div className="border-t border-line pt-3">
        <SidebarRow to="/settings" icon={<Settings className="h-4 w-4" />}>Settings</SidebarRow>
      </div>
    </aside>
  );
}

function SidebarRow({ to, icon, children }: { to: string; icon: ReactNode; children: ReactNode }) {
  return (
    <div className="[&_a]:gap-2.5 [&_a]:px-2">
      <SidebarItem to={to}>
        <span className="inline-flex items-center gap-2.5">
          <span className="text-[#52514e]">{icon}</span>
          <span>{children}</span>
        </span>
      </SidebarItem>
    </div>
  );
}

function TopBar({ config }: { config: ConsoleConfig }) {
  return (
    <div className="mb-6 flex h-9 items-center justify-between gap-3">
      <div className="min-w-0">
        <div className="text-xs uppercase tracking-[0.08em] text-muted">Administrator workspace</div>
        <div className="truncate text-sm text-[#52514e]">{config.apiBase.trim() || "Using same-origin registry API"}</div>
      </div>
      <div className="flex items-center gap-2">
        <Badge tone={config.adminToken.trim() ? "green" : "warning"}>{config.adminToken.trim() ? "Admin token set" : "Read-only until token is set"}</Badge>
        <Link to="/settings">
          <Button variant="secondary" size="sm">
            <KeyRound className="h-3.5 w-3.5" />
            Connection
          </Button>
        </Link>
      </div>
    </div>
  );
}

function PageHeader({ title, description, actions }: { title: string; description: string; actions?: ReactNode }) {
  return (
    <div className="mb-5 flex items-start justify-between gap-4">
      <div className="min-w-0">
        <h1 className="text-[28px] leading-8 [font-weight:650]">{title}</h1>
        <p className="mt-1 max-w-3xl text-sm text-muted">{description}</p>
      </div>
      {actions ? <div className="flex shrink-0 items-center gap-2">{actions}</div> : null}
    </div>
  );
}

function OverviewPage({ config }: { config: ConsoleConfig }) {
  const [health, setHealth] = useState<HealthResponse | null>(null);
  const [entries, setEntries] = useState<ListResponse | null>(null);
  const [reviews, setReviews] = useState<ListResponse | null>(null);
  const [audit, setAudit] = useState<AdminAuditEvent[]>([]);
  const [auditVerification, setAuditVerification] = useState<AdminAuditVerification | null>(null);
  const [facets, setFacets] = useState<ExploreResponse | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function load() {
    setLoading(true);
    setError("");
    const [healthResult, exploreResult] = await Promise.allSettled([
      getHealth(config),
      exploreCatalog(config)
    ]);
    if (healthResult.status === "fulfilled") setHealth(healthResult.value);
    if (exploreResult.status === "fulfilled") setFacets(exploreResult.value);

    const rejected = [healthResult, exploreResult].find((result) => result.status === "rejected");
    if (rejected?.status === "rejected") setError(messageFromError(rejected.reason));

    if (config.adminToken.trim()) {
      const [entriesResult, reviewsResult, auditResult, verifyResult] = await Promise.allSettled([
        adminListEntries(config, { pageSize: 100 }),
        adminReviews(config, { pageSize: 20 }),
        adminAudit(config, { pageSize: 8 }),
        adminVerifyAudit(config)
      ]);
      if (entriesResult.status === "fulfilled") setEntries(entriesResult.value);
      if (reviewsResult.status === "fulfilled") setReviews(reviewsResult.value);
      if (auditResult.status === "fulfilled") setAudit(auditResult.value.items);
      if (verifyResult.status === "fulfilled") setAuditVerification(verifyResult.value);
      const adminRejected = [entriesResult, reviewsResult, auditResult, verifyResult].find((result) => result.status === "rejected");
      if (adminRejected?.status === "rejected") setError(messageFromError(adminRejected.reason));
    } else {
      setEntries(null);
      setReviews(null);
      setAudit([]);
      setAuditVerification(null);
    }
    setLoading(false);
  }

  useEffect(() => {
    void load();
  }, [config.apiBase, config.adminToken]);

  const statusCounts = useMemo(() => countStatuses(entries?.items ?? []), [entries]);

  return (
    <>
      <PageHeader
        title="Overview"
        description="Operate the self-hosted registry: inventory state, pending governance work, health, and recent administrator changes."
        actions={<RefreshButton loading={loading} onClick={load} />}
      />
      <ErrorBanner message={error} />
      {!config.adminToken.trim() ? <div className="mb-4 rounded-cds border border-[#e9c77a] bg-[#fff8e8] px-3 py-2 text-sm text-[#734500]">Set an admin bearer token in Settings to load protected inventory, review, and audit data.</div> : null}
      <div className="grid grid-cols-4 gap-3">
        <Stat label="Health" value={health?.status ?? "unknown"} detail={`${health?.entries ?? 0} active entries`} tone={health?.status === "ok" ? "green" : "warning"} />
        <Stat label="Total entries" value={`${entries?.total ?? entries?.items.length ?? 0}`} detail={`${statusCounts.active} active / ${statusCounts.pending} pending / ${statusCounts.disabled} disabled`} />
        <Stat label="Pending reviews" value={`${reviews?.total ?? reviews?.items.length ?? 0}`} detail="Entries hidden from public discovery" tone={(reviews?.total ?? 0) > 0 ? "warning" : "green"} />
        <Stat label="Audit chain" value={auditVerification?.valid ? "valid" : "unknown"} detail={`${auditVerification?.total ?? 0} events verified`} tone={auditVerification?.valid ? "green" : "warning"} />
      </div>
      <div className="mt-5 grid grid-cols-[minmax(0,1.25fr)_minmax(320px,0.75fr)] gap-5">
        <section className="rounded-cds border border-line bg-white p-4">
          <SectionTitle title="Catalog distribution" detail="Public facets from active entries" />
          <div className="mt-4 grid grid-cols-3 gap-4">
            <FacetList title="Types" buckets={facets?.facets.type?.buckets ?? []} />
            <FacetList title="Publishers" buckets={facets?.facets.publisherId?.buckets ?? []} />
            <FacetList title="Tags" buckets={facets?.facets.tags?.buckets ?? []} />
          </div>
        </section>
        <section className="rounded-cds border border-line bg-white p-4">
          <SectionTitle title="Recent audit" detail="Latest administrator mutations" />
          <div className="mt-3 space-y-3">
            {audit.length === 0 ? <EmptyText>No audit events available.</EmptyText> : audit.map((event) => <AuditEventRow key={event.id} event={event} />)}
          </div>
        </section>
      </div>
    </>
  );
}

function DiscoverPage({ config }: { config: ConsoleConfig }) {
  const [query, setQuery] = useState("weather");
  const [federation, setFederation] = useState("auto");
  const [results, setResults] = useState<SearchResult[]>([]);
  const [referrals, setReferrals] = useState<CatalogEntry[]>([]);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function runSearch(event?: FormEvent) {
    event?.preventDefault();
    if (!query.trim()) return;
    setLoading(true);
    setError("");
    try {
      const response = await searchEntries(config, { text: query.trim(), federation, pageSize: 25 });
      setResults(response.results);
      setReferrals(response.referrals ?? []);
    } catch (err) {
      setError(messageFromError(err));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void runSearch();
  }, [config.apiBase]);

  return (
    <>
      <PageHeader title="Discover" description="Search the public registry surface exactly as agent clients do, including bounded federation modes." />
      <ErrorBanner message={error} />
      <form onSubmit={runSearch} className="mb-4 flex items-center gap-2">
        <TextInput value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Search by intent, capability, tag, or publisher" className="max-w-xl" />
        <FieldSelect label="Federation" value={federation} options={federationModes} onValueChange={setFederation} />
        <Button type="submit" disabled={loading}>
          <Search className="h-4 w-4" />
          Search
        </Button>
      </form>
      <EntryTable entries={results} loading={loading} showScore />
      {referrals.length > 0 ? (
        <section className="mt-5 rounded-cds border border-line bg-white p-4">
          <SectionTitle title="Registry referrals" detail="Returned for clients that follow registry federation themselves" />
          <div className="mt-3 grid grid-cols-2 gap-3">
            {referrals.map((entry) => <EntrySummary key={entry.identifier} entry={entry} />)}
          </div>
        </section>
      ) : null}
    </>
  );
}

function CatalogPage({ config }: { config: ConsoleConfig }) {
  const [status, setStatus] = useState("All");
  const [kind, setKind] = useState("All");
  const [entries, setEntries] = useState<CatalogEntry[]>([]);
  const [total, setTotal] = useState(0);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function load() {
    setLoading(true);
    setError("");
    try {
      const response = await adminListEntries(config, {
        pageSize: 100,
        status: status === "All" ? undefined : status,
        kind: kind === "All" ? undefined : kind
      });
      setEntries(response.items);
      setTotal(response.total ?? response.items.length);
    } catch (err) {
      setError(messageFromError(err));
    } finally {
      setLoading(false);
    }
  }

  async function setEntryStatus(entry: CatalogEntry, nextStatus: string) {
    setError("");
    try {
      await adminSetStatus(config, entry.identifier, nextStatus);
      await load();
    } catch (err) {
      setError(messageFromError(err));
    }
  }

  async function deleteEntry(entry: CatalogEntry) {
    setError("");
    try {
      await adminDeleteEntry(config, entry.identifier);
      await load();
    } catch (err) {
      setError(messageFromError(err));
    }
  }

  async function exportCatalog() {
    setError("");
    try {
      const catalog = await adminExportCatalog(config);
      downloadJSON("ai-catalog.json", catalog);
    } catch (err) {
      setError(messageFromError(err));
    }
  }

  useEffect(() => {
    void load();
  }, [config.apiBase, config.adminToken, status, kind]);

  return (
    <>
      <PageHeader
        title="Catalog"
        description="Manage the complete registry inventory, including entries hidden from public discovery by lifecycle status."
        actions={
          <>
            <Button variant="secondary" size="sm" onClick={exportCatalog}>
              <Download className="h-3.5 w-3.5" />
              Export
            </Button>
            <RefreshButton loading={loading} onClick={load} />
          </>
        }
      />
      <ErrorBanner message={error} />
      <div className="mb-3 flex items-center justify-between gap-3">
        <div className="flex items-center gap-2">
          <FieldSelect label="Status" value={status} options={lifecycleStatuses} onValueChange={setStatus} />
          <FieldSelect label="Kind" value={kind} options={resourceKinds} onValueChange={setKind} />
        </div>
        <div className="text-sm text-muted">{total} entries</div>
      </div>
      <AdminEntryTable entries={entries} loading={loading} onStatusChange={setEntryStatus} onDelete={deleteEntry} />
    </>
  );
}

function AddResourcePage({ config }: { config: ConsoleConfig }) {
  const [mode, setMode] = useState("Catalog");
  const [body, setBody] = useState(sampleCatalogJSON);
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    setBody(mode === "Catalog" ? sampleCatalogJSON : sampleEntryJSON);
  }, [mode]);

  async function submit(event: FormEvent) {
    event.preventDefault();
    setSubmitting(true);
    setMessage("");
    setError("");
    try {
      const parsed = JSON.parse(body) as Catalog | CatalogEntry;
      if (mode === "Catalog") {
        const response = await adminImportCatalog(config, parsed as Catalog);
        setMessage(`Imported ${response.entries} catalog entries.`);
      } else {
        const entry = await adminImportEntry(config, parsed as CatalogEntry);
        setMessage(`Imported ${entry.identifier}.`);
      }
    } catch (err) {
      setError(messageFromError(err));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <>
      <PageHeader title="Add resource" description="Import ARD-native catalog JSON or a single CatalogEntry through the protected admin API." />
      <ErrorBanner message={error} />
      {message ? <div className="mb-4 rounded-cds border border-[#b8deb5] bg-[#f1faef] px-3 py-2 text-sm text-[#006300]">{message}</div> : null}
      <form onSubmit={submit} className="rounded-cds border border-line bg-white">
        <div className="flex items-center justify-between border-b border-line px-4 py-3">
          <FieldSelect label="Import type" value={mode} options={["Catalog", "Entry"]} onValueChange={setMode} />
          <Button type="submit" disabled={submitting}>
            <Plus className="h-4 w-4" />
            Import
          </Button>
        </div>
        <textarea
          value={body}
          onChange={(event) => setBody(event.target.value)}
          spellCheck={false}
          className="min-h-[520px] w-full resize-y border-0 bg-[#111] p-4 font-mono text-xs leading-5 text-[#f5f5f5] outline-none"
        />
      </form>
    </>
  );
}

function ReviewsPage({ config }: { config: ConsoleConfig }) {
  const [entries, setEntries] = useState<CatalogEntry[]>([]);
  const [reason, setReason] = useState("Reviewed publisher metadata and lifecycle policy.");
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");
  const [loading, setLoading] = useState(false);

  async function load() {
    setLoading(true);
    setError("");
    try {
      const response = await adminReviews(config, { pageSize: 100 });
      setEntries(response.items);
    } catch (err) {
      setError(messageFromError(err));
    } finally {
      setLoading(false);
    }
  }

  async function decide(entry: CatalogEntry, action: "approve" | "reject") {
    setError("");
    setMessage("");
    try {
      const response = action === "approve"
        ? await adminApproveReview(config, entry.identifier, reason)
        : await adminRejectReview(config, entry.identifier, reason);
      setMessage(`${entry.displayName} is ${response.status}${response.approvals ? ` (${response.approvals}/${response.requiredApprovals} approvals)` : ""}.`);
      await load();
    } catch (err) {
      setError(messageFromError(err));
    }
  }

  useEffect(() => {
    void load();
  }, [config.apiBase, config.adminToken]);

  return (
    <>
      <PageHeader title="Reviews" description="Approve or reject pending entries before they become visible on public search, browse, explore, and catalog export surfaces." actions={<RefreshButton loading={loading} onClick={load} />} />
      <ErrorBanner message={error} />
      {message ? <div className="mb-4 rounded-cds border border-[#b8deb5] bg-[#f1faef] px-3 py-2 text-sm text-[#006300]">{message}</div> : null}
      <div className="mb-3 flex items-center gap-2">
        <TextInput value={reason} onChange={(event) => setReason(event.target.value)} placeholder="Review reason recorded on audit events" />
      </div>
      <AdminEntryTable entries={entries} loading={loading} reviewMode onApprove={(entry) => decide(entry, "approve")} onReject={(entry) => decide(entry, "reject")} />
    </>
  );
}

function AuditPage({ config }: { config: ConsoleConfig }) {
  const [events, setEvents] = useState<AdminAuditEvent[]>([]);
  const [verification, setVerification] = useState<AdminAuditVerification | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function load() {
    setLoading(true);
    setError("");
    try {
      const [auditResponse, verificationResponse] = await Promise.all([
        adminAudit(config, { pageSize: 100 }),
        adminVerifyAudit(config)
      ]);
      setEvents(auditResponse.items);
      setVerification(verificationResponse);
    } catch (err) {
      setError(messageFromError(err));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void load();
  }, [config.apiBase, config.adminToken]);

  return (
    <>
      <PageHeader title="Audit log" description="Inspect admin mutations and verify the persisted audit hash chain." actions={<RefreshButton loading={loading} onClick={load} />} />
      <ErrorBanner message={error} />
      <div className="mb-4 grid grid-cols-3 gap-3">
        <Stat label="Chain status" value={verification?.valid ? "valid" : "unknown"} detail={verification?.message || "Hash chain verification result"} tone={verification?.valid ? "green" : "warning"} />
        <Stat label="Events" value={`${verification?.total ?? events.length}`} detail="Persisted admin events" />
        <Stat label="Last hash" value={verification?.lastHash ? shortHash(verification.lastHash) : "none"} detail="Tamper-evident chain tip" />
      </div>
      <AuditTable events={events} loading={loading} />
    </>
  );
}

function OperationsPage({ config }: { config: ConsoleConfig }) {
  const [health, setHealth] = useState<HealthResponse | null>(null);
  const [metrics, setMetrics] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function load() {
    setLoading(true);
    setError("");
    try {
      const [healthResponse, metricsResponse] = await Promise.all([getHealth(config), getMetrics(config)]);
      setHealth(healthResponse);
      setMetrics(metricsResponse);
    } catch (err) {
      setError(messageFromError(err));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void load();
  }, [config.apiBase]);

  const parsed = useMemo(() => parseMetrics(metrics), [metrics]);

  return (
    <>
      <PageHeader title="Operations" description="Public registry health and Prometheus metrics for local operational checks." actions={<RefreshButton loading={loading} onClick={load} />} />
      <ErrorBanner message={error} />
      <div className="grid grid-cols-4 gap-3">
        <Stat label="Status" value={health?.status ?? "unknown"} detail={`${health?.entries ?? 0} active entries`} tone={health?.status === "ok" ? "green" : "warning"} />
        <Stat label="Uptime" value={formatSeconds(parsed.ard_registry_uptime_seconds)} detail="Registry process uptime" />
        <Stat label="In-flight" value={parsed.ard_http_requests_in_flight ?? "0"} detail="Current HTTP requests" />
        <Stat label="Goroutines" value={parsed.ard_runtime_goroutines ?? parsed.go_goroutines ?? "n/a"} detail="Go runtime activity" />
      </div>
      <section className="mt-5 rounded-cds border border-line bg-white">
        <div className="border-b border-line px-4 py-3">
          <SectionTitle title="Raw metrics" detail="Prometheus text exposed by /metrics" />
        </div>
        <pre className="max-h-[520px] overflow-auto bg-[#111] p-4 font-mono text-xs leading-5 text-[#f5f5f5]">{metrics || "No metrics loaded."}</pre>
      </section>
    </>
  );
}

function SettingsPage({ config, onSave }: { config: ConsoleConfig; onSave: (config: ConsoleConfig) => void }) {
  const [apiBase, setApiBase] = useState(config.apiBase);
  const [adminToken, setAdminToken] = useState(config.adminToken);
  const [message, setMessage] = useState("");

  function submit(event: FormEvent) {
    event.preventDefault();
    onSave({ apiBase: apiBase.trim(), adminToken });
    setMessage("Connection settings saved locally in this browser.");
  }

  return (
    <>
      <PageHeader title="Settings" description="Configure the registry endpoint and admin bearer token for this browser session. Tokens are only stored in browser local storage." />
      {message ? <div className="mb-4 rounded-cds border border-[#b8deb5] bg-[#f1faef] px-3 py-2 text-sm text-[#006300]">{message}</div> : null}
      <form onSubmit={submit} className="max-w-2xl rounded-cds border border-line bg-white p-4">
        <label className="block text-sm [font-weight:600]">Registry API base URL</label>
        <p className="mb-2 text-sm text-muted">Leave blank when using the Vite dev proxy or same-origin deployment.</p>
        <TextInput value={apiBase} onChange={(event) => setApiBase(event.target.value)} placeholder="http://127.0.0.1:8080" />
        <label className="mt-5 block text-sm [font-weight:600]">Admin bearer token</label>
        <p className="mb-2 text-sm text-muted">Required for Catalog, Reviews, Audit, and protected management actions.</p>
        <TextInput value={adminToken} onChange={(event) => setAdminToken(event.target.value)} placeholder="ARD_ADMIN_TOKEN" type="password" />
        <div className="mt-5 flex items-center gap-2">
          <Button type="submit">
            <Check className="h-4 w-4" />
            Save settings
          </Button>
          <Button variant="secondary" onClick={() => { setApiBase(""); setAdminToken(""); onSave({ apiBase: "", adminToken: "" }); }}>
            Clear
          </Button>
        </div>
      </form>
    </>
  );
}

function EntryTable({ entries, loading, showScore = false }: { entries: (CatalogEntry | SearchResult)[]; loading?: boolean; showScore?: boolean }) {
  return (
    <div className="rounded-cds border border-line bg-white">
      <DataTable
        rows={entries}
        loading={loading}
        showSelection={false}
        showActions={false}
        getKey={(entry) => entry.identifier}
        columns={[
          { key: "name", header: "Name", width: "28%", render: (entry) => <EntryName entry={entry} /> },
          { key: "type", header: "Type", width: "18%", render: (entry) => <TypeBadge type={entry.type} /> },
          { key: "publisher", header: "Publisher", width: "16%", render: (entry) => <span className="text-[#52514e]">{publisherFromIdentifier(entry.identifier)}</span> },
          { key: "tags", header: "Tags", width: "22%", render: (entry) => <TagList tags={entry.tags ?? []} /> },
          { key: "score", header: showScore ? "Score" : "Updated", width: "10%", align: "right", render: (entry) => <span className="text-[#52514e]">{showScore ? ((entry as SearchResult).score ?? "-") : formatDate(entry.updatedAt)}</span> }
        ]}
      />
    </div>
  );
}

function AdminEntryTable({
  entries,
  loading,
  reviewMode = false,
  onStatusChange,
  onDelete,
  onApprove,
  onReject
}: {
  entries: CatalogEntry[];
  loading?: boolean;
  reviewMode?: boolean;
  onStatusChange?: (entry: CatalogEntry, status: string) => void;
  onDelete?: (entry: CatalogEntry) => void;
  onApprove?: (entry: CatalogEntry) => void;
  onReject?: (entry: CatalogEntry) => void;
}) {
  return (
    <div className="rounded-cds border border-line bg-white">
      <DataTable
        rows={entries}
        loading={loading}
        showSelection={false}
        getKey={(entry) => entry.identifier}
        actionsWidth={reviewMode ? "190px" : "220px"}
        columns={[
          { key: "status", header: "Status", width: "88px", render: (entry) => <StatusBadge status={entryStatus(entry)} /> },
          { key: "name", header: "Name", width: "28%", render: (entry) => <EntryName entry={entry} /> },
          { key: "type", header: "Type", width: "18%", render: (entry) => <TypeBadge type={entry.type} /> },
          { key: "publisher", header: "Publisher", width: "14%", render: (entry) => <span className="text-[#52514e]">{publisherFromIdentifier(entry.identifier)}</span> },
          { key: "updated", header: "Updated", width: "106px", render: (entry) => <span className="text-[#52514e]">{formatDate(entry.updatedAt)}</span> }
        ]}
        renderActions={(entry) => reviewMode ? (
          <div className="flex items-center gap-1">
            <Button variant="secondary" size="sm" onClick={() => onApprove?.(entry)}>
              <Check className="h-3.5 w-3.5" />
              Approve
            </Button>
            <Button variant="ghost" size="sm" onClick={() => onReject?.(entry)}>
              <X className="h-3.5 w-3.5" />
              Reject
            </Button>
          </div>
        ) : (
          <div className="flex items-center gap-1">
            <Button variant="secondary" size="sm" onClick={() => onStatusChange?.(entry, entryStatus(entry) === "active" ? "disabled" : "active")}>
              <Archive className="h-3.5 w-3.5" />
              {entryStatus(entry) === "active" ? "Disable" : "Activate"}
            </Button>
            <Button variant="ghost" size="sm" onClick={() => onStatusChange?.(entry, "pending")}>Pend</Button>
            <Button variant="ghost" size="sm" onClick={() => onDelete?.(entry)} aria-label={`Delete ${entry.displayName}`}>
              <Trash2 className="h-3.5 w-3.5" />
            </Button>
          </div>
        )}
      />
    </div>
  );
}

function AuditTable({ events, loading }: { events: AdminAuditEvent[]; loading?: boolean }) {
  return (
    <div className="rounded-cds border border-line bg-white">
      <DataTable
        rows={events}
        loading={loading}
        showSelection={false}
        showActions={false}
        getKey={(event) => event.id}
        columns={[
          { key: "action", header: "Action", width: "180px", render: (event) => <span className="font-mono text-xs">{event.action}</span> },
          { key: "identifier", header: "Identifier", width: "34%", render: (event) => <span className="font-mono text-xs text-[#52514e]">{event.identifier ?? "-"}</span> },
          { key: "status", header: "Status", width: "90px", render: (event) => event.status ? <StatusBadge status={event.status} /> : <span className="text-muted">-</span> },
          { key: "request", header: "Request", width: "150px", render: (event) => <span className="font-mono text-xs text-[#52514e]">{event.requestId ? shortHash(event.requestId) : "-"}</span> },
          { key: "created", header: "Created", width: "160px", render: (event) => <span className="text-[#52514e]">{formatDateTime(event.createdAt)}</span> }
        ]}
      />
    </div>
  );
}

function EntryName({ entry }: { entry: CatalogEntry }) {
  return (
    <div className="min-w-0">
      <div className="truncate [font-weight:550]">{entry.displayName}</div>
      <div className="truncate font-mono text-xs text-muted">{entry.identifier}</div>
    </div>
  );
}

function EntrySummary({ entry }: { entry: CatalogEntry }) {
  return (
    <div className="rounded-cds border border-line p-3">
      <EntryName entry={entry} />
      <p className="mt-2 cds-line-clamp-2 text-sm text-[#52514e]">{entry.description || "No description."}</p>
    </div>
  );
}

function Stat({ label, value, detail, tone = "neutral" }: { label: string; value: string; detail: string; tone?: "neutral" | "green" | "warning" }) {
  const valueClass = tone === "green" ? "text-[#006300]" : tone === "warning" ? "text-[#734500]" : "text-ink";
  return (
    <section className="rounded-cds border border-line bg-white p-4">
      <div className="text-xs uppercase tracking-[0.06em] text-muted">{label}</div>
      <div className={`mt-2 truncate text-2xl leading-7 [font-weight:650] ${valueClass}`}>{value}</div>
      <div className="mt-1 truncate text-sm text-[#52514e]">{detail}</div>
    </section>
  );
}

function SectionTitle({ title, detail }: { title: string; detail: string }) {
  return (
    <div>
      <h2 className="text-sm [font-weight:650]">{title}</h2>
      <p className="mt-0.5 text-sm text-muted">{detail}</p>
    </div>
  );
}

function FacetList({ title, buckets }: { title: string; buckets: { value: string; count: number }[] }) {
  return (
    <div>
      <div className="mb-2 text-xs uppercase tracking-[0.06em] text-muted">{title}</div>
      <div className="space-y-2">
        {buckets.length === 0 ? <EmptyText>No facet data.</EmptyText> : buckets.slice(0, 6).map((bucket) => (
          <div key={bucket.value} className="flex items-center justify-between gap-2 text-sm">
            <span className="min-w-0 truncate">{bucket.value}</span>
            <Badge>{bucket.count}</Badge>
          </div>
        ))}
      </div>
    </div>
  );
}

function AuditEventRow({ event }: { event: AdminAuditEvent }) {
  return (
    <div className="min-w-0 border-b border-line pb-3 last:border-0 last:pb-0">
      <div className="flex items-center gap-2">
        <span className="truncate text-sm [font-weight:550]">{event.action}</span>
        {event.status ? <StatusBadge status={event.status} /> : null}
      </div>
      <div className="mt-1 truncate font-mono text-xs text-muted">{event.identifier ?? event.id}</div>
    </div>
  );
}

function TypeBadge({ type }: { type: string }) {
  return <Badge tone="blue">{displayType(type)}</Badge>;
}

function StatusBadge({ status }: { status: string }) {
  const tone = status === "active" ? "green" : status === "pending" ? "warning" : status === "disabled" ? "red" : "neutral";
  return <Badge tone={tone}>{status}</Badge>;
}

function TagList({ tags }: { tags: string[] }) {
  if (tags.length === 0) return <span className="text-muted">-</span>;
  return (
    <div className="flex min-w-0 flex-wrap gap-1">
      {tags.slice(0, 3).map((tag) => <Badge key={tag}>{tag}</Badge>)}
      {tags.length > 3 ? <Badge>+{tags.length - 3}</Badge> : null}
    </div>
  );
}

function RefreshButton({ loading, onClick }: { loading?: boolean; onClick: () => void }) {
  return (
    <Button variant="secondary" size="sm" onClick={onClick} disabled={loading}>
      <RefreshCcw className={`h-3.5 w-3.5 ${loading ? "animate-spin" : ""}`} />
      Refresh
    </Button>
  );
}

function ErrorBanner({ message }: { message: string }) {
  if (!message) return null;
  return (
    <div className="mb-4 rounded-cds border border-[#f3b8ae] bg-[#fff1ef] px-3 py-2 text-sm text-[#8e2626]">
      {message}
    </div>
  );
}

function EmptyText({ children }: { children: ReactNode }) {
  return <div className="text-sm text-muted">{children}</div>;
}

function entryStatus(entry: CatalogEntry) {
  const value = entry.metadata?.["ard.status"];
  return typeof value === "string" && value ? value : "active";
}

function countStatuses(entries: CatalogEntry[]) {
  return entries.reduce(
    (counts, entry) => {
      const status = entryStatus(entry);
      if (status === "pending") counts.pending += 1;
      else if (status === "disabled") counts.disabled += 1;
      else counts.active += 1;
      return counts;
    },
    { active: 0, pending: 0, disabled: 0 }
  );
}

function publisherFromIdentifier(identifier: string) {
  const match = identifier.match(/^urn:air:([^:]+)/);
  return match?.[1] ?? "-";
}

function displayType(type: string) {
  if (type.includes("mcp-server")) return "MCP";
  if (type.includes("a2a-agent")) return "A2A";
  if (type.includes("agent-skills")) return "Skill";
  if (type.includes("openapi")) return "OpenAPI";
  if (type.includes("ai-registry")) return "Registry";
  if (type.includes("ai-catalog")) return "Catalog";
  return type;
}

function formatDate(value?: string) {
  if (!value) return "-";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleDateString();
}

function formatDateTime(value?: string) {
  if (!value) return "-";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString();
}

function shortHash(value: string) {
  if (value.length <= 14) return value;
  return `${value.slice(0, 7)}...${value.slice(-7)}`;
}

function messageFromError(error: unknown) {
  if (error instanceof Error) return error.message;
  return String(error);
}

function downloadJSON(filename: string, data: unknown) {
  const blob = new Blob([JSON.stringify(data, null, 2)], { type: "application/json" });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  link.click();
  URL.revokeObjectURL(url);
}

function parseMetrics(metrics: string) {
  const parsed: Record<string, string> = {};
  for (const line of metrics.split("\n")) {
    if (!line || line.startsWith("#")) continue;
    const [rawName, value] = line.trim().split(/\s+/);
    if (!rawName || value === undefined) continue;
    const name = rawName.split("{")[0];
    if (!(name in parsed)) parsed[name] = value;
  }
  return parsed;
}

function formatSeconds(value?: string) {
  if (!value) return "n/a";
  const seconds = Number(value);
  if (Number.isNaN(seconds)) return value;
  if (seconds < 60) return `${Math.round(seconds)}s`;
  if (seconds < 3600) return `${Math.round(seconds / 60)}m`;
  return `${Math.round(seconds / 3600)}h`;
}

const sampleCatalogJSON = JSON.stringify({
  specVersion: "1.0",
  host: {
    displayName: "Example Publisher"
  },
  entries: [
    {
      identifier: "urn:air:example.com:server:weather",
      displayName: "Weather MCP Server",
      type: "application/mcp-server-card+json",
      url: "https://example.com/mcp/weather.json",
      description: "Weather capability for agent workflows.",
      tags: ["weather", "mcp"],
      capabilities: ["forecast"],
      trustManifest: {
        identity: "https://example.com"
      }
    }
  ]
}, null, 2);

const sampleEntryJSON = JSON.stringify({
  identifier: "urn:air:example.com:api:weather",
  displayName: "Weather OpenAPI",
  type: "application/openapi+json",
  url: "https://example.com/openapi/weather.json",
  description: "OpenAPI weather capability for agent workflows.",
  tags: ["weather", "openapi"],
  trustManifest: {
    identity: "https://example.com"
  }
}, null, 2);
