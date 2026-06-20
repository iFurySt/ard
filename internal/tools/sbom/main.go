package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	spdxVersion     = "SPDX-2.3"
	documentID      = "SPDXRef-DOCUMENT"
	mainPackageID   = "SPDXRef-Package-github.com-ifuryst-ard"
	defaultCreated  = "1970-01-01T00:00:00Z"
	defaultVersion  = "dev"
	defaultOut      = "dist/sbom.spdx.json"
	defaultRepoName = "github.com/ifuryst/ard"
)

type module struct {
	Path     string  `json:"Path"`
	Version  string  `json:"Version"`
	Main     bool    `json:"Main"`
	Indirect bool    `json:"Indirect"`
	Replace  *module `json:"Replace"`
}

type document struct {
	SPDXVersion       string         `json:"spdxVersion"`
	DataLicense       string         `json:"dataLicense"`
	SPDXID            string         `json:"SPDXID"`
	Name              string         `json:"name"`
	DocumentNamespace string         `json:"documentNamespace"`
	CreationInfo      creationInfo   `json:"creationInfo"`
	Packages          []spdxPackage  `json:"packages"`
	Relationships     []relationship `json:"relationships"`
}

type creationInfo struct {
	Created  string   `json:"created"`
	Creators []string `json:"creators"`
}

type spdxPackage struct {
	Name              string        `json:"name"`
	SPDXID            string        `json:"SPDXID"`
	VersionInfo       string        `json:"versionInfo,omitempty"`
	DownloadLocation  string        `json:"downloadLocation"`
	FilesAnalyzed     bool          `json:"filesAnalyzed"`
	LicenseConcluded  string        `json:"licenseConcluded"`
	LicenseDeclared   string        `json:"licenseDeclared"`
	CopyrightText     string        `json:"copyrightText"`
	ExternalRefs      []externalRef `json:"externalRefs,omitempty"`
	Supplier          string        `json:"supplier"`
	PackageComment    string        `json:"comment,omitempty"`
	PackageSourceInfo string        `json:"sourceInfo,omitempty"`
}

type externalRef struct {
	ReferenceCategory string `json:"referenceCategory"`
	ReferenceType     string `json:"referenceType"`
	ReferenceLocator  string `json:"referenceLocator"`
}

type relationship struct {
	SPDXElementID      string `json:"spdxElementId"`
	RelationshipType   string `json:"relationshipType"`
	RelatedSPDXElement string `json:"relatedSpdxElement"`
}

var spdxIDRe = regexp.MustCompile(`[^A-Za-z0-9.-]+`)

func main() {
	version := flag.String("version", defaultVersion, "release version to record for the main module")
	created := flag.String("created", defaultCreated, "SPDX creation timestamp in RFC3339 format")
	out := flag.String("out", defaultOut, "output SPDX JSON path")
	flag.Parse()

	if err := run(*version, *created, *out); err != nil {
		fmt.Fprintf(os.Stderr, "sbom: %v\n", err)
		os.Exit(1)
	}
}

func run(version, created, out string) error {
	if _, err := time.Parse(time.RFC3339, created); err != nil {
		return fmt.Errorf("created timestamp must be RFC3339: %w", err)
	}

	modules, err := listModules()
	if err != nil {
		return err
	}
	if len(modules) == 0 {
		return fmt.Errorf("go list returned no modules")
	}

	doc := buildDocument(modules, version, created)
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	if dir := filepath.Dir(out); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(out, data, 0o644)
}

func listModules() ([]module, error) {
	command := exec.Command("go", "list", "-m", "-json", "all")
	output, err := command.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("go list -m -json all failed: %s", strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, err
	}

	decoder := json.NewDecoder(strings.NewReader(string(output)))
	var modules []module
	for {
		var current module
		if err := decoder.Decode(&current); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		if current.Path == "" {
			continue
		}
		modules = append(modules, current)
	}
	return modules, nil
}

func buildDocument(modules []module, version, created string) document {
	sort.SliceStable(modules, func(i, j int) bool {
		if modules[i].Main != modules[j].Main {
			return modules[i].Main
		}
		return modules[i].Path < modules[j].Path
	})

	packages := make([]spdxPackage, 0, len(modules))
	relationships := []relationship{
		{
			SPDXElementID:      documentID,
			RelationshipType:   "DESCRIBES",
			RelatedSPDXElement: mainPackageID,
		},
	}

	for _, mod := range modules {
		pkg := packageForModule(mod, version)
		packages = append(packages, pkg)
		if !mod.Main {
			relationships = append(relationships, relationship{
				SPDXElementID:      mainPackageID,
				RelationshipType:   "DEPENDS_ON",
				RelatedSPDXElement: pkg.SPDXID,
			})
		}
	}

	return document{
		SPDXVersion:       spdxVersion,
		DataLicense:       "CC0-1.0",
		SPDXID:            documentID,
		Name:              "ard release SBOM",
		DocumentNamespace: "https://github.com/iFurySt/ard/sbom/" + sanitizeID(version),
		CreationInfo: creationInfo{
			Created: created,
			Creators: []string{
				"Tool: ard internal/tools/sbom",
				"Organization: iFurySt",
			},
		},
		Packages:      packages,
		Relationships: relationships,
	}
}

func packageForModule(mod module, releaseVersion string) spdxPackage {
	version := mod.Version
	if mod.Main && version == "" {
		version = releaseVersion
	}

	pkg := spdxPackage{
		Name:             mod.Path,
		SPDXID:           spdxIDForModule(mod),
		VersionInfo:      version,
		DownloadLocation: downloadLocation(mod),
		FilesAnalyzed:    false,
		LicenseConcluded: "NOASSERTION",
		LicenseDeclared:  "NOASSERTION",
		CopyrightText:    "NOASSERTION",
		Supplier:         "NOASSERTION",
		ExternalRefs: []externalRef{
			{
				ReferenceCategory: "PACKAGE-MANAGER",
				ReferenceType:     "purl",
				ReferenceLocator:  packageURL(mod, version),
			},
		},
	}

	if mod.Replace != nil {
		pkg.PackageComment = fmt.Sprintf("Replaced by %s %s", mod.Replace.Path, mod.Replace.Version)
	}
	if mod.Indirect {
		pkg.PackageSourceInfo = "Go module marked indirect in module graph."
	}
	return pkg
}

func spdxIDForModule(mod module) string {
	if mod.Main {
		return mainPackageID
	}
	return "SPDXRef-Package-" + sanitizeID(mod.Path+"-"+mod.Version)
}

func sanitizeID(value string) string {
	value = strings.TrimSpace(value)
	value = spdxIDRe.ReplaceAllString(value, "-")
	value = strings.Trim(value, "-.")
	if value == "" {
		return "unknown"
	}
	return value
}

func packageURL(mod module, version string) string {
	path := strings.TrimPrefix(mod.Path, "/")
	escapedPath := strings.ReplaceAll(url.PathEscape(path), "%2F", "/")
	if version == "" {
		return "pkg:golang/" + escapedPath
	}
	return "pkg:golang/" + escapedPath + "@" + url.QueryEscape(version)
}

func downloadLocation(mod module) string {
	if mod.Main {
		return "git+https://github.com/iFurySt/ard.git"
	}
	return "https://pkg.go.dev/" + strings.TrimPrefix(mod.Path, "/")
}
