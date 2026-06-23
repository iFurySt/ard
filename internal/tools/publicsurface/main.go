package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/ifuryst/ard/internal/cli"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type packageSurface struct {
	Consts  []string
	Funcs   []string
	Types   []string
	Methods []string
}

var expectedPackages = map[string]packageSurface{
	"pkg/ard": {
		Consts: []string{
			"TypeA2AAgentCard",
			"TypeAICatalog",
			"TypeAIRegistry",
			"TypeAIRegistryBare",
			"TypeAISkill",
			"TypeMCPServerCard",
			"TypeOpenAPI",
		},
		Funcs: []string{
			"Publisher",
			"ValidateCatalog",
			"ValidateCatalogEntry",
			"ValidateExploreRequest",
			"ValidateIdentifier",
			"ValidateSearchRequest",
		},
		Types: []string{
			"Catalog",
			"CatalogEntry",
			"ExploreFacet",
			"ExploreFacetBucket",
			"ExploreFacetRequest",
			"ExploreRequest",
			"ExploreResponse",
			"ExploreResultType",
			"Filter",
			"HostInfo",
			"ListResponse",
			"SearchQuery",
			"SearchRequest",
			"SearchResponse",
			"SearchResult",
		},
	},
	"pkg/client": {
		Funcs: []string{
			"New",
			"WithAdminToken",
			"WithHTTPClient",
			"WithHeader",
			"WithUserAgent",
		},
		Types: []string{
			"AdminAuditEvent",
			"AdminAuditOptions",
			"AdminAuditResponse",
			"AdminAuditVerification",
			"AdminCatalogImportResponse",
			"AdminListOptions",
			"AdminReviewOptions",
			"AdminStatusResponse",
			"BrowseOptions",
			"Client",
			"HTTPError",
			"HealthResponse",
			"Option",
		},
		Methods: []string{
			"Client.AdminApproveReview",
			"Client.AdminAudit",
			"Client.AdminDeleteEntry",
			"Client.AdminExportCatalog",
			"Client.AdminList",
			"Client.AdminRejectReview",
			"Client.AdminReviews",
			"Client.AdminSetStatus",
			"Client.AdminUpsertCatalog",
			"Client.AdminUpsertEntry",
			"Client.AdminVerifyAudit",
			"Client.Browse",
			"Client.Catalog",
			"Client.Explore",
			"Client.Health",
			"Client.Metrics",
			"Client.Search",
			"HTTPError.Error",
		},
	},
}

type cliSurface struct {
	Use          string
	Commands     []string
	Flags        []string
	CommandFlags map[string][]string
}

var expectedCLI = map[string]cliSurface{
	"ard": {
		Use: "ard",
		Commands: []string{
			"add",
			"admin",
			"browse",
			"crawl",
			"export",
			"health",
			"list",
			"metrics",
			"remove",
			"search",
			"serve",
			"verify",
			"version",
		},
		Flags: []string{"database-url", "policy-file"},
		CommandFlags: map[string][]string{
			"verify catalog": []string{
				"attestation-digests",
				"json",
				"jws-discover-did-web",
				"jws-discover-oidc",
				"jws-discover-spiffe",
				"jws-discover-tls-cert",
				"jws-remote-jwks",
				"jws-trust-anchors",
				"jws-tls-spki-pin",
				"provenance-digests",
				"require-attestation-digests",
				"require-jws-signatures",
				"require-jws-tls-spki-pins",
				"require-provenance-digests",
				"require-source-digests",
				"source-digests",
			},
		},
	},
	"ardctl": {
		Use: "ardctl",
		Commands: []string{
			"add",
			"admin",
			"browse",
			"crawl",
			"export",
			"health",
			"list",
			"metrics",
			"remove",
			"search",
			"verify",
			"version",
		},
		Flags: []string{"database-url", "policy-file"},
		CommandFlags: map[string][]string{
			"verify catalog": []string{
				"attestation-digests",
				"json",
				"jws-discover-did-web",
				"jws-discover-oidc",
				"jws-discover-spiffe",
				"jws-discover-tls-cert",
				"jws-remote-jwks",
				"jws-trust-anchors",
				"jws-tls-spki-pin",
				"provenance-digests",
				"require-attestation-digests",
				"require-jws-signatures",
				"require-jws-tls-spki-pins",
				"require-provenance-digests",
				"require-source-digests",
				"source-digests",
			},
		},
	},
	"ard-server": {
		Use:      "ard-server",
		Commands: []string{"version"},
		Flags: []string{
			"addr",
			"admin-token",
			"admin-tokens-file",
			"console-dir",
			"database-url",
			"otlp-traces-endpoint",
			"policy-file",
		},
	},
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "publicsurface: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	for path, expected := range expectedPackages {
		actual, err := collectPackageSurface(path)
		if err != nil {
			return err
		}
		if err := comparePackageSurface(path, expected, actual); err != nil {
			return err
		}
	}
	commands := map[string]*cobra.Command{
		"ard":        cli.NewRootCommand(),
		"ardctl":     cli.NewCLICommand(),
		"ard-server": cli.NewServerCommand(),
	}
	for name, command := range commands {
		if err := compareCLISurface(name, expectedCLI[name], command); err != nil {
			return err
		}
	}
	return nil
}

