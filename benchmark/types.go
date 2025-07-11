package benchmark

type command string

const (
	Apply   command = "terraform apply --auto-approve"
	Destroy command = "terraform destroy --auto-approve"
	Init    command = "terraform init"
	Plan    command = "terraform plan"
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

type Benchmark struct {
	// TfCommand Terraform command to run
	TfCommand command

	// References can be commit hashes, tags, or branches
	References []string

	// ProjectPath is the absolute path to the locally cloned project
	ProjectPath string

	// RequireConfirmation controls whether destructive operations require user confirmation
	RequireConfirmation bool

	// LogLevel controls the verbosity of logging
	LogLevel LogLevel

	// TerraformRcFilePath is the path to the .terraformrc file (Defaults to "./.terraformrc" which is to say we assume it is in the current working directory)
	TerraformRcFilePath string

	// OutputDir is the directory to write the output to (Defaults to "output")
	OutputDir string

	// TfConfigDir is the directory containing the Terraform configuration to run commands against (Defaults to current working directory)
	TfConfigDir string

	logsDir             string
	performanceDir      string
	performanceFilePath string
	destroyLogFilePath  string
	initLogFilePath     string
}

// PlanDetails stores details about each Terraform plan execution
type PlanDetails struct {
	Version  string  `json:"version"`
	Duration float64 `json:"duration"`
}
