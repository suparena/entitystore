# EntityStore Prerequisites

## System Requirements

### Required Software

1. **Go Programming Language**
   - Version: 1.21 or higher
   - Download: https://golang.org/dl/
   - Verify: `go version`

2. **AWS CLI**
   - Version: 2.x recommended
   - Install: https://aws.amazon.com/cli/
   - Verify: `aws --version`

3. **Just (Command Runner)**
   - Install:
     ```bash
     # macOS
     brew install just
     
     # Linux
     curl --proto '=https' --tlsv1.2 -sSf https://just.systems/install.sh | bash -s -- --to /usr/local/bin
     
     # Windows (via Scoop)
     scoop install just
     ```
   - Verify: `just --version`

4. **jq (JSON processor)**
   - Install:
     ```bash
     # macOS
     brew install jq
     
     # Linux
     apt-get install jq  # Debian/Ubuntu
     yum install jq      # CentOS/RHEL
     
     # Windows
     choco install jq
     ```
   - Verify: `jq --version`

### Optional Tools

1. **Docker** (for local DynamoDB)
   - Install: https://docs.docker.com/get-docker/
   - Used for: Running DynamoDB locally for testing

2. **GitHub CLI** (for releases)
   - Install: https://cli.github.com/
   - Used for: Creating GitHub releases

3. **golangci-lint** (for code quality)
   - Install: `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`
   - Used for: Code linting

## AWS Requirements

### AWS Account
- Active AWS account with billing enabled
- Access to DynamoDB service

### IAM Permissions
Your AWS user/role needs the following DynamoDB permissions:

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
                "arn:aws:dynamodb:*:*:table/${TableName}",
                "arn:aws:dynamodb:*:*:table/${TableName}/index/*"
            ]
        }
    ]
}
```

### AWS Credentials Configuration
Configure AWS credentials using one of these methods:

1. **AWS CLI Configuration**
   ```bash
   aws configure
   ```

2. **Environment Variables**
   ```bash
   export AWS_ACCESS_KEY_ID=your-access-key
   export AWS_SECRET_ACCESS_KEY=your-secret-key
   export AWS_REGION=us-east-1
   ```

3. **AWS Credentials File**
   ```ini
   # ~/.aws/credentials
   [default]
   aws_access_key_id = your-access-key
   aws_secret_access_key = your-secret-key
   
   # ~/.aws/config
   [default]
   region = us-east-1
   ```

## DynamoDB Table Requirements

### Table Schema
EntityStore requires a specific DynamoDB table structure:

| Attribute | Type | Description |
|-----------|------|-------------|
| PK | String (S) | Primary partition key |
| SK | String (S) | Primary sort key |
| PK1 | String (S) | GSI1 partition key |
| SK1 | String (S) | GSI1 sort key |

### Global Secondary Index (GSI)
- **Index Name**: GSI1
- **Partition Key**: PK1 (String)
- **Sort Key**: SK1 (String)
- **Projection Type**: ALL

### Capacity Settings
- **Provisioned Mode**:
  - Read Capacity: 5 RCU (minimum)
  - Write Capacity: 5 WCU (minimum)
- **On-Demand Mode**: Supported (pay-per-request)

## Quick Verification

Run this command to verify all prerequisites:

```bash
# Check all required tools
for cmd in go aws just jq; do
    if command -v $cmd &> /dev/null; then
        echo "✓ $cmd is installed"
    else
        echo "✗ $cmd is NOT installed"
    fi
done

# Check Go version
go_version=$(go version | awk '{print $3}' | sed 's/go//')
echo "Go version: $go_version (need 1.21+)"

# Check AWS credentials
if aws sts get-caller-identity &> /dev/null; then
    echo "✓ AWS credentials configured"
else
    echo "✗ AWS credentials NOT configured"
fi
```

## Common Issues

### macOS Specific
- Ensure Xcode Command Line Tools are installed: `xcode-select --install`
- If using Homebrew, keep it updated: `brew update && brew upgrade`

### Linux Specific
- May need to add Go to PATH: `export PATH=$PATH:/usr/local/go/bin`
- Ensure you have build-essential: `apt-get install build-essential`

### Windows Specific
- Use Git Bash or WSL2 for better compatibility
- Ensure Go and tools are in System PATH

## Next Steps

Once all prerequisites are met:
1. Clone the EntityStore repository
2. Run `just setup` to install dependencies
3. Run `just verify-dynamodb` to check DynamoDB setup
4. Start developing with EntityStore!