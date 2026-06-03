package benchmark

import (
	"fmt"
	"time"
)

// testCommitHashes tests different versions of the project by commit hash
func (b *Benchmark) testReferences() error {
	var data []commandResult

	if err := b.initialiseTerraform(); err != nil {
		return fmt.Errorf("terraform init failed: %v", err)
	}

	// Iterate through versions, testing each one
	for i, ref := range b.References {
		b.logMessage(LogLevelInfo, "Starting benchmark for reference %s (%d/%d)", ref, i+1, len(b.References))

		if err := b.makeSideload(ref); err != nil {
			return err
		}

		if b.TfCommand != Plan {
			if err := b.destroy(); err != nil {
				return fmt.Errorf("destroy failed: %v", err)
			}
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
		result := commandResult{
			Version:  ref,
			Duration: duration,
		}
		data = append(data, result)
	}

	return b.writeDataToFile(data)
}

func (b *Benchmark) Run() (err error) {
	b.logMessage(LogLevelInfo, "Starting benchmark with %d references", len(b.References))

	if err = b.setupConfiguration(); err != nil {
		return fmt.Errorf("pre-config failed: %w", err)
	}

	if err = b.createOutputDirectories(); err != nil {
		return fmt.Errorf("failed to create output directories: %w", err)
	}

	if !b.shouldSkipConfirmationOfDestructiveOperations() {
		if err = b.confirmDestructiveOperation(); err != nil {
			return fmt.Errorf("failed to confirm destructive operation: %w", err)
		}
	}

	if err = b.testReferences(); err != nil {
		return fmt.Errorf("failed to test commit hashes: %w", err)
	}

	b.logMessage(LogLevelInfo, "🎉 Benchmark completed successfully")
	b.logMessage(LogLevelInfo, "📈 All results were written to the %s directory", b.OutputDir)

	return
}
