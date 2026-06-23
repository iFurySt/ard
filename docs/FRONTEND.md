# Frontend Guide

OpenARD Console is the administrator-facing web UI under `apps/console`. It is a
Vite/React workspace that reuses a compact CDS-style component layer and talks to the
existing registry HTTP API.

## Local Development

Install dependencies once:

```sh
npm install
```

Run the console:

```sh
make console-dev
```

The Vite dev server listens on `http://localhost:5173/console/` and proxies OpenARD
API routes to `http://localhost:8080` by default. Override the proxy target when
needed:

```sh
ARD_CONSOLE_PROXY_TARGET=http://127.0.0.1:9090 make console-dev
```

In the console Settings page, leave the Registry API base URL empty when using the dev
proxy or same-origin deployment. Set an admin bearer token to unlock protected Catalog,
Reviews, Audit, and management actions.

## Registry-Hosted Console

Production builds default to the `/console/` base path so `ard-server` can serve them
from the registry origin:

```sh
make console-build
bin/ard-server --console-dir apps/console/dist
```

`--console-dir` defaults to `ARD_CONSOLE_DIR` when the flag is omitted. Keep the
console Settings API base URL empty for this same-origin mode. Use
`ARD_CONSOLE_BASE=/` only for a standalone static host that serves the app from `/`.

The Docker image builds and includes the console at `/usr/share/openard/console`; the
compose stack serves it at `http://127.0.0.1:18080/console/`.

## Build And Checks

```sh
make console-lint
make console-build
```

`console-lint` runs TypeScript with no emit. `console-build` runs the TypeScript build
and Vite production build.

## Browser Verification

For UI changes, verify through the in-app browser with DOM/style inspection, screenshots,
and console/network checks. At minimum, cover:

- first render at `/console/overview`
- sidebar navigation across administrator sections
- Settings save/clear behavior
- an API-backed page against a local registry when available
- resource details dialogs from Discover, Catalog, and Reviews when entry-shape data
  changes
- browser console has no runtime errors

## Component Boundaries

- `src/components/cds.tsx` contains the shared component primitives copied from the
  reference console and should stay product-neutral.
- `src/api.ts` is the only fetch layer. Keep auth header handling and endpoint building
  there.
- `src/App.tsx` owns the first administrator workflows: Overview, Discover, Catalog,
  Add Resource, Reviews, Audit Log, Operations, Settings, and resource detail
  inspection.
