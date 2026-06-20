# ARD Spec Working Notes

Date captured: 2026-06-20

These notes summarize the local and upstream `ards-project/ard-spec` review used to
ground `ard` implementation planning.

## Source

- GitHub: https://github.com/ards-project/ard-spec
- Rendered spec: https://agenticresourcediscovery.org/spec/
- Local checkout reviewed during planning: `a78be70`
- Upstream status observed from README: v0.9 draft, Apache-2.0, no GitHub releases yet.

The upstream repository describes itself as the canonical home of the Agentic Resource
Discovery specification and contains:

- `spec/ard.md`: main specification.
- `spec/schemas/`: CDDL, JSON Schema, and OpenAPI definitions.
- `adr/`: architecture decision records.
- `conformance/`: manifest and registry conformance tooling.

## Impact On Our Previous Judgment

The previous product direction still holds:

- A neutral self-hosted registry is a strong open-source wedge.
- B2B users need internal discovery, verification, policy, and federation control.
- The project should be a neutral self-hosted registry and toolkit for Agentic Resource
  Discovery, not a public marketplace or agent runtime.

The spec review adds important corrections and priorities:

- Use `urn:air:` identifiers. Older `urn:ai:` material is superseded by ADR-0009 and the
  current schema.
- Use `application/mcp-server-card+json` for MCP discovery cards.
- Treat `score` as semantic relevance only; trust and safety must remain separate
  verification concerns.
- Implement `POST /search` first; `POST /explore` and `GET /agents` are optional but
  valuable for enterprise portals.
- Support web ingestion of `ai-catalog.json`; the spec says ARD implementations must
  support this.
- Keep protocol wrappers such as MCP or A2A search tools optional and secondary to the
  REST API.

## Implementation Requirements To Carry Forward

- `GET /.well-known/ai-catalog.json` discovery documents should advertise registry and
  catalog entries.
- Registry base URLs are discovered through catalog entries with
  `application/ai-registry+json` or `application/ai-registry`.
- `POST /search` requires `query.text`; `query.filter` is optional.
- `SearchRequest.federation` is root-level and supports `auto`, `referrals`, and `none`.
- `SearchRequest.pageSize` and `pageToken` are root-level.
- `SearchResponse.results[]` are catalog entries plus `score` and `source`.
- `SearchResponse.referrals[]` can return registry catalog entries for client-followed
  federation.
- `POST /explore` shares the query model, returns facets, does not federate, and may
  return `501`.
- `GET /agents` is optional deterministic browsing and uses its own filter syntax.
- Catalog entries require `identifier`, `displayName`, and `type`, plus exactly one of
  `url` or `data`.
- `representativeQueries` should contain 2 to 5 examples when present.
- Trust metadata belongs in `trustManifest`; cryptographic verification is separate from
  search ranking.

## Known Upstream Drift To Watch

The reviewed checkout has some internal version drift:

- README and main spec say v0.9 draft.
- OpenAPI and conformance tooling still include v0.5.0 labels in places.
- Some older ADR context text mentions `urn:ai:` even though the accepted current form is
  `urn:air:`.

Implementation should follow the current main spec, schemas, and conformance behavior,
not older explanatory text from superseded ADR context.

## Source Management Recommendation

Do not add `ard-spec` as a git submodule yet.

Submodules are useful when a repository must build or test against an exact upstream tree
on every checkout. At this stage we mostly need a human-readable reference and a few
machine-readable schemas/conformance scripts. A submodule would add clone and CI friction
before the implementation actually consumes those files.

Preferred path:

1. Keep links and the observed source commit in `docs/references/`.
2. During implementation, integrate against a local sibling checkout or a pinned download.
3. Once code generation or conformance checks require stable files in this repo, vendor
   only the needed artifacts under a clear third-party path and record:
   - upstream URL,
   - source commit,
   - license,
   - update command,
   - local modifications, if any.
4. Reconsider a submodule only if we repeatedly need the full upstream repository,
   including ADRs, schemas, examples, and conformance scripts, as live test fixtures.

## Follow-Up Tasks

- Decide where third-party spec artifacts should live if vendored.
- Add a script to run upstream conformance against local manifests and registry endpoints.
- Add validation tests for `urn:air:` and `application/mcp-server-card+json`.
- Keep release notes explicit when upstream spec revisions require breaking changes.
