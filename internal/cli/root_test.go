package cli

import "testing"

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
	if _, _, err := command.Find([]string{"search"}); err != nil {
		t.Fatalf("expected ardctl search command: %v", err)
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
	if len(command.Commands()) != 0 {
		t.Fatalf("ard-server should not expose management subcommands, got %d", len(command.Commands()))
	}
}
