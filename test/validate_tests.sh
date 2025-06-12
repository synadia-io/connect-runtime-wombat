#!/bin/bash

set -e

echo "=== Validating All Test Commands ==="
echo

# Test benchmark command
echo "Testing: task test:benchmark"
if task test:benchmark > /dev/null 2>&1; then
    echo "✓ test:benchmark works"
else
    echo "✗ test:benchmark failed"
    exit 1
fi

# Test integration command
echo "Testing: task test:integration"
if task test:integration > /dev/null 2>&1; then
    echo "✓ test:integration works"
else
    echo "✗ test:integration failed"
    exit 1
fi

# Test stress command (just compile check)
echo "Testing: task test:stress (compile only)"
if go run github.com/onsi/ginkgo/v2/ginkgo --dry-run ./test/stress > /dev/null 2>&1; then
    echo "✓ test:stress compiles"
else
    echo "✗ test:stress failed to compile"
    exit 1
fi

echo
echo "All test commands are working correctly!"