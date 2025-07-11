package benchmark

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBenchmark_configureDefaults(t *testing.T) {
	tests := []struct {
		name      string
		benchmark *Benchmark
		expected  *Benchmark
	}{
		{
			name: "all defaults",
			benchmark: &Benchmark{
				References:  []string{"test"},
				ProjectPath: "/test/path",
			},
			expected: &Benchmark{
				References:          []string{"test"},
				ProjectPath:         "/test/path",
				OutputDir:           "output",
				TerraformRcFilePath: "./.terraformrc",
				TfConfigDir:         ".",
			},
		},
		{
			name: "custom values",
			benchmark: &Benchmark{
				References:          []string{"test"},
				ProjectPath:         "/test/path",
				OutputDir:           "custom-output",
				TerraformRcFilePath: "/custom/.terraformrc",
				TfConfigDir:         "/custom/config",
			},
			expected: &Benchmark{
				References:          []string{"test"},
				ProjectPath:         "/test/path",
				OutputDir:           "custom-output",
				TerraformRcFilePath: "/custom/.terraformrc",
				TfConfigDir:         "/custom/config",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.benchmark.configureDefaults()

			if tt.benchmark.OutputDir != tt.expected.OutputDir {
				t.Errorf("OutputDir = %v, want %v", tt.benchmark.OutputDir, tt.expected.OutputDir)
			}
			if tt.benchmark.TerraformRcFilePath != tt.expected.TerraformRcFilePath {
				t.Errorf("TerraformRcFilePath = %v, want %v", tt.benchmark.TerraformRcFilePath, tt.expected.TerraformRcFilePath)
			}
			if tt.benchmark.TfConfigDir != tt.expected.TfConfigDir {
				t.Errorf("TfConfigDir = %v, want %v", tt.benchmark.TfConfigDir, tt.expected.TfConfigDir)
			}
		})
	}
}

func TestBenchmark_configureOutputPaths(t *testing.T) {
	b := &Benchmark{
		OutputDir: "test-output",
	}

	b.configureOutputPaths()

	expectedLogsDir := filepath.Join(".", "test-output", "logs")
	expectedPerformanceDir := filepath.Join(".", "test-output", "performance")
	expectedDestroyLogPath := filepath.Join(expectedLogsDir, destroyLogFileName)
	expectedPerformancePath := filepath.Join(expectedPerformanceDir, performanceDataFileName)
	expectedInitLogPath := filepath.Join(expectedLogsDir, initLogFileName)

	if b.logsDir != expectedLogsDir {
		t.Errorf("logsDir = %v, want %v", b.logsDir, expectedLogsDir)
	}
	if b.performanceDir != expectedPerformanceDir {
		t.Errorf("performanceDir = %v, want %v", b.performanceDir, expectedPerformanceDir)
	}
	if b.destroyLogFilePath != expectedDestroyLogPath {
		t.Errorf("destroyLogFilePath = %v, want %v", b.destroyLogFilePath, expectedDestroyLogPath)
	}
	if b.performanceFilePath != expectedPerformancePath {
		t.Errorf("performanceFilePath = %v, want %v", b.performanceFilePath, expectedPerformancePath)
	}
	if b.initLogFilePath != expectedInitLogPath {
		t.Errorf("initLogFilePath = %v, want %v", b.initLogFilePath, expectedInitLogPath)
	}
}

