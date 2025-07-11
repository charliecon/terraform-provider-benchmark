package benchmark

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// writeDataToFile writes collected timing data to JSON file
func (b *Benchmark) writeDataToFile(data []PlanDetails) error {
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
	for _, ref := range b.References {
		// Create or truncate the file
		file, err := os.Create(b.generateLogFilePath(ref))
		if err != nil {
			return fmt.Errorf("failed to create log file %s: %w", b.generateLogFilePath(ref), err)
		}
		file.Close()
	}

	// Create destroy.log file
	file, err := os.Create(b.destroyLogFilePath)
	if err != nil {
		return fmt.Errorf("failed to create destroy log file: %w", err)
	}
	file.Close()

	// Create data.json file
	file, err = os.Create(b.performanceFilePath)
	if err != nil {
		return fmt.Errorf("failed to create data file: %w", err)
	}
	file.Close()

	// Create init.log file
	file, err = os.Create(b.initLogFilePath)
	if err != nil {
		return fmt.Errorf("failed to create init log file: %w", err)
	}
	file.Close()

	b.logMessage(LogLevelInfo, "🏗️ Output directories and files created")
	return nil
}
