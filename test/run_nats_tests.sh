#!/usr/bin/env bash

set -e

echo "=== Running NATS Component Tests ==="
echo

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to run tests and check results
run_test() {
    local test_name=$1
    local test_path=$2

    echo -e "${YELLOW}Running $test_name...${NC}"
    if ginkgo -v $test_path; then
        echo -e "${GREEN}✓ $test_name passed${NC}"
    else
        echo -e "${RED}✗ $test_name failed${NC}"
        exit 1
    fi
    echo
}

# Run integration tests
echo -e "${YELLOW}Running Integration Tests...${NC}"
if go run github.com/onsi/ginkgo/v2/ginkgo -v ./test/integration; then
    echo -e "${GREEN}✓ Integration Tests passed${NC}"
else
    echo -e "${RED}✗ Integration Tests failed${NC}"
    exit 1
fi
echo

# Run benchmarks
echo -e "${YELLOW}Running Performance Benchmarks...${NC}"
if go test -run=^$ -bench=BenchmarkSimple -benchmem ./test/benchmark -benchtime=10x; then
    echo -e "${GREEN}✓ Benchmarks completed${NC}"
else
    echo -e "${RED}✗ Benchmarks failed${NC}"
    exit 1
fi
echo

# Run stress tests (only run a subset for quick validation)
echo -e "${YELLOW}Running Stress Tests (subset)...${NC}"
if go run github.com/onsi/ginkgo/v2/ginkgo -v --focus="should handle 100K messages" --timeout=60s ./stress_tests; then
    echo -e "${GREEN}✓ Stress Tests passed${NC}"
else
    echo -e "${RED}✗ Stress Tests failed${NC}"
    exit 1
fi

echo -e "${GREEN}All tests passed successfully!${NC}"

# Optional: Generate coverage report
if [ "$1" == "--coverage" ]; then
    echo
    echo -e "${YELLOW}Generating coverage report...${NC}"
    go test -coverprofile=coverage.out ./test/...
    go tool cover -html=coverage.out -o coverage.html
    echo -e "${GREEN}Coverage report generated: coverage.html${NC}"
fi
