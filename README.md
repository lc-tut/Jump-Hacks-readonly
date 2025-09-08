## Frontend (React + Vite)

A React app lives in `frontend/` using Vite (TypeScript).

### Local dev (without Docker)

1. Start the API (Docker or `make run`) so it's available on `http://localhost:8080`.
2. In another terminal:

```
cd frontend
npm install
npm run dev
```

The app runs at `http://localhost:5173`. API requests to `/health` and `/api/*` are proxied to the API.

### Dev with Docker Compose

Run the full stack:

```
docker compose up --build
```

Then open:
- Frontend: `http://localhost:5173`
- API: `http://localhost:8080`
- DB Adminer: `http://localhost:8081`

The frontend container uses `VITE_API_BASE_URL=http://api:8080` to reach the API service.

# Start everything with one command
./scripts/start-dev.sh

# Test it works
API - curl http://localhost:8080/health
Localstack Bucket - docker-compose exec localstack awslocal s3 ls

- **API**: http://localhost:8080
- **Database Admin**: http://localhost:8081 (Adminer)
```
    System: PostgreSQL
    Server: postgres
    Username: postgres
    Password: password
    Database: hackathon_db
```
- **PostgreSQL**: localhost:5432
- **Redis**: localhost:6379 (for future caching needs)

API:
[POST] http://localhost:8080/api/v1/auth/login - authenticate and get token
Request Body:
{ 
  "email": "test@example.com", 
  "password": "password123"
}
[GET] http://localhost:8080/api/v1/profile - get current profile

## File Uploads

Organizer-hosted Presign (App Runner):

```bash
# Get presigned PUT URL
curl -s -X POST "https://bpijpynumu.ap-northeast-1.awsapprunner.com/presign-upload" \
  -H "X-Team-Id: <your-team-id>" \
  -H "X-Team-Token: <your-team-token>" \
  -H 'Content-Type: application/json' \
  -d '{"filename":"hello.txt","content_type":"text/plain"}'

# Upload your file with the returned URL
printf 'Hello, Hackathon!\n' > hello.txt
curl -X PUT --upload-file ./hello.txt -H 'Content-Type: text/plain' "<url-from-step-1>"

# Presign GET to download
curl -s -X POST "https://bpijpynumu.ap-northeast-1.awsapprunner.com/presign-get" \
  -H "X-Team-Id: <your-team-id>" \
  -H "X-Team-Token: <your-team-token>" \
  -H 'Content-Type: application/json' \
  -d '{"key":"uploads/<your-team-id>/<object-key>"}'
```

Local API + LocalStack (optional for local dev):

```bash
# Get presigned PUT URL (JWT required)
TOKEN=<your-jwt>
RES=$(curl -s -X POST http://localhost:8080/api/v1/storage/presign \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"filename":"hello.txt","content_type":"text/plain"}')
URL=$(echo "$RES" | jq -r .url)

# Host override: the API now supports AWS_PUBLIC_ENDPOINT_URL. By default, it uses http://localhost:4566 for presigned URLs so you can upload from your host without edits.
printf 'Hello, Hackathon!\n' > hello.txt
curl -X PUT --upload-file ./hello.txt -H 'Content-Type: text/plain' "$URL"

# One-step multipart upload (server-side)
curl -X POST http://localhost:8080/api/v1/upload \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@hello.txt"
```

LocalStack note (ap-northeast-1):

```bash
docker-compose exec localstack awslocal s3api create-bucket \
  --bucket hackathon-uploads \
  --create-bucket-configuration LocationConstraint=ap-northeast-1
```

```
digi-con-hackathon2025/
├── cmd/api/                    # 🚀 Application entry point
│   └── main.go                # Server startup and routing setup
│
├── internal/                   # 🔒 Private application code
│   ├── auth/                  # 🔐 JWT token handling & password hashing
│   ├── config/                # ⚙️ Environment-based configuration
│   ├── database/              # 🗄️ Database models, migrations, connection
│   ├── handlers/              # 🌐 HTTP request handlers (controllers)
│   ├── middleware/            # 🛡️ CORS, authentication, recovery
│   └── upload/                # 📁 File upload utilities
│
├── scripts/                   # 🔧 Development & deployment scripts
│   ├── start-dev.sh          # One-command development setup
│   ├── test-api.sh           # API endpoint testing
│   ├── init-db.sql           # Database initialization
│   └── (more scripts)
│
├── pkg/                       # 📦 (Ready for reusable packages)
├── api/                       # 📋 (Ready for API documentation)
├── tests/                     # 🧪 (Ready for integration tests)
├── tools/                     # 🛠️ (Ready for development tools)
│
├── docker-compose.yml         # 🐳 Local development environment
├── Dockerfile                 # 🏭 Production container
├── Dockerfile.dev             # 🔄 Development container with hot reload
├── Makefile                   # 🎯 Common development commands
└── .air.toml                  # ♨️ Hot reload configuration
```
