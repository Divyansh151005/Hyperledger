# Quick Reference Guide - Medical Information Sharing Demo

## Prerequisites Checklist

- [ ] Hyperledger Fabric network is running (`docker ps` shows all containers)
- [ ] Channel `medical-main-channel` exists and peers have joined
- [ ] Chaincodes are committed: `medical-records`, `authorization-consent`, `anonymous-sharing`
- [ ] Identities are enrolled and registered in middleware database
- [ ] Middleware server is running on `localhost:8080`
- [ ] PostgreSQL database is accessible
- [ ] S3-compatible storage is accessible

## Identity Setup Quick Commands

### Register Identities in Database

```sql
INSERT INTO identity_mappings (certificate_id, msp_id, role, user_id) VALUES
('hospital1admin', 'HospitalMSP1', 'HOSPITAL', 'hospital1admin'),
('patient1', 'HospitalMSP1', 'PATIENT', 'patient1'),
('researcher1', 'ResearchOrgMSP', 'RESEARCH', 'researcher1'),
('regulator1', 'RegulatorMSP', 'REGULATOR', 'regulator1')
ON CONFLICT (certificate_id) DO UPDATE SET msp_id = EXCLUDED.msp_id, role = EXCLUDED.role, user_id = EXCLUDED.user_id;
```

## Network Verification Quick Commands

```bash
# Check containers
docker ps --format "table {{.Names}}\t{{.Status}}"

# List channels
peer channel list

# Query committed chaincodes
peer lifecycle chaincode querycommitted --channelID medical-main-channel --peerAddresses peer0.hospital1.example.com:7051 --tlsRootCertFiles /path/to/tls/ca.crt
```

## API Endpoints Quick Reference

| Endpoint | Method | Certificate ID | Purpose |
|----------|--------|----------------|---------|
| `/api/v1/records` | POST | Hospital | Create medical record |
| `/api/v1/records/:id` | GET | Hospital/Regulator | Get medical record |
| `/api/v1/authorizations/request` | POST | Research | Request access |
| `/api/v1/authorizations/:id/approve/hospital` | POST | Hospital | Hospital approval |
| `/api/v1/authorizations/:id/approve/patient` | POST | Patient | Patient approval |
| `/api/v1/authorizations/:id` | GET | Any authorized | Get authorization |
| `/api/v1/sharing/share` | POST | Hospital | Share record |
| `/api/v1/sharing/:recordID/:authID` | GET | Research/Hospital | Get shared record |
| `/api/v1/audit/logs` | GET | Regulator | Get audit logs |

## Complete Demo Flow (One Command)

```bash
./demo.sh
```

## Manual Step-by-Step Demo

### Step A: Create Record
```bash
curl -X POST http://localhost:8080/api/v1/records \
  -H "Content-Type: application/json" \
  -H "X-Certificate-ID: hospital1admin" \
  -d '{"patient_id":"patient1","data":{"diagnosis":"Type 2 Diabetes"}}'
```

### Step B: Request Access
```bash
export RECORD_ID="<from_step_a>"
curl -X POST http://localhost:8080/api/v1/authorizations/request \
  -H "Content-Type: application/json" \
  -H "X-Certificate-ID: researcher1" \
  -d "{\"record_id\":\"$RECORD_ID\",\"duration\":720}"
```

### Step C: Hospital Approves
```bash
export AUTH_ID="<from_step_b>"
curl -X POST http://localhost:8080/api/v1/authorizations/$AUTH_ID/approve/hospital \
  -H "X-Certificate-ID: hospital1admin" \
  -d '{}'
```

### Step D: Patient Approves
```bash
curl -X POST http://localhost:8080/api/v1/authorizations/$AUTH_ID/approve/patient \
  -H "X-Certificate-ID: patient1" \
  -d '{}'
```

### Step E: Share Record
```bash
curl -X POST http://localhost:8080/api/v1/sharing/share \
  -H "Content-Type: application/json" \
  -H "X-Certificate-ID: hospital1admin" \
  -d "{\"record_id\":\"$RECORD_ID\",\"authorization_request_id\":\"$AUTH_ID\"}"
```

