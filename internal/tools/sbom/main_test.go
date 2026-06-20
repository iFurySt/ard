package main

import "testing"

func TestSanitizeID(t *testing.T) {
	t.Parallel()

	got := sanitizeID("github.com/ifuryst/ard v0.1.0")
	if got != "github.com-ifuryst-ard-v0.1.0" {
		t.Fatalf("unexpected sanitized ID: %s", got)
	}
}

func TestBuildDocument(t *testing.T) {
	t.Parallel()

	doc := buildDocument([]module{
		{Path: defaultRepoName, Main: true},
		{Path: "github.com/spf13/cobra", Version: "v1.10.2"},
	}, "v0.1.0", "2026-06-21T00:00:00Z")

	if doc.SPDXVersion != spdxVersion {
		t.Fatalf("unexpected SPDX version: %s", doc.SPDXVersion)
	}
	if len(doc.Packages) != 2 {
		t.Fatalf("unexpected package count: %d", len(doc.Packages))
	}
	if doc.Packages[0].SPDXID != mainPackageID {
		t.Fatalf("main package should sort first, got %s", doc.Packages[0].SPDXID)
	}
	if doc.Packages[0].VersionInfo != "v0.1.0" {
		t.Fatalf("main package should use release version, got %s", doc.Packages[0].VersionInfo)
	}
	if len(doc.Relationships) != 2 {
		t.Fatalf("unexpected relationship count: %d", len(doc.Relationships))
	}
}
