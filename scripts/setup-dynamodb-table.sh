#!/bin/bash

# EntityStore DynamoDB Table Setup Script
# Usage: ./setup-dynamodb-table.sh [table-name]

set -e

# Configuration
TABLE_NAME="${1:-entitystore-table}"
REGION="${AWS_REGION:-us-east-1}"
READ_CAPACITY="${READ_CAPACITY:-5}"
WRITE_CAPACITY="${WRITE_CAPACITY:-5}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}EntityStore DynamoDB Table Setup${NC}"
echo "=================================="
echo "Table Name: $TABLE_NAME"
echo "Region: $REGION"
echo "Read Capacity: $READ_CAPACITY"
echo "Write Capacity: $WRITE_CAPACITY"
echo ""

# Check AWS CLI is installed
if ! command -v aws &> /dev/null; then
    echo -e "${RED}Error: AWS CLI is not installed${NC}"
    echo "Please install AWS CLI: https://aws.amazon.com/cli/"
    exit 1
fi

# Check AWS credentials
echo -n "Checking AWS credentials... "
if aws sts get-caller-identity &>/dev/null; then
    echo -e "${GREEN}OK${NC}"
else
    echo -e "${RED}FAILED${NC}"
    echo "Please configure AWS credentials"
    exit 1
fi

# Check if table exists
echo -n "Checking if table exists... "
if aws dynamodb describe-table --table-name "$TABLE_NAME" --region "$REGION" &>/dev/null; then
    echo -e "${YELLOW}Table already exists${NC}"
    
    # Show table info
    echo -e "\nExisting table structure:"
    aws dynamodb describe-table --table-name "$TABLE_NAME" --region "$REGION" \
        --query 'Table.{TableName:TableName,Status:TableStatus,Keys:KeySchema,GSIs:GlobalSecondaryIndexes[*].{IndexName:IndexName,Status:IndexStatus,Keys:KeySchema}}' \
        --output table
    
    read -p "Do you want to continue with the existing table? (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 0
    fi
else
    echo -e "${GREEN}Table does not exist${NC}"
    
    # Create table
    echo -e "\nCreating table..."
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
        --region "$REGION" \
        --output json > /dev/null

    echo -e "${GREEN}Table creation initiated${NC}"
    
    # Wait for table to be active
    echo -n "Waiting for table to become active"
    while true; do
        STATUS=$(aws dynamodb describe-table --table-name "$TABLE_NAME" --region "$REGION" \
            --query 'Table.TableStatus' --output text 2>/dev/null || echo "CREATING")
        
        if [ "$STATUS" = "ACTIVE" ]; then
            echo -e " ${GREEN}ACTIVE${NC}"
            break
        fi
        
        echo -n "."
        sleep 2
    done
    
    # Wait for GSI to be active
    echo -n "Waiting for GSI1 to become active"
    while true; do
        GSI_STATUS=$(aws dynamodb describe-table --table-name "$TABLE_NAME" --region "$REGION" \
            --query 'Table.GlobalSecondaryIndexes[?IndexName==`GSI1`].IndexStatus' --output text 2>/dev/null || echo "CREATING")
        
        if [ "$GSI_STATUS" = "ACTIVE" ]; then
            echo -e " ${GREEN}ACTIVE${NC}"
            break
        fi
        
        echo -n "."
        sleep 2
    done
fi

# Final verification
echo -e "\n${GREEN}Setup Complete!${NC}"
echo -e "\nFinal table structure:"
aws dynamodb describe-table --table-name "$TABLE_NAME" --region "$REGION" \
    --query 'Table.{TableName:TableName,Status:TableStatus,ItemCount:ItemCount,Keys:KeySchema,GSIs:GlobalSecondaryIndexes[*].{IndexName:IndexName,Status:IndexStatus,Keys:KeySchema}}' \
    --output table

echo -e "\n${GREEN}Table $TABLE_NAME is ready for use with EntityStore!${NC}"