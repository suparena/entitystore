# EntityStore Setup Guide

## Prerequisites

Before using EntityStore, ensure you have the following:

### 1. Go Environment
- Go 1.21 or higher
- Properly configured GOPATH and GOROOT

### 2. AWS Account and Credentials
- AWS account with DynamoDB access
- AWS credentials configured (via environment variables or AWS credentials file)
- IAM permissions for DynamoDB operations

### 3. DynamoDB Table Setup

EntityStore requires a DynamoDB table with specific structure:

#### Table Schema
```
Table Name: your-table-name
Partition Key: PK (String)
Sort Key: SK (String)

Global Secondary Index 1 (GSI1):
  Name: GSI1
  Partition Key: PK1 (String)
  Sort Key: SK1 (String)
  Projection: ALL

Optional GSIs:
GSI2: PK2/SK2
GSI3: PK3/SK3
```

## Installation

### 1. Install EntityStore Library

```bash
go get github.com/suparena/entitystore@latest
```

### 2. Install Development Tools

```bash
# Install Just (command runner)
# macOS
brew install just

# Linux
curl --proto '=https' --tlsv1.2 -sSf https://just.systems/install.sh | bash -s -- --to /usr/local/bin

# Install other tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

## DynamoDB Table Setup Script

Save this script as `setup-dynamodb-table.sh`:

```bash
#!/bin/bash

# Configuration
TABLE_NAME="${1:-entitystore-table}"
REGION="${AWS_REGION:-us-east-1}"
READ_CAPACITY="${READ_CAPACITY:-5}"
WRITE_CAPACITY="${WRITE_CAPACITY:-5}"

echo "Setting up DynamoDB table: $TABLE_NAME in region: $REGION"

# Check if table exists
if aws dynamodb describe-table --table-name "$TABLE_NAME" --region "$REGION" 2>/dev/null; then
    echo "Table $TABLE_NAME already exists"
    exit 0
fi

# Create table with GSI1
aws dynamodb create-table \
    --table-name "$TABLE_NAME" \
    --attribute-definitions \
        AttributeName=PK,AttributeType=S \
        AttributeName=SK,AttributeType=S \
        AttributeName=PK1,AttributeType=S \
        AttributeName=SK1,AttributeType=S \
    --key-schema \
        AttributeName=PK,KeyType=HASH \
        AttributeName=SK,KeyType=RANGE \
    --global-secondary-indexes \
        "[
            {
                \"IndexName\": \"GSI1\",
                \"Keys\": [
                    {\"AttributeName\":\"PK1\",\"KeyType\":\"HASH\"},
                    {\"AttributeName\":\"SK1\",\"KeyType\":\"RANGE\"}
                ],
                \"Projection\": {\"ProjectionType\":\"ALL\"},
                \"ProvisionedThroughput\": {
                    \"ReadCapacityUnits\": $READ_CAPACITY,
                    \"WriteCapacityUnits\": $WRITE_CAPACITY
                }
            }
        ]" \
    --provisioned-throughput \
        ReadCapacityUnits=$READ_CAPACITY,WriteCapacityUnits=$WRITE_CAPACITY \
    --region "$REGION"

echo "Waiting for table to be active..."
aws dynamodb wait table-exists --table-name "$TABLE_NAME" --region "$REGION"

echo "Table $TABLE_NAME created successfully!"

# Verify table structure
echo -e "\nTable structure:"
aws dynamodb describe-table --table-name "$TABLE_NAME" --region "$REGION" \
    --query 'Table.{TableName:TableName,Keys:KeySchema,GSIs:GlobalSecondaryIndexes[*].{IndexName:IndexName,Keys:KeySchema}}'
```

## Verification Script

Save this script as `verify-dynamodb-setup.sh`:

```bash
#!/bin/bash

TABLE_NAME="${1:-entitystore-table}"
REGION="${AWS_REGION:-us-east-1}"

echo "Verifying DynamoDB setup for table: $TABLE_NAME"
echo "Region: $REGION"
echo "================================================"

# Check AWS credentials
echo -e "\n1. Checking AWS credentials..."
if aws sts get-caller-identity >/dev/null 2>&1; then
    echo "✓ AWS credentials are configured"
    aws sts get-caller-identity --query '{Account:Account,UserId:UserId}'
else
    echo "✗ AWS credentials not configured properly"
    exit 1
fi

# Check table exists
echo -e "\n2. Checking if table exists..."
if aws dynamodb describe-table --table-name "$TABLE_NAME" --region "$REGION" >/dev/null 2>&1; then
    echo "✓ Table $TABLE_NAME exists"
else
    echo "✗ Table $TABLE_NAME does not exist"
    echo "  Run: ./setup-dynamodb-table.sh $TABLE_NAME"
    exit 1
fi

# Verify table structure
echo -e "\n3. Verifying table structure..."
TABLE_INFO=$(aws dynamodb describe-table --table-name "$TABLE_NAME" --region "$REGION" 2>/dev/null)

