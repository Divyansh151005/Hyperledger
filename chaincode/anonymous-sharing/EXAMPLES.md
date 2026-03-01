# Anonymous Sharing Chaincode - Usage Examples

This document provides detailed examples for using the anonymous-sharing chaincode.

## Prerequisites

Before using the anonymous-sharing chaincode, ensure:
1. Authorization request is in `GRANTED` status (via `authorization-consent` chaincode)
2. Medical record exists (via `medical-records` chaincode)
3. Proper client identity and MSP configuration

## Example 1: Complete Sharing Flow

### Step 1: Create Medical Record

```bash
# As HospitalMSP1
peer chaincode invoke \
  -C medical-main-channel \
  -n medical-records \
  -c '{"function":"CreateMedicalRecord","Args":["REC-2024-001","patient123","a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0e1f2"]}' \
  --peerAddresses peer0.hospital1.medical-network.com:8051 \
  --tlsRootCertFiles /path/to/hospital1/tls/ca.crt
```

### Step 2: Request Access

```bash
# As ResearchOrgMSP
peer chaincode invoke \
  -C medical-main-channel \
  -n authorization-consent \
  -c '{"function":"RequestAccess","Args":["REC-2024-001","168"]}' \
  --peerAddresses peer0.research.medical-network.com:10051 \
  --tlsRootCertFiles /path/to/research/tls/ca.crt
```

**Response**: `"AUTH-REC-2024-001-1234567890123456789"`

### Step 3: Hospital Approves

```bash
# As HospitalMSP1
peer chaincode invoke \
  -C medical-main-channel \
  -n authorization-consent \
  -c '{"function":"ApproveByHospital","Args":["AUTH-REC-2024-001-1234567890123456789"]}' \
  --peerAddresses peer0.hospital1.medical-network.com:8051 \
  --tlsRootCertFiles /path/to/hospital1/tls/ca.crt
```

### Step 4: Patient Approves

```bash
# As Patient (using patient client identity)
peer chaincode invoke \
  -C medical-main-channel \
  -n authorization-consent \
  -c '{"function":"ApproveByPatient","Args":["AUTH-REC-2024-001-1234567890123456789"]}' \
  --peerAddresses peer0.hospital1.medical-network.com:8051 \
  --tlsRootCertFiles /path/to/hospital1/tls/ca.crt
```

### Step 5: Share Record

```bash
# As HospitalMSP1
# Note: encryptedPointer should be encrypted by middleware before calling
peer chaincode invoke \
  -C medical-main-channel \
  -n anonymous-sharing \
  -c '{"function":"ShareRecord","Args":["REC-2024-001","AUTH-REC-2024-001-1234567890123456789","https://ipfs.io/ipfs/QmXxxEncryptedPointer"]}' \
  --peerAddresses peer0.hospital1.medical-network.com:8051 \
  --tlsRootCertFiles /path/to/hospital1/tls/ca.crt
```

**What Happens**:
- Payload stored in Private Data Collection (visible to HospitalMSP1, ResearchOrgMSP, RegulatorMSP)
- Hash commitment stored on public ledger (visible to all)
- `RecordShared` event emitted

### Step 6: Research Organization Accesses Record

```bash
# As ResearchOrgMSP
peer chaincode query \
  -C medical-main-channel \
  -n anonymous-sharing \
  -c '{"function":"GetSharedRecord","Args":["REC-2024-001","AUTH-REC-2024-001-1234567890123456789"]}' \
  --peerAddresses peer0.research.medical-network.com:10051 \
  --tlsRootCertFiles /path/to/research/tls/ca.crt
```

**Response**:
```json
{
  "recordID": "REC-2024-001",
  "authorizationRequestID": "AUTH-REC-2024-001-1234567890123456789",
  "encryptedPointer": "https://ipfs.io/ipfs/QmXxxEncryptedPointer",
  "sharedAt": "2024-01-15T10:30:00Z",
  "sharedBy": "HospitalMSP1"
}
```

### Step 7: Verify Hash (Optional)

```bash
# As any organization
peer chaincode query \
  -C medical-main-channel \
  -n anonymous-sharing \
  -c '{"function":"VerifySharedRecord","Args":["REC-2024-001","AUTH-REC-2024-001-1234567890123456789","a3f5b8c9d2e1f4a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0"]}' \
  --peerAddresses peer0.hospital2.medical-network.com:9051 \
  --tlsRootCertFiles /path/to/hospital2/tls/ca.crt
```

**Response**: `true` or `false`

## Example 2: Regulatory Audit

### Regulator Accesses Shared Record

```bash
# As RegulatorMSP
peer chaincode query \
  -C medical-main-channel \
  -n anonymous-sharing \
  -c '{"function":"GetSharedRecord","Args":["REC-2024-001","AUTH-REC-2024-001-1234567890123456789"]}' \
  --peerAddresses peer0.regulator.medical-network.com:7051 \
  --tlsRootCertFiles /path/to/regulator/tls/ca.crt
```

