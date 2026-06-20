# ARD Market Context

Date captured: 2026-06-20

This note summarizes the launch context that shaped the initial direction for `ard`.

## Primary Sources

- Google Developers Blog, "Announcing the Agentic Resource Discovery specification",
  published 2026-06-17:
  https://developers.googleblog.com/announcing-the-agentic-resource-discovery-specification/
- Hugging Face Blog, "Agentic Resource Discovery: Let agents search for tools, skills,
  and other agents", published 2026-06-17:
  https://huggingface.co/blog/agentic-resource-discovery-launch
- ARD specification repository:
  https://github.com/ards-project/ard-spec
- Reference implementation studied during planning: Hugging Face `hf-discover`.

## What ARD Adds

ARD is the discovery layer in front of agentic resources. MCP, Skills, A2A, OpenAPI, and
similar protocols define how a capability is used. ARD defines how a client finds,
filters, verifies, and resolves capabilities before using their native protocol.

The specification centers on two primitives:

- Catalogs: publisher-hosted `ai-catalog.json` files, commonly exposed under
  `/.well-known/ai-catalog.json`.
- Registries: searchable services that crawl, index, rank, and return catalog entries
  through standard ARD search APIs.

The strategic shift is from preinstalled static tool lists to intent-based discovery.
Agents can search for a capability at runtime, inspect metadata, verify the publisher,
and then connect directly through MCP, A2A, Skills, OpenAPI, or another artifact-specific
protocol.

## Signals From Google

Google frames ARD around three enterprise questions:

- Where does the right capability live?
- Which capability should the agent use?
- How does the agent verify that it is safe to connect?

The announcement emphasizes catalogs, registries, publisher-owned domains, cryptographic
verification, trust manifests, globally unique URNs, egress policies, and enterprise
governance. Google Cloud Agent Registry is positioned as a hosted enterprise product in
Gemini Enterprise Agent Platform.

Implication for `ard`: avoid competing with a cloud-hosted enterprise product head-on.
Build the neutral open-source distribution that organizations can self-host and adapt.

## Signals From Hugging Face

Hugging Face Discover is a reference implementation that adapts existing Hugging Face Hub
search into the ARD envelope. It exposes Hub Spaces, generated Skills, and MCP-tagged
Spaces through ARD `SearchResponse` results. It demonstrates that a registry can wrap an
existing ecosystem without inventing a new artifact protocol.

Implication for `ard`: the registry should be adapter-oriented. The core should normalize
existing resources into ARD entries instead of forcing resource owners to rewrite their
tools.

## Strategic Conclusion

The strongest open-source opportunity is still not a public marketplace and not a
platform-specific registry. The upstream ARD spec reinforces this direction: registries
are a required dynamic discovery layer, web catalog ingestion is required, protocol
wrappers are optional, and enterprise trust is delegated into catalog metadata rather
than owned by any one cloud platform.

`ard` should therefore be a neutral self-hosted registry and toolkit for Agentic
Resource Discovery:

> Neutral self-hosted registry and toolkit for Agentic Resource Discovery.

The project should make it easy for organizations to:

- Publish internal and external agentic resources.
- Crawl and index ARD catalogs.
- Search by natural-language intent and structured filters.
- Verify publisher identity and artifact metadata.
- Apply local policy before agents connect.
- Federate with other ARD catalogs and registries where appropriate.

## Product Shape

The initial distribution should include:

- Registry server: `POST /search`, `GET /.well-known/ai-catalog.json`, health endpoint,
  and optional `POST /explore`.
- CLI: `serve`, `add`, `crawl`, `verify`, `search`, and `export`.
- Client library: standards-aligned catalog fetch, registry search, and navigation.
- Publisher kit: adapters for MCP, Skills, A2A, OpenAPI, and direct URL artifacts.
- Verification engine: schema validation, domain-anchored URNs, media types,
  `url`/`data` exclusivity, trust metadata, and artifact pinning.

## Specification Corrections

The newer `ards-project/ard-spec` draft supersedes some details from older reference
implementations:

- Identifier NID is `urn:air:`, not `urn:ai:`.
- MCP discovery entries use `application/mcp-server-card+json`.
- `score` is semantic relevance only and must not be interpreted as trust, compliance, or
  safety.
- `POST /explore` and `GET /agents` are optional registry endpoints.
- Web ingestion of `ai-catalog.json` catalogs is required for ARD implementations.

## Naming

Use `ard` as the repository and project name. The CLI should also be `ard` unless a
packaging conflict requires a longer executable name.

## License Direction

The repository uses Apache-2.0 for enterprise adoption and patent grant clarity.
