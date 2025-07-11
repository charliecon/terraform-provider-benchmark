package benchmark

import (
	"fmt"
	"path/filepath"
	"strings"
)

const (
	destroyLogFileName      = "destroy.log"
	performanceDataFileName = "data.json"
	initLogFileName         = "init.log"
)

// configureDefaults sets the default values for the benchmark
func (b *Benchmark) configureDefaults() {
	if b.OutputDir == "" {
		b.OutputDir = "output"
	}
	if b.TerraformRcFilePath == "" {
		b.TerraformRcFilePath = "./.terraformrc"
	}
	if b.TfConfigDir == "" {
		b.TfConfigDir = "."
	}
}

// setDefaults sets the default values for the benchmark
func (b *Benchmark) configureOutputPaths() {
	if b.OutputDir == "" {
		b.OutputDir = "output"
	}
	b.logsDir = filepath.Join(".", b.OutputDir, "logs")
	b.performanceDir = filepath.Join(".", b.OutputDir, "performance")
	b.destroyLogFilePath = filepath.Join(b.logsDir, destroyLogFileName)
	b.performanceFilePath = filepath.Join(b.performanceDir, performanceDataFileName)
	b.initLogFilePath = filepath.Join(b.logsDir, initLogFileName)
}

// validate validates the benchmark configuration
func (b *Benchmark) validate() error {
	b.logMessage(LogLevelInfo, "Validating benchmark configuration")
	if b.TfCommand == "" {
		return fmt.Errorf("terraform command is required")
	}
	if len(b.References) == 0 {
		return fmt.Errorf("at least one reference is required")
	}
	if b.ProjectPath == "" {
		return fmt.Errorf("project path is required")
	}
	return nil
}

// setupConfiguration validates the benchmark configuration and sets the default values
func (b *Benchmark) setupConfiguration() error {
	if err := b.validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	b.logMessage(LogLevelInfo, "Configuring benchmark default values")
	b.configureDefaults()
	b.configureOutputPaths()
	return nil
}

// generateLogFilePath generates the path to the log file for a given reference
func (b *Benchmark) generateLogFilePath(reference string) string {
	filename := strings.ReplaceAll(reference, ".", "_")
	return filepath.Join(b.logsDir, fmt.Sprintf("%s.log", filename))
}
