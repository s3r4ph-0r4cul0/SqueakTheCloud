package output

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func WriteResult(filename string, data interface{}) error {
	resultsDir := "./results"

	if err := os.MkdirAll(resultsDir, 0o755); err != nil {
		return fmt.Errorf("failed to create results directory: %w", err)
	}

	jsonPayload, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal result as JSON: %w", err)
	}

	fullPath := filepath.Join(resultsDir, filename)
	if err := os.WriteFile(fullPath, jsonPayload, 0o644); err != nil {
		return fmt.Errorf("failed to write result file %s: %w", fullPath, err)
	}

	return nil
}
