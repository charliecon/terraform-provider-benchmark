package benchmark

type command string

const (
	Apply command = "terraform apply --auto-approve"
	Init  command = "terraform init"
	Plan  command = "terraform plan"
)

// LogLevel represents the logging level
type LogLevel int

const (
	LogLevelQuiet LogLevel = iota
	LogLevelInfo
	LogLevelDebug
)

// String returns the string representation of the LogLevel
func (l LogLevel) String() string {
	return []string{"Quiet", "Info", "Debug"}[l]
}

// BenchmarkTarget is a git ref (branch, tag, or commit) to benchmark, with optional
// environment variables applied while that target is active.
type BenchmarkTarget struct {
	// Id can be used to identify the execution in the output e.g. "main_with_env_var" / "main_without_env_var"
	Id string

	// Ref is the git branch, tag, or commit to check out and sideload.
	Ref string

	// Env sets extra environment variables for subprocesses run while benchmarking
	// this target (make sideload and Terraform plan/apply/destroy). TF_CLI_CONFIG_FILE
	// is still set automatically for Terraform commands.
	Env map[string]string

	// Parallelism sets -parallelism on Terraform plan/apply/destroy for this target.
	// Zero uses the Terraform default.
	Parallelism int
}

// Target returns a BenchmarkTarget for ref with no extra environment variables.
func Target(ref string) BenchmarkTarget {
	return BenchmarkTarget{Ref: ref}
}

// outputKey returns Id when set, otherwise Ref. Used for log file names and output identity.
func (t BenchmarkTarget) outputKey() string {
	if t.Id != "" {
		return t.Id
	}
	return t.Ref
}

type Benchmark struct {
	// TfCommand Terraform command to run
	TfCommand command

	// Targets lists git refs to benchmark, each with optional per-target environment.
	Targets []BenchmarkTarget

	// ProjectPath is the absolute path to the locally cloned project
	ProjectPath string

	// SkipDestroyConfirmation controls whether to skip user confirmation for destructive operations
	SkipDestroyConfirmation bool

	// LogLevel controls the verbosity of logging
	LogLevel LogLevel

	// TerraformRcFilePath is the path to the .terraformrc file (Defaults to "./.terraformrc" which is to say we assume it is in the current working directory)
	TerraformRcFilePath string

	// OutputDir is the directory to write the output to (Defaults to "output")
	OutputDir string

	// TfConfigDir is the directory containing the Terraform configuration to run commands against (Defaults to current working directory)
	TfConfigDir string

	// RequireConfirmation controls whether to require user confirmation for destructive operations (Deprecated. Use SkipDestroyConfirmation instead.)
	RequireConfirmation bool

	logsDir             string
	performanceDir      string
	performanceFilePath string
	destroyLogFilePath  string
	initLogFilePath     string
}

// commandResult stores details about each Terraform command execution
type commandResult struct {
	Id       string  `json:"id,omitempty"`
	Version  string  `json:"version"`
	Duration float64 `json:"duration"`
}
