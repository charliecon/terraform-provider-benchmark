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
}

// PlanDetails stores details about each Terraform plan execution
type PlanDetails struct {
	Version  string  `json:"version"`
	Duration float64 `json:"duration"`
}
