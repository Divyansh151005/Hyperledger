# Medical Records Chaincode

## Overview

This chaincode implements a blockchain-based medical information sharing system for the Hyperledger Fabric network. It provides secure, auditable, and compliant management of medical record metadata with MSP-based access control.

**Chaincode Name**: `medical-records`  
**Language**: Go  
**Fabric Version**: v2.x  
**Channel**: `medical-main-channel`

## Design Principles

### 1. Metadata-Only Storage
- **No raw medical data on-chain**: Only metadata (recordID, patientID, hospitalID, recordHash, timestamp, status) is stored
- **Off-chain data storage**: Actual medical records are stored off-chain, with only their hash stored on the blockchain
- **Privacy compliance**: Aligns with HIPAA, GDPR, and other healthcare privacy regulations

### 2. MSP-Based Access Control
- **Chaincode-level enforcement**: Access control is enforced within the chaincode using Client Identity (CID) APIs
- **Not relying on endorsement policies alone**: While endorsement policies control transaction validation, chaincode-level checks provide fine-grained access control
- **MSP verification**: Every function verifies the invoker's MSP ID before processing

### 3. Auditability
- **Event emission**: All state-changing operations emit events for external systems to track
- **Complete audit trail**: Events include invoker MSP, timestamp, and relevant record information
- **Regulatory compliance**: Enables compliance with healthcare audit requirements

## Data Model

### MedicalRecord Structure

```json
{
  "recordID": "REC-2024-001",
  "patientID": "PAT-12345",
  "hospitalID": "HospitalMSP1",
  "recordHash": "a1b2c3d4e5f6...",
  "timestamp": "2024-01-15T10:30:00Z",
  "status": "ACTIVE"
}
```

**Fields**:
- `recordID`: Unique identifier for the medical record (used as ledger key)
- `patientID`: Patient identifier (should be pseudonymized for privacy)
- `hospitalID`: MSP ID of the hospital that created the record
- `recordHash`: SHA-256 hash of the off-chain medical record (64-character hex string)
- `timestamp`: ISO 8601 timestamp of record creation
- `status`: Record status (`ACTIVE` or `REVOKED`)

## Chaincode Functions

### 1. CreateMedicalRecord

Creates a new medical record metadata entry on the ledger.

**Function Signature**:
```go
CreateMedicalRecord(ctx, recordID, patientID, recordHash)
```

**Parameters**:
- `recordID` (string): Unique identifier for the medical record
- `patientID` (string): Patient identifier
- `recordHash` (string): SHA-256 hash of the off-chain medical record (64-character hex)

**Access Control**:
- ✅ **Allowed**: HospitalMSP1, HospitalMSP2
- ❌ **Denied**: RegulatorMSP, ResearchOrgMSP, any other MSP

**Validation**:
- Verifies invoker is a Hospital MSP
- Checks recordID doesn't already exist
- Validates all parameters are non-empty
- Validates recordHash format (64-character hex string)

**Event**: Emits `MedicalRecordCreated` event

**Example**:
```bash
peer chaincode invoke \
  -C medical-main-channel \
  -n medical-records \
  -c '{"function":"CreateMedicalRecord","Args":["REC-2024-001","PAT-12345","a1b2c3d4e5f6789012345678901234567890123456789012345678901234567890"]}' \
  --peerAddresses peer0.hospital1.medical-network.com:8051 \
  --tlsRootCertFiles /path/to/tls/cert
```

### 2. GetMedicalRecord

Retrieves a medical record from the ledger (read-only).

**Function Signature**:
```go
GetMedicalRecord(ctx, recordID)
```

**Parameters**:
- `recordID` (string): Unique identifier for the medical record

**Access Control**:
- ✅ **Allowed**: 
  - RegulatorMSP (can access any record - regulatory oversight)
  - HospitalMSP that created the record (can only access own records)
- ❌ **Denied**: 
  - Other HospitalMSPs (cannot access records created by other hospitals)
  - ResearchOrgMSP
  - Any other MSP

**Returns**: MedicalRecord JSON object

**Example**:
```bash
peer chaincode query \
  -C medical-main-channel \
  -n medical-records \
  -c '{"function":"GetMedicalRecord","Args":["REC-2024-001"]}' \
  --peerAddresses peer0.regulator.medical-network.com:7051 \
  --tlsRootCertFiles /path/to/tls/cert
```

