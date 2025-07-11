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
	var dataFilePath = filepath.Join(".", b.OutputDir, "performance", "data.json")
	b.logMessage(LogLevelInfo, "Writing data to %s", dataFilePath)

	// Create the parent directory for the data file
	// Ensure directory exists
	dir := filepath.Dir(dataFilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %v", dir, err)
	}

	jsonData, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %v", err)
	}

	return os.WriteFile(dataFilePath, jsonData, 0644)
}

// testCommitHashes tests different versions of the project by commit hash
func (b *Benchmark) testReferences() error {
	var data []PlanDetails

	// Request confirmation if required
	if err := b.confirmDestructiveOperation(); err != nil {
		return err
	}

	// Iterate through versions, testing each one
	for i, ref := range b.References {
		b.logMessage(LogLevelInfo, "Starting benchmark for reference %s (%d/%d)", ref, i+1, len(b.References))

		if err := b.destroy(); err != nil {
			return fmt.Errorf("destroy failed: %v", err)
		}

		b.logMessage(LogLevelDebug, "Sleeping for 1 second...")
		time.Sleep(1 * time.Second)

		if err := b.makeSideload(ref); err != nil {
			return err
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

// runTerraformCommand executes terraform command and captures output
func (b *Benchmark) runTerraformCommand(reference string) error {
	logFileName := strings.ReplaceAll(reference, ".", "_")
	outputFileName := filepath.Join(".", b.OutputDir, "logs", fmt.Sprintf("%s.log", logFileName))

	// Create the parent directory for the log file
	dir := filepath.Dir(outputFileName)
	b.logMessage(LogLevelDebug, "Creating directory %s", dir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %v", dir, err)
	}

	b.logMessage(LogLevelDebug, "Creating output file %s", outputFileName)
	outputFile, err := os.Create(outputFileName)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer outputFile.Close()

	// Split the command into executable and arguments
	commandParts := strings.Fields(string(b.TfCommand))
	if len(commandParts) == 0 {
		return fmt.Errorf("invalid command: %s", string(b.TfCommand))
	}

	cmd := exec.Command(commandParts[0], commandParts[1:]...)
	cmd.Stdout = outputFile
	cmd.Stderr = outputFile
	cmd.Dir = b.TfConfigDir

	// checking if file exists
	if _, err := os.Stat(b.TerraformRcFilePath); os.IsNotExist(err) {
		return fmt.Errorf("terraformrc file does not exist where we expect it to")
	}

	// Set TF_CLI_CONFIG_FILE to b.TerraformRcFilePath
	b.logMessage(LogLevelDebug, "Setting TF_CLI_CONFIG_FILE to "+b.TerraformRcFilePath)
	env := os.Environ()
	env = append(env, "TF_CLI_CONFIG_FILE="+b.TerraformRcFilePath)
	cmd.Env = env

	b.logMessage(LogLevelInfo, "Running %s for version %s in directory %s", string(b.TfCommand), reference, b.TfConfigDir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("terraform command failed: %w", err)
	}

	b.logMessage(LogLevelInfo, "Successfully completed command: %s", string(b.TfCommand))
	return nil
}

// createDirectories creates output directories if they don't exist
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

	b.logMessage(LogLevelInfo, "Output directories created")
	return nil
}

// makeSideload checks out the specified ref and runs make sideload
func (b *Benchmark) makeSideload(ref string) (err error) {
	const devBranchName = "dev"

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

	// If not dev branch, checkout dev branch
	if ref != devBranchName {
		b.logMessage(LogLevelDebug, "Checking out dev branch after sideload")
		// Checkout dev branch
		cmd = exec.Command("git", "checkout", devBranchName)
		cmd.Dir = b.ProjectPath
		if err = cmd.Run(); err != nil {
			return fmt.Errorf("git checkout %s failed: %w", devBranchName, err)
		}
	}

	return err
}

// destroy runs terraform destroy with optional confirmation
func (b *Benchmark) destroy() error {
	command := []string{"terraform", "destroy", "--auto-approve"}
	b.logMessage(LogLevelInfo, "Running %v in directory %s", command, b.TfConfigDir)

	outputFileName := filepath.Join(b.logsDir, "destroy.log")

	// Ensure directory exists
	dir := filepath.Dir(outputFileName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %v", dir, err)
	}

	outputFile, err := os.Create(outputFileName)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer outputFile.Close()

	cmd := exec.Command(command[0], command[1:]...)
	cmd.Stdout = outputFile
	cmd.Stderr = outputFile
	cmd.Dir = b.TfConfigDir

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("destroy failed: %v", err)
	}

	b.logMessage(LogLevelInfo, "Destroy successful")
	return nil
}

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
func (b *Benchmark) configureOutputDirectories() {
	if b.OutputDir == "" {
		b.OutputDir = "output"
	}
	b.logsDir = filepath.Join(".", b.OutputDir, "logs")
	b.performanceDir = filepath.Join(".", b.OutputDir, "performance")
}

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

func (b *Benchmark) setupConfiguration() error {
	if err := b.validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	b.logMessage(LogLevelInfo, "Configuring benchmark default values")
	b.configureDefaults()
	b.configureOutputDirectories()
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
