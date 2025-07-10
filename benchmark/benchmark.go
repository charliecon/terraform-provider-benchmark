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

// File to store timing data results
var dataFilePath = filepath.Join(".", "output", "performance", "data.json")

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
func writeDataToFile(data []PlanDetails) error {
	fmt.Printf("Writing data to %s\n", dataFilePath)

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

	return writeDataToFile(data)
}

// runTerraformCommand executes terraform command and captures output
func (b *Benchmark) runTerraformCommand(reference string) error {
	logFileName := strings.ReplaceAll(reference, ".", "_")
	outputFileName := filepath.Join(".", "output", "logs", fmt.Sprintf("%s.log", logFileName))

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

	// checking if file exists
	if _, err := os.Stat("./terraformrc"); os.IsNotExist(err) {
		return fmt.Errorf("terraformrc file does not exist where we expect it to")
	}

	// Set TF_CLI_CONFIG_FILE to ./terraformrc
	b.logMessage(LogLevelDebug, "Setting TF_CLI_CONFIG_FILE to ./terraformrc")
	env := os.Environ()
	env = append(env, "TF_CLI_CONFIG_FILE=./terraformrc")
	cmd.Env = env

	b.logMessage(LogLevelInfo, "Running %s for version %s", string(b.TfCommand), reference)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("terraform command failed: %w", err)
	}

	b.logMessage(LogLevelInfo, "Successfully completed command: %s", string(b.TfCommand))
	return nil
}

// createDirectories creates output directories if they don't exist
func createDirectories() error {
	directories := []string{
		"./output/performance/",
		"./output/logs/",
	}

	for _, directory := range directories {
		if err := os.MkdirAll(directory, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %v", directory, err)
		}
		log.Printf("Directory ready: %s", directory)
	}

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
	b.logMessage(LogLevelInfo, "Running %v", command)

	outputFileName := filepath.Join(".", "output", "logs", "destroy.log")

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

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("destroy failed: %v", err)
	}

	b.logMessage(LogLevelInfo, "Destroy successful")
	return nil
}

func (b *Benchmark) Run() error {
	b.logMessage(LogLevelInfo, "Starting benchmark with %d references", len(b.References))

	if err := createDirectories(); err != nil {
		return fmt.Errorf("failed to create directories: %v", err)
	}

	if err := b.testReferences(); err != nil {
		return fmt.Errorf("failed to test commit hashes: %v", err)
	}

	b.logMessage(LogLevelInfo, "Benchmark completed successfully")
	return nil
}
