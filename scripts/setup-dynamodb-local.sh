#!/bin/bash

# Setup script for DynamoDB Local testing

set -e

echo "Setting up DynamoDB Local for testing..."

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo "Docker is required but not installed. Please install Docker first."
    exit 1
fi

# Stop and remove existing container if it exists
echo "Cleaning up existing DynamoDB Local container..."
docker stop dynamodb-local 2>/dev/null || true
docker rm dynamodb-local 2>/dev/null || true

# Start DynamoDB Local
echo "Starting DynamoDB Local..."
docker run -d \
    --name dynamodb-local \
    -p 8000:8000 \
    amazon/dynamodb-local:latest \
    -jar DynamoDBLocal.jar \
    -inMemory \
    -sharedDb

# Wait for DynamoDB to be ready
echo "Waiting for DynamoDB Local to be ready..."
for i in {1..30}; do
    if curl -s http://localhost:8000 > /dev/null; then
        echo "DynamoDB Local is ready!"
        break
    fi
    echo -n "."
    sleep 1
done

# Create test table
echo "Creating test table..."
aws dynamodb create-table \
    --table-name entitystore-test \
    --attribute-definitions \
        AttributeName=PK,AttributeType=S \
        AttributeName=SK,AttributeType=S \
        AttributeName=GSI1PK,AttributeType=S \
        AttributeName=GSI1SK,AttributeType=S \
        AttributeName=GSI2PK,AttributeType=S \
        AttributeName=GSI2SK,AttributeType=S \
    --key-schema \
        AttributeName=PK,KeyType=HASH \
        AttributeName=SK,KeyType=RANGE \
    --global-secondary-indexes \
        '[
            {
                "IndexName": "GSI1",
                "Keys": [
                    {"AttributeName": "GSI1PK", "KeyType": "HASH"},
                    {"AttributeName": "GSI1SK", "KeyType": "RANGE"}
                ],
                "Projection": {"ProjectionType": "ALL"},
                "ProvisionedThroughput": {"ReadCapacityUnits": 5, "WriteCapacityUnits": 5}
            },
            {
                "IndexName": "GSI2",
                "Keys": [
                    {"AttributeName": "GSI2PK", "KeyType": "HASH"},
                    {"AttributeName": "GSI2SK", "KeyType": "RANGE"}
                ],
                "Projection": {"ProjectionType": "ALL"},
                "ProvisionedThroughput": {"ReadCapacityUnits": 5, "WriteCapacityUnits": 5}
            }
        ]' \
    --provisioned-throughput ReadCapacityUnits=5,WriteCapacityUnits=5 \
    --endpoint-url http://localhost:8000 \
    --region us-east-1 \
    2>/dev/null || echo "Table might already exist"

echo "DynamoDB Local setup complete!"
echo ""
echo "To run integration tests, use:"
echo "export AWS_ACCESS_KEY_ID=test"
echo "export AWS_SECRET_ACCESS_KEY=test"
echo "export AWS_REGION=us-east-1"
echo "export DDB_TEST_TABLE_NAME=entitystore-test"
echo "export DYNAMODB_ENDPOINT=http://localhost:8000"
echo "go test -v -tags=integration ./..."