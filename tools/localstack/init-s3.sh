#!/bin/sh

# BusyBox /bin/sh doesn't support 'pipefail'
set -eu

BUCKET_NAME=${S3_BUCKET_NAME:-hackathon-uploads}
REGION=${AWS_REGION:-ap-northeast-1}

echo "[localstack] Ensuring S3 bucket '$BUCKET_NAME' exists in region '$REGION'"

if awslocal s3api head-bucket --bucket "$BUCKET_NAME" >/dev/null 2>&1; then
  echo "[localstack] Bucket already exists"
else
  if [ "$REGION" = "us-east-1" ]; then
    awslocal s3api create-bucket \
      --bucket "$BUCKET_NAME" || true
  else
    awslocal s3api create-bucket \
      --bucket "$BUCKET_NAME" \
      --create-bucket-configuration LocationConstraint=$REGION || true
  fi
fi

echo "[localstack] Bucket ready: s3://$BUCKET_NAME"





