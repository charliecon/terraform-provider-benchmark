package benchmark

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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

// validate the benchmark configuration
func (b *Benchmark) validate() error {
	b.logMessage(LogLevelInfo, "Validating benchmark configuration")

	if b.RequireConfirmation {
		b.logMessage(LogLevelInfo, "⚠️ RequireConfirmation is deprecated and has no effect. Use SkipDestroyConfirmation instead.")
	}
	if b.TfCommand == "" {
		return errors.New("terraform command is required")
	}
	if len(b.Targets) == 0 {
		return errors.New("at least one target is required")
	}
	for i, target := range b.Targets {
		if target.Ref == "" {
			return fmt.Errorf("target %d: ref is required", i)
		}
		if target.Parallelism < 0 {
			return fmt.Errorf("target %d: parallelism must be zero or positive", i)
		}
	}
	if b.ProjectPath == "" {
		return errors.New("project path is required")
	}
	if b.TerraformRcFilePath == "" {
		return errors.New("terraformrc file path is required")
	}
	if _, err := os.Stat(b.TerraformRcFilePath); os.IsNotExist(err) {
		return fmt.Errorf("terraformrc file does not exist at %s", b.TerraformRcFilePath)
	}
	if b.TfConfigDir == "" {
		return errors.New("terraform config directory is required")
	}
	if _, err := os.Stat(b.TfConfigDir); os.IsNotExist(err) {
		return fmt.Errorf("terraform config directory does not exist at %s", b.TfConfigDir)
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

func appendParallelism(args []string, parallelism int) []string {
	if parallelism > 0 {
		return append(args, "-parallelism", strconv.Itoa(parallelism))
	}
	return args
}

func terraformCommandParts(tfCommand command, parallelism int) ([]string, error) {
	parts := strings.Fields(string(tfCommand))
	if len(parts) == 0 {
		return nil, fmt.Errorf("invalid command: %s", string(tfCommand))
	}
	return appendParallelism(parts, parallelism), nil
}

// generateLogFilePath generates the path to the log file for a given reference
func (b *Benchmark) generateLogFilePath(reference string) string {
	filename := strings.ReplaceAll(reference, ".", "_")
	return filepath.Join(b.logsDir, fmt.Sprintf("%s.log", filename))
}

// commandEnv builds the process environment, optionally including per-target variables
// and TF_CLI_CONFIG_FILE for dev-overridden Terraform runs.
func (b *Benchmark) commandEnv(extraEnv map[string]string, useDevOverride bool) []string {
	env := os.Environ()
	for key, value := range extraEnv {
		env = append(env, key+"="+value)
	}
	if !useDevOverride {
		return env
	}

	if _, err := os.Stat(b.TerraformRcFilePath); os.IsNotExist(err) {
		b.logMessage(LogLevelDebug, "terraformrc file does not exist where we expect it to")
	}

	b.logMessage(LogLevelDebug, "Setting TF_CLI_CONFIG_FILE to "+b.TerraformRcFilePath)
	return append(env, "TF_CLI_CONFIG_FILE="+b.TerraformRcFilePath)
}

// terraformEnv merges per-target environment with benchmark-level Terraform settings.
func (b *Benchmark) terraformEnv(extraEnv map[string]string) map[string]string {
	if b.LogLevel < LogLevelDebug {
		return extraEnv
	}

	merged := make(map[string]string, len(extraEnv)+1)
	for key, value := range extraEnv {
		merged[key] = value
	}
	merged["TF_LOG"] = "debug"
	return merged
}

// setupTerraformCommand creates and configures a terraform command with proper environment
func (b *Benchmark) setupTerraformCommand(command []string, outputFile *os.File, useDevOverride bool, extraEnv map[string]string) *exec.Cmd {
	if b.LogLevel >= LogLevelDebug {
		b.logMessage(LogLevelDebug, "Setting TF_LOG=debug for Terraform command")
	}

	cmd := exec.Command(command[0], command[1:]...)
	cmd.Stdout = outputFile
	cmd.Stderr = outputFile
	cmd.Dir = b.TfConfigDir
	cmd.Env = b.commandEnv(b.terraformEnv(extraEnv), useDevOverride)
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

func (b *Benchmark) shouldSkipConfirmationOfDestructiveOperations() bool {
	return b.SkipDestroyConfirmation || b.TfCommand == Plan
}
