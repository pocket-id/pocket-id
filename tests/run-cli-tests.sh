#!/bin/bash

# Pocket-ID CLI Test Runner
# This script runs the comprehensive CLI tests for Pocket-ID

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
TESTS_DIR="$SCRIPT_DIR"
SETUP_DIR="$SCRIPT_DIR/setup"

# Default values
TEST_FILE=""
RUN_MODE="all"  # all, separated, comprehensive, export
VERBOSE=false
CLEANUP=false
DOCKER_COMPOSE_FILE="docker-compose.yml"

# Function to print usage
print_usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Run Pocket-ID CLI tests"
    echo ""
    echo "Options:"
    echo "  -f, --file FILE        Test file to run (default: run all separated tests)"
    echo "                         Options: cli-comprehensive.spec.ts, cli.spec.ts, or any cli-*.spec.ts"
    echo "  -m, --mode MODE        Test mode (default: all)"
    echo "                         Options: all, separated, comprehensive, export"
    echo "  -v, --verbose          Enable verbose output"
    echo "  -c, --cleanup          Clean up test resources before running"
    echo "  --postgres             Use PostgreSQL mode"
    echo "  --s3                   Use S3 mode"
    echo "  -h, --help             Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0                      # Run all separated CLI tests"
    echo "  $0 -m separated         # Run all separated CLI tests"
    echo "  $0 -m comprehensive     # Run comprehensive integration tests"
    echo "  $0 -m export            # Run only export/import tests"
    echo "  $0 -f cli-user-management.spec.ts  # Run specific test file"
    echo "  $0 --postgres           # Run tests in PostgreSQL mode"
    echo "  $0 -v -c                # Run with verbose output and cleanup"
}

# Function to print colored messages
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check prerequisites
check_prerequisites() {
    print_info "Checking prerequisites..."

    # Check if Docker is installed
    if ! command -v docker &> /dev/null; then
        print_error "Docker is not installed. Please install Docker first."
        exit 1
    fi

    # Check if Docker Compose is installed
    if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
        print_error "Docker Compose is not installed. Please install Docker Compose first."
        exit 1
    fi

    # Check if Node.js is installed
    if ! command -v node &> /dev/null; then
        print_error "Node.js is not installed. Please install Node.js first."
        exit 1
    fi

    # Check if pnpm is available (either globally or via npx)
    if ! command -v pnpm &> /dev/null; then
        print_warning "pnpm is not installed globally. Using npx pnpm..."
        # We'll use npx pnpm in the test command
    fi

    # Check if setup directory exists
    if [ ! -d "$SETUP_DIR" ]; then
        print_warning "Setup directory not found: $SETUP_DIR"
        print_warning "Creating setup directory..."
        mkdir -p "$SETUP_DIR"
    fi

    # Check if tests directory exists
    if [ ! -d "$TESTS_DIR" ]; then
        print_error "Tests directory not found: $TESTS_DIR"
        exit 1
    fi

    print_success "All prerequisites are met"
}

# Function to check if Docker containers are running
check_docker_containers() {
    print_info "Checking Docker containers..."

    # Check if pocket-id container is running
    if ! docker ps --format '{{.Names}}' | grep -q "pocket-id"; then
        print_warning "Pocket-ID container is not running. Starting containers..."

        cd "$SETUP_DIR"

        # Determine which compose file to use
        COMPOSE_CMD="docker-compose"
        if command -v docker-compose &> /dev/null; then
            COMPOSE_CMD="docker-compose"
        elif docker compose version &> /dev/null; then
            COMPOSE_CMD="docker compose"
        fi

        if [ "$DOCKER_COMPOSE_FILE" != "docker-compose.yml" ]; then
            $COMPOSE_CMD -f "$DOCKER_COMPOSE_FILE" up -d
        else
            $COMPOSE_CMD up -d
        fi

        # Wait for container to be ready
        print_info "Waiting for Pocket-ID to be ready..."
        sleep 10

        cd - > /dev/null
    else
        print_success "Pocket-ID container is running"
    fi
}

# Function to detect test mode
detect_test_mode() {
    print_info "Detecting test mode..."

    cd "$SETUP_DIR"

    # Check which Docker Compose files are being used
    if docker ps --format '{{.Names}}' | grep -q "postgres"; then
        DOCKER_COMPOSE_FILE="docker-compose-postgres.yml"
        print_info "Detected PostgreSQL mode"
    elif docker ps --format '{{.Names}}' | grep -q "minio"; then
        DOCKER_COMPOSE_FILE="docker-compose-s3.yml"
        print_info "Detected S3 mode"
    else
        DOCKER_COMPOSE_FILE="docker-compose.yml"
        print_info "Detected SQLite mode"
    fi

    cd - > /dev/null
}

