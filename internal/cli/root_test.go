package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootCommandIncludesServe(t *testing.T) {
	command := NewRootCommand()
	if command.Use != "ard" {
		t.Fatalf("unexpected root use: %s", command.Use)
	}
	if command.CommandPath() != "ard" {
		t.Fatalf("unexpected command path: %s", command.CommandPath())
	}
	if _, _, err := command.Find([]string{"serve"}); err != nil {
		t.Fatalf("expected ard serve command: %v", err)
	}
	if _, _, err := command.Find([]string{"version"}); err != nil {
		t.Fatalf("expected ard version command: %v", err)
	}
}

func TestCLICommandOmitsServe(t *testing.T) {
	command := NewCLICommand()
	if command.Use != "ardctl" {
		t.Fatalf("unexpected cli use: %s", command.Use)
	}
	if found, _, err := command.Find([]string{"serve"}); err == nil && found.Name() == "serve" {
		t.Fatal("ardctl should not expose the server command")
	}
	if _, _, err := command.Find([]string{"add"}); err != nil {
		t.Fatalf("expected ardctl add command: %v", err)
	}
	if _, _, err := command.Find([]string{"admin"}); err != nil {
		t.Fatalf("expected ardctl admin command: %v", err)
	}
	if _, _, err := command.Find([]string{"browse"}); err != nil {
		t.Fatalf("expected ardctl browse command: %v", err)
	}
	if _, _, err := command.Find([]string{"search"}); err != nil {
		t.Fatalf("expected ardctl search command: %v", err)
	}
	if _, _, err := command.Find([]string{"health"}); err != nil {
		t.Fatalf("expected ardctl health command: %v", err)
	}
	if _, _, err := command.Find([]string{"metrics"}); err != nil {
		t.Fatalf("expected ardctl metrics command: %v", err)
	}
	if _, _, err := command.Find([]string{"version"}); err != nil {
		t.Fatalf("expected ardctl version command: %v", err)
	}
	if _, _, err := command.Find([]string{"export"}); err != nil {
		t.Fatalf("expected ardctl export command: %v", err)
	}
	if _, _, err := command.Find([]string{"list"}); err != nil {
		t.Fatalf("expected ardctl list command: %v", err)
	}
	if _, _, err := command.Find([]string{"remove"}); err != nil {
		t.Fatalf("expected ardctl remove command: %v", err)
	}
	if command.Flag("admin-token") != nil {
		t.Fatal("ardctl should not expose server admin token flag")
	}
	if command.Flag("admin-tokens-file") != nil {
		t.Fatal("ardctl should not expose server admin tokens file flag")
	}
}

func TestServerCommandRunsAtRoot(t *testing.T) {
	command := NewServerCommand()
	if command.Use != "ard-server" {
		t.Fatalf("unexpected server use: %s", command.Use)
	}
	if command.RunE == nil {
		t.Fatal("ard-server should run the registry server at the root command")
	}
	if command.Flag("admin-token") == nil {
		t.Fatal("ard-server should expose admin token flag")
	}
	if command.Flag("admin-tokens-file") == nil {
		t.Fatal("ard-server should expose admin tokens file flag")
	}
	if _, _, err := command.Find([]string{"version"}); err != nil {
		t.Fatalf("expected ard-server version command: %v", err)
	}
	if found, _, err := command.Find([]string{"admin"}); err == nil && found.Name() == "admin" {
		t.Fatal("ard-server should not expose management subcommands")
	}
	if found, _, err := command.Find([]string{"health"}); err == nil && found.Name() == "health" {
		t.Fatal("ard-server should not expose client health subcommands")
	}
	if found, _, err := command.Find([]string{"metrics"}); err == nil && found.Name() == "metrics" {
		t.Fatal("ard-server should not expose client metrics subcommands")
	}
}

func TestVersionCommandTextAndJSON(t *testing.T) {
	command := NewRootCommand()
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetErr(&output)
	command.SetArgs([]string{"version"})
	if err := command.Execute(); err != nil {
		t.Fatalf("version command: %v", err)
	}
	if got := output.String(); !strings.Contains(got, "version=") || !strings.Contains(got, "commit=") {
		t.Fatalf("unexpected version output: %s", got)
	}

	output.Reset()
	command = NewRootCommand()
	command.SetOut(&output)
	command.SetErr(&output)
	command.SetArgs([]string{"version", "--json"})
	if err := command.Execute(); err != nil {
		t.Fatalf("version --json command: %v", err)
	}
	if got := output.String(); !strings.Contains(got, `"version"`) || !strings.Contains(got, `"commit"`) {
		t.Fatalf("unexpected JSON version output: %s", got)
	}
}
