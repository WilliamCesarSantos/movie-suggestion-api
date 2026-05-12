#!/bin/bash
set -e

echo "Initializing LocalStack..."

# Create SQS queue
awslocal sqs create-queue \
  --queue-name movie-import \
  --attributes VisibilityTimeout=60,MessageRetentionPeriod=86400

echo "SQS queue 'movie-import' created"

# Put JWT secret in SSM Parameter Store
awslocal ssm put-parameter \
  --name /movie-suggestion/jwt-secret \
  --value "dev-secret" \
  --type SecureString \
  --overwrite

echo "SSM parameter '/movie-suggestion/jwt-secret' created"

# Create Lambda functions (zipped)
echo "LocalStack initialization complete"
