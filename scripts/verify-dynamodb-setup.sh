#!/bin/bash

# EntityStore DynamoDB Verification Script
# Usage: ./verify-dynamodb-setup.sh [table-name]

set -e

# Configuration
TABLE_NAME="${1:-entitystore-table}"
REGION="${AWS_REGION:-us-east-1}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}EntityStore DynamoDB Setup Verification${NC}"
echo "========================================"
echo "Table Name: $TABLE_NAME"
echo "Region: $REGION"
echo ""

# Track overall status
OVERALL_STATUS=0

# Function to check status
check_status() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✓${NC} $2"
    else
        echo -e "${RED}✗${NC} $2"
        OVERALL_STATUS=1
    fi
}

# 1. Check AWS CLI
echo -e "${YELLOW}1. Checking Prerequisites${NC}"
if command -v aws &> /dev/null; then
    check_status 0 "AWS CLI installed"
else
    check_status 1 "AWS CLI not installed"
    echo "  Install from: https://aws.amazon.com/cli/"
    exit 1
fi

if command -v jq &> /dev/null; then
    check_status 0 "jq installed"
else
    check_status 1 "jq not installed (required for JSON parsing)"
    echo "  Install: brew install jq (macOS) or apt-get install jq (Linux)"
    exit 1
fi

# 2. Check AWS credentials
echo -e "\n${YELLOW}2. Checking AWS Credentials${NC}"
if AWS_OUTPUT=$(aws sts get-caller-identity 2>&1); then
    check_status 0 "AWS credentials configured"
    ACCOUNT_ID=$(echo "$AWS_OUTPUT" | jq -r '.Account')
    USER_ID=$(echo "$AWS_OUTPUT" | jq -r '.UserId')
    echo "  Account: $ACCOUNT_ID"
    echo "  User: $USER_ID"
else
    check_status 1 "AWS credentials not configured"
    echo "  Configure with: aws configure"
    exit 1
fi

# 3. Check table exists
echo -e "\n${YELLOW}3. Checking Table Existence${NC}"
if TABLE_INFO=$(aws dynamodb describe-table --table-name "$TABLE_NAME" --region "$REGION" 2>&1); then
    check_status 0 "Table '$TABLE_NAME' exists"
    
    # Extract table status
    TABLE_STATUS=$(echo "$TABLE_INFO" | jq -r '.Table.TableStatus')
    if [ "$TABLE_STATUS" = "ACTIVE" ]; then
        check_status 0 "Table status: ACTIVE"
    else
        check_status 1 "Table status: $TABLE_STATUS (expected: ACTIVE)"
    fi
else
    check_status 1 "Table '$TABLE_NAME' does not exist"
    echo "  Create with: just setup-dynamodb $TABLE_NAME"
    exit 1
fi

# 4. Verify table structure
echo -e "\n${YELLOW}4. Verifying Table Structure${NC}"

# Check primary keys
PK_NAME=$(echo "$TABLE_INFO" | jq -r '.Table.KeySchema[] | select(.KeyType=="HASH") | .AttributeName')
SK_NAME=$(echo "$TABLE_INFO" | jq -r '.Table.KeySchema[] | select(.KeyType=="RANGE") | .AttributeName')

if [[ "$PK_NAME" == "PK" && "$SK_NAME" == "SK" ]]; then
    check_status 0 "Primary keys: PK (HASH), SK (RANGE)"
else
    check_status 1 "Primary keys incorrect. Expected: PK/SK, Got: $PK_NAME/$SK_NAME"
fi

# Check GSI1
GSI1_EXISTS=$(echo "$TABLE_INFO" | jq -r '.Table.GlobalSecondaryIndexes[]? | select(.IndexName=="GSI1") | .IndexName' || echo "")
if [ "$GSI1_EXISTS" = "GSI1" ]; then
    check_status 0 "GSI1 exists"
    
    # Check GSI1 keys
    GSI1_PK=$(echo "$TABLE_INFO" | jq -r '.Table.GlobalSecondaryIndexes[] | select(.IndexName=="GSI1") | .KeySchema[] | select(.KeyType=="HASH") | .AttributeName')
    GSI1_SK=$(echo "$TABLE_INFO" | jq -r '.Table.GlobalSecondaryIndexes[] | select(.IndexName=="GSI1") | .KeySchema[] | select(.KeyType=="RANGE") | .AttributeName')
    
    if [[ "$GSI1_PK" == "PK1" && "$GSI1_SK" == "SK1" ]]; then
        check_status 0 "GSI1 keys: PK1 (HASH), SK1 (RANGE)"
    else
        check_status 1 "GSI1 keys incorrect. Expected: PK1/SK1, Got: $GSI1_PK/$GSI1_SK"
    fi
    
    # Check GSI1 status
    GSI1_STATUS=$(echo "$TABLE_INFO" | jq -r '.Table.GlobalSecondaryIndexes[] | select(.IndexName=="GSI1") | .IndexStatus')
    if [ "$GSI1_STATUS" = "ACTIVE" ]; then
        check_status 0 "GSI1 status: ACTIVE"
    else
        check_status 1 "GSI1 status: $GSI1_STATUS (expected: ACTIVE)"
    fi
