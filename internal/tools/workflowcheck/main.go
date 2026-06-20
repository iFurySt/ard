package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	attestAction   = "actions/attest@59d89421af93a897026c735860bf21b6eb4f7b26"
	checkoutAction = "actions/checkout@34e114876b0b11c390a56381ad16ebd13914f8d5"
	setupGoAction  = "actions/setup-go@40f1582b2485089dde7abd97c1529aa768e1baff"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "workflowcheck: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	paths, err := filepath.Glob(".github/workflows/*")
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		return fmt.Errorf("no GitHub Actions workflows found")
	}

	foundCI := false
	foundRelease := false

	for _, path := range paths {
		extension := filepath.Ext(path)
		if extension != ".yml" && extension != ".yaml" {
			continue
		}
		root, err := readWorkflow(path)
		if err != nil {
			return err
		}
		if filepath.Base(path) == "ci.yml" {
			foundCI = true
			if err := checkCI(root); err != nil {
				return fmt.Errorf("%s: %w", path, err)
			}
		}
		if filepath.Base(path) == "release.yml" {
			foundRelease = true
			if err := checkRelease(root); err != nil {
				return fmt.Errorf("%s: %w", path, err)
			}
		}
	}
	if !foundCI {
		return fmt.Errorf("missing .github/workflows/ci.yml")
	}
	if !foundRelease {
		return fmt.Errorf("missing .github/workflows/release.yml")
	}

	return nil
}

func readWorkflow(path string) (*yaml.Node, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var document yaml.Node
	if err := yaml.Unmarshal(data, &document); err != nil {
		return nil, err
	}
	if len(document.Content) != 1 || document.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("workflow root must be a mapping")
	}
	return document.Content[0], nil
}

func checkCI(root *yaml.Node) error {
	steps := jobSteps(root, "go")
	if steps == nil {
		return fmt.Errorf("missing go job steps")
	}
	if !hasRunStep(steps, "make package") {
		return fmt.Errorf("CI must run make package")
	}
	return nil
}

func checkRelease(root *yaml.Node) error {
	if !releaseTagTrigger(root) {
		return fmt.Errorf("release workflow must trigger on v* tags")
	}
	requiredPermissions := map[string]string{
		"contents":          "write",
		"id-token":          "write",
		"attestations":      "write",
		"artifact-metadata": "write",
	}
	for name, value := range requiredPermissions {
		if scalarValue(mappingValue(mappingValue(root, "permissions"), name)) != value {
			return fmt.Errorf("missing permission %s: %s", name, value)
		}
	}

	steps := jobSteps(root, "release")
	if steps == nil {
		return fmt.Errorf("missing release job steps")
	}
	if !hasUsesStep(steps, checkoutAction) {
		return fmt.Errorf("release job must use pinned checkout action")
	}
	if !hasUsesStep(steps, setupGoAction) {
		return fmt.Errorf("release job must use pinned setup-go action")
	}
	if !hasRunStep(steps, "make package") {
		return fmt.Errorf("release job must run make package")
	}
	if !hasRunStep(steps, "shasum -a 256 -c checksums.txt") {
		return fmt.Errorf("release job must verify checksums")
	}
	if attestStepCount(steps) < 2 {
		return fmt.Errorf("release job must include provenance and SBOM attestations")
	}
	if !hasAttestSBOMStep(steps) {
		return fmt.Errorf("release job must include an SBOM attestation")
	}
	if !hasRunStep(steps, "gh release create") {
		return fmt.Errorf("release job must publish a GitHub release")
	}

	return nil
}

func releaseTagTrigger(root *yaml.Node) bool {
	on := mappingValue(root, "on")
	push := mappingValue(on, "push")
	tags := mappingValue(push, "tags")
	for _, value := range sequenceValues(tags) {
		if value == "v*" {
			return true
		}
	}
	return false
}

func jobSteps(root *yaml.Node, jobName string) *yaml.Node {
	jobs := mappingValue(root, "jobs")
	job := mappingValue(jobs, jobName)
	steps := mappingValue(job, "steps")
	if steps == nil || steps.Kind != yaml.SequenceNode {
		return nil
	}
	return steps
}

func hasRunStep(steps *yaml.Node, contains string) bool {
	for _, step := range steps.Content {
		if strings.Contains(scalarValue(mappingValue(step, "run")), contains) {
			return true
		}
	}
	return false
}

func attestStepCount(steps *yaml.Node) int {
	count := 0
	for _, step := range steps.Content {
		if stepUses(step, attestAction) {
			count++
		}
	}
	return count
}

func hasAttestSBOMStep(steps *yaml.Node) bool {
	for _, step := range steps.Content {
		if !stepUses(step, attestAction) {
			continue
		}
		with := mappingValue(step, "with")
		if scalarValue(mappingValue(with, "sbom-path")) == "dist/sbom.spdx.json" {
			return true
		}
	}
	return false
}

func hasUsesStep(steps *yaml.Node, uses string) bool {
	for _, step := range steps.Content {
		if stepUses(step, uses) {
			return true
		}
	}
	return false
}

func stepUses(step *yaml.Node, uses string) bool {
	return scalarValue(mappingValue(step, "uses")) == uses
}

func mappingValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for index := 0; index+1 < len(node.Content); index += 2 {
		if node.Content[index].Value == key {
			return node.Content[index+1]
		}
	}
	return nil
}

func sequenceValues(node *yaml.Node) []string {
	if node == nil || node.Kind != yaml.SequenceNode {
		return nil
	}
	values := make([]string, 0, len(node.Content))
	for _, child := range node.Content {
		values = append(values, scalarValue(child))
	}
	return values
}

func scalarValue(node *yaml.Node) string {
	if node == nil || node.Kind != yaml.ScalarNode {
		return ""
	}
	return node.Value
}
