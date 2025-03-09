#!/bin/bash

# Set default options
FORMAT=false
LINT=false
VET=false
ALL=true
VERBOSE=false

# Print usage information
usage() {
    echo "Usage: $0 [OPTIONS]"
    echo "Options:"
    echo "  -f, --format       Run go fmt to format the code."
    echo "  -l, --lint         Run golint to check for lint issues."
    echo "  -v, --vet          Run go vet to check for potential issues."
    echo "  -a, --all          Run all checks (format, lint, and vet)."
    echo "  -V, --verbose      Enable verbose logging."
    echo "  -h, --help         Show this help message and exit."
}

# Verbose logging function
log() {
    if [ "$VERBOSE" = true ]; then
        echo "$@"
    fi
}

# Function to run go fmt
run_format() {
    log "Running go fmt on the codebase..."
    go fmt ./...
    if [ $? -ne 0 ]; then
        echo "ERROR: go fmt failed."
        exit 1
    fi
    echo "go fmt completed successfully."
}

# Function to run golint
run_lint() {
    log "Running golint to check for lint issues..."
    if ! [ -x "$(command -v golint)" ]; then
        echo "golint is not installed. Installing golint..."
        go install golang.org/x/lint/golint@latest
    fi
    golint ./...
    if [ $? -ne 0 ]; then
        echo "ERROR: golint failed."
        exit 1
    fi
    echo "golint completed successfully."
}

# Function to run go vet
run_vet() {
    log "Running go vet to check for potential issues..."
    go vet ./...
    if [ $? -ne 0 ]; then
        echo "ERROR: go vet failed."
        exit 1
    fi
    echo "go vet completed successfully."
}

# Parse the command-line arguments
while [[ "$#" -gt 0 ]]; do
    case $1 in
        -f|--format) FORMAT=true; ALL=false ;;
        -l|--lint) LINT=true; ALL=false ;;
        -v|--vet) VET=true; ALL=false ;;
        -a|--all) ALL=true ;;
        -V|--verbose) VERBOSE=true ;;
        -h|--help) usage; exit 0 ;;
        *) echo "Unknown parameter passed: $1"; usage; exit 1 ;;
    esac
    shift
done

# Run selected or all checks
if [ "$ALL" = true ]; then
    log "Running all checks: format, lint, and vet..."
    run_format
    run_lint
    run_vet
else
    if [ "$FORMAT" = true ]; then
        run_format
    fi
    if [ "$LINT" = true ]; then
        run_lint
    fi
    if [ "$VET" = true ]; then
        run_vet
    fi
fi

echo "All selected checks completed successfully."
