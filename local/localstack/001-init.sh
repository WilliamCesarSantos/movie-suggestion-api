#!/bin/bash
set -euo pipefail

echo "Initializing LocalStack..."

awslocal sqs create-queue \
  --queue-name movie-import \
  --attributes VisibilityTimeout=60,MessageRetentionPeriod=86400

echo "SQS queue 'movie-import' created"

awslocal ssm put-parameter \
  --name /movie-suggestion-api/jwt-secret \
  --value "dev-secret" \
  --type SecureString \
  --overwrite

echo "SSM parameter '/movie-suggestion-api/jwt-secret' created"

echo "LocalStack initialization complete"