# Check primary keys
PK_NAME=$(echo "$TABLE_INFO" | jq -r '.Table.KeySchema[] | select(.KeyType=="HASH") | .AttributeName')
SK_NAME=$(echo "$TABLE_INFO" | jq -r '.Table.KeySchema[] | select(.KeyType=="RANGE") | .AttributeName')

if [[ "$PK_NAME" == "PK" && "$SK_NAME" == "SK" ]]; then
    echo "✓ Primary keys correct: PK (HASH), SK (RANGE)"
else
    echo "✗ Primary keys incorrect. Expected: PK/SK, Got: $PK_NAME/$SK_NAME"
    exit 1
fi

# Check GSI1
GSI1_INFO=$(echo "$TABLE_INFO" | jq -r '.Table.GlobalSecondaryIndexes[] | select(.IndexName=="GSI1")')
if [[ -n "$GSI1_INFO" ]]; then
    GSI1_PK=$(echo "$GSI1_INFO" | jq -r '.KeySchema[] | select(.KeyType=="HASH") | .AttributeName')
    GSI1_SK=$(echo "$GSI1_INFO" | jq -r '.KeySchema[] | select(.KeyType=="RANGE") | .AttributeName')
    
    if [[ "$GSI1_PK" == "PK1" && "$GSI1_SK" == "SK1" ]]; then
        echo "✓ GSI1 keys correct: PK1 (HASH), SK1 (RANGE)"
    else
        echo "✗ GSI1 keys incorrect. Expected: PK1/SK1, Got: $GSI1_PK/$GSI1_SK"
        exit 1
    fi
else
    echo "✗ GSI1 not found"
    exit 1
fi

# Check table status
TABLE_STATUS=$(echo "$TABLE_INFO" | jq -r '.Table.TableStatus')
if [[ "$TABLE_STATUS" == "ACTIVE" ]]; then
    echo "✓ Table status: ACTIVE"
else
    echo "⚠ Table status: $TABLE_STATUS"
fi

# Test IAM permissions
echo -e "\n4. Testing IAM permissions..."
TEST_ITEM='{
    "PK": {"S": "TEST#PERMISSION"},
    "SK": {"S": "TEST#PERMISSION"},
    "TestData": {"S": "Permission test"},
    "PK1": {"S": "TEST#GSI"},
    "SK1": {"S": "TEST#GSI"}
}'

# Try to put an item
if aws dynamodb put-item \
    --table-name "$TABLE_NAME" \
    --item "$TEST_ITEM" \
    --region "$REGION" >/dev/null 2>&1; then
    echo "✓ PutItem permission: OK"
    
    # Try to get the item
    if aws dynamodb get-item \
        --table-name "$TABLE_NAME" \
        --key '{"PK": {"S": "TEST#PERMISSION"}, "SK": {"S": "TEST#PERMISSION"}}' \
        --region "$REGION" >/dev/null 2>&1; then
        echo "✓ GetItem permission: OK"
    else
        echo "✗ GetItem permission: FAILED"
    fi
    
    # Try to query
    if aws dynamodb query \
        --table-name "$TABLE_NAME" \
        --key-condition-expression "PK = :pk" \
        --expression-attribute-values '{":pk": {"S": "TEST#PERMISSION"}}' \
        --region "$REGION" >/dev/null 2>&1; then
        echo "✓ Query permission: OK"
    else
        echo "✗ Query permission: FAILED"
    fi
    
    # Try to query GSI
    if aws dynamodb query \
        --table-name "$TABLE_NAME" \
        --index-name "GSI1" \
        --key-condition-expression "PK1 = :pk" \
        --expression-attribute-values '{":pk": {"S": "TEST#GSI"}}' \
        --region "$REGION" >/dev/null 2>&1; then
        echo "✓ Query GSI permission: OK"
    else
        echo "✗ Query GSI permission: FAILED"
    fi
    
    # Clean up test item
    aws dynamodb delete-item \
        --table-name "$TABLE_NAME" \
        --key '{"PK": {"S": "TEST#PERMISSION"}, "SK": {"S": "TEST#PERMISSION"}}' \
        --region "$REGION" >/dev/null 2>&1
    echo "✓ DeleteItem permission: OK"
else
    echo "✗ PutItem permission: FAILED"
    echo "  Check your IAM permissions for DynamoDB"
fi

echo -e "\n================================================"
echo "Verification complete!"
```

## Just Build System

Create a `Justfile` in your project root:

```just
# EntityStore Justfile
# Run 'just' to see available commands

# Default command - show help
default:
    @just --list

# Development environment setup
setup:
    @echo "Setting up development environment..."
    go mod download
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
    @echo "Setup complete!"

# Build the project
build:
    go build ./...

# Build the indexmap preprocessor
build-indexmap:
    go build -o indexmap-pps ./cmd/indexmap

# Run all tests
test:
    go test ./...

# Run tests with coverage
test-coverage:
    go test -v -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out -o coverage.html
    @echo "Coverage report generated: coverage.html"

