package benchmark

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

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

// testCommitHashes tests different versions of the project by commit hash
func (b *Benchmark) testReferences() error {
	var data []PlanDetails

	if err := b.initialiseTerraform(); err != nil {
		return fmt.Errorf("terraform init failed: %v", err)
	}

	// Iterate through versions, testing each one
	for i, ref := range b.References {
		b.logMessage(LogLevelInfo, "Starting benchmark for reference %s (%d/%d)", ref, i+1, len(b.References))

		if err := b.makeSideload(ref); err != nil {
			return err
		}

		if err := b.destroy(); err != nil {
			return fmt.Errorf("destroy failed: %v", err)
		}

		// Time the execution of terraform command
		b.logMessage(LogLevelInfo, "Running Terraform command for reference %s", ref)
		start := time.Now()
		if err := b.runTerraformCommand(ref); err != nil {
			return err
		}
		end := time.Now()

		duration := end.Sub(start).Seconds()
		b.logMessage(LogLevelInfo, "Completed reference %s in %.2f seconds", ref, duration)

		// Store results
		plan := PlanDetails{
			Version:  ref,
			Duration: duration,
		}
		data = append(data, plan)
	}

	return b.writeDataToFile(data)
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

func (b *Benchmark) initialiseTerraform() error {
	command := []string{"terraform", "init"}
	b.logMessage(LogLevelInfo, "Running %v in directory %s", command, b.TfConfigDir)

	outputFile, err := os.OpenFile(b.initLogFilePath, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open output file: %v", err)
	}
	defer outputFile.Close()

	cmd := b.setupTerraformCommand(command, outputFile, false)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("terraform init failed: %v", err)
	}

	return nil
}

// runTerraformCommand executes terraform command and captures output
func (b *Benchmark) runTerraformCommand(reference string) error {
	outputFileName := b.generateLogFilePath(reference)

	b.logMessage(LogLevelDebug, "Opening output file %s", outputFileName)
	outputFile, err := os.OpenFile(outputFileName, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open output file: %v", err)
	}
	defer outputFile.Close()

	// Split the command into executable and arguments
	commandParts := strings.Fields(string(b.TfCommand))
	if len(commandParts) == 0 {
		return fmt.Errorf("invalid command: %s", string(b.TfCommand))
	}

	cmd := b.setupTerraformCommand(commandParts, outputFile, true)

	b.logMessage(LogLevelInfo, "Running %s for version %s in directory %s", string(b.TfCommand), reference, b.TfConfigDir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("terraform command failed: %w", err)
	}

	b.logMessage(LogLevelInfo, "Successfully completed command: %s", string(b.TfCommand))
	return nil
}

// createOutputDirectories creates output directories and placeholder files
func (b *Benchmark) createOutputDirectories() error {
	b.logMessage(LogLevelInfo, "Creating output directories")
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

	b.logMessage(LogLevelInfo, "Output directories and files created")
	return nil
}

// makeSideload checks out the specified ref and runs make sideload
func (b *Benchmark) makeSideload(ref string) (err error) {
	b.logMessage(LogLevelInfo, "Checking out reference %s in %s", ref, b.ProjectPath)
	// Checkout specific hash
	cmd := exec.Command("git", "checkout", ref)
	cmd.Dir = b.ProjectPath
	if err = cmd.Run(); err != nil {
		return fmt.Errorf("git checkout failed: %w", err)
	}

	b.logMessage(LogLevelInfo, "Running make sideload in %s", b.ProjectPath)
	// Run make sideload
	cmd = exec.Command("make", "sideload")
	cmd.Dir = b.ProjectPath
	if err = cmd.Run(); err != nil {
		return fmt.Errorf("make sideload failed: %w", err)
	}

	return err
}

// destroy runs terraform destroy with optional confirmation
func (b *Benchmark) destroy() error {
	command := []string{"terraform", "destroy", "--auto-approve"}
	b.logMessage(LogLevelInfo, "Running %v in directory %s", command, b.TfConfigDir)

	outputFile, err := os.OpenFile(b.destroyLogFilePath, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open output file: %v", err)
	}
	defer outputFile.Close()

	cmd := b.setupTerraformCommand(command, outputFile, true)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("destroy failed: %v", err)
	}

	b.logMessage(LogLevelInfo, "Destroy successful")
	return nil
}

func (b *Benchmark) Run() error {
	b.logMessage(LogLevelInfo, "Starting benchmark with %d references", len(b.References))

	if err := b.setupConfiguration(); err != nil {
		return fmt.Errorf("pre-config failed: %w", err)
	}

	if err := b.createOutputDirectories(); err != nil {
		return fmt.Errorf("failed to create output directories: %w", err)
	}

	if err := b.confirmDestructiveOperation(); err != nil {
		return fmt.Errorf("failed to confirm destructive operation: %w", err)
	}

	if err := b.testReferences(); err != nil {
		return fmt.Errorf("failed to test commit hashes: %w", err)
	}

	b.logMessage(LogLevelInfo, "Benchmark completed successfully")
	return nil
}
