#!/bin/bash
# Demo script for Medical Records API
# Prerequisites: Network running, chaincode deployed, backend running, MinIO running

BASE_URL="${BASE_URL:-http://localhost:3000}"
TEST_FILE="/tmp/demo-record.pdf"

# Create a test PDF file if it doesn't exist
if [ ! -f "$TEST_FILE" ]; then
    echo "Creating test PDF..."
    echo "%PDF-1.4 Demo Medical Record - Patient 001" > "$TEST_FILE"
fi

echo "=== Medical Records Demo ==="
echo "Base URL: $BASE_URL"
echo ""

# Step 1: Hospital uploads record
echo "[1] Hospital uploads medical record..."
UPLOAD_RESPONSE=$(curl -s -X POST "$BASE_URL/hospital/upload-record" \
  -F "patientID=patient-001" \
  -F "file=@$TEST_FILE" \
  -H "x-org: hospital1")
echo "$UPLOAD_RESPONSE" | jq .
RECORD_ID=$(echo "$UPLOAD_RESPONSE" | jq -r '.recordID')
echo "Record ID: $RECORD_ID"
echo ""

# Step 2: Researcher requests access
echo "[2] Researcher requests access..."
REQUEST_RESPONSE=$(curl -s -X POST "$BASE_URL/researcher/request-access" \
  -H "Content-Type: application/json" \
  -d "{\"recordID\": \"$RECORD_ID\"}")
echo "$REQUEST_RESPONSE" | jq .
REQUEST_ID=$(echo "$REQUEST_RESPONSE" | jq -r '.requestID')
RESEARCHER_ID=$(echo "$REQUEST_RESPONSE" | jq -r '.researcherID')
echo "Request ID: $REQUEST_ID"
echo "Researcher ID: $RESEARCHER_ID"
echo ""

# Step 3: Patient approves (expiry = now + 24 hours)
echo "[3] Patient approves request..."
EXPIRY=$(($(date +%s) * 1000 + 86400000))
curl -s -X POST "$BASE_URL/patient/approve-request" \
  -H "Content-Type: application/json" \
  -d "{\"requestID\": \"$REQUEST_ID\", \"expiryTimestamp\": $EXPIRY}" | jq .
echo ""

# Step 4: Researcher retrieves record (with integrity verification)
echo "[4] Researcher retrieves record..."
curl -s -o /tmp/retrieved-record.pdf "$BASE_URL/researcher/retrieve-record?recordID=$RECORD_ID&researcherID=$(echo $RESEARCHER_ID | jq -sRr @uri)"
if [ -f /tmp/retrieved-record.pdf ]; then
    echo "File downloaded. Size: $(wc -c < /tmp/retrieved-record.pdf) bytes"
fi
echo ""

# Step 5: Regulator audits
echo "[5] Regulator audits records..."
curl -s "$BASE_URL/regulator/audit-records" | jq .
echo ""
echo "[6] Regulator audits consents..."
CONSENTS=$(curl -s "$BASE_URL/regulator/audit-consents")
echo "$CONSENTS" | jq .
CONSENT_ID=$(echo "$CONSENTS" | jq -r '.consents[0].consentID')
echo ""

# Step 7: Regulator revokes consent (optional)
if [ -n "$CONSENT_ID" ] && [ "$CONSENT_ID" != "null" ]; then
    echo "[7] Regulator revokes consent..."
    curl -s -X POST "$BASE_URL/regulator/revoke-consent" \
      -H "Content-Type: application/json" \
      -d "{\"consentID\": \"$CONSENT_ID\"}" | jq .
fi

echo ""
echo "=== Demo complete ==="
