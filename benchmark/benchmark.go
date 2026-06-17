package benchmark

import (
	"fmt"
	"time"
)

// testCommitHashes tests different versions of the project by commit hash
func (b *Benchmark) testTargets() error {
	var data []commandResult

	if err := b.initialiseTerraform(); err != nil {
		return fmt.Errorf("terraform init failed: %v", err)
	}

	// Iterate through targets, testing each one
	for i, target := range b.Targets {
		b.logMessage(LogLevelInfo, "Starting benchmark for %s (%d/%d)", target.Ref, i+1, len(b.Targets))

		if err := b.makeSideload(target); err != nil {
			return err
		}

		if b.TfCommand != Plan {
			if err := b.destroy(target); err != nil {
				return fmt.Errorf("destroy failed: %v", err)
			}
		}

		// Time the execution of terraform command
		b.logMessage(LogLevelInfo, "Running Terraform command for %s", target.Ref)
		start := time.Now()
		if err := b.runTerraformCommand(target); err != nil {
			return err
		}
		end := time.Now()

		duration := end.Sub(start).Seconds()
		b.logMessage(LogLevelInfo, "Completed %s in %.2f seconds", target.Ref, duration)

		// Store results
		result := commandResult{
			Id:          target.Id,
			Version:     target.Ref,
			Parallelism: target.Parallelism,
			Duration:    duration,
		}
		data = append(data, result)

		if b.SleepBetweenTargets > 0 && i < len(b.Targets)-1 {
			b.logMessage(LogLevelInfo, "Sleeping %s before next target", b.SleepBetweenTargets)
			time.Sleep(b.SleepBetweenTargets)
		}
	}

	return b.writeDataToFile(data)
}

func (b *Benchmark) Run() (err error) {
	b.logMessage(LogLevelInfo, "Starting benchmark with %d targets", len(b.Targets))

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

	if err = b.testTargets(); err != nil {
		return fmt.Errorf("failed to test commit hashes: %w", err)
	}

	b.logMessage(LogLevelInfo, "🎉 Benchmark completed successfully")
	b.logMessage(LogLevelInfo, "📈 All results were written to the %s directory", b.OutputDir)

	return
}