**Note**: RegulatorMSP can access any shared record for regulatory oversight.

## Example 3: Error Scenarios

### Error: Authorization Not GRANTED

```bash
# Attempting to share before authorization is GRANTED
peer chaincode invoke \
  -C medical-main-channel \
  -n anonymous-sharing \
  -c '{"function":"ShareRecord","Args":["REC-2024-001","AUTH-REC-2024-001-1234567890123456789","https://ipfs.io/ipfs/QmXxx"]}' \
  --peerAddresses peer0.hospital1.medical-network.com:8051 \
  --tlsRootCertFiles /path/to/hospital1/tls/ca.crt
```

**Error**: `"authorization request AUTH-REC-2024-001-1234567890123456789 is not GRANTED (current status: REQUESTED)"`

### Error: Wrong Hospital

```bash
# HospitalMSP2 attempting to share HospitalMSP1's record
peer chaincode invoke \
  -C medical-main-channel \
  -n anonymous-sharing \
  -c '{"function":"ShareRecord","Args":["REC-2024-001","AUTH-REC-2024-001-1234567890123456789","https://ipfs.io/ipfs/QmXxx"]}' \
  --peerAddresses peer0.hospital2.medical-network.com:9051 \
  --tlsRootCertFiles /path/to/hospital2/tls/ca.crt
```

**Error**: `"access denied: only the owning hospital (HospitalMSP1) can share this record, invoker is HospitalMSP2"`

### Error: Unauthorized Access

```bash
# ResearchOrgMSP attempting to access another org's shared record
peer chaincode query \
  -C medical-main-channel \
  -n anonymous-sharing \
  -c '{"function":"GetSharedRecord","Args":["REC-2024-002","AUTH-REC-2024-002-9876543210987654321"]}' \
  --peerAddresses peer0.research.medical-network.com:10051 \
  --tlsRootCertFiles /path/to/research/tls/ca.crt
```

**Error**: `"access denied: GetSharedRecord can only be invoked by RegulatorMSP, owning HospitalMSP (HospitalMSP2), or requesting ResearchOrgMSP (ResearchOrgMSP2). Invoker is ResearchOrgMSP"`

## Example 4: Multiple Records Sharing

### Share Multiple Records for Same Authorization

```bash
# Share record REC-2024-001
peer chaincode invoke \
  -C medical-main-channel \
  -n anonymous-sharing \
  -c '{"function":"ShareRecord","Args":["REC-2024-001","AUTH-REC-2024-001-1234567890123456789","https://ipfs.io/ipfs/QmXxx1"]}' \
  --peerAddresses peer0.hospital1.medical-network.com:8051 \
  --tlsRootCertFiles /path/to/hospital1/tls/ca.crt

# Share record REC-2024-002 (different authorization)
peer chaincode invoke \
  -C medical-main-channel \
  -n anonymous-sharing \
  -c '{"function":"ShareRecord","Args":["REC-2024-002","AUTH-REC-2024-002-9876543210987654321","https://ipfs.io/ipfs/QmXxx2"]}' \
  --peerAddresses peer0.hospital1.medical-network.com:8051 \
  --tlsRootCertFiles /path/to/hospital1/tls/ca.crt
```

## Example 5: Event Monitoring

### Listen for RecordShared Events

```javascript
// Example event listener (pseudo-code)
const eventHub = channel.newChannelEventHub(peer);
const listener = async (event) => {
  if (event.eventName === 'RecordShared') {
    const payload = JSON.parse(event.payload.toString());
    console.log('Record shared:', payload);
    // Process event: recordID, authorizationRequestID, invokerMSP, etc.
  }
};
eventHub.registerChaincodeEvent('anonymous-sharing', 'RecordShared', listener);
eventHub.connect();
```

### Listen for RecordAccessed Events

```javascript
// Example event listener (pseudo-code)
const listener = async (event) => {
  if (event.eventName === 'RecordAccessed') {
    const payload = JSON.parse(event.payload.toString());
    console.log('Record accessed:', payload);
    // Audit: who accessed what record, when
  }
};
eventHub.registerChaincodeEvent('anonymous-sharing', 'RecordAccessed', listener);
```

## Example 6: Hash Verification Workflow

### Verify Integrity Without Revealing Payload

```bash
# Step 1: Get hash commitment from public ledger (any organization)
peer chaincode query \
  -C medical-main-channel \
  -n anonymous-sharing \
  -c '{"function":"GetState","Args":["SHARED_REC-2024-001_AUTH-REC-2024-001-1234567890123456789"]}' \
  --peerAddresses peer0.hospital2.medical-network.com:9051 \
  --tlsRootCertFiles /path/to/hospital2/tls/ca.crt

# Response contains payloadHash

# Step 2: Compute hash of payload (if you have access to payload)
# (This would be done by middleware that has access to the payload)

# Step 3: Verify hash matches
peer chaincode query \
  -C medical-main-channel \
  -n anonymous-sharing \
  -c '{"function":"VerifySharedRecord","Args":["REC-2024-001","AUTH-REC-2024-001-1234567890123456789","computed_hash"]}' \
  --peerAddresses peer0.hospital2.medical-network.com:9051 \
  --tlsRootCertFiles /path/to/hospital2/tls/ca.crt
```

