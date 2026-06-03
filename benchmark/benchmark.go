package benchmark

import (
	"fmt"
	"time"
)

// testCommitHashes tests different versions of the project by commit hash
func (b *Benchmark) testTargets() error {
	var data []PlanDetails

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
			if err := b.destroy(target.Env); err != nil {
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
		plan := PlanDetails{
			Version:  target.Ref,
			Duration: duration,
		}
		data = append(data, plan)
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
