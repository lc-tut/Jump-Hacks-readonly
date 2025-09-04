# Presign Service (Organizer-hosted)

A minimal service to issue S3 presigned URLs for a central bucket with team prefixes.

## Endpoints
- POST /presign-upload
  - Headers: X-Team-Id, X-Team-Token
  - Body: { "filename": "test.png", "content_type": "image/png" }
  - Returns: { method, url, headers, expires_in, key, bucket }
- POST /presign-get
  - Headers: X-Team-Id, X-Team-Token
  - Body: { "key": "uploads/team-01/..." }
  - Returns: { method, url, expires_in, key, bucket }
- POST /upload-multipart (one-step upload; Bruno/Postman friendly)
  - Headers: X-Team-Id, X-Team-Token
  - Body: multipart/form-data with field `file` (and optional field `content_type`)
  - Returns: { message, bucket, key, size, content_type }

## Env vars
- PORT=8080
- AWS_REGION=ap-northeast-1
- S3_BUCKET_NAME=digicon-hackathon-2025-uploads
- AWS_ENDPOINT_URL= (empty in prod; set to http://localstack:4566 for local)
- AWS_S3_FORCE_PATH_STYLE=true
- PRESIGN_UPLOAD_EXPIRES_SECONDS=600
- PRESIGN_GET_EXPIRES_SECONDS=600
- ALLOWED_FILE_TYPES=jpg,jpeg,png,gif,pdf
- TEAM_TOKENS="team-01:secret1,team-02:secret2"

## Run locally
```bash
PORT=8080 AWS_REGION=ap-northeast-1 S3_BUCKET_NAME=digicon-hackathon-2025-uploads \
TEAM_TOKENS="team-01:secret1" go run ./cmd/presign
```

## Docker
```Dockerfile
FROM golang:1.22-alpine AS build
WORKDIR /app
COPY . .
RUN go build -o presign ./cmd/presign

FROM alpine:3.20
WORKDIR /app
COPY --from=build /app/presign /usr/local/bin/presign
EXPOSE 8080
CMD ["/usr/local/bin/presign"]
```

## App Runner (high level)
1) Build/push to ECR (image containing the binary above)
2) App Runner → Create service from ECR
3) Set env vars (above) and create an instance role with S3 permissions
4) Test /health, /presign-upload, /presign-get

## IAM policy (attach to instance role)
```json
{
  "Version": "2012-10-17",
  "Statement": [
    { "Effect": "Allow", "Action": ["s3:HeadBucket","s3:ListBucket"], "Resource": "arn:aws:s3:::digicon-hackathon-2025-uploads" },
    { "Effect": "Allow", "Action": ["s3:PutObject","s3:GetObject","s3:DeleteObject"], "Resource": "arn:aws:s3:::digicon-hackathon-2025-uploads/*" }
  ]
}
```