### 3. VerifyMedicalRecordHash

Verifies if a provided hash matches the stored hash for a medical record.

**Function Signature**:
```go
VerifyMedicalRecordHash(ctx, recordID, providedHash)
```

**Parameters**:
- `recordID` (string): Unique identifier for the medical record
- `providedHash` (string): The hash to verify against the stored record hash

**Access Control**:
- Same as `GetMedicalRecord`:
  - ✅ RegulatorMSP (can verify any record)
  - ✅ HospitalMSP that created the record (can only verify own records)
  - ❌ Other MSPs

**Returns**: `true` if hashes match, `false` otherwise

**Use Case**: Verify integrity of off-chain medical data by comparing hashes

**Example**:
```bash
peer chaincode query \
  -C medical-main-channel \
  -n medical-records \
  -c '{"function":"VerifyMedicalRecordHash","Args":["REC-2024-001","a1b2c3d4e5f6789012345678901234567890123456789012345678901234567890"]}' \
  --peerAddresses peer0.hospital1.medical-network.com:8051 \
  --tlsRootCertFiles /path/to/tls/cert
```

## Access Control Architecture

### Why Chaincode-Level Access Control?

While Hyperledger Fabric provides endorsement policies at the network level, **chaincode-level access control** provides several critical benefits:

1. **Fine-Grained Control**: Endorsement policies control *who can endorse*, but chaincode controls *who can invoke specific functions*
2. **Function-Specific Rules**: Different functions can have different access rules (e.g., only hospitals can create, but regulators can read)
3. **Ownership-Based Access**: Hospitals can only access their own records, not records from other hospitals
4. **Defense in Depth**: Multiple layers of security (endorsement policy + chaincode checks)

### Access Control Implementation

The chaincode uses Fabric's **Client Identity (CID) API** to retrieve the invoker's MSP ID:

```go
clientIdentity, err := ctx.GetClientIdentity().GetMSPID()
```

This MSP ID is then checked against the access control rules for each function.

### Access Control Matrix

| Function | RegulatorMSP | HospitalMSP1 | HospitalMSP2 | ResearchOrgMSP |
|----------|-------------|--------------|--------------|----------------|
| CreateMedicalRecord | ❌ | ✅ | ✅ | ❌ |
| GetMedicalRecord | ✅ (all) | ✅ (own only) | ✅ (own only) | ❌ |
| VerifyMedicalRecordHash | ✅ (all) | ✅ (own only) | ✅ (own only) | ❌ |

**Legend**:
- ✅ = Allowed
- ❌ = Denied
- (all) = Can access any record
- (own only) = Can only access records created by that MSP

## Events

### MedicalRecordCreated Event

Emitted when a new medical record is created.

**Event Name**: `MedicalRecordCreated`

**Event Payload**:
```json
{
  "recordID": "REC-2024-001",
  "action": "MedicalRecordCreated",
  "invokerMSP": "HospitalMSP1",
  "timestamp": "2024-01-15T10:30:00Z",
  "patientID": "PAT-12345",
  "hospitalID": "HospitalMSP1"
}
```

**Use Cases**:
- External audit systems can listen to events
- Compliance monitoring
- Real-time notifications
- Analytics and reporting

## Transaction Flow Examples

### Example 1: Hospital Creates Medical Record

**Scenario**: HospitalMSP1 creates a medical record for patient PAT-12345

**Flow**:
1. Client application (authenticated as HospitalMSP1) prepares transaction
2. Transaction is sent to peer0.hospital1.medical-network.com
3. Chaincode validates:
   - ✅ Invoker is HospitalMSP1 (access control check)
   - ✅ RecordID doesn't exist
   - ✅ Parameters are valid
4. Chaincode creates MedicalRecord object
5. Chaincode stores record in world state (key: recordID)
6. Chaincode emits `MedicalRecordCreated` event
7. Transaction is endorsed and submitted to ordering service
8. Transaction is committed to all peers

**Command**:
```bash
peer chaincode invoke \
  -C medical-main-channel \
  -n medical-records \
  -c '{"function":"CreateMedicalRecord","Args":["REC-2024-001","PAT-12345","a1b2c3d4e5f6789012345678901234567890123456789012345678901234567890"]}' \
  --peerAddresses peer0.hospital1.medical-network.com:8051 \
  --tlsRootCertFiles crypto-config/peerOrganizations/hospital1.medical-network.com/peers/peer0.hospital1.medical-network.com/tls/ca.crt
```

