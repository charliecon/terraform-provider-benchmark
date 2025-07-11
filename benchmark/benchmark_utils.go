package benchmark

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
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

// setupTerraformCommand creates and configures a terraform command with proper environment
func (b *Benchmark) setupTerraformCommand(command []string, outputFile *os.File, useDevOverride bool) *exec.Cmd {
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Stdout = outputFile
	cmd.Stderr = outputFile
	cmd.Dir = b.TfConfigDir

	if !useDevOverride {
		return cmd
	}

	// checking if file exists
	if _, err := os.Stat(b.TerraformRcFilePath); os.IsNotExist(err) {
		b.logMessage(LogLevelDebug, "terraformrc file does not exist where we expect it to")
	}

	// Set TF_CLI_CONFIG_FILE to b.TerraformRcFilePath
	b.logMessage(LogLevelDebug, "Setting TF_CLI_CONFIG_FILE to "+b.TerraformRcFilePath)
	env := os.Environ()
	env = append(env, "TF_CLI_CONFIG_FILE="+b.TerraformRcFilePath)
	cmd.Env = env

	return cmd
}

// logMessage provides structured logging based on the benchmark's log level
func (b *Benchmark) logMessage(level LogLevel, format string, args ...interface{}) {
	if b.LogLevel >= level {
		if level == LogLevelDebug {
			log.Printf("[DEBUG] "+format, args...)
		} else {
			log.Printf("[INFO] "+format, args...)
		}
	}
}

// confirmDestructiveOperation prompts the user for confirmation before destructive operations
func (b *Benchmark) confirmDestructiveOperation() error {
	if !b.RequireConfirmation {
		return nil
	}

	fmt.Printf("\n⚠️  WARNING: About to run destructive terraform operation\n")
	fmt.Printf("This will destroy any existing Terraform state.\n")
	fmt.Printf("Are you sure you want to continue? (yes/no): ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read user input: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response != "yes" && response != "y" {
		return fmt.Errorf("operation cancelled by user")
	}

	fmt.Println("✅ Confirmed. Proceeding with operation...")
	return nil
}
