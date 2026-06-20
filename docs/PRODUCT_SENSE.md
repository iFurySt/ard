# Product Sense

`ard` is a neutral self-hosted registry and toolkit for Agentic Resource Discovery. It
is built for organizations that need to publish, discover, verify, and govern agentic
resources without depending on a single platform vendor.

## Market Read

ARD is positioned as the discovery layer after MCP, Skills, A2A, and OpenAPI. The major
reference implementations and product announcements point in two directions:

- Platform registries: Google Cloud Agent Registry and Hugging Face Discover.
- Publisher-side catalogs: `/.well-known/ai-catalog.json` documents hosted under an
  organization's own domain.

The open-source opportunity is the middle: a neutral registry distribution that any
enterprise, agent platform, or internal tools team can fork and self-host.

## Primary Users

- Platform teams building internal agent infrastructure.
- Developer experience teams standardizing internal MCP servers, Skills, A2A agents, and
  APIs.
- Security and governance teams that need verification, policy, and auditability before
  agents connect to newly discovered capabilities.
- Open-source agent framework maintainers that need a standards-aligned registry and
  client they can embed or recommend.

## Positioning

The project should be framed as:

> Neutral self-hosted registry and toolkit for Agentic Resource Discovery.

The primary value is not "another search engine." The primary value is a deployable
control plane for agentic resource discovery:

- Find the right capability by intent.
- Verify the publisher and artifact metadata before use.
- Govern which capabilities agents may connect to.
- Federate with other ARD catalogs and registries without surrendering internal control.

## Product Principles

- Self-hosted first: enterprises should be able to run this in an internal network.
- Neutral by default: do not bind the core to one model provider, agent framework, cloud,
  or resource protocol.
- Standards-aligned: implement ARD primitives directly instead of inventing new discovery
  semantics.
- Spec-disciplined: treat `ards-project/ard-spec` as the primary source for schemas,
  protocol details, conformance behavior, and naming rules.
- Trust-aware: verification, publisher identity, artifact pinning, and policy should be
  first-class, not post-launch add-ons.
- Fast to adopt: a single binary or simple container should be enough for local and
  internal trials.
- Adapter-rich over UI-rich: early impact comes from supporting existing MCP, Skills,
  A2A, and OpenAPI assets, not from a large dashboard.

## MVP

The first release should prove the core loop:

- Run `ard serve` to expose an ARD-compatible registry.
- Add catalogs and artifacts with `ard add`.
- Crawl `/.well-known/ai-catalog.json` catalogs and nested registry referrals.
- Search through `POST /search` and `ard search`.
- Validate ARD schemas, media types, `url`/`data` exclusivity, and domain-anchored
  `urn:air:` identifiers.
- Perform lightweight verification for publisher domain, trust metadata, and pinned
  artifacts.
- Run the upstream ARD conformance suite against manifests and registry endpoints.

## Non-Goals

- Do not start as a hosted public marketplace.
- Do not replace MCP, Skills, A2A, OpenAPI, or agent runtimes.
- Do not make vector search mandatory for the first release.
- Do not build a full enterprise governance product before the registry core is useful.
- Do not require external SaaS dependencies for a basic deployment.

## Success Signals

- Agent framework authors can use the CLI or Go client without platform-specific
  assumptions.
- Internal platform teams can deploy the registry in less than 10 minutes.
- Existing MCP servers, Skills, A2A agents, and OpenAPI specs can be published without
  rewriting them.
- The project is cited as the neutral self-hosted ARD option alongside platform-native
  registries.
