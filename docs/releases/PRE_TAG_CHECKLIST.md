# Pre-Tag Checklist

Use this before creating a public `v*` tag.

## Inputs

- Version: `vMAJOR.MINOR.PATCH`, for example `v0.1.0`.
- Target commit: the current `main` commit intended for release.
- Release note: a matching entry in `docs/releases/feature-release-notes.md`.

## Required Checks

```sh
git status --short
git fetch --tags origin
git tag --list "$VERSION"
VERSION="$VERSION" make release-dry-run
make test-e2e
```

Required result:

- Working tree is clean.
- The version tag does not already exist locally or on origin.
- `VERSION="$VERSION" make release-dry-run` passes.
- `make test-e2e` passes against live MCP, Skill, OpenAPI, and checked-in A2A fixtures.

## Review

- README and GitHub About still match the intended public positioning.
- `docs/SDK_COMPATIBILITY.md` still reflects the public Go SDK boundary.
- `docs/QUALITY_SCORE.md` has no first-tag blocker that should be fixed now.
- Release notes describe user-visible changes without leaking local paths, tokens, or
  private infrastructure details.
- Any known breaking change is intentional and documented.

## Tag

Only create the tag after the checks and review pass:

```sh
git tag -a "$VERSION" -m "$VERSION"
git push origin "$VERSION"
```

## Post-Release

```sh
gh release create "$VERSION" dist/* --repo iFurySt/ard --verify-tag --title "$VERSION" --generate-notes
gh release download "$VERSION" --repo iFurySt/ard --dir /tmp/ard-release-check --clobber
(cd /tmp/ard-release-check && shasum -a 256 -c checksums.txt)
```

Record the final decision in the relevant history entry or release note.