func TestBenchmark_validate(t *testing.T) {
	tests := []struct {
		name      string
		benchmark *Benchmark
		wantErr   bool
		errMsg    string
	}{
		{
			name: "valid configuration",
			benchmark: &Benchmark{
				TfCommand:   Plan,
				References:  []string{"test"},
				ProjectPath: "/test/path",
			},
			wantErr: false,
		},
		{
			name: "missing terraform command",
			benchmark: &Benchmark{
				References:  []string{"test"},
				ProjectPath: "/test/path",
			},
			wantErr: true,
			errMsg:  "terraform command is required",
		},
		{
			name: "missing references",
			benchmark: &Benchmark{
				TfCommand:   Plan,
				ProjectPath: "/test/path",
			},
			wantErr: true,
			errMsg:  "at least one reference is required",
		},
		{
			name: "missing project path",
			benchmark: &Benchmark{
				TfCommand:  Plan,
				References: []string{"test"},
			},
			wantErr: true,
			errMsg:  "project path is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.benchmark.validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("validate() error message = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestBenchmark_generateLogFilePath(t *testing.T) {
	b := &Benchmark{
		logsDir: "/test/logs",
	}

	tests := []struct {
		reference string
		expected  string
	}{
		{
			reference: "v1.66.0",
			expected:  filepath.Join("/test/logs", "v1_66_0.log"),
		},
		{
			reference: "main",
			expected:  filepath.Join("/test/logs", "main.log"),
		},
		{
			reference: "abc1234",
			expected:  filepath.Join("/test/logs", "abc1234.log"),
		},
		{
			reference: "feature.branch",
			expected:  filepath.Join("/test/logs", "feature_branch.log"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.reference, func(t *testing.T) {
			result := b.generateLogFilePath(tt.reference)
			if result != tt.expected {
				t.Errorf("generateLogFilePath(%s) = %v, want %v", tt.reference, result, tt.expected)
			}
		})
	}
}

func TestBenchmark_writeDataToFile(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "benchmark_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	b := &Benchmark{
		performanceDir: tempDir,
	}

	testData := []PlanDetails{
		{Version: "v1.0.0", Duration: 10.5},
		{Version: "v1.1.0", Duration: 9.8},
		{Version: "main", Duration: 11.2},
	}

	err = b.writeDataToFile(testData)
	if err != nil {
		t.Fatalf("writeDataToFile() error = %v", err)
	}

	// Verify file was created
	dataFilePath := filepath.Join(tempDir, "data.json")
	if _, err := os.Stat(dataFilePath); os.IsNotExist(err) {
		t.Fatalf("data.json file was not created")
	}

	// Read and verify content
	content, err := os.ReadFile(dataFilePath)
	if err != nil {
		t.Fatalf("Failed to read data.json: %v", err)
	}

	var result []PlanDetails
	err = json.Unmarshal(content, &result)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if len(result) != len(testData) {
		t.Errorf("Expected %d records, got %d", len(testData), len(result))
	}

	for i, expected := range testData {
		if result[i].Version != expected.Version {
			t.Errorf("Record %d: Version = %v, want %v", i, result[i].Version, expected.Version)
		}
		if result[i].Duration != expected.Duration {
			t.Errorf("Record %d: Duration = %v, want %v", i, result[i].Duration, expected.Duration)
		}
	}
}

func TestBenchmark_createOutputDirectories(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "benchmark_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	b := &Benchmark{
		OutputDir:  "test-output",
		References: []string{"v1.0.0", "main", "feature.branch"},
	}
	b.configureOutputPaths()

	// Change to temp directory for test
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	err = b.createOutputDirectories()
	if err != nil {
		t.Fatalf("createOutputDirectories() error = %v", err)
	}

	// Verify directories were created
	expectedDirs := []string{
		filepath.Join("test-output", "logs"),
		filepath.Join("test-output", "performance"),
	}

	for _, dir := range expectedDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("Directory %s was not created", dir)
		}
	}

	// Verify log files were created
	expectedFiles := []string{
		filepath.Join("test-output", "logs", "v1_0_0.log"),
		filepath.Join("test-output", "logs", "main.log"),
		filepath.Join("test-output", "logs", "feature_branch.log"),
		filepath.Join("test-output", "logs", "destroy.log"),
		filepath.Join("test-output", "logs", "init.log"),
		filepath.Join("test-output", "performance", "data.json"),
	}

	for _, file := range expectedFiles {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			t.Errorf("File %s was not created", file)
		}
	}
}