else
    check_status 1 "GSI1 not found"
fi

# 5. Test IAM permissions
echo -e "\n${YELLOW}5. Testing IAM Permissions${NC}"

# Create test item
TEST_ITEM='{
    "PK": {"S": "TEST#ENTITYSTORE#VERIFY"},
    "SK": {"S": "TEST#ENTITYSTORE#VERIFY"},
    "TestData": {"S": "EntityStore verification test"},
    "PK1": {"S": "TEST#GSI#VERIFY"},
    "SK1": {"S": "TEST#GSI#VERIFY"},
    "Timestamp": {"S": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'"}
}'

# Test PutItem
if aws dynamodb put-item \
    --table-name "$TABLE_NAME" \
    --item "$TEST_ITEM" \
    --region "$REGION" &>/dev/null; then
    check_status 0 "PutItem permission"
else
    check_status 1 "PutItem permission DENIED"
fi

# Test GetItem
if aws dynamodb get-item \
    --table-name "$TABLE_NAME" \
    --key '{"PK": {"S": "TEST#ENTITYSTORE#VERIFY"}, "SK": {"S": "TEST#ENTITYSTORE#VERIFY"}}' \
    --region "$REGION" &>/dev/null; then
    check_status 0 "GetItem permission"
else
    check_status 1 "GetItem permission DENIED"
fi

# Test Query on main table
if aws dynamodb query \
    --table-name "$TABLE_NAME" \
    --key-condition-expression "PK = :pk" \
    --expression-attribute-values '{":pk": {"S": "TEST#ENTITYSTORE#VERIFY"}}' \
    --region "$REGION" &>/dev/null; then
    check_status 0 "Query permission (main table)"
else
    check_status 1 "Query permission DENIED (main table)"
fi

# Test Query on GSI1
if aws dynamodb query \
    --table-name "$TABLE_NAME" \
    --index-name "GSI1" \
    --key-condition-expression "PK1 = :pk" \
    --expression-attribute-values '{":pk": {"S": "TEST#GSI#VERIFY"}}' \
    --region "$REGION" &>/dev/null; then
    check_status 0 "Query permission (GSI1)"
else
    check_status 1 "Query permission DENIED (GSI1)"
fi

# Test UpdateItem
if aws dynamodb update-item \
    --table-name "$TABLE_NAME" \
    --key '{"PK": {"S": "TEST#ENTITYSTORE#VERIFY"}, "SK": {"S": "TEST#ENTITYSTORE#VERIFY"}}' \
    --update-expression "SET TestData = :data" \
    --expression-attribute-values '{":data": {"S": "Updated"}}' \
    --region "$REGION" &>/dev/null; then
    check_status 0 "UpdateItem permission"
else
    check_status 1 "UpdateItem permission DENIED"
fi

# Test DeleteItem (cleanup)
if aws dynamodb delete-item \
    --table-name "$TABLE_NAME" \
    --key '{"PK": {"S": "TEST#ENTITYSTORE#VERIFY"}, "SK": {"S": "TEST#ENTITYSTORE#VERIFY"}}' \
    --region "$REGION" &>/dev/null; then
    check_status 0 "DeleteItem permission"
else
    check_status 1 "DeleteItem permission DENIED"
fi

# 6. EntityStore compatibility check
echo -e "\n${YELLOW}6. EntityStore Compatibility${NC}"

# Check if we can create entities with GSI attributes
ENTITY_TEST='{
    "PK": {"S": "USER#123"},
    "SK": {"S": "PROFILE"},
    "PK1": {"S": "EMAIL#user@example.com"},
    "SK1": {"S": "USER"},
    "EntityType": {"S": "UserProfile"},
    "Email": {"S": "user@example.com"},
    "Name": {"S": "Test User"}
}'

if aws dynamodb put-item \
    --table-name "$TABLE_NAME" \
    --item "$ENTITY_TEST" \
    --region "$REGION" &>/dev/null; then
    check_status 0 "EntityStore entity format compatible"
    
    # Clean up test entity
    aws dynamodb delete-item \
        --table-name "$TABLE_NAME" \
        --key '{"PK": {"S": "USER#123"}, "SK": {"S": "PROFILE"}}' \
        --region "$REGION" &>/dev/null
else
    check_status 1 "Failed to create EntityStore-format entity"
fi

# Summary
echo -e "\n${BLUE}Verification Summary${NC}"
echo "===================="

if [ $OVERALL_STATUS -eq 0 ]; then
    echo -e "${GREEN}✓ All checks passed!${NC}"
    echo -e "\nYour DynamoDB table '$TABLE_NAME' is properly configured for EntityStore."
    echo -e "\nYou can now use this table with EntityStore by configuring:"
    echo -e "  Table Name: $TABLE_NAME"
    echo -e "  Region: $REGION"
else
    echo -e "${RED}✗ Some checks failed!${NC}"
    echo -e "\nPlease fix the issues above before using EntityStore."
    exit 1
fi