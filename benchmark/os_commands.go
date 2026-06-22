package benchmark

import (
	"fmt"
	"os"
	"os/exec"
)

func (b *Benchmark) initialiseTerraform() error {
	command := []string{"terraform", "init"}
	b.logMessage(LogLevelInfo, "Running %v in directory %s", command, b.TfConfigDir)

	outputFile, err := os.OpenFile(b.initLogFilePath, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open output file: %v", err)
	}
	defer func() { _ = outputFile.Close() }()

	cmd := b.setupTerraformCommand(command, outputFile, false, nil)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("terraform init failed: %v", err)
	}

	return nil
}

// runTerraformCommand executes terraform command and captures output
func (b *Benchmark) runTerraformCommand(target BenchmarkTarget) error {
	outputFileName := b.generateLogFilePath(target.outputKey())

	b.logMessage(LogLevelDebug, "Opening output file %s", outputFileName)
	outputFile, err := os.OpenFile(outputFileName, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open output file: %v", err)
	}
	defer func() { _ = outputFile.Close() }()

	commandParts, err := terraformCommandParts(b.TfCommand, target.Parallelism)
	if err != nil {
		return err
	}

	cmd := b.setupTerraformCommand(commandParts, outputFile, true, target.Env)

	b.logTargetStep("⌛️ Running %s in directory %s", string(b.TfCommand), b.TfConfigDir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("terraform command failed: %w", err)
	}

	b.logTargetStep("✅ Successfully completed command: %s", string(b.TfCommand))
	return nil
}

// makeSideload checks out the specified ref and runs make sideload
func (b *Benchmark) makeSideload(target BenchmarkTarget) (err error) {
	b.logTargetStep("Checking out %s in %s", target.Ref, b.ProjectPath)
	// Checkout specific hash
	cmd := exec.Command("git", "checkout", target.Ref)
	cmd.Dir = b.ProjectPath
	if err = cmd.Run(); err != nil {
		return fmt.Errorf("git checkout failed: %w", err)
	}

	b.logTargetStep("Running make sideload in %s", b.ProjectPath)
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
func (b *Benchmark) destroy(target BenchmarkTarget) error {
	command := appendParallelism([]string{"terraform", "destroy", "--auto-approve"}, target.Parallelism)
	b.logTargetStep("🔥 Running %v in directory %s", command, b.TfConfigDir)

	outputFile, err := os.OpenFile(b.destroyLogFilePath, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open output file: %v", err)
	}
	defer func() { _ = outputFile.Close() }()

	cmd := b.setupTerraformCommand(command, outputFile, true, target.Env)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("destroy failed: %v", err)
	}

	b.logTargetStep("🔥 Destroy successful")
	return nil
}