# Function to cleanup test resources
cleanup_resources() {
    print_info "Cleaning up test resources..."

    # This would typically involve:
    # 1. Stopping and removing test containers
    # 2. Cleaning up test databases
    # 3. Removing test files

    print_warning "Cleanup functionality not yet implemented"
    print_warning "Manual cleanup may be required between test runs"
}

# Function to run tests
run_tests() {
    print_info "Running CLI tests..."

    cd "$TESTS_DIR"

    # Build test command - use npx pnpm if pnpm is not available globally
    if command -v pnpm &> /dev/null; then
        TEST_CMD="pnpm test"
    else
        TEST_CMD="npx pnpm test"
    fi

    # Determine which test files to run
    if [ -n "$TEST_FILE" ]; then
        # Run specific test file
        TEST_FILES="$TEST_FILE"
    elif [ "$RUN_MODE" == "export" ]; then
        TEST_FILES="cli.spec.ts"
    elif [ "$RUN_MODE" == "comprehensive" ]; then
        TEST_FILES="cli-comprehensive.spec.ts"
    elif [ "$RUN_MODE" == "separated" ] || [ "$RUN_MODE" == "all" ]; then
        # Run all separated CLI test files
        TEST_FILES="cli-user-management.spec.ts cli-oidc-client.spec.ts cli-user-groups.spec.ts cli-api-keys.spec.ts cli-app-config.spec.ts cli-scim.spec.ts cli-one-time-token.spec.ts cli-output-formats.spec.ts cli-error-handling.spec.ts cli-custom-claims.spec.ts cli-setup.spec.ts"
    fi

    if [ -n "$TEST_FILES" ]; then
        TEST_CMD="$TEST_CMD $TEST_FILES"
    fi

    # Add project filter for CLI tests
    TEST_CMD="$TEST_CMD --project=cli"

    # Add verbose flag if requested
    if [ "$VERBOSE" = true ]; then
        TEST_CMD="$TEST_CMD --reporter=verbose"
    fi

    print_info "Test command: $TEST_CMD"
    print_info "Test mode: $RUN_MODE"
    print_info "Test files: $TEST_FILES"
    print_info "Docker Compose file: $DOCKER_COMPOSE_FILE"

    # Export environment variables for tests
    export DOCKER_COMPOSE_FILE

    # Run the tests
    print_info "Starting test execution..."
    echo "========================================"

    if eval "$TEST_CMD"; then
        echo "========================================"
        print_success "All tests passed!"
        return 0
    else
        echo "========================================"
        print_error "Some tests failed"
        return 1
    fi
}

# Function to display test summary
display_summary() {
    print_info "Test Summary"
    print_info "============"
    print_info "Test File: ${TEST_FILE:-all separated tests}"
    print_info "Test Mode: $RUN_MODE"
    print_info "Docker Mode: $(basename "$DOCKER_COMPOSE_FILE" .yml)"
    print_info "Verbose: $VERBOSE"
    print_info "Cleanup: $CLEANUP"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -f|--file)
            TEST_FILE="$2"
            shift 2
            ;;
        -m|--mode)
            RUN_MODE="$2"
            shift 2
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -c|--cleanup)
            CLEANUP=true
            shift
            ;;
        --postgres)
            DOCKER_COMPOSE_FILE="docker-compose-postgres.yml"
            shift
            ;;
        --s3)
            DOCKER_COMPOSE_FILE="docker-compose-s3.yml"
            shift
            ;;
        -h|--help)
            print_usage
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            print_usage
            exit 1
            ;;
    esac
done

# Main execution
main() {
    print_info "Starting Pocket-ID CLI Test Runner"
    print_info "=================================="

    # Check prerequisites
    check_prerequisites

    # Detect test mode if not explicitly set
    if [ "$DOCKER_COMPOSE_FILE" = "docker-compose.yml" ]; then
        detect_test_mode
    fi

    # Check Docker containers
    check_docker_containers

    # Cleanup if requested
    if [ "$CLEANUP" = true ]; then
        cleanup_resources
    fi

    # Display test summary
    display_summary

    # Run tests
    if run_tests; then
        print_success "Test execution completed successfully"
        exit 0
    else
        print_error "Test execution failed"
        exit 1
    fi
}

# Run main function
main
