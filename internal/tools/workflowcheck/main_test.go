package main

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestReleaseWorkflowShape(t *testing.T) {
	t.Parallel()

	root := parseWorkflow(t, `
name: Release
on:
  push:
    tags:
      - "v*"
permissions:
  contents: write
  id-token: write
  attestations: write
  artifact-metadata: write
jobs:
  release:
    steps:
      - uses: actions/checkout@34e114876b0b11c390a56381ad16ebd13914f8d5
      - uses: actions/setup-go@40f1582b2485089dde7abd97c1529aa768e1baff
      - run: make package
      - run: shasum -a 256 -c checksums.txt
      - uses: actions/attest@59d89421af93a897026c735860bf21b6eb4f7b26
        with:
          subject-checksums: dist/checksums.txt
      - uses: actions/attest@59d89421af93a897026c735860bf21b6eb4f7b26
        with:
          subject-checksums: /tmp/release-archives.checksums.txt
          sbom-path: dist/sbom.spdx.json
      - run: gh release create "$GITHUB_REF_NAME" dist/*
`)

	if err := checkRelease(root); err != nil {
		t.Fatalf("expected release workflow to pass: %v", err)
	}
}

func TestReleaseWorkflowRequiresSBOMAttestation(t *testing.T) {
	t.Parallel()

	root := parseWorkflow(t, `
name: Release
on:
  push:
    tags: ["v*"]
permissions:
  contents: write
  id-token: write
  attestations: write
  artifact-metadata: write
jobs:
  release:
    steps:
      - uses: actions/checkout@34e114876b0b11c390a56381ad16ebd13914f8d5
      - uses: actions/setup-go@40f1582b2485089dde7abd97c1529aa768e1baff
      - run: make package
      - run: shasum -a 256 -c checksums.txt
      - uses: actions/attest@59d89421af93a897026c735860bf21b6eb4f7b26
        with:
          subject-checksums: dist/checksums.txt
      - run: gh release create "$GITHUB_REF_NAME" dist/*
`)

	if err := checkRelease(root); err == nil || !strings.Contains(err.Error(), "SBOM") {
		t.Fatalf("expected missing SBOM attestation error, got %v", err)
	}
}

func parseWorkflow(t *testing.T, content string) *yaml.Node {
	t.Helper()

	var document yaml.Node
	if err := yaml.Unmarshal([]byte(content), &document); err != nil {
		t.Fatal(err)
	}
	if len(document.Content) != 1 {
		t.Fatalf("unexpected document content length: %d", len(document.Content))
	}
	return document.Content[0]
}
