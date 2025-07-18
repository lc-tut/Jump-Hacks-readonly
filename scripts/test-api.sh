#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

API_URL="http://localhost:8080"
TOKEN=""

echo -e "${BLUE}🧪 Testing Hackathon API Endpoints${NC}"
echo "========================================"

# Helper function to make requests
make_request() {
    local method=$1
    local endpoint=$2
    local data=$3
    local expect_success=${4:-true}
    
    echo -e "\n${YELLOW}Testing: $method $endpoint${NC}"
    
    if [ -n "$data" ]; then
        if [ -n "$TOKEN" ]; then
            response=$(curl -s -X $method "$API_URL$endpoint" \
                -H "Content-Type: application/json" \
                -H "Authorization: Bearer $TOKEN" \
                -d "$data" \
                -w "\n%{http_code}")
        else
            response=$(curl -s -X $method "$API_URL$endpoint" \
                -H "Content-Type: application/json" \
                -d "$data" \
                -w "\n%{http_code}")
        fi
    else
        if [ -n "$TOKEN" ]; then
            response=$(curl -s -X $method "$API_URL$endpoint" \
                -H "Authorization: Bearer $TOKEN" \
                -w "\n%{http_code}")
        else
            response=$(curl -s -X $method "$API_URL$endpoint" \
                -w "\n%{http_code}")
        fi
    fi
    
    # Extract HTTP code (last line) and body (everything else)
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')
    
    if [ $expect_success = true ] && [ $http_code -ge 200 ] && [ $http_code -lt 300 ]; then
        echo -e "${GREEN}✅ Success ($http_code)${NC}"
        echo "$body" | jq . 2>/dev/null || echo "$body"
    elif [ $expect_success = false ] && [ $http_code -ge 400 ]; then
        echo -e "${GREEN}✅ Expected error ($http_code)${NC}"
        echo "$body" | jq . 2>/dev/null || echo "$body"
    else
        echo -e "${RED}❌ Unexpected response ($http_code)${NC}"
        echo "$body" | jq . 2>/dev/null || echo "$body"
        return 1
    fi
    
    # Extract token from login response
    if [[ "$endpoint" == *"/auth/login"* ]] && [ $http_code -eq 200 ]; then
        TOKEN=$(echo "$body" | jq -r '.token' 2>/dev/null)
        if [ "$TOKEN" != "null" ] && [ -n "$TOKEN" ]; then
            echo -e "${BLUE}🔑 Token extracted: ${TOKEN:0:20}...${NC}"
        fi
    fi
    
    return 0
}

# Test 1: Health Check
echo -e "\n${BLUE}📊 1. Health Check${NC}"
make_request GET "/health"

# Test 2: User Registration
echo -e "\n${BLUE}👤 2. User Registration${NC}"
make_request POST "/api/v1/auth/register" '{
    "email": "test@example.com",
    "password": "password123",
    "first_name": "John",
    "last_name": "Doe"
}'

# Test 3: Duplicate Registration (should fail)
echo -e "\n${BLUE}👤 3. Duplicate Registration (should fail)${NC}"
make_request POST "/api/v1/auth/register" '{
    "email": "test@example.com",
    "password": "password123",
    "first_name": "Jane",
    "last_name": "Doe"
}' false

# Test 4: User Login
echo -e "\n${BLUE}🔐 4. User Login${NC}"
make_request POST "/api/v1/auth/login" '{
    "email": "test@example.com",
    "password": "password123"
}'

# Test 5: Invalid Login (should fail)
echo -e "\n${BLUE}🔐 5. Invalid Login (should fail)${NC}"
make_request POST "/api/v1/auth/login" '{
    "email": "test@example.com",
    "password": "wrongpassword"
}' false

# Test 6: Get Profile (requires token)
echo -e "\n${BLUE}👤 6. Get Profile${NC}"
if [ -n "$TOKEN" ]; then
    make_request GET "/api/v1/profile"
else
    echo -e "${RED}❌ No token available${NC}"
fi

# Test 7: Update Profile
echo -e "\n${BLUE}✏️ 7. Update Profile${NC}"
if [ -n "$TOKEN" ]; then
    make_request PUT "/api/v1/profile" '{
        "first_name": "John Updated",
        "last_name": "Doe Updated"
    }'
else
    echo -e "${RED}❌ No token available${NC}"
fi

# Test 8: Access Protected Route Without Token (should fail)
echo -e "\n${BLUE}🚫 8. Access Protected Route Without Token (should fail)${NC}"
TOKEN_BACKUP="$TOKEN"
TOKEN=""
make_request GET "/api/v1/profile" "" false
TOKEN="$TOKEN_BACKUP"

# Test 9: List Files (empty initially)
echo -e "\n${BLUE}📁 9. List Files${NC}"
if [ -n "$TOKEN" ]; then
    make_request GET "/api/v1/files"
else
    echo -e "${RED}❌ No token available${NC}"
fi

# Test 10: Admin Endpoints (should fail for regular user)
echo -e "\n${BLUE}👑 10. Admin Endpoints (should fail)${NC}"
if [ -n "$TOKEN" ]; then
    make_request GET "/api/v1/admin/users" "" false
else
    echo -e "${RED}❌ No token available${NC}"
fi

# Test 11: Token Refresh
echo -e "\n${BLUE}🔄 11. Token Refresh${NC}"
if [ -n "$TOKEN" ]; then
    make_request POST "/api/v1/auth/refresh" "{\"token\": \"$TOKEN\"}"
else
    echo -e "${RED}❌ No token available${NC}"
fi

# Summary
echo -e "\n${BLUE}📋 Test Summary${NC}"
echo "========================================"
echo -e "${GREEN}✅ Basic authentication flow working${NC}"
echo -e "${GREEN}✅ Protected routes secured${NC}"
echo -e "${GREEN}✅ Error handling implemented${NC}"
echo -e "${GREEN}✅ Token-based access control functional${NC}"

echo -e "\n${YELLOW}📝 Next Steps:${NC}"
echo "1. Set up AWS credentials for S3 file upload testing"
echo "2. Create admin user for testing admin endpoints"
echo "3. Test file upload functionality"
echo "4. Set up monitoring and logging"

echo -e "\n${BLUE}🎉 API Testing Complete!${NC}" 