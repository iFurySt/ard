package cli

import (
	"fmt"

	"github.com/ifuryst/ard/internal/ard"
	"github.com/ifuryst/ard/internal/config"
	"github.com/ifuryst/ard/internal/policy"
)

func evaluatePolicy(root *rootOptions, catalog ard.Catalog) (map[string]string, error) {
	policyFile := config.PolicyFile(root.policyFile)
	if policyFile == "" {
		return nil, nil
	}
	loadedPolicy, err := policy.LoadFile(policyFile)
	if err != nil {
		return nil, fmt.Errorf("load policy: %w", err)
	}
	statuses, _, err := loadedPolicy.EvaluateCatalog(catalog)
	if err != nil {
		return nil, err
	}
	return statuses, nil
}
