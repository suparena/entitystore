#!/bin/bash

# Script to generate and view test coverage reports

set -e

echo "Running tests with coverage..."

# Clean previous coverage files
rm -f coverage.out coverage.html

# Run tests with coverage
echo "Generating coverage report..."
go test -v -coverprofile=coverage.out -covermode=atomic ./...

# Generate coverage summary
echo ""
echo "Coverage Summary:"
go tool cover -func=coverage.out | grep -E '^total:|^github.com/suparena/entitystore/[^/]+\s' | sort

# Generate HTML report
go tool cover -html=coverage.out -o coverage.html

# Calculate total coverage
TOTAL_COVERAGE=$(go tool cover -func=coverage.out | grep '^total:' | awk '{print $3}')
echo ""
echo "Total Coverage: $TOTAL_COVERAGE"

# Open coverage report in browser (macOS)
if [[ "$OSTYPE" == "darwin"* ]]; then
    echo "Opening coverage report in browser..."
    open coverage.html
elif command -v xdg-open &> /dev/null; then
    # Linux
    xdg-open coverage.html
else
    echo "Coverage report generated: coverage.html"
fi

# Check if coverage meets threshold
THRESHOLD=80
COVERAGE_NUM=$(echo $TOTAL_COVERAGE | sed 's/%//')
if (( $(echo "$COVERAGE_NUM < $THRESHOLD" | bc -l) )); then
    echo ""
    echo "⚠️  Warning: Coverage ($TOTAL_COVERAGE) is below threshold ($THRESHOLD%)"
    exit 1
else
    echo ""
    echo "✅ Coverage ($TOTAL_COVERAGE) meets threshold ($THRESHOLD%)"
fi