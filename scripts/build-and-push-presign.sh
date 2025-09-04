#!/usr/bin/env bash
set -euo pipefail

REGION=${REGION:-ap-northeast-1}
REPO_NAME=${REPO_NAME:-digicon-hackathon-presign}

ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
ECR_URI="$ACCOUNT_ID.dkr.ecr.$REGION.amazonaws.com/$REPO_NAME"

echo "Logging in to ECR: $ECR_URI"
aws ecr get-login-password --region "$REGION" | docker login --username AWS --password-stdin "$ECR_URI"

echo "Building linux/amd64 image (forcing fresh base pull)..."
docker buildx build --pull --no-cache --platform linux/amd64 -f cmd/presign/Dockerfile -t "$REPO_NAME:latest" .

echo "Tagging image..."
docker tag "$REPO_NAME:latest" "$ECR_URI:latest"

echo "Ensuring ECR repo exists..."
aws ecr describe-repositories --repository-names "$REPO_NAME" --region "$REGION" >/dev/null 2>&1 || \
  aws ecr create-repository --repository-name "$REPO_NAME" --region "$REGION" >/dev/null

echo "Pushing image to ECR..."
docker push "$ECR_URI:latest"

echo "Done. Image: $ECR_URI:latest"


