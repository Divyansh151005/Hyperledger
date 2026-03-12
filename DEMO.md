# Medical Records Demo Guide

Complete workflow for the medical data sharing research demo.

## Prerequisites

1. **Fabric network running** - `./network/scripts/start-network.sh` (or equivalent)
2. **Channel created** - `./network/scripts/create-channel.sh`
3. **Chaincode deployed** - `./network/scripts/deploy-chaincode.sh`
4. **MinIO running** - `docker-compose -f docker-compose.minio.yaml up -d`
5. **Backend running** - `cd backend && npm install && npm start`

## Environment Variables

```bash
# Backend (optional - defaults shown)
export PORT=3000
export PEER_HOST=localhost
export PEER_PORT=11051
export CRYPTO_PATH=./network/crypto-config
export MINIO_ENDPOINT=localhost
export MINIO_PORT=9000
export MINIO_ACCESS_KEY=minioadmin
export MINIO_SECRET_KEY=minioadmin
```

## Demo Flow

### 1. Hospital Uploads Record

```bash
curl -X POST http://localhost:3000/hospital/upload-record \
  -F "patientID=patient-001" \
  -F "file=@/path/to/record.pdf" \
  -H "x-org: hospital1"
```

**Response:** `{ "recordID": "REC_1234567890", "patientID": "patient-001", ... }`

### 2. Researcher Requests Access

```bash
curl -X POST http://localhost:3000/researcher/request-access \
  -H "Content-Type: application/json" \
  -d '{"recordID": "REC_1234567890"}'
```

**Response:** `{ "requestID": "REQ_1234567891", "researcherID": "...", ... }`

**Save `researcherID`** - required for retrieve-record.

### 3. Patient Approves Request

```bash
# expiryTimestamp = Unix ms (e.g. now + 24 hours)
EXPIRY=$(($(date +%s) * 1000 + 86400000))
curl -X POST http://localhost:3000/patient/approve-request \
  -H "Content-Type: application/json" \
  -d "{\"requestID\": \"REQ_1234567891\", \"expiryTimestamp\": $EXPIRY}"
```

### 4. Researcher Retrieves Record (with integrity verification)

```bash
curl -o record.pdf "http://localhost:3000/researcher/retrieve-record?recordID=REC_1234567890&researcherID=YOUR_RESEARCHER_ID"
```

*Note: URL-encode researcherID if it contains special characters.*

### 5. Regulator Audits

```bash
# All records
curl http://localhost:3000/regulator/audit-records

# All consents
curl http://localhost:3000/regulator/audit-consents
```

### 6. Regulator Revokes Consent

```bash
curl -X POST http://localhost:3000/regulator/revoke-consent \
  -H "Content-Type: application/json" \
  -d '{"consentID": "CONSENT_1234567892"}'
```

### 7. Patient Rejects Request (alternative to approve)

```bash
curl -X POST http://localhost:3000/patient/reject-request \
  -H "Content-Type: application/json" \
  -d '{"requestID": "REQ_1234567891"}'
```

## Demo Logging

The backend emits clear console logs:

- `[Hospital] Uploaded record REC_xxx for patient patient-001`
- `[Researcher] Access requested for record REC_xxx`
- `[Patient] Consent granted for request REQ_xxx`
- `[Researcher] Integrity verified for record REC_xxx`
- `[Regulator] Consent revoked: CONSENT_xxx`
- `[Event] RecordUploaded: {...}`
- `[Event] AccessRequested: {...}`
- `[Event] ConsentGranted: {...}`
- `[Event] ConsentRevoked: {...}`

## Automated Demo Script

```bash
chmod +x demo-requests.sh
./demo-requests.sh
```

## Chaincode Events

The chaincode emits:

- **RecordUploaded** - when hospital uploads
- **AccessRequested** - when researcher requests
- **ConsentGranted** - when patient approves
- **ConsentRevoked** - when regulator revokes

The backend listens and logs these events.
