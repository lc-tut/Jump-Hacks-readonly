#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}🚀 Starting Digi-Con Hackathon Template Development Environment${NC}"

# Check if Docker is running
if ! docker info >/dev/null 2>&1; then
    echo -e "${RED}❌ Docker is not running. Please start Docker and try again.${NC}"
    exit 1
fi

# Check if docker-compose is available
if ! command -v docker-compose &> /dev/null; then
    echo -e "${YELLOW}⚠️  docker-compose not found, trying docker compose...${NC}"
    if ! docker compose version &> /dev/null; then
        echo -e "${RED}❌ Neither docker-compose nor 'docker compose' is available.${NC}"
        exit 1
    fi
    DOCKER_COMPOSE="docker compose"
else
    DOCKER_COMPOSE="docker-compose"
fi

# Create .env file if it doesn't exist
if [ ! -f .env ]; then
    echo -e "${YELLOW}📝 Creating .env file from .env.example...${NC}"
    cp .env.example .env
fi

# Pull latest images
echo -e "${GREEN}📦 Pulling latest images...${NC}"
$DOCKER_COMPOSE pull

# Build and start services
echo -e "${GREEN}🔨 Building and starting services...${NC}"
$DOCKER_COMPOSE up --build -d

# Wait for services to be ready
echo -e "${GREEN}⏳ Waiting for services to be ready...${NC}"
sleep 10

# Check if API is responding
if curl -f http://localhost:8080/health >/dev/null 2>&1; then
    echo -e "${GREEN}✅ API is ready at http://localhost:8080${NC}"
    echo -e "${GREEN}✅ Database admin at http://localhost:8081${NC}"
    echo -e "${GREEN}📚 API Documentation will be at http://localhost:8080/swagger${NC}"
else
    echo -e "${YELLOW}⚠️  API might still be starting up. Check logs with: $DOCKER_COMPOSE logs api${NC}"
fi

# Frontend hint
echo -e "${GREEN}✅ Frontend is available at http://localhost:5173${NC}"

echo -e "${GREEN}🎉 Development environment is ready!${NC}"
echo ""
echo -e "Available commands:"
echo -e "  ${YELLOW}$DOCKER_COMPOSE logs api${NC}     - View API logs"
echo -e "  ${YELLOW}$DOCKER_COMPOSE logs postgres${NC} - View database logs"
echo -e "  ${YELLOW}$DOCKER_COMPOSE down${NC}          - Stop all services"
echo -e "  ${YELLOW}$DOCKER_COMPOSE restart api${NC}   - Restart API service"
echo ""

LOCAL_DIR="./private_files/"

docker exec -it $(docker compose ps -q localstack) \
  awslocal s3 mb s3://hackathon-uploads --region ${AWS_REGION:-ap-northeast-1}

docker cp "$LOCAL_DIR" $(docker compose ps -q localstack):/tmp/uploads
docker exec -it $(docker compose ps -q localstack) \
  awslocal s3 cp /tmp/uploads s3://hackathon-uploads/uploads/ --recursive --exclude "*" --include "*.jpg" --include "*.png"
  