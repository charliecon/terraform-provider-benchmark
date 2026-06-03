package benchmark

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func (b *Benchmark) initialiseTerraform() error {
	command := []string{"terraform", "init"}
	b.logMessage(LogLevelInfo, "Running %v in directory %s", command, b.TfConfigDir)

	outputFile, err := os.OpenFile(b.initLogFilePath, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open output file: %v", err)
	}
	defer outputFile.Close()

	cmd := b.setupTerraformCommand(command, outputFile, false, nil)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("terraform init failed: %v", err)
	}

	return nil
}

// runTerraformCommand executes terraform command and captures output
func (b *Benchmark) runTerraformCommand(target BenchmarkTarget) error {
	outputFileName := b.generateLogFilePath(target.Ref)

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

	cmd := b.setupTerraformCommand(commandParts, outputFile, true, target.Env)

	b.logMessage(LogLevelInfo, "⌛️ Running %s for version %s in directory %s", string(b.TfCommand), target.Ref, b.TfConfigDir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("terraform command failed: %w", err)
	}

	b.logMessage(LogLevelInfo, "✅ Successfully completed command: %s", string(b.TfCommand))
	return nil
}

// makeSideload checks out the specified ref and runs make sideload
func (b *Benchmark) makeSideload(target BenchmarkTarget) (err error) {
	b.logMessage(LogLevelInfo, "Checking out %s in %s", target.Ref, b.ProjectPath)
	// Checkout specific hash
	cmd := exec.Command("git", "checkout", target.Ref)
	cmd.Dir = b.ProjectPath
	if err = cmd.Run(); err != nil {
		return fmt.Errorf("git checkout failed: %w", err)
	}

	b.logMessage(LogLevelInfo, "Running make sideload in %s", b.ProjectPath)
	// Run make sideload
	cmd = exec.Command("make", "sideload")
	cmd.Dir = b.ProjectPath
	cmd.Env = b.commandEnv(target.Env, false)
	if err = cmd.Run(); err != nil {
		return fmt.Errorf("make sideload failed: %w", err)
	}

	return err
}

// destroy runs terraform destroy with optional confirmation
func (b *Benchmark) destroy(extraEnv map[string]string) error {
	command := []string{"terraform", "destroy", "--auto-approve"}
	b.logMessage(LogLevelInfo, "🔥 Running %v in directory %s", command, b.TfConfigDir)

	outputFile, err := os.OpenFile(b.destroyLogFilePath, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open output file: %v", err)
	}
	defer outputFile.Close()

	cmd := b.setupTerraformCommand(command, outputFile, true, extraEnv)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("destroy failed: %v", err)
	}

	b.logMessage(LogLevelInfo, "🔥 Destroy successful")
	return nil
}