# Run integration tests (requires AWS credentials)
test-integration:
    AWS_REGION=${AWS_REGION:-us-east-1} go test -tags=integration ./...

# Run specific test pattern
test-run pattern:
    go test -run {{pattern}} -v ./...

# Lint the code
lint:
    golangci-lint run

# Format code
fmt:
    go fmt ./...
    goimports -w .

# Clean build artifacts
clean:
    rm -rf dist/ coverage.out coverage.html indexmap-pps

# Setup DynamoDB table
setup-dynamodb table_name='entitystore-table':
    @chmod +x scripts/setup-dynamodb-table.sh
    ./scripts/setup-dynamodb-table.sh {{table_name}}

# Verify DynamoDB setup
verify-dynamodb table_name='entitystore-table':
    @chmod +x scripts/verify-dynamodb-setup.sh
    ./scripts/verify-dynamodb-setup.sh {{table_name}}

# Run local DynamoDB for testing
dynamodb-local:
    docker run -d -p 8000:8000 \
        --name dynamodb-local \
        amazon/dynamodb-local \
        -jar DynamoDBLocal.jar -sharedDb -inMemory

# Stop local DynamoDB
dynamodb-local-stop:
    docker stop dynamodb-local && docker rm dynamodb-local

# Generate code from OpenAPI
generate:
    cd openapi && ./generate-server.sh

# Build for all platforms
build-all: clean
    @echo "Building for all platforms..."
    GOOS=linux GOARCH=amd64 go build -o dist/indexmap-pps-linux-amd64 ./cmd/indexmap
    GOOS=linux GOARCH=arm64 go build -o dist/indexmap-pps-linux-arm64 ./cmd/indexmap
    GOOS=darwin GOARCH=amd64 go build -o dist/indexmap-pps-darwin-amd64 ./cmd/indexmap
    GOOS=darwin GOARCH=arm64 go build -o dist/indexmap-pps-darwin-arm64 ./cmd/indexmap
    GOOS=windows GOARCH=amd64 go build -o dist/indexmap-pps-windows-amd64.exe ./cmd/indexmap

# Create a release
release version:
    @echo "Creating release {{version}}..."
    git tag -a v{{version}} -m "Release v{{version}}"
    git push origin v{{version}}
    just build-all
    gh release create v{{version}} dist/* --title "v{{version}}" --generate-notes

# Run benchmarks
bench:
    go test -bench=. -benchmem ./...

# Check for security vulnerabilities
security:
    go list -json -m all | nancy sleuth
    gosec ./...

# Update dependencies
update-deps:
    go get -u ./...
    go mod tidy

# Run a full CI check
ci: fmt lint test security
    @echo "CI checks passed!"

# Start development environment
dev:
    @echo "Starting development environment..."
    just setup
    just verify-dynamodb
    @echo "Ready for development!"
```

## Environment Variables

Create a `.env.example` file:

```bash
# AWS Configuration
AWS_REGION=us-east-1
AWS_ACCESS_KEY_ID=your-access-key
AWS_SECRET_ACCESS_KEY=your-secret-key

# DynamoDB Configuration
DYNAMODB_TABLE_NAME=entitystore-table

# Optional: Local DynamoDB
DYNAMODB_ENDPOINT=http://localhost:8000
```

## Quick Start with Just

```bash
# Setup development environment
just setup

# Setup DynamoDB table
just setup-dynamodb my-table

# Verify setup
just verify-dynamodb my-table

# Run tests
just test

# Build the project
just build

# Run full CI checks
just ci
```

## Troubleshooting

### Common Issues

1. **Table already exists error**
   - The table name is already in use
   - Use a different table name or delete the existing table

2. **Access denied errors**
   - Check IAM permissions
   - Ensure your AWS credentials have DynamoDB full access

3. **GSI key mismatch**
   - Ensure GSI1 uses PK1/SK1 as key attributes
   - Run the verification script to check table structure

4. **Build errors**
   - Ensure Go 1.21+ is installed
   - Run `go mod tidy` to resolve dependencies

### Required IAM Permissions

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "dynamodb:CreateTable",
                "dynamodb:DescribeTable",
                "dynamodb:ListTables",
                "dynamodb:PutItem",
                "dynamodb:GetItem",
                "dynamodb:Query",
                "dynamodb:UpdateItem",
                "dynamodb:DeleteItem",
                "dynamodb:BatchGetItem",
                "dynamodb:BatchWriteItem"
            ],
            "Resource": [
                "arn:aws:dynamodb:*:*:table/entitystore-*",
                "arn:aws:dynamodb:*:*:table/entitystore-*/index/*"
            ]
        }
    ]
}
```

## Next Steps

1. Review the [Quick Reference Guide](entitystore-quick-reference.md)
2. Check the [GSI Optimization Guide](GSI_OPTIMIZATION_GUIDE.md)
3. Explore example implementations in the `examples/` directory