func TestBenchmark_setupTerraformCommand(t *testing.T) {
	// Create temporary terraformrc file
	tempDir, err := os.MkdirTemp("", "benchmark_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	terraformrcPath := filepath.Join(tempDir, ".terraformrc")
	err = os.WriteFile(terraformrcPath, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create terraformrc file: %v", err)
	}

	b := &Benchmark{
		TerraformRcFilePath: terraformrcPath,
		TfConfigDir:         tempDir,
	}

	// Test with dev override
	outputFile, err := os.CreateTemp("", "test_output")
	if err != nil {
		t.Fatalf("Failed to create temp output file: %v", err)
	}
	defer os.Remove(outputFile.Name())
	defer outputFile.Close()

	cmd := b.setupTerraformCommand([]string{"terraform", "plan"}, outputFile, true)

	if cmd.Dir != tempDir {
		t.Errorf("Command directory = %v, want %v", cmd.Dir, tempDir)
	}

	// Check if TF_CLI_CONFIG_FILE is set in environment
	found := false
	for _, env := range cmd.Env {
		if strings.HasPrefix(env, "TF_CLI_CONFIG_FILE=") {
			found = true
			break
		}
	}
	if !found {
		t.Error("TF_CLI_CONFIG_FILE environment variable not set")
	}

	// Test without dev override
	cmd = b.setupTerraformCommand([]string{"terraform", "plan"}, outputFile, false)

	// Should not have TF_CLI_CONFIG_FILE set
	for _, env := range cmd.Env {
		if strings.HasPrefix(env, "TF_CLI_CONFIG_FILE=") {
			t.Error("TF_CLI_CONFIG_FILE should not be set when useDevOverride is false")
		}
	}
}

func TestBenchmark_logMessage(t *testing.T) {
	tests := []struct {
		name     string
		logLevel LogLevel
		message  string
		args     []any
	}{
		{
			name:     "info level message",
			logLevel: LogLevelInfo,
			message:  "Test info message",
		},
		{
			name:     "debug level message",
			logLevel: LogLevelDebug,
			message:  "Test debug message",
		},
		{
			name:     "message with args",
			logLevel: LogLevelInfo,
			message:  "Test message with %s and %d",
			args:     []any{"string", 42},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Benchmark{
				LogLevel: LogLevelInfo,
			}

			// This should not panic
			b.logMessage(tt.logLevel, tt.message, tt.args...)
		})
	}
}

func TestPlanDetails_JSON(t *testing.T) {
	plan := PlanDetails{
		Version:  "v1.0.0",
		Duration: 10.5,
	}

	data, err := json.Marshal(plan)
	if err != nil {
		t.Fatalf("Failed to marshal PlanDetails: %v", err)
	}

	var result PlanDetails
	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("Failed to unmarshal PlanDetails: %v", err)
	}

	if result.Version != plan.Version {
		t.Errorf("Version = %v, want %v", result.Version, plan.Version)
	}
	if result.Duration != plan.Duration {
		t.Errorf("Duration = %v, want %v", result.Duration, plan.Duration)
	}
}

func TestCommand_String(t *testing.T) {
	tests := []struct {
		command  command
		expected string
	}{
		{Apply, "terraform apply --auto-approve"},
		{Destroy, "terraform destroy --auto-approve"},
		{Init, "terraform init"},
		{Plan, "terraform plan"},
	}

	for _, tt := range tests {
		t.Run(string(tt.command), func(t *testing.T) {
			result := string(tt.command)
			if result != tt.expected {
				t.Errorf("command.String() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestLogLevel_String(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{LogLevelQuiet, "LogLevelQuiet"},
		{LogLevelInfo, "LogLevelInfo"},
		{LogLevelDebug, "LogLevelDebug"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			_ = tt.level.String()
		})
	}
}

// Test helper function to create a temporary benchmark configuration
func createTestBenchmark() *Benchmark {
	return &Benchmark{
		TfCommand:           Plan,
		References:          []string{"v1.0.0", "main"},
		ProjectPath:         "/test/project/path",
		RequireConfirmation: false,
		LogLevel:            LogLevelInfo,
		OutputDir:           "test-output",
		TerraformRcFilePath: "./.terraformrc",
		TfConfigDir:         ".",
	}
}

func TestBenchmark_Integration(t *testing.T) {
	// This is a basic integration test that verifies the benchmark can be created
	// and configured without errors. In a real scenario, you'd want to mock
	// the external dependencies (git, terraform, make commands)

	b := createTestBenchmark()

	// Test setup configuration
	err := b.setupConfiguration()
	if err != nil {
		t.Fatalf("setupConfiguration() failed: %v", err)
	}

	// Verify configuration was set up correctly
	if b.logsDir == "" {
		t.Error("logsDir was not configured")
	}
	if b.performanceDir == "" {
		t.Error("performanceDir was not configured")
	}
	if b.destroyLogFilePath == "" {
		t.Error("destroyLogFilePath was not configured")
	}
	if b.performanceFilePath == "" {
		t.Error("performanceFilePath was not configured")
	}
	if b.initLogFilePath == "" {
		t.Error("initLogFilePath was not configured")
	}
}

// Benchmark tests for performance
func BenchmarkPlanDetails_Marshal(b *testing.B) {
	plan := PlanDetails{
		Version:  "v1.0.0",
		Duration: 10.5,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(plan)
		if err != nil {
			b.Fatalf("Failed to marshal: %v", err)
		}
	}
}

func BenchmarkBenchmark_generateLogFilePath(b *testing.B) {
	benchmark := &Benchmark{
		logsDir: "/test/logs",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchmark.generateLogFilePath("v1.66.0")
	}
}
