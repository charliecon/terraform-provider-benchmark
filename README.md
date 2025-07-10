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

Create a `terraformrc` file in your working directory (where you'll run the benchmark) with the following content:

```hcl
provider_installation {
  dev_overrides {
    "mypurecloud/genesyscloud" = "/absolute/path/to/your/terraform-provider-genesyscloud"
  }
}
```

**Important**: Replace `/absolute/path/to/your/terraform-provider-genesyscloud` with the actual absolute path to your cloned provider repository.

### 3. Prepare Your Terraform Configuration

Place your Terraform configuration files (`.tf` files) in the same directory where you'll run the benchmark. This directory should contain:

- Your Terraform configuration files (e.g., `main.tf`, `variables.tf`, etc.)
- The `terraformrc` file (from step 2)
- The Go file that imports and uses the benchmark package

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
            "main",           // branch name
            "v1.66.0",        // tag
            "abc1234",       // commit hash
        },
        ProjectPath: "/absolute/path/to/your/terraform-provider-genesyscloud",
        RequireConfirmation: true,  // Ask for confirmation before destructive operations
        LogLevel: benchmark.LogLevelInfo,  // Set logging verbosity
    }
    
    // Run the benchmark
    if err := b.Run(); err != nil {
        log.Fatal(err)
    }
}
```

### Configuration Options

#### RequireConfirmation
When set to `true`, the benchmark will prompt for user confirmation before running destructive operations like `terraform destroy`. This helps prevent accidental data loss.

```go
b := &benchmark.Benchmark{
    // ... other fields ...
    RequireConfirmation: true,  // Will prompt before destroy operations
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

### Available Commands

The benchmark supports the following Terraform commands:

- `benchmark.Plan` - Runs `terraform plan`
- `benchmark.Apply` - Runs `terraform apply --auto-approve`
- `benchmark.Init` - Runs `terraform init`
- `benchmark.Destroy` - Runs `terraform destroy --auto-approve`

### Running the Benchmark

1. Ensure you're in the directory containing your Terraform files and the Go file
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

1. **Setup**: Creates necessary output directories
2. **Iteration**: For each reference (commit/branch/tag):
   - Runs `terraform destroy` to clean up any existing state (with optional confirmation)
   - Checks out the specified reference in the provider repository
   - Runs `make sideload` to build and install the provider
   - Executes the specified Terraform command and measures execution time
   - Records the results
3. **Output**: Saves timing data to JSON file and logs to individual files

## Safety Features

- **Confirmation Prompts**: When `RequireConfirmation` is enabled, the tool will ask for confirmation before running destructive operations
- **Structured Logging**: All operations are logged with appropriate levels for better debugging
- **Progress Tracking**: Shows progress through references being tested

## Notes

- The tool expects a `terraformrc` file in the current working directory
- Each benchmark run will destroy any existing Terraform state before testing (unless cancelled)
- The provider repository will be switched between different references during testing
- All Terraform command output is logged to individual files for debugging