### Example 2: Regulator Accesses Medical Record

**Scenario**: RegulatorMSP queries a medical record for audit purposes

**Flow**:
1. Client application (authenticated as RegulatorMSP) prepares query
2. Query is sent to peer0.regulator.medical-network.com
3. Chaincode validates:
   - ✅ Invoker is RegulatorMSP (has access to all records)
4. Chaincode retrieves record from world state
5. Chaincode returns MedicalRecord JSON

**Command**:
```bash
peer chaincode query \
  -C medical-main-channel \
  -n medical-records \
  -c '{"function":"GetMedicalRecord","Args":["REC-2024-001"]}' \
  --peerAddresses peer0.regulator.medical-network.com:7051 \
  --tlsRootCertFiles crypto-config/peerOrganizations/regulator.medical-network.com/peers/peer0.regulator.medical-network.com/tls/ca.crt
```

### Example 3: Hospital Verifies Record Integrity

**Scenario**: HospitalMSP1 verifies that off-chain medical data matches the stored hash

**Flow**:
1. Hospital computes SHA-256 hash of off-chain medical record
2. Client application (authenticated as HospitalMSP1) calls VerifyMedicalRecordHash
3. Chaincode validates:
   - ✅ Invoker is HospitalMSP1 (can verify own records)
   - ✅ Record exists and is owned by HospitalMSP1
4. Chaincode compares provided hash with stored hash
5. Chaincode returns `true` if hashes match, `false` otherwise

**Command**:
```bash
peer chaincode query \
  -C medical-main-channel \
  -n medical-records \
  -c '{"function":"VerifyMedicalRecordHash","Args":["REC-2024-001","computed_hash_here"]}' \
  --peerAddresses peer0.hospital1.medical-network.com:8051 \
  --tlsRootCertFiles crypto-config/peerOrganizations/hospital1.medical-network.com/peers/peer0.hospital1.medical-network.com/tls/ca.crt
```

### Example 4: Unauthorized Access Attempt

**Scenario**: ResearchOrgMSP attempts to create a medical record

**Flow**:
1. Client application (authenticated as ResearchOrgMSP) prepares transaction
2. Transaction is sent to peer0.research.medical-network.com
3. Chaincode validates:
   - ❌ Invoker is ResearchOrgMSP (NOT a Hospital MSP)
4. Chaincode returns error: `"access denied: CreateMedicalRecord can only be invoked by HospitalMSP organizations"`
5. Transaction fails

## Deployment

### Prerequisites

- Hyperledger Fabric v2.x network running
- Channel `medical-main-channel` created and peers joined
- Go 1.20+ installed
- Fabric chaincode dependencies

### Build Chaincode

```bash
cd chaincode/medical-records
go mod download
go mod vendor  # Optional: vendor dependencies
```

### Package Chaincode

```bash
peer lifecycle chaincode package medical-records.tar.gz \
  --path ./chaincode/medical-records \
  --lang golang \
  --label medical-records_1.0
```

### Install Chaincode

Install on all peer organizations:

```bash
# Install on RegulatorMSP peer
peer lifecycle chaincode install medical-records.tar.gz

# Install on HospitalMSP1 peer
peer lifecycle chaincode install medical-records.tar.gz

# Install on HospitalMSP2 peer
peer lifecycle chaincode install medical-records.tar.gz

# Install on ResearchOrgMSP peer
peer lifecycle chaincode install medical-records.tar.gz
```

### Approve Chaincode

Approve from each organization (example for RegulatorMSP):

```bash
peer lifecycle chaincode approveformyorg \
  -o orderer0.medical-network.com:7050 \
  --channelID medical-main-channel \
  --name medical-records \
  --version 1.0 \
  --package-id <PACKAGE_ID> \
  --sequence 1 \
  --tls \
  --cafile /path/to/orderer/tls/ca.crt
```

### Commit Chaincode

