# terraform-provider-benchmark

## Overview

`terraform-provider-benchmark` is a Go package designed to benchmark the performance of different versions (commits, branches, or tags) of the [terraform-provider-genesyscloud](https://github.com/mypurecloud/terraform-provider-genesyscloud) provider. It automates the process of switching provider versions, running Terraform commands, and collecting timing data for each run.

## Prerequisites

- **Go** installed on your system
- **Terraform** installed and available in your `PATH`
- **git** and **make** installed
- A local clone of [`github.com/mypurecloud/terraform-provider-genesyscloud`](https://github.com/mypurecloud/terraform-provider-genesyscloud)

## Installation

### Adding to Your Go Module

To use this package in your Go project, add it to your module:

```bash
go get github.com/charliecon/terraform-provider-benchmark
```

### Using in Your Code

Import the package in your Go file:

```go
import "github.com/charliecon/terraform-provider-benchmark/benchmark"
```

## Setup

### 1. Clone the Provider Repository

First, clone the terraform-provider-genesyscloud repository to your local machine:

```bash
git clone https://github.com/mypurecloud/terraform-provider-genesyscloud.git
cd terraform-provider-genesyscloud
```

### 2. Configure Terraform

Create a `.terraformrc` file with the following content:

```hcl
provider_installation {
  dev_overrides {
    "mypurecloud/genesyscloud" = "/absolute/path/to/your/terraform-provider-genesyscloud/dist/"
  }
}
```

**Important**: Replace `/absolute/path/to/your/terraform-provider-genesyscloud/dist/` with the actual absolute path of the `dist` folder in your cloned provider repository. The `dist` folder will be created automatically by the `make sideload` process.

### 3. Prepare Your Terraform Configuration

Place your Terraform configuration files (`.tf` files) in a directory that you'll reference in your benchmark configuration.

## Usage

### Basic Example

Create a Go file (e.g., `main.go`) in your working directory:

```go
package main

import (
    "log"
    "path/filepath"
    
    "github.com/charliecon/terraform-provider-benchmark/benchmark"
)

func main() {
    // Define the benchmark configuration
    b := &benchmark.Benchmark{
        TfCommand: benchmark.Plan, // or benchmark.Apply, benchmark.Init, benchmark.Destroy
        References: []string{
            "main",    // branch name
            "v1.66.0", // tag
            "abc1234", // commit hash
        },
        ProjectPath: "/absolute/path/to/your/terraform-provider-genesyscloud",
        SkipDestroyConfirmation: false, // Default: requires confirmation before destructive operations
        LogLevel: benchmark.LogLevelInfo, // Set logging verbosity
        TfConfigDir: "/full/path/to/terraform_config/folder", // Directory containing your Terraform configuration
        TerraformRcFilePath: "/full/path/to/.terraformrc", // Path to your .terraformrc file
    }
    
    // Run the benchmark
    if err := b.Run(); err != nil {
        log.Fatal(err)
    }
}
```

### Configuration Options

#### TfConfigDir (Required)
Specify the directory containing your Terraform configuration files. This is a required field.

```go
b := &benchmark.Benchmark{
    // ... other fields ...
    TfConfigDir: "/full/path/to/your/terraform/config",  // Required: Terraform config directory
}
```

#### TerraformRcFilePath (Required)
Specify the path to your `.terraformrc` file. This is a required field.

```go
b := &benchmark.Benchmark{
    // ... other fields ...
    TerraformRcFilePath: "/full/path/to/.terraformrc",  // Required: terraformrc file location
}
```

#### SkipDestroyConfirmation
Controls whether to skip user confirmation for destructive operations. Defaults to `false` (require confirmation). Set to `true` to skip manual confirmation before running destructive operations like `terraform destroy`.

```go
b := &benchmark.Benchmark{
    // ... other fields ...
    SkipDestroyConfirmation: true,  // Will skip confirmation prompts
}
```

#### LogLevel
Control the verbosity of logging output:

- `benchmark.LogLevelQuiet` - Minimal output
- `benchmark.LogLevelInfo` - Standard informational messages (default)
- `benchmark.LogLevelDebug` - Detailed debug information

```go
b := &benchmark.Benchmark{
    // ... other fields ...
    LogLevel: benchmark.LogLevelDebug,  // Verbose logging
}
```

#### OutputDir
Specify a custom directory for benchmark output files. If not provided, defaults to `output`.

```go
b := &benchmark.Benchmark{
    // ... other fields ...
    OutputDir: "custom-output",  // Custom output directory
}
```

### Available Commands

The benchmark supports the following Terraform commands:

- `benchmark.Plan` - Runs `terraform plan`
- `benchmark.Apply` - Runs `terraform apply --auto-approve`
- `benchmark.Init` - Runs `terraform init`
- `benchmark.Destroy` - Runs `terraform destroy --auto-approve`

### Running the Benchmark

1. Ensure you have configured all required fields in your benchmark configuration
2. Run the benchmark:

```bash
go run main.go
```

## Output

The benchmark will create the following directory structure:

```
.
├── output/
│   ├── performance/
│   │   └── data.json          # Timing results in JSON format
│   └── logs/
│       ├── destroy.log        # Destroy command output
│       ├── init.log          # Terraform init command output
│       ├── main.log          # Log for 'main' reference
│       ├── v1.66.0.log       # Log for 'v1.66.0' reference
│       └── abc1234.log      # Log for 'abc1234' reference
├── terraformrc               # Your Terraform configuration
├── main.tf                   # Your Terraform configuration
└── main.go                   # Your benchmark script
```

### Results Format

The `data.json` file contains timing results in the following format:

```json
[
    {
        "version": "main",
        "duration": 12.345
    },
    {
        "version": "v1.66.0",
        "duration": 11.234
    },
    {
        "version": "abc1234",
        "duration": 13.456
    }
]
```

## How It Works

1. **Setup**: Creates necessary output directories and placeholder files
2. **Initialization**: Runs `terraform init` to initialize the Terraform working directory
3. **Iteration**: For each reference (commit/branch/tag):
   - Checks out the specified reference in the provider repository
   - Runs `make sideload` to build and install the provider
   - Runs `terraform destroy` to clean up any existing state (with optional confirmation)
   - Executes the specified Terraform command and measures execution time
   - Records the results
4. **Output**: Saves timing data to JSON file and logs to individual files

## Safety Features

- **Confirmation Prompts**: By default, the tool will ask for confirmation before running destructive operations. Set `SkipDestroyConfirmation: true` to skip confirmation prompts.
- **Structured Logging**: All operations are logged with appropriate levels for better debugging
- **Progress Tracking**: Shows progress through references being tested

## Notes

- The tool requires a `terraformrc` file path to be specified via `TerraformRcFilePath`
- Each benchmark run will destroy any existing Terraform state before testing (unless cancelled)
- The provider repository will be switched between different references during testing
- All Terraform command output is logged to individual files for debugging
- The benchmark automatically initializes Terraform before running commands. This means you should have a provider block set up in your tf configuration.
- Output files are created fresh on each run to ensure clean results