func collectPackageSurface(path string) (packageSurface, error) {
	root, err := repoRoot()
	if err != nil {
		return packageSurface{}, err
	}
	files, err := filepath.Glob(filepath.Join(root, path, "*.go"))
	if err != nil {
		return packageSurface{}, err
	}
	fileSet := token.NewFileSet()
	surface := packageSurface{}
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		parsed, err := parser.ParseFile(fileSet, file, nil, 0)
		if err != nil {
			return packageSurface{}, err
		}
		for _, declaration := range parsed.Decls {
			switch declaration := declaration.(type) {
			case *ast.GenDecl:
				for _, spec := range declaration.Specs {
					switch spec := spec.(type) {
					case *ast.ValueSpec:
						for _, name := range spec.Names {
							if !name.IsExported() {
								continue
							}
							if declaration.Tok == token.CONST {
								surface.Consts = append(surface.Consts, name.Name)
							}
						}
					case *ast.TypeSpec:
						if spec.Name.IsExported() {
							surface.Types = append(surface.Types, spec.Name.Name)
						}
					}
				}
			case *ast.FuncDecl:
				if !declaration.Name.IsExported() {
					continue
				}
				if declaration.Recv == nil {
					surface.Funcs = append(surface.Funcs, declaration.Name.Name)
					continue
				}
				receiver := receiverName(declaration.Recv)
				if receiver != "" && ast.IsExported(receiver) {
					surface.Methods = append(surface.Methods, receiver+"."+declaration.Name.Name)
				}
			}
		}
	}
	sortSurface(&surface)
	return surface, nil
}

func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find repository root from %s", dir)
		}
		dir = parent
	}
}

func receiverName(receiver *ast.FieldList) string {
	if receiver == nil || len(receiver.List) == 0 {
		return ""
	}
	switch expr := receiver.List[0].Type.(type) {
	case *ast.Ident:
		return expr.Name
	case *ast.StarExpr:
		if ident, ok := expr.X.(*ast.Ident); ok {
			return ident.Name
		}
	}
	return ""
}

func comparePackageSurface(path string, expected packageSurface, actual packageSurface) error {
	sortSurface(&expected)
	sortSurface(&actual)
	if !reflect.DeepEqual(expected, actual) {
		return fmt.Errorf("%s public surface drift\nexpected: %#v\nactual:   %#v", path, expected, actual)
	}
	return nil
}

func sortSurface(surface *packageSurface) {
	sort.Strings(surface.Consts)
	sort.Strings(surface.Funcs)
	sort.Strings(surface.Types)
	sort.Strings(surface.Methods)
}

func compareCLISurface(name string, expected cliSurface, command *cobra.Command) error {
	actual := cliSurface{
		Use:      command.Use,
		Commands: commandNames(command),
		Flags:    flagNames(command),
	}
	sort.Strings(expected.Commands)
	sort.Strings(expected.Flags)
	sortCommandFlags(expected.CommandFlags)
	if expected.CommandFlags != nil {
		actual.CommandFlags = map[string][]string{}
		for path := range expected.CommandFlags {
			nested, err := commandByPath(command, path)
			if err != nil {
				return fmt.Errorf("%s CLI surface drift: %w", name, err)
			}
			actual.CommandFlags[path] = flagNames(nested)
		}
		sortCommandFlags(actual.CommandFlags)
	}
	if !reflect.DeepEqual(expected, actual) {
		return fmt.Errorf("%s CLI surface drift\nexpected: %#v\nactual:   %#v", name, expected, actual)
	}
	return nil
}

func commandNames(command *cobra.Command) []string {
	names := []string{}
	for _, child := range command.Commands() {
		if child.Hidden {
			continue
		}
		names = append(names, child.Name())
	}
	sort.Strings(names)
	return names
}

func flagNames(command *cobra.Command) []string {
	seen := map[string]struct{}{}
	command.LocalFlags().VisitAll(func(flag *pflag.Flag) {
		seen[flag.Name] = struct{}{}
	})
	command.PersistentFlags().VisitAll(func(flag *pflag.Flag) {
		seen[flag.Name] = struct{}{}
	})
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func commandByPath(root *cobra.Command, path string) (*cobra.Command, error) {
	command := root
	for _, part := range strings.Fields(path) {
		next := (*cobra.Command)(nil)
		for _, child := range command.Commands() {
			if child.Name() == part {
				next = child
				break
			}
		}
		if next == nil {
			return nil, fmt.Errorf("missing command path %q", path)
		}
		command = next
	}
	return command, nil
}

func sortCommandFlags(commandFlags map[string][]string) {
	for path := range commandFlags {
		sort.Strings(commandFlags[path])
	}
}
