package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/squeak-the-cloud/squeak/banner"
	"github.com/squeak-the-cloud/squeak/output"
	"github.com/squeak-the-cloud/squeak/providers/aws"
	"github.com/squeak-the-cloud/squeak/providers/azure"
	"github.com/squeak-the-cloud/squeak/providers/gcp"
)

func main() {
	provider := flag.String("provider", "", "cloud provider: aws, azure, gcp")
	flag.Parse()

	normalizedProvider := strings.ToLower(strings.TrimSpace(*provider))
	if normalizedProvider == "" {
		output.LogCritical("provider flag is required. Use --provider aws|azure|gcp")
		os.Exit(1)
	}

	if normalizedProvider != "aws" && normalizedProvider != "azure" && normalizedProvider != "gcp" {
		output.LogCritical("invalid provider value. Use --provider aws|azure|gcp")
		os.Exit(1)
	}

	banner.Render(normalizedProvider)
	output.LogInfo(fmt.Sprintf("starting audit for provider: %s", normalizedProvider))

	switch normalizedProvider {
	case "aws":
		aws.Run()
	case "azure":
		azure.Run()
	case "gcp":
		gcp.Run()
	}

	if err := consolidateResults(normalizedProvider); err != nil {
		output.LogCritical(fmt.Sprintf("failed to consolidate results: %v", err))
		os.Exit(1)
	}

	output.LogSuccess("audit completed and results consolidated")
}

func consolidateResults(provider string) error {
	const resultsDir = "./results"

	entries, err := os.ReadDir(resultsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("results directory does not exist: %w", err)
		}

		return fmt.Errorf("failed to read results directory: %w", err)
	}

	jsonFiles := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if strings.HasSuffix(name, ".json") && name != "consolidated_results.json" && strings.HasPrefix(name, provider) {
			jsonFiles = append(jsonFiles, name)
		}
	}

	if len(jsonFiles) == 0 {
		return fmt.Errorf("no JSON result files were generated")
	}

	payload := struct {
		Provider string   `json:"provider"`
		Files    []string `json:"files"`
	}{
		Provider: provider,
		Files:    jsonFiles,
	}

	if err := output.WriteResult("consolidated_results.json", payload); err != nil {
		return fmt.Errorf("failed to write consolidated JSON: %w", err)
	}

	for _, fileName := range jsonFiles {
		output.LogInfo(fmt.Sprintf("result file available: %s", filepath.Join(resultsDir, fileName)))
	}

	return nil
}