### Step F: Retrieve Shared Record
```bash
curl -X GET http://localhost:8080/api/v1/sharing/$RECORD_ID/$AUTH_ID \
  -H "X-Certificate-ID: researcher1"
```

### Step G: Audit
```bash
curl -X GET http://localhost:8080/api/v1/audit/logs \
  -H "X-Certificate-ID: regulator1"
```

## Verification Commands

### Verify On-Chain State
```bash
# Medical Record
peer chaincode query -C medical-main-channel -n medical-records \
  -c '{"function":"GetMedicalRecord","Args":["<record_id>"]}'

# Authorization
peer chaincode query -C medical-main-channel -n authorization-consent \
  -c "{\"function\":\"GetAuthorization\",\"Args\":[\"<auth_id>\"]}"

# Shared Record (from authorized peer)
peer chaincode query -C medical-main-channel -n anonymous-sharing \
  -c "{\"function\":\"GetSharedRecord\",\"Args\":[\"<record_id>\",\"<auth_id>\"]}"
```

### Verify PDC Access Control
```bash
# From authorized peer (ResearchOrgMSP) - should succeed
export CORE_PEER_LOCALMSPID=ResearchOrgMSP
peer chaincode query -C medical-main-channel -n anonymous-sharing \
  -c "{\"function\":\"GetSharedRecord\",\"Args\":[\"<record_id>\",\"<auth_id>\"]}"

# From unauthorized peer (HospitalMSP2) - should fail
export CORE_PEER_LOCALMSPID=HospitalMSP2
peer chaincode query -C medical-main-channel -n anonymous-sharing \
  -c "{\"function\":\"GetSharedRecord\",\"Args\":[\"<record_id>\",\"<auth_id>\"]}"
```

## Expected Status Transitions

| Step | Authorization Status |
|------|---------------------|
| B: Request Access | `REQUESTED` |
| C: Hospital Approves | `HOSPITAL_APPROVED` (if patient hasn't approved) or `GRANTED` (if patient already approved) |
| D: Patient Approves | `GRANTED` |
| E: Share Record | Record shared (status remains `GRANTED`) |

## Troubleshooting Quick Fixes

| Issue | Solution |
|-------|----------|
| "identity not found" | Check `identity_mappings` table in database |
| "access denied" | Verify certificate ID matches identity role |
| "chaincode not found" | Verify chaincode is committed: `peer lifecycle chaincode querycommitted` |
| "connection refused" | Check middleware is running: `curl http://localhost:8080/health` |
| "PDC access denied" | Verify requesting MSP is in collection policy |

## Environment Variables for Peer CLI

```bash
# HospitalMSP1
export CORE_PEER_LOCALMSPID=HospitalMSP1
export CORE_PEER_MSPCONFIGPATH=/path/to/crypto-config/peerOrganizations/hospital1.example.com/users/Admin@hospital1.example.com/msp
export CORE_PEER_ADDRESS=peer0.hospital1.example.com:7051
export CORE_PEER_TLS_ROOTCERT_FILE=/path/to/crypto-config/peerOrganizations/hospital1.example.com/peers/peer0.hospital1.example.com/tls/ca.crt

# ResearchOrgMSP
export CORE_PEER_LOCALMSPID=ResearchOrgMSP
export CORE_PEER_MSPCONFIGPATH=/path/to/crypto-config/peerOrganizations/research.example.com/users/Admin@research.example.com/msp
export CORE_PEER_ADDRESS=peer0.research.example.com:9051
export CORE_PEER_TLS_ROOTCERT_FILE=/path/to/crypto-config/peerOrganizations/research.example.com/peers/peer0.research.example.com/tls/ca.crt
```

## Success Indicators

- ✅ HTTP 201 for creation endpoints
- ✅ HTTP 200 for query endpoints
- ✅ Status transitions: REQUESTED → HOSPITAL_APPROVED → GRANTED
- ✅ Shared record retrievable by research org
- ✅ Audit logs contain all events
- ✅ PDC accessible by authorized orgs only
