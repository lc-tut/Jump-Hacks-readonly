# Start everything with one command
./scripts/start-dev.sh

# Test it works
curl http://localhost:8080/health

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
├── deployments/               # 🚢 Deployment configurations
│   ├── docker/               # (Empty - ready for production Docker configs)
│   ├── kubernetes/           # (Empty - ready for K8s manifests)
│   └── terraform/            # (Empty - ready for AWS infrastructure)
│
├── scripts/                   # 🔧 Development & deployment scripts
│   ├── start-dev.sh          # One-command development setup
│   ├── test-api.sh           # API endpoint testing
│   └── init-db.sql           # Database initialization
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
