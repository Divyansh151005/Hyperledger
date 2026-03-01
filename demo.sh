#!/bin/bash

################################################################################
# Hyperledger Fabric Medical Information Sharing - End-to-End Demo Script
# This script demonstrates the complete flow from record creation to audit
################################################################################

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Configuration
API_BASE="http://localhost:8080/api/v1"
HOSPITAL_CERT="hospital1admin"
PATIENT_CERT="patient1"
RESEARCHER_CERT="researcher1"
REGULATOR_CERT="regulator1"

# Variables to store IDs
RECORD_ID=""
AUTH_REQUEST_ID=""

# Function to print section header
print_section() {
    echo -e "\n${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}\n"
}

# Function to print step
print_step() {
    echo -e "${YELLOW}[$1] $2...${NC}"
}

# Function to print success
print_success() {
    echo -e "${GREEN}✓ $1${NC}\n"
}

# Function to print error
print_error() {
    echo -e "${RED}✗ $1${NC}\n"
    exit 1
}

# Check if jq is installed
if ! command -v jq &> /dev/null; then
    print_error "jq is required but not installed. Install it with: brew install jq (macOS) or apt-get install jq (Linux)"
fi

# Check if API is accessible
if ! curl -s -f "$API_BASE/../health" > /dev/null; then
    print_error "API server is not accessible at $API_BASE. Make sure middleware is running."
fi

print_section "Hyperledger Fabric Medical Demo"

# Step A: Hospital Creates Medical Record
print_step "A" "Hospital creates medical record"
RESPONSE=$(curl -s -X POST "$API_BASE/records" \
  -H "Content-Type: application/json" \
  -H "X-Certificate-ID: $HOSPITAL_CERT" \
  -d '{
    "patient_id": "patient1",
    "data": {
      "diagnosis": "Type 2 Diabetes",
      "medications": ["Metformin 500mg", "Insulin"],
      "blood_glucose": 180,
      "notes": "Patient requires regular monitoring"
    }
  }')

if [ $? -ne 0 ]; then
    print_error "Failed to create medical record"
fi

RECORD_ID=$(echo $RESPONSE | jq -r '.record_id')
if [ -z "$RECORD_ID" ] || [ "$RECORD_ID" = "null" ]; then
    print_error "Failed to get record ID from response: $RESPONSE"
fi

print_success "Record created: $RECORD_ID"
echo "Response: $RESPONSE" | jq '.'

# Step B: Research Organization Requests Access
print_step "B" "Research organization requests access"
RESPONSE=$(curl -s -X POST "$API_BASE/authorizations/request" \
  -H "Content-Type: application/json" \
  -H "X-Certificate-ID: $RESEARCHER_CERT" \
  -d "{
    \"record_id\": \"$RECORD_ID\",
    \"duration\": 720
  }")

if [ $? -ne 0 ]; then
    print_error "Failed to request access"
fi

AUTH_REQUEST_ID=$(echo $RESPONSE | jq -r '.request_id')
if [ -z "$AUTH_REQUEST_ID" ] || [ "$AUTH_REQUEST_ID" = "null" ]; then
    print_error "Failed to get authorization request ID from response: $RESPONSE"
fi

print_success "Access requested: $AUTH_REQUEST_ID"
echo "Response: $RESPONSE" | jq '.'

# Step C: Hospital Approves Request
print_step "C" "Hospital approves request"
RESPONSE=$(curl -s -X POST "$API_BASE/authorizations/$AUTH_REQUEST_ID/approve/hospital" \
  -H "Content-Type: application/json" \
  -H "X-Certificate-ID: $HOSPITAL_CERT" \
  -d '{}')

if [ $? -ne 0 ]; then
    print_error "Failed to approve by hospital"
fi

print_success "Hospital approved"
echo "Response: $RESPONSE" | jq '.'

# Verify status
STATUS=$(curl -s -X GET "$API_BASE/authorizations/$AUTH_REQUEST_ID" \
  -H "X-Certificate-ID: $HOSPITAL_CERT" | jq -r '.status')
echo "Current status: $STATUS"

# Step D: Patient Approves Request
print_step "D" "Patient approves request"
RESPONSE=$(curl -s -X POST "$API_BASE/authorizations/$AUTH_REQUEST_ID/approve/patient" \
  -H "Content-Type: application/json" \
  -H "X-Certificate-ID: $PATIENT_CERT" \
  -d '{}')

if [ $? -ne 0 ]; then
    print_error "Failed to approve by patient"
fi

print_success "Patient approved"
echo "Response: $RESPONSE" | jq '.'

# Verify status is GRANTED
STATUS=$(curl -s -X GET "$API_BASE/authorizations/$AUTH_REQUEST_ID" \
  -H "X-Certificate-ID: $HOSPITAL_CERT" | jq -r '.status')
print_success "Authorization status: $STATUS"

if [ "$STATUS" != "GRANTED" ]; then
    print_error "Expected status GRANTED, got $STATUS"
fi

# Step E: Hospital Shares Record
print_step "E" "Hospital shares record anonymously"
RESPONSE=$(curl -s -X POST "$API_BASE/sharing/share" \
  -H "Content-Type: application/json" \
  -H "X-Certificate-ID: $HOSPITAL_CERT" \
  -d "{
    \"record_id\": \"$RECORD_ID\",
    \"authorization_request_id\": \"$AUTH_REQUEST_ID\"
  }")

if [ $? -ne 0 ]; then
    print_error "Failed to share record"
fi

print_success "Record shared"
echo "Response: $RESPONSE" | jq '.'

# Step F: Research Org Retrieves Shared Record
print_step "F" "Research organization retrieves shared record"
RESPONSE=$(curl -s -X GET "$API_BASE/sharing/$RECORD_ID/$AUTH_REQUEST_ID" \
  -H "X-Certificate-ID: $RESEARCHER_CERT")

if [ $? -ne 0 ]; then
    print_error "Failed to retrieve shared record"
fi

print_success "Record retrieved"
echo -e "${BLUE}Retrieved Data:${NC}"
echo $RESPONSE | jq '.data'

# Step G: Regulator Audit
print_step "G" "Regulator performs audit"
AUDIT_LOGS=$(curl -s -X GET "$API_BASE/audit/logs" \
  -H "X-Certificate-ID: $REGULATOR_CERT")

if [ $? -ne 0 ]; then
    print_error "Failed to get audit logs"
fi

LOG_COUNT=$(echo $AUDIT_LOGS | jq '. | length')
print_success "Audit complete: $LOG_COUNT events found"

# Show summary
print_section "Demo Summary"
echo -e "${GREEN}✓ All steps completed successfully!${NC}\n"
echo -e "Record ID: ${BLUE}$RECORD_ID${NC}"
echo -e "Authorization Request ID: ${BLUE}$AUTH_REQUEST_ID${NC}"
echo -e "Final Authorization Status: ${BLUE}$STATUS${NC}"
echo -e "Audit Events: ${BLUE}$LOG_COUNT${NC}\n"

echo -e "${BLUE}========================================${NC}"
echo -e "${GREEN}Demo completed successfully!${NC}"
echo -e "${BLUE}========================================${NC}\n"
