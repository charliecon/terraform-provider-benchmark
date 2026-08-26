package benchmark

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// writeDataToFile writes collected timing data to JSON file
func (b *Benchmark) writeDataToFile(data []commandResult) error {
	var dataFilePath = filepath.Join(b.performanceDir, "data.json")
	b.logMessage(LogLevelInfo, "Writing data to %s", dataFilePath)

	jsonData, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %v", err)
	}

	return os.WriteFile(dataFilePath, jsonData, 0644)
}

// createOutputDirectories creates output directories and placeholder files
func (b *Benchmark) createOutputDirectories() error {
	b.logMessage(LogLevelInfo, "🏗️ Creating output directories")
	directories := []string{
		b.logsDir,
		b.performanceDir,
	}

	for _, directory := range directories {
		if err := os.MkdirAll(directory, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", directory, err)
		}
	}

	b.logMessage(LogLevelInfo, "Creating output files")

	// Create placeholder files for all expected log files
	for _, target := range b.Targets {
		// Create or truncate the file
		file, err := os.Create(b.generateLogFilePath(target.outputKey()))
		if err != nil {
			return fmt.Errorf("failed to create log file %s: %w", b.generateLogFilePath(target.outputKey()), err)
		}
		_ = file.Close()
	}

	// Create destroy.log file
	file, err := os.Create(b.destroyLogFilePath)
	if err != nil {
		return fmt.Errorf("failed to create destroy log file: %w", err)
	}
	_ = file.Close()

	// Create data.json file
	file, err = os.Create(b.performanceFilePath)
	if err != nil {
		return fmt.Errorf("failed to create data file: %w", err)
	}
	_ = file.Close()

	// Create init.log file
	file, err = os.Create(b.initLogFilePath)
	if err != nil {
		return fmt.Errorf("failed to create init log file: %w", err)
	}
	_ = file.Close()

	b.logMessage(LogLevelInfo, "🏗️ Output directories and files created")
	return nil
}

type terraformJSON struct {
	Resource map[string]map[string]json.RawMessage `json:"resource"`
}

// exportedResourceCounts reads *.tf.json files in ExportDir and returns resource counts by type.
// Returns nil if ExportDir is unset.
func (b *Benchmark) exportedResourceCounts() (map[string]int, error) {
	if b.ExportDir == "" {
		return nil, nil
	}

	counts, err := countExportedResources(b.ExportDir)
	if err != nil {
		return nil, err
	}

	b.logExportedResources(counts)
	return counts, nil
}

func (b *Benchmark) logExportedResources(counts map[string]int) {
	if len(counts) == 0 {
		b.logTargetStep("No exported resources found in %s", b.ExportDir)
		return
	}

	types := make([]string, 0, len(counts))
	for resourceType := range counts {
		types = append(types, resourceType)
	}
	sort.Strings(types)

	for _, resourceType := range types {
		b.logTargetStep("%d of %s", counts[resourceType], resourceType)
	}
}

func countExportedResources(dir string) (map[string]int, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("export directory %s: %w", dir, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("export path %s is not a directory", dir)
	}

	matches, err := filepath.Glob(filepath.Join(dir, "*.tf.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to list exported terraform json in %s: %w", dir, err)
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("no .tf.json files found in %s", dir)
	}

	counts := make(map[string]int)
	for _, path := range matches {
		fileCounts, err := countResourcesInFile(path)
		if err != nil {
			return nil, err
		}
		for resourceType, count := range fileCounts {
			counts[resourceType] += count
		}
	}

	return counts, nil
}

func countResourcesInFile(path string) (map[string]int, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read exported terraform json %s: %w", path, err)
	}

	var parsed terraformJSON
	if err := json.Unmarshal(content, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse exported terraform json %s: %w", path, err)
	}

	counts := make(map[string]int, len(parsed.Resource))
	for resourceType, instances := range parsed.Resource {
		counts[resourceType] = len(instances)
	}
	return counts, nil
}
