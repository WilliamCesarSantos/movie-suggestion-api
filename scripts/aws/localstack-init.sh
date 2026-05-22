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

# Build and deploy auth-lambda
echo "Building auth-lambda..."
AUTH_BUILD=$(mktemp -d)
cp /opt/auth-lambda/handler.py /opt/auth-lambda/jwt_service.py "$AUTH_BUILD/"
pip install --quiet --target "$AUTH_BUILD" PyJWT==2.8.0 cryptography==46.0.5
(cd "$AUTH_BUILD" && zip -r /tmp/auth-lambda.zip .)
rm -rf "$AUTH_BUILD"

awslocal lambda create-function \
  --function-name auth-function \
  --runtime python3.11 \
  --handler handler.lambda_handler \
  --role arn:aws:iam::000000000000:role/lambda-role \
  --zip-file fileb:///tmp/auth-lambda.zip \
  --environment 'Variables={JWT_SECRET=dev-secret,JWT_ALGORITHM=HS256,JWT_EXPIRY_HOURS=24}'

echo "auth-function deployed"

# Build and deploy import-lambda
echo "Building import-lambda..."
IMPORT_BUILD=$(mktemp -d)
cp /opt/import-lambda/handler.py /opt/import-lambda/omdb_client.py /opt/import-lambda/sqs_publisher.py "$IMPORT_BUILD/"
pip install --quiet --target "$IMPORT_BUILD" requests==2.31.0 boto3==1.34.0
(cd "$IMPORT_BUILD" && zip -r /tmp/import-lambda.zip .)
rm -rf "$IMPORT_BUILD"

awslocal lambda create-function \
  --function-name import-function \
  --runtime python3.11 \
  --handler handler.lambda_handler \
  --role arn:aws:iam::000000000000:role/lambda-role \
  --zip-file fileb:///tmp/import-lambda.zip \
  --environment 'Variables={AWS_REGION=us-east-1,AWS_ENDPOINT=http://localhost:4566,SQS_QUEUE_URL=http://localhost:4566/000000000000/movie-import}'

echo "import-function deployed"

echo "LocalStack initialization complete"