## Example 7: Middleware Integration

### Encrypt Pointer Before Sharing

```python
# Example middleware (pseudo-code)
import encryption_library

def share_record(record_id, auth_request_id, data_pointer):
    # Encrypt the pointer
    encrypted_pointer = encryption_library.encrypt(
        data_pointer,
        key=shared_key
    )
    
    # Call chaincode
    response = invoke_chaincode(
        chaincode_name='anonymous-sharing',
        function='ShareRecord',
        args=[record_id, auth_request_id, encrypted_pointer]
    )
    
    return response
```

### Decrypt Pointer After Retrieval

```python
# Example middleware (pseudo-code)
def access_record(record_id, auth_request_id):
    # Query chaincode
    payload = query_chaincode(
        chaincode_name='anonymous-sharing',
        function='GetSharedRecord',
        args=[record_id, auth_request_id]
    )
    
    # Decrypt the pointer
    decrypted_pointer = encryption_library.decrypt(
        payload['encryptedPointer'],
        key=shared_key
    )
    
    # Access actual data using decrypted pointer
    actual_data = fetch_data(decrypted_pointer)
    
    return actual_data
```

## Example 8: Batch Operations

### Share Multiple Records (Sequential)

```bash
# Share multiple records one by one
for record_id in REC-2024-001 REC-2024-002 REC-2024-003; do
  peer chaincode invoke \
    -C medical-main-channel \
    -n anonymous-sharing \
    -c "{\"function\":\"ShareRecord\",\"Args\":[\"$record_id\",\"AUTH-$record_id-123\",\"https://ipfs.io/ipfs/QmXxx$record_id\"]}" \
    --peerAddresses peer0.hospital1.medical-network.com:8051 \
    --tlsRootCertFiles /path/to/hospital1/tls/ca.crt
done
```

## Example 9: Query Public Commitments

### View All Hash Commitments (Public Ledger)

```bash
# Query all shared record commitments (public data)
peer chaincode query \
  -C medical-main-channel \
  -n anonymous-sharing \
  -c '{"function":"GetStateByRange","Args":["SHARED_", "SHARED_zzz"]}' \
  --peerAddresses peer0.hospital2.medical-network.com:9051 \
  --tlsRootCertFiles /path/to/hospital2/tls/ca.crt
```

**Note**: This returns hash commitments (public), not the actual payloads (private).

## Example 10: Access Control Verification

### Test Access Control

```bash
# Test 1: HospitalMSP1 can share their own records
peer chaincode invoke \
  -C medical-main-channel \
  -n anonymous-sharing \
  -c '{"function":"ShareRecord","Args":["REC-2024-001","AUTH-REC-2024-001-123","https://ipfs.io/ipfs/QmXxx"]}' \
  --peerAddresses peer0.hospital1.medical-network.com:8051 \
  --tlsRootCertFiles /path/to/hospital1/tls/ca.crt
# Expected: Success

# Test 2: HospitalMSP2 cannot share HospitalMSP1's records
peer chaincode invoke \
  -C medical-main-channel \
  -n anonymous-sharing \
  -c '{"function":"ShareRecord","Args":["REC-2024-001","AUTH-REC-2024-001-123","https://ipfs.io/ipfs/QmXxx"]}' \
  --peerAddresses peer0.hospital2.medical-network.com:9051 \
  --tlsRootCertFiles /path/to/hospital2/tls/ca.crt
# Expected: Access denied error

# Test 3: ResearchOrgMSP can access their requested records
peer chaincode query \
  -C medical-main-channel \
  -n anonymous-sharing \
  -c '{"function":"GetSharedRecord","Args":["REC-2024-001","AUTH-REC-2024-001-123"]}' \
  --peerAddresses peer0.research.medical-network.com:10051 \
  --tlsRootCertFiles /path/to/research/tls/ca.crt
# Expected: Success (if they requested it)

# Test 4: RegulatorMSP can access any record
peer chaincode query \
  -C medical-main-channel \
  -n anonymous-sharing \
  -c '{"function":"GetSharedRecord","Args":["REC-2024-001","AUTH-REC-2024-001-123"]}' \
  --peerAddresses peer0.regulator.medical-network.com:7051 \
  --tlsRootCertFiles /path/to/regulator/tls/ca.crt
# Expected: Success
```

## Notes

1. **Encryption**: The `encryptedPointer` parameter should be encrypted by middleware before calling `ShareRecord`
2. **Authorization**: Authorization must be `GRANTED` before sharing
3. **Expiration**: Expired authorizations cannot be used for sharing
4. **Privacy**: Private data is only visible to authorized organizations (HospitalMSP, ResearchOrgMSP, RegulatorMSP)
5. **Public Data**: Hash commitments are visible to all channel members
6. **Events**: All operations emit events for auditability
