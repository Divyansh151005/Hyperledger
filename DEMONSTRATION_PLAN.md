# Hyperledger Fabric Medical Information Sharing System
## End-to-End Demonstration Plan

This document provides a comprehensive, step-by-step guide to demonstrate and verify that the Hyperledger Fabric-based medical information sharing system is working correctly.

---

## Table of Contents

1. [Network Verification](#1-network-verification)
2. [Identity Setup](#2-identity-setup)
3. [Start Middleware](#3-start-middleware)
4. [Live End-to-End Demo](#4-live-end-to-end-demo)
5. [Visibility Proofs](#5-visibility-proofs)
6. [Expected Outputs](#6-expected-outputs)
7. [One-Command Demo Script](#7-one-command-demo-script)

---

## 1. Network Verification

### 1.1 Verify All Peers and Orderers Are Running

```bash
# Check all Docker containers are up
docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"

# Expected output should show:
# - Orderer containers (orderer0, orderer1, orderer2)
# - Peer containers for each organization (peer0.hospital1, peer0.hospital2, peer0.research, peer0.regulator)
# - CA containers (ca.hospital1, ca.hospital2, ca.research, ca.regulator)

# Check specific peer containers
docker ps | grep -E "peer0|orderer"

# Check container health
docker ps --filter "health=unhealthy"  # Should return empty
```

### 1.2 Verify Channel Exists and Peers Have Joined

```bash
# Set environment variables for peer CLI
export FABRIC_CFG_PATH=/path/to/fabric/config
export CORE_PEER_LOCALMSPID=HospitalMSP1
export CORE_PEER_MSPCONFIGPATH=/path/to/crypto-config/peerOrganizations/hospital1.example.com/users/Admin@hospital1.example.com/msp
export CORE_PEER_ADDRESS=peer0.hospital1.example.com:7051
export CORE_PEER_TLS_ROOTCERT_FILE=/path/to/crypto-config/peerOrganizations/hospital1.example.com/peers/peer0.hospital1.example.com/tls/ca.crt

# List channels (should show medical-main-channel)
peer channel list

# Expected output:
# Channels peers has joined:
# medical-main-channel

# Get channel info
peer channel getinfo -c medical-main-channel

# Expected output shows:
# - Block height
# - Current block hash
# - Previous block hash
```

### 1.3 Verify Chaincodes Are Committed

```bash
# Check committed chaincodes using peer lifecycle
peer lifecycle chaincode querycommitted --channelID medical-main-channel --peerAddresses peer0.hospital1.example.com:7051 --tlsRootCertFiles /path/to/crypto-config/peerOrganizations/hospital1.example.com/peers/peer0.hospital1.example.com/tls/ca.crt

# Expected output should show:
# Committed chaincode definitions on channel 'medical-main-channel':
# Name: medical-records, Version: 1.0, Sequence: 1, Endorsement Plugin: escc, Validation Plugin: vscc
# Name: authorization-consent, Version: 1.0, Sequence: 1, Endorsement Plugin: escc, Validation Plugin: vscc
# Name: anonymous-sharing, Version: 1.0, Sequence: 1, Endorsement Plugin: escc, Validation Plugin: vscc

# Alternative: Query using peer chaincode
peer chaincode list --installed
peer chaincode list --instantiated -C medical-main-channel
```

### 1.4 Verify Chaincode Endorsement Policies

```bash
# Query chaincode definition details
peer lifecycle chaincode querycommitted --channelID medical-main-channel --name medical-records --peerAddresses peer0.hospital1.example.com:7051 --tlsRootCertFiles /path/to/crypto-config/peerOrganizations/hospital1.example.com/peers/peer0.hospital1.example.com/tls/ca.crt --output json

# This will show endorsement policies, collections, etc.
```

---

## 2. Identity Setup

### 2.1 Enroll Hospital Admin Identity

```bash
# Set environment for Hospital1 CA
export FABRIC_CA_CLIENT_HOME=/path/to/crypto-config/peerOrganizations/hospital1.example.com
export FABRIC_CA_CLIENT_TLS_CERTFILES=/path/to/crypto-config/peerOrganizations/hospital1.example.com/ca/ca.hospital1.example.com-cert.pem

# Enroll admin user
fabric-ca-client enroll -u https://admin:adminpw@ca.hospital1.example.com:7054 --caname ca.hospital1.example.com

# Register a hospital admin user
fabric-ca-client register --caname ca.hospital1.example.com --id.name hospital1admin --id.secret hospital1adminpw --id.attrs '"hf.Registrar.Roles=user,admin","hf.Registrar.Attributes=*"'

# Enroll the hospital admin
fabric-ca-client enroll -u https://hospital1admin:hospital1adminpw@ca.hospital1.example.com:7054 --caname ca.hospital1.example.com -M /path/to/crypto-config/peerOrganizations/hospital1.example.com/users/hospital1admin@hospital1.example.com/msp

# Verify certificate
ls -la /path/to/crypto-config/peerOrganizations/hospital1.example.com/users/hospital1admin@hospital1.example.com/msp/signcerts/

# Extract certificate ID (CN from certificate)
openssl x509 -in /path/to/crypto-config/peerOrganizations/hospital1.example.com/users/hospital1admin@hospital1.example.com/msp/signcerts/cert.pem -noout -subject

# Expected: subject= /CN=hospital1admin
# Certificate ID: hospital1admin
```

### 2.2 Enroll Patient Identity

```bash
# Set environment for Hospital1 CA (patients enroll through hospital CA)
export FABRIC_CA_CLIENT_HOME=/path/to/crypto-config/peerOrganizations/hospital1.example.com

# Register a patient
fabric-ca-client register --caname ca.hospital1.example.com --id.name patient1 --id.secret patient1pw --id.attrs '"role=patient"'

# Enroll the patient
fabric-ca-client enroll -u https://patient1:patient1pw@ca.hospital1.example.com:7054 --caname ca.hospital1.example.com -M /path/to/crypto-config/peerOrganizations/hospital1.example.com/users/patient1@hospital1.example.com/msp

# Extract certificate ID
openssl x509 -in /path/to/crypto-config/peerOrganizations/hospital1.example.com/users/patient1@hospital1.example.com/msp/signcerts/cert.pem -noout -subject

# Expected: subject= /CN=patient1
# Certificate ID: patient1
```

### 2.3 Enroll Research Organization User

```bash
# Set environment for ResearchOrg CA
export FABRIC_CA_CLIENT_HOME=/path/to/crypto-config/peerOrganizations/research.example.com
export FABRIC_CA_CLIENT_TLS_CERTFILES=/path/to/crypto-config/peerOrganizations/research.example.com/ca/ca.research.example.com-cert.pem

# Enroll admin
fabric-ca-client enroll -u https://admin:adminpw@ca.research.example.com:8054 --caname ca.research.example.com

# Register research user
fabric-ca-client register --caname ca.research.example.com --id.name researcher1 --id.secret researcher1pw --id.attrs '"role=researcher"'

# Enroll researcher
fabric-ca-client enroll -u https://researcher1:researcher1pw@ca.research.example.com:8054 --caname ca.research.example.com -M /path/to/crypto-config/peerOrganizations/research.example.com/users/researcher1@research.example.com/msp

# Extract certificate ID
openssl x509 -in /path/to/crypto-config/peerOrganizations/research.example.com/users/researcher1@research.example.com/msp/signcerts/cert.pem -noout -subject

# Expected: subject= /CN=researcher1
# Certificate ID: researcher1
```

### 2.4 Enroll Regulator Identity

```bash
# Set environment for Regulator CA
export FABRIC_CA_CLIENT_HOME=/path/to/crypto-config/peerOrganizations/regulator.example.com
export FABRIC_CA_CLIENT_TLS_CERTFILES=/path/to/crypto-config/peerOrganizations/regulator.example.com/ca/ca.regulator.example.com-cert.pem

# Enroll admin
fabric-ca-client enroll -u https://admin:adminpw@ca.regulator.example.com:9054 --caname ca.regulator.example.com

# Register regulator user
fabric-ca-client register --caname ca.regulator.example.com --id.name regulator1 --id.secret regulator1pw --id.attrs '"role=regulator"'

# Enroll regulator
fabric-ca-client enroll -u https://regulator1:regulator1pw@ca.regulator.example.com:9054 --caname ca.regulator.example.com -M /path/to/crypto-config/peerOrganizations/regulator.example.com/users/regulator1@regulator.example.com/msp

# Extract certificate ID
openssl x509 -in /path/to/crypto-config/peerOrganizations/regulator.example.com/users/regulator1@regulator.example.com/msp/signcerts/cert.pem -noout -subject

# Expected: subject= /CN=regulator1
# Certificate ID: regulator1
```

### 2.5 Register Identities in Middleware Database

The middleware needs to map certificate IDs to roles and MSPs. You'll need to insert these into the `identity_mappings` table:

```sql
-- Connect to PostgreSQL
psql -U user -d medical_middleware

-- Insert identities
INSERT INTO identity_mappings (certificate_id, msp_id, role, user_id) VALUES
('hospital1admin', 'HospitalMSP1', 'HOSPITAL', 'hospital1admin'),
('patient1', 'HospitalMSP1', 'PATIENT', 'patient1'),
('researcher1', 'ResearchOrgMSP', 'RESEARCH', 'researcher1'),
('regulator1', 'RegulatorMSP', 'REGULATOR', 'regulator1')
ON CONFLICT (certificate_id) DO UPDATE SET msp_id = EXCLUDED.msp_id, role = EXCLUDED.role, user_id = EXCLUDED.user_id;

-- Verify
SELECT * FROM identity_mappings;
```

---

## 3. Start Middleware

### 3.1 Environment Variables

Create a `.env` file in the `middleware/` directory:

```bash
# API Configuration
API_HOST=0.0.0.0
API_PORT=8080
API_LOG_LEVEL=info

# Database
DATABASE_URL=postgresql://user:password@localhost:5432/medical_middleware?sslmode=disable

# S3-Compatible Storage (MinIO example)
S3_ENDPOINT=http://localhost:9000
S3_BUCKET=medical-records
S3_ACCESS_KEY=minioadmin
S3_SECRET_KEY=minioadmin
S3_REGION=us-east-1

# Fabric Network
FABRIC_NETWORK_CONFIG_PATH=./network/connection-profile.json
FABRIC_CHANNEL_NAME=medical-main-channel
FABRIC_CHAINCODE_MEDICAL_RECORDS=medical-records
FABRIC_CHAINCODE_AUTHORIZATION=authorization-consent
FABRIC_CHAINCODE_ANONYMOUS_SHARING=anonymous-sharing

# Encryption
ENCRYPTION_KEY_STORE_PATH=./keystore
ENCRYPTION_ALGORITHM=AES-256-GCM
ENCRYPTION_KEY_SIZE=32
```

### 3.2 Start the Middleware Server

```bash
cd middleware
go run main.go

# Expected output:
# INFO[0000] Starting API server on 0.0.0.0:8080
```

### 3.3 Verify Middleware is Connected

```bash
# Health check
curl http://localhost:8080/health

# Expected response:
# {"status":"healthy"}
```

### 3.4 Verify Fabric Connection

Check middleware logs for:
- "Fabric gateway initialized successfully"
- "Event listener started"
- No connection errors

---

## 4. Live End-to-End Demo

### Step A: Hospital Creates a Medical Record

#### REST Call

```bash
curl -X POST http://localhost:8080/api/v1/records \
  -H "Content-Type: application/json" \
  -H "X-Certificate-ID: hospital1admin" \
  -d '{
    "patient_id": "patient1",
    "data": {
      "diagnosis": "Type 2 Diabetes",
      "medications": ["Metformin 500mg", "Insulin"],
      "blood_glucose": 180,
      "notes": "Patient requires regular monitoring"
    }
  }'
```

#### Expected Response

```json
{
  "record_id": "550e8400-e29b-41d4-a716-446655440000",
  "record_hash": "a3f5e8b2c1d4f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1",
  "status": "ACTIVE"
}
```

#### What Gets Stored

**Off-Chain (PostgreSQL + S3):**
- Encrypted medical data stored in S3 at `records/{record_id}`
- Metadata in PostgreSQL (if tracked)

**On-Chain (Blockchain):**
- Record metadata with hash commitment
- Event: `MedicalRecordCreated`

#### Verify On-Chain State

```bash
# Set peer environment
export CORE_PEER_LOCALMSPID=HospitalMSP1
export CORE_PEER_MSPCONFIGPATH=/path/to/crypto-config/peerOrganizations/hospital1.example.com/users/Admin@hospital1.example.com/msp
export CORE_PEER_ADDRESS=peer0.hospital1.example.com:7051
export CORE_PEER_TLS_ROOTCERT_FILE=/path/to/crypto-config/peerOrganizations/hospital1.example.com/peers/peer0.hospital1.example.com/tls/ca.crt

# Query chaincode
peer chaincode query -C medical-main-channel -n medical-records -c '{"function":"GetMedicalRecord","Args":["550e8400-e29b-41d4-a716-446655440000"]}'

# Expected output:
# {"recordID":"550e8400-e29b-41d4-a716-446655440000","patientID":"patient1","hospitalID":"HospitalMSP1","recordHash":"a3f5e8b2c1d4f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1","timestamp":"2024-01-15T10:30:00Z","status":"ACTIVE"}
```

---

### Step B: Research Organization Requests Access

#### REST Call

```bash
curl -X POST http://localhost:8080/api/v1/authorizations/request \
  -H "Content-Type: application/json" \
  -H "X-Certificate-ID: researcher1" \
  -d '{
    "record_id": "550e8400-e29b-41d4-a716-446655440000",
    "duration": 720
  }'
```

#### Expected Response

```json
{
  "request_id": "AUTH-550e8400-e29b-41d4-a716-446655440000-1705312200000000000",
  "status": "REQUESTED"
}
```

**Save the `request_id` for subsequent steps:**
```bash
export AUTH_REQUEST_ID="AUTH-550e8400-e29b-41d4-a716-446655440000-1705312200000000000"
```

#### Query Authorization State On-Chain

```bash
# Query using peer CLI
peer chaincode query -C medical-main-channel -n authorization-consent -c "{\"function\":\"GetAuthorization\",\"Args\":[\"$AUTH_REQUEST_ID\"]}"

# Expected output:
# {"requestID":"AUTH-550e8400-e29b-41d4-a716-446655440000-1705312200000000000","recordID":"550e8400-e29b-41d4-a716-446655440000","requesterOrg":"ResearchOrgMSP","hospitalID":"HospitalMSP1","patientID":"patient1","status":"REQUESTED","createdAt":"2024-01-15T10:35:00Z","expiresAt":"2024-02-14T10:35:00Z"}
```

---

### Step C: Hospital Approves Request

#### REST Call

```bash
curl -X POST http://localhost:8080/api/v1/authorizations/$AUTH_REQUEST_ID/approve/hospital \
  -H "Content-Type: application/json" \
  -H "X-Certificate-ID: hospital1admin" \
  -d '{}'
```

#### Expected Response

```json
{
  "status": "approved"
}
```

#### State Transition Verification

```bash
# Query authorization again
curl -X GET http://localhost:8080/api/v1/authorizations/$AUTH_REQUEST_ID \
  -H "X-Certificate-ID: hospital1admin"

# Expected response:
# {
#   "request_id": "AUTH-550e8400-e29b-41d4-a716-446655440000-1705312200000000000",
#   "record_id": "550e8400-e29b-41d4-a716-446655440000",
#   "requester_org": "ResearchOrgMSP",
#   "hospital_id": "HospitalMSP1",
#   "patient_id": "patient1",
#   "status": "HOSPITAL_APPROVED",
#   "created_at": "2024-01-15T10:35:00Z",
#   "expires_at": "2024-02-14T10:35:00Z"
# }
```

---

### Step D: Patient Approves Request

#### REST Call

```bash
curl -X POST http://localhost:8080/api/v1/authorizations/$AUTH_REQUEST_ID/approve/patient \
  -H "Content-Type: application/json" \
  -H "X-Certificate-ID: patient1" \
  -d '{}'
```

#### Expected Response

```json
{
  "status": "approved"
}
```

#### Transition to GRANTED - Verify Consent Exists On-Chain

```bash
# Query authorization status
curl -X GET http://localhost:8080/api/v1/authorizations/$AUTH_REQUEST_ID \
  -H "X-Certificate-ID: patient1"

# Expected response:
# {
#   "request_id": "AUTH-550e8400-e29b-41d4-a716-446655440000-1705312200000000000",
#   "record_id": "550e8400-e29b-41d4-a716-446655440000",
#   "requester_org": "ResearchOrgMSP",
#   "hospital_id": "HospitalMSP1",
#   "patient_id": "patient1",
#   "status": "GRANTED",
#   "created_at": "2024-01-15T10:35:00Z",
#   "expires_at": "2024-02-14T10:35:00Z"
# }

# Verify on-chain using peer CLI
peer chaincode query -C medical-main-channel -n authorization-consent -c "{\"function\":\"GetAuthorization\",\"Args\":[\"$AUTH_REQUEST_ID\"]}"

# Status should be "GRANTED"
```

---

### Step E: Hospital Shares Record Anonymously

#### REST Call

```bash
curl -X POST http://localhost:8080/api/v1/sharing/share \
  -H "Content-Type: application/json" \
  -H "X-Certificate-ID: hospital1admin" \
  -d '{
    "record_id": "550e8400-e29b-41d4-a716-446655440000",
    "authorization_request_id": "'$AUTH_REQUEST_ID'"
  }'
```

#### Expected Response

```json
{
  "record_id": "550e8400-e29b-41d4-a716-446655440000",
  "authorization_request_id": "AUTH-550e8400-e29b-41d4-a716-446655440000-1705312200000000000",
  "status": "SHARED"
}
```

#### What Is Stored

**Private Data Collection (PDC):**
- Encrypted pointer to off-chain data
- Only visible to: HospitalMSP1, ResearchOrgMSP, RegulatorMSP

**Public Ledger (Hash Commitment):**
- Key: `SHARED_{recordID}_{authRequestID}`
- Value: Hash of the payload (visible to all)

#### Prove Unrelated Orgs Cannot See Payload

```bash
# Try to access PDC from an unrelated peer (HospitalMSP2)
export CORE_PEER_LOCALMSPID=HospitalMSP2
export CORE_PEER_MSPCONFIGPATH=/path/to/crypto-config/peerOrganizations/hospital2.example.com/users/Admin@hospital2.example.com/msp
export CORE_PEER_ADDRESS=peer0.hospital2.example.com:8051
export CORE_PEER_TLS_ROOTCERT_FILE=/path/to/crypto-config/peerOrganizations/hospital2.example.com/peers/peer0.hospital2.example.com/tls/ca.crt

# Attempt to query private data (should fail or return empty)
peer chaincode query -C medical-main-channel -n anonymous-sharing -c "{\"function\":\"GetSharedRecord\",\"Args\":[\"550e8400-e29b-41d4-a716-446655440000\",\"$AUTH_REQUEST_ID\"]}"

# Expected: Error - access denied or empty result
```

---

### Step F: Research Org Retrieves Shared Record

#### REST Call

```bash
curl -X GET "http://localhost:8080/api/v1/sharing/550e8400-e29b-41d4-a716-446655440000/$AUTH_REQUEST_ID" \
  -H "X-Certificate-ID: researcher1"
```

#### Expected Response

```json
{
  "record_id": "550e8400-e29b-41d4-a716-446655440000",
  "authorization_request_id": "AUTH-550e8400-e29b-41d4-a716-446655440000-1705312200000000000",
  "data": {
    "diagnosis": "Type 2 Diabetes",
    "medications": ["Metformin 500mg", "Insulin"],
    "blood_glucose": 180,
    "notes": "Patient requires regular monitoring"
  },
  "shared_at": "2024-01-15T10:45:00Z"
}
```

#### Decryption Flow Explanation

1. Middleware retrieves encrypted pointer from PDC
2. Decrypts pointer using authorization-specific key
3. Extracts storage location from pointer
4. Retrieves encrypted medical data from S3
5. Decrypts medical data using hospital key
6. Returns decrypted data to research org

---

### Step G: Regulator Audit

#### Query All Records

```bash
# Query all medical records (regulator has access to all)
curl -X GET http://localhost:8080/api/v1/records/550e8400-e29b-41d4-a716-446655440000 \
  -H "X-Certificate-ID: regulator1"
```

#### Query All Authorizations

```bash
# Query authorization
curl -X GET http://localhost:8080/api/v1/authorizations/$AUTH_REQUEST_ID \
  -H "X-Certificate-ID: regulator1"
```

#### Query Audit Logs

```bash
# Get all audit logs
curl -X GET "http://localhost:8080/api/v1/audit/logs" \
  -H "X-Certificate-ID: regulator1"

# Filter by event type
curl -X GET "http://localhost:8080/api/v1/audit/logs?event_type=MedicalRecordCreated" \
  -H "X-Certificate-ID: regulator1"

# Filter by chaincode
curl -X GET "http://localhost:8080/api/v1/audit/logs?chaincode_name=medical-records" \
  -H "X-Certificate-ID: regulator1"
```

#### Show Audit Trail from Fabric Events

```bash
# Query events using peer CLI (if event listener is configured)
# Or check middleware database audit_logs table

psql -U user -d medical_middleware -c "SELECT * FROM audit_logs ORDER BY timestamp DESC LIMIT 10;"

# Expected columns:
# - event_type (MedicalRecordCreated, AuthorizationRequested, etc.)
# - chaincode_name
# - transaction_id
# - timestamp
# - payload (JSON)
```

---

## 5. Visibility Proofs

### 5.1 World State Entries

```bash
# Query world state for medical record
peer chaincode query -C medical-main-channel -n medical-records -c '{"function":"GetMedicalRecord","Args":["550e8400-e29b-41d4-a716-446655440000"]}'

# Query world state for authorization
peer chaincode query -C medical-main-channel -n authorization-consent -c "{\"function\":\"GetAuthorization\",\"Args\":[\"$AUTH_REQUEST_ID\"]}"

# Query world state for sharing commitment
peer chaincode query -C medical-main-channel -n anonymous-sharing -c "{\"function\":\"VerifySharedRecord\",\"Args\":[\"550e8400-e29b-41d4-a716-446655440000\",\"$AUTH_REQUEST_ID\",\"<hash>\"]}"
```

### 5.2 PDC Data Access - Allowed vs Denied Org

#### Allowed Org (ResearchOrgMSP) - Can Access

```bash
# Set environment for ResearchOrg peer
export CORE_PEER_LOCALMSPID=ResearchOrgMSP
export CORE_PEER_MSPCONFIGPATH=/path/to/crypto-config/peerOrganizations/research.example.com/users/Admin@research.example.com/msp
export CORE_PEER_ADDRESS=peer0.research.example.com:9051
export CORE_PEER_TLS_ROOTCERT_FILE=/path/to/crypto-config/peerOrganizations/research.example.com/peers/peer0.research.example.com/tls/ca.crt

# Query shared record (should succeed)
peer chaincode query -C medical-main-channel -n anonymous-sharing -c "{\"function\":\"GetSharedRecord\",\"Args\":[\"550e8400-e29b-41d4-a716-446655440000\",\"$AUTH_REQUEST_ID\"]}"

# Expected: Returns encrypted pointer
```

#### Denied Org (HospitalMSP2) - Cannot Access

```bash
# Set environment for HospitalMSP2 peer
export CORE_PEER_LOCALMSPID=HospitalMSP2
export CORE_PEER_MSPCONFIGPATH=/path/to/crypto-config/peerOrganizations/hospital2.example.com/users/Admin@hospital2.example.com/msp
export CORE_PEER_ADDRESS=peer0.hospital2.example.com:8051
export CORE_PEER_TLS_ROOTCERT_FILE=/path/to/crypto-config/peerOrganizations/hospital2.example.com/peers/peer0.hospital2.example.com/tls/ca.crt

# Query shared record (should fail)
peer chaincode query -C medical-main-channel -n anonymous-sharing -c "{\"function\":\"GetSharedRecord\",\"Args\":[\"550e8400-e29b-41d4-a716-446655440000\",\"$AUTH_REQUEST_ID\"]}"

# Expected: Error - access denied
```

### 5.3 Hash Verification Using VerifySharedRecord

```bash
# Compute hash of payload (for demonstration)
# In practice, you'd compute this from the actual payload

# Query commitment hash from public ledger
peer chaincode query -C medical-main-channel -n anonymous-sharing -c "{\"function\":\"VerifySharedRecord\",\"Args\":[\"550e8400-e29b-41d4-a716-446655440000\",\"$AUTH_REQUEST_ID\",\"<computed_hash>\"]}"

# Expected: true if hash matches, false otherwise
```

### 5.4 What Unrelated Peer Sees vs Authorized Peer

#### Unrelated Peer (HospitalMSP2) - Public Ledger Only

```bash
# Query public commitment (visible to all)
peer chaincode query -C medical-main-channel -n anonymous-sharing -c "{\"function\":\"VerifySharedRecord\",\"Args\":[\"550e8400-e29b-41d4-a716-446655440000\",\"$AUTH_REQUEST_ID\",\"<hash>\"]}"

# Can see: Hash commitment, timestamp
# Cannot see: Encrypted pointer, actual data
```

#### Authorized Peer (ResearchOrgMSP) - PDC Access

```bash
# Query private data (accessible)
peer chaincode query -C medical-main-channel -n anonymous-sharing -c "{\"function\":\"GetSharedRecord\",\"Args\":[\"550e8400-e29b-41d4-a716-446655440000\",\"$AUTH_REQUEST_ID\"]}"

# Can see: Encrypted pointer, full payload
```

---

## 6. Expected Outputs

### 6.1 Sample JSON Responses

#### Medical Record Creation
```json
{
  "record_id": "550e8400-e29b-41d4-a716-446655440000",
  "record_hash": "a3f5e8b2c1d4f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1",
  "status": "ACTIVE"
}
```

#### Authorization Request
```json
{
  "request_id": "AUTH-550e8400-e29b-41d4-a716-446655440000-1705312200000000000",
  "status": "REQUESTED"
}
```

#### Authorization Status (GRANTED)
```json
{
  "request_id": "AUTH-550e8400-e29b-41d4-a716-446655440000-1705312200000000000",
  "record_id": "550e8400-e29b-41d4-a716-446655440000",
  "requester_org": "ResearchOrgMSP",
  "hospital_id": "HospitalMSP1",
  "patient_id": "patient1",
  "status": "GRANTED",
  "created_at": "2024-01-15T10:35:00Z",
  "expires_at": "2024-02-14T10:35:00Z"
}
```

#### Shared Record Retrieval
```json
{
  "record_id": "550e8400-e29b-41d4-a716-446655440000",
  "authorization_request_id": "AUTH-550e8400-e29b-41d4-a716-446655440000-1705312200000000000",
  "data": {
    "diagnosis": "Type 2 Diabetes",
    "medications": ["Metformin 500mg", "Insulin"],
    "blood_glucose": 180,
    "notes": "Patient requires regular monitoring"
  },
  "shared_at": "2024-01-15T10:45:00Z"
}
```

### 6.2 Sample Fabric Event Logs

Events are emitted by chaincodes and captured by the middleware event listener. Check middleware logs or database:

```json
{
  "event_type": "MedicalRecordCreated",
  "chaincode_name": "medical-records",
  "transaction_id": "tx1234567890",
  "timestamp": "2024-01-15T10:30:00Z",
  "payload": {
    "recordID": "550e8400-e29b-41d4-a716-446655440000",
    "action": "MedicalRecordCreated",
    "invokerMSP": "HospitalMSP1",
    "timestamp": "2024-01-15T10:30:00Z",
    "patientID": "patient1",
    "hospitalID": "HospitalMSP1"
  }
}
```

### 6.3 What Success Looks Like at Each Step

| Step | Success Indicator |
|------|-------------------|
| A: Create Record | HTTP 201, record_id returned, on-chain state queryable |
| B: Request Access | HTTP 201, request_id returned, status=REQUESTED |
| C: Hospital Approves | HTTP 200, status=HOSPITAL_APPROVED or GRANTED |
| D: Patient Approves | HTTP 200, status=GRANTED |
| E: Share Record | HTTP 201, status=SHARED, PDC contains payload |
| F: Retrieve Record | HTTP 200, decrypted data returned |
| G: Audit | HTTP 200, audit logs returned |

---

## 7. One-Command Demo Script

Create a bash script `demo.sh` that runs the entire flow:

```bash
#!/bin/bash

# Hyperledger Fabric Medical Information Sharing - End-to-End Demo Script
# This script demonstrates the complete flow from record creation to audit

set -e

# Colors
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

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Hyperledger Fabric Medical Demo${NC}"
echo -e "${BLUE}========================================${NC}\n"

# Step A: Hospital Creates Medical Record
echo -e "${YELLOW}[Step A] Hospital creates medical record...${NC}"
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

RECORD_ID=$(echo $RESPONSE | jq -r '.record_id')
echo -e "${GREEN}✓ Record created: $RECORD_ID${NC}\n"

# Step B: Research Organization Requests Access
echo -e "${YELLOW}[Step B] Research organization requests access...${NC}"
RESPONSE=$(curl -s -X POST "$API_BASE/authorizations/request" \
  -H "Content-Type: application/json" \
  -H "X-Certificate-ID: $RESEARCHER_CERT" \
  -d "{
    \"record_id\": \"$RECORD_ID\",
    \"duration\": 720
  }")

AUTH_REQUEST_ID=$(echo $RESPONSE | jq -r '.request_id')
echo -e "${GREEN}✓ Access requested: $AUTH_REQUEST_ID${NC}\n"

# Step C: Hospital Approves Request
echo -e "${YELLOW}[Step C] Hospital approves request...${NC}"
curl -s -X POST "$API_BASE/authorizations/$AUTH_REQUEST_ID/approve/hospital" \
  -H "Content-Type: application/json" \
  -H "X-Certificate-ID: $HOSPITAL_CERT" \
  -d '{}' > /dev/null
echo -e "${GREEN}✓ Hospital approved${NC}\n"

# Step D: Patient Approves Request
echo -e "${YELLOW}[Step D] Patient approves request...${NC}"
curl -s -X POST "$API_BASE/authorizations/$AUTH_REQUEST_ID/approve/patient" \
  -H "Content-Type: application/json" \
  -H "X-Certificate-ID: $PATIENT_CERT" \
  -d '{}' > /dev/null
echo -e "${GREEN}✓ Patient approved${NC}\n"

# Verify status is GRANTED
STATUS=$(curl -s -X GET "$API_BASE/authorizations/$AUTH_REQUEST_ID" \
  -H "X-Certificate-ID: $HOSPITAL_CERT" | jq -r '.status')
echo -e "${GREEN}✓ Authorization status: $STATUS${NC}\n"

# Step E: Hospital Shares Record
echo -e "${YELLOW}[Step E] Hospital shares record anonymously...${NC}"
curl -s -X POST "$API_BASE/sharing/share" \
  -H "Content-Type: application/json" \
  -H "X-Certificate-ID: $HOSPITAL_CERT" \
  -d "{
    \"record_id\": \"$RECORD_ID\",
    \"authorization_request_id\": \"$AUTH_REQUEST_ID\"
  }" > /dev/null
echo -e "${GREEN}✓ Record shared${NC}\n"

# Step F: Research Org Retrieves Shared Record
echo -e "${YELLOW}[Step F] Research organization retrieves shared record...${NC}"
RESPONSE=$(curl -s -X GET "$API_BASE/sharing/$RECORD_ID/$AUTH_REQUEST_ID" \
  -H "X-Certificate-ID: $RESEARCHER_CERT")
echo -e "${GREEN}✓ Record retrieved${NC}"
echo -e "${BLUE}Data:${NC}"
echo $RESPONSE | jq '.data'
echo ""

# Step G: Regulator Audit
echo -e "${YELLOW}[Step G] Regulator performs audit...${NC}"
AUDIT_LOGS=$(curl -s -X GET "$API_BASE/audit/logs" \
  -H "X-Certificate-ID: $REGULATOR_CERT")
LOG_COUNT=$(echo $AUDIT_LOGS | jq '. | length')
echo -e "${GREEN}✓ Audit complete: $LOG_COUNT events found${NC}\n"

echo -e "${BLUE}========================================${NC}"
echo -e "${GREEN}Demo completed successfully!${NC}"
echo -e "${BLUE}========================================${NC}"
echo -e "\nRecord ID: $RECORD_ID"
echo -e "Authorization Request ID: $AUTH_REQUEST_ID"
```

### Usage

```bash
chmod +x demo.sh
./demo.sh
```

---

## Troubleshooting

### Common Issues

1. **Middleware not connecting to Fabric**
   - Check `FABRIC_NETWORK_CONFIG_PATH` points to valid connection profile
   - Verify network is running: `docker ps`
   - Check middleware logs for connection errors

2. **Identity not found**
   - Verify identity is registered in database: `SELECT * FROM identity_mappings;`
   - Check certificate ID matches exactly

3. **Access denied errors**
   - Verify MSP IDs match network configuration
   - Check role assignments in identity_mappings table
   - Ensure correct certificate ID is used in headers

4. **Chaincode query fails**
   - Verify chaincode is committed: `peer lifecycle chaincode querycommitted`
   - Check peer environment variables are set correctly
   - Ensure TLS certificates are valid

5. **PDC access denied**
   - Verify collection configuration matches MSPs
   - Check that requesting organization is in collection policy
   - Ensure private data was stored correctly

---

## Conclusion

This demonstration plan provides a complete, verifiable proof that the Hyperledger Fabric medical information sharing system is working correctly. Each step includes:

- Exact commands to run
- Expected outputs
- Verification methods
- Visibility proofs

Follow the steps sequentially to demonstrate the full system functionality.
