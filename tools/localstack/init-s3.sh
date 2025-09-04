#!/bin/sh

set -euo pipefail

BUCKET_NAME=${S3_BUCKET_NAME:-hackathon-uploads}
REGION=${AWS_REGION:-us-east-1}

echo "[localstack] Ensuring S3 bucket '$BUCKET_NAME' exists in region '$REGION'"

awslocal s3api head-bucket --bucket "$BUCKET_NAME" 2>/dev/null || \
  awslocal s3api create-bucket \
    --bucket "$BUCKET_NAME" \
    --create-bucket-configuration LocationConstraint=$REGION || true

echo "[localstack] Bucket ready: s3://$BUCKET_NAME"