```bash
peer lifecycle chaincode commit \
  -o orderer0.medical-network.com:7050 \
  --channelID medical-main-channel \
  --name medical-records \
  --version 1.0 \
  --sequence 1 \
  --tls \
  --cafile /path/to/orderer/tls/ca.crt \
  --peerAddresses peer0.regulator.medical-network.com:7051 \
  --peerAddresses peer0.hospital1.medical-network.com:8051 \
  --peerAddresses peer0.hospital2.medical-network.com:9051 \
  --peerAddresses peer0.research.medical-network.com:10051 \
  --tlsRootCertFiles /path/to/regulator/tls/ca.crt \
  --tlsRootCertFiles /path/to/hospital1/tls/ca.crt \
  --tlsRootCertFiles /path/to/hospital2/tls/ca.crt \
  --tlsRootCertFiles /path/to/research/tls/ca.crt
```

## Testing

### Test CreateMedicalRecord

```bash
# As HospitalMSP1
peer chaincode invoke \
  -C medical-main-channel \
  -n medical-records \
  -c '{"function":"CreateMedicalRecord","Args":["REC-TEST-001","PAT-TEST-001","a1b2c3d4e5f6789012345678901234567890123456789012345678901234567890"]}' \
  --peerAddresses peer0.hospital1.medical-network.com:8051 \
  --tlsRootCertFiles /path/to/tls/ca.crt
```

### Test GetMedicalRecord

```bash
# As RegulatorMSP
peer chaincode query \
  -C medical-main-channel \
  -n medical-records \
  -c '{"function":"GetMedicalRecord","Args":["REC-TEST-001"]}' \
  --peerAddresses peer0.regulator.medical-network.com:7051 \
  --tlsRootCertFiles /path/to/tls/ca.crt
```

### Test VerifyMedicalRecordHash

```bash
# As HospitalMSP1 (owner)
peer chaincode query \
  -C medical-main-channel \
  -n medical-records \
  -c '{"function":"VerifyMedicalRecordHash","Args":["REC-TEST-001","a1b2c3d4e5f6789012345678901234567890123456789012345678901234567890"]}' \
  --peerAddresses peer0.hospital1.medical-network.com:8051 \
  --tlsRootCertFiles /path/to/tls/ca.crt
```

## Security Considerations

### Access Control
- ✅ MSP-based access control enforced at chaincode level
- ✅ Client Identity API used to verify invoker MSP
- ✅ Function-specific access rules
- ✅ Ownership-based record access

### Data Privacy
- ✅ Only metadata stored on-chain (no raw medical data)
- ✅ Patient IDs should be pseudonymized
- ✅ Hash-based integrity verification

### Auditability
- ✅ All state-changing operations emit events
- ✅ Events include invoker MSP and timestamp
- ✅ Complete transaction history on ledger

### Future Enhancements
- Private data collections for sensitive metadata
- Encryption for additional security layers
- Patient consent management
- Record revocation functionality

## Compliance Alignment

This chaincode design aligns with:

- **HIPAA**: Privacy and security of health information
- **GDPR**: Right to be forgotten (status field supports REVOKED)
- **Medical Data Privacy Laws**: Metadata-only storage, access control
- **Regulatory Oversight**: RegulatorMSP can access all records for compliance

## Extensibility

The chaincode is designed for future extensions:

1. **Record Revocation**: Status field supports REVOKED status (functionality can be added)
2. **Additional Hospitals**: `isHospitalMSP()` function can be extended
3. **Research Access**: ResearchOrgMSP access can be added with appropriate functions
4. **Patient Consent**: Patient identity management can be integrated
5. **Encryption**: Additional encryption layers can be added

## Troubleshooting

### Access Denied Errors

If you receive "access denied" errors:
1. Verify the client identity is correctly authenticated
2. Check that the MSP ID matches expected values (HospitalMSP1, HospitalMSP2, RegulatorMSP)
3. For GetMedicalRecord, ensure the record is owned by the invoking HospitalMSP (or use RegulatorMSP)

### Hash Verification Failures

If hash verification fails:
1. Ensure the hash is computed correctly (SHA-256, hex-encoded)
2. Verify the hash format (64-character hex string)
3. Check that the record exists and is accessible

### Event Not Received

If events are not received:
1. Verify the transaction was successful (check transaction status)
2. Ensure event listeners are properly configured
3. Check that the event name matches: `MedicalRecordCreated`

## References

- [Hyperledger Fabric Chaincode Documentation](https://hyperledger-fabric.readthedocs.io/en/latest/chaincode4ade.html)
- [Fabric Contract API Go](https://github.com/hyperledger/fabric-contract-api-go)
- [Client Identity Library](https://github.com/hyperledger/fabric-chaincode-go)
