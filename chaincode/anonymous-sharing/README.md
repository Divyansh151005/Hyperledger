# Anonymous Information Sharing Chaincode

## Overview

This chaincode implements **supervised anonymous information sharing** using Private Data Collections (PDCs) on Hyperledger Fabric v2.x, aligned with academic medical blockchain paper requirements.

**Chaincode Name**: `anonymous-sharing`  
**Language**: Go  
**Fabric Version**: v2.x  
**Channel**: `medical-main-channel`

## Design Principles

### 1. Privacy by Design
- **Private Data Collections**: Sensitive sharing payloads stored in PDCs, invisible to unrelated organizations
- **Hash Commitments**: Only hash + metadata stored on public ledger
- **Access Control**: MSP-based access control enforced at chaincode level
- **Regulatory Oversight**: RegulatorMSP can audit all transactions and access payloads

### 2. Authorization Enforcement
- **Pre-requisite**: Authorization status must be `GRANTED` before data sharing
- **Integration**: Verifies authorization via `authorization-consent` chaincode
- **Expiration Check**: Validates authorization has not expired
- **Ownership Verification**: Only owning HospitalMSP can share records

### 3. Supervised Sharing
- **Hospital Control**: Only HospitalMSP organizations can initiate sharing
- **Research Access**: ResearchOrgMSP can access shared records for which they have authorization
- **Regulatory Audit**: RegulatorMSP can access all shared records for compliance

### 4. Full Auditability
- **Event Emission**: All share and access operations emit events
- **Complete Context**: Events include invoker MSP, client ID, timestamp, recordID
- **Compliance Tracking**: Enables regulatory compliance and audit requirements

## Privacy Model

### Private Data Collection Architecture

**Collection Name**: `shared_records_collection`  
*(Note: Requirement specified pattern `collection_record_<recordID>`, but Fabric requires static collection definitions. A single collection with recordID in keys provides equivalent privacy guarantees.)*

**Member Organizations**:
- Owning HospitalMSP (HospitalMSP1 or HospitalMSP2)
- Requesting ResearchOrgMSP
- RegulatorMSP

**Access Control**:
- **Read/Write**: Restricted to member organizations only
- **Others**: See only hash commitments on public ledger
- **Privacy Guarantee**: Unrelated organizations cannot view shared content

### Data Separation

#### Private Data (PDC)
Stored in Private Data Collection, visible only to authorized organizations:

```json
{
  "recordID": "REC-2024-001",
  "authorizationRequestID": "AUTH-REC-2024-001-1234567890",
  "encryptedPointer": "https://ipfs.io/ipfs/QmXxx...",
  "sharedAt": "2024-01-15T10:30:00Z",
  "sharedBy": "HospitalMSP1"
}
```

#### Public Ledger (Hash Commitment)
Stored on public ledger, visible to all channel members:

```json
{
  "recordID": "REC-2024-001",
  "authorizationRequestID": "AUTH-REC-2024-001-1234567890",
  "payloadHash": "a3f5b8c9d2e1f4a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

## Data Model

### SharedRecordPayload (Private Data)

Stored in Private Data Collection, accessible only to authorized organizations.

| Field | Type | Description |
|-------|------|-------------|
| `recordID` | string | Unique identifier for the medical record |
| `authorizationRequestID` | string | ID of the authorization request (must be GRANTED) |
| `encryptedPointer` | string | Encrypted pointer to actual data (URL/IPFS, encrypted by middleware) |
| `sharedAt` | string | ISO 8601 timestamp when record was shared |
| `sharedBy` | string | MSP ID of the hospital that shared the record |

### SharedRecordCommitment (Public Ledger)

Stored on public ledger, visible to all channel members.

| Field | Type | Description |
|-------|------|-------------|
| `recordID` | string | Unique identifier for the medical record |
| `authorizationRequestID` | string | ID of the authorization request |
| `payloadHash` | string | SHA-256 hash of the SharedRecordPayload |
| `timestamp` | string | ISO 8601 timestamp |

## Chaincode Functions

### 1. ShareRecord

Shares a medical record with authorized research organization.

**Function Signature**:
```go
ShareRecord(ctx, recordID, authorizationRequestID, encryptedPointer) error
```

**Parameters**:
- `recordID` (string): Unique identifier for the medical record
- `authorizationRequestID` (string): ID of the authorization request (must be GRANTED)
- `encryptedPointer` (string): Encrypted pointer to actual data (URL/IPFS, already encrypted by middleware)

**Access Control**:
- ✅ **Allowed**: HospitalMSP organizations only
- ❌ **Denied**: All other MSPs (RegulatorMSP, ResearchOrgMSP, patients)

**Validation**:
- Verifies invoker is HospitalMSP
- Verifies authorization status is `GRANTED`
- Verifies recordID matches authorization request
- Verifies invoker is the owning hospital
- Checks authorization has not expired
- Prevents duplicate sharing

**Privacy Operations**:
1. Creates `SharedRecordPayload` with encrypted pointer
2. Computes SHA-256 hash of payload
3. Stores payload in Private Data Collection
4. Stores hash commitment on public ledger

**Event**: Emits `RecordShared` event

**Example**:
```bash
peer chaincode invoke \
  -C medical-main-channel \
  -n anonymous-sharing \
  -c '{"function":"ShareRecord","Args":["REC-2024-001","AUTH-REC-2024-001-1234567890","https://ipfs.io/ipfs/QmXxx..."]}' \
  --peerAddresses peer0.hospital1.medical-network.com:8051 \
  --tlsRootCertFiles /path/to/tls/ca.crt
```

### 2. GetSharedRecord

Retrieves a shared record from the Private Data Collection.

**Function Signature**:
```go
GetSharedRecord(ctx, recordID, authorizationRequestID) (*SharedRecordPayload, error)
```

**Parameters**:
- `recordID` (string): Unique identifier for the medical record
- `authorizationRequestID` (string): ID of the authorization request

**Access Control**:
- ✅ **Allowed**:
  - Owning HospitalMSP (can access records they shared)
  - Requesting ResearchOrgMSP (can access records for which they have authorization)
  - RegulatorMSP (can access any shared record - regulatory oversight)
- ❌ **Denied**: 
  - Other HospitalMSPs (cannot access other hospitals' shared records)
  - Other ResearchOrgs (cannot access records they didn't request)

**Privacy**:
- Reads from Private Data Collection
- Only accessible to authorized organizations
- Emits access event for auditability

**Event**: Emits `RecordAccessed` event

**Example**:
```bash
peer chaincode query \
  -C medical-main-channel \
  -n anonymous-sharing \
  -c '{"function":"GetSharedRecord","Args":["REC-2024-001","AUTH-REC-2024-001-1234567890"]}' \
  --peerAddresses peer0.research.medical-network.com:10051 \
  --tlsRootCertFiles /path/to/tls/ca.crt
```

### 3. VerifySharedRecord

Verifies if a provided hash matches the stored hash commitment.

**Function Signature**:
```go
VerifySharedRecord(ctx, recordID, authorizationRequestID, providedHash) (bool, error)
```

**Parameters**:
- `recordID` (string): Unique identifier for the medical record
- `authorizationRequestID` (string): ID of the authorization request
- `providedHash` (string): The hash to verify against the stored commitment

**Access Control**:
- ✅ **Public Function**: Any organization can verify hash commitments
- **Purpose**: Enables integrity verification without revealing payload

**Privacy**:
- Only reads from public ledger (hash commitments)
- Does not access Private Data Collection
- Enables verification without revealing sensitive data

**Returns**: `true` if hashes match, `false` otherwise, or error

**Example**:
```bash
peer chaincode query \
  -C medical-main-channel \
  -n anonymous-sharing \
  -c '{"function":"VerifySharedRecord","Args":["REC-2024-001","AUTH-REC-2024-001-1234567890","a3f5b8c9..."]}' \
  --peerAddresses peer0.hospital2.medical-network.com:9051 \
  --tlsRootCertFiles /path/to/tls/ca.crt
```

## Access Control Matrix

| Function | RegulatorMSP | HospitalMSP (own) | HospitalMSP (other) | ResearchOrgMSP (own request) | ResearchOrgMSP (other) |
|----------|-------------|-------------------|---------------------|------------------------------|------------------------|
| **ShareRecord** | ❌ | ✅ | ❌ | ❌ | ❌ |
| **GetSharedRecord** | ✅ (all) | ✅ (own) | ❌ | ✅ (own requests) | ❌ |
| **VerifySharedRecord** | ✅ | ✅ | ✅ | ✅ | ✅ |

**Legend**:
- ✅ = Allowed
- ❌ = Denied
- (own) = Only for own records/requests
- (all) = Can access any shared record

## Events

### RecordShared Event

Emitted when a record is shared.

**Event Name**: `RecordShared`

**Event Payload**:
```json
{
  "recordID": "REC-2024-001",
  "authorizationRequestID": "AUTH-REC-2024-001-1234567890",
  "action": "RecordShared",
  "invokerMSP": "HospitalMSP1",
  "clientID": "x509::CN=doctor1...",
  "timestamp": "2024-01-15T10:30:00Z",
  "payloadHash": "a3f5b8c9d2e1f4a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0"
}
```

### RecordAccessed Event

Emitted when a shared record is accessed.

**Event Name**: `RecordAccessed`

**Event Payload**:
```json
{
  "recordID": "REC-2024-001",
  "authorizationRequestID": "AUTH-REC-2024-001-1234567890",
  "action": "RecordAccessed",
  "invokerMSP": "ResearchOrgMSP",
  "clientID": "x509::CN=researcher1...",
  "timestamp": "2024-01-15T11:00:00Z"
}
```

## End-to-End Anonymous Sharing Flow

### Complete Workflow Example

**Scenario**: ResearchOrgMSP requests access, hospital and patient approve, hospital shares record, research org accesses it.

#### Step 1: Research Organization Requests Access

```bash
# As ResearchOrgMSP (via authorization-consent chaincode)
peer chaincode invoke \
  -C medical-main-channel \
  -n authorization-consent \
  -c '{"function":"RequestAccess","Args":["REC-2024-001","168"]}' \
  --peerAddresses peer0.research.medical-network.com:10051 \
  --tlsRootCertFiles /path/to/tls/ca.crt
```

**Result**: Authorization request created with status `REQUESTED`, requestID: `AUTH-REC-2024-001-1234567890`

#### Step 2: Hospital Approves

```bash
# As HospitalMSP1 (owner of REC-2024-001)
peer chaincode invoke \
  -C medical-main-channel \
  -n authorization-consent \
  -c '{"function":"ApproveByHospital","Args":["AUTH-REC-2024-001-1234567890"]}' \
  --peerAddresses peer0.hospital1.medical-network.com:8051 \
  --tlsRootCertFiles /path/to/tls/ca.crt
```

**Result**: Status changes to `HOSPITAL_APPROVED`

#### Step 3: Patient Approves

```bash
# As Patient (using patient client identity)
peer chaincode invoke \
  -C medical-main-channel \
  -n authorization-consent \
  -c '{"function":"ApproveByPatient","Args":["AUTH-REC-2024-001-1234567890"]}' \
  --peerAddresses peer0.hospital1.medical-network.com:8051 \
  --tlsRootCertFiles /path/to/tls/ca.crt
```

**Result**: Status changes to `GRANTED` (both parties approved)

#### Step 4: Hospital Shares Record

```bash
# As HospitalMSP1 (owner of REC-2024-001)
# encryptedPointer is already encrypted by middleware (e.g., IPFS URL with encryption)
peer chaincode invoke \
  -C medical-main-channel \
  -n anonymous-sharing \
  -c '{"function":"ShareRecord","Args":["REC-2024-001","AUTH-REC-2024-001-1234567890","https://ipfs.io/ipfs/QmXxxEncryptedPointer"]}' \
  --peerAddresses peer0.hospital1.medical-network.com:8051 \
  --tlsRootCertFiles /path/to/tls/ca.crt
```

**Result**: 
- Payload stored in Private Data Collection (visible to HospitalMSP1, ResearchOrgMSP, RegulatorMSP)
- Hash commitment stored on public ledger (visible to all)
- `RecordShared` event emitted

#### Step 5: Research Organization Accesses Record

```bash
# As ResearchOrgMSP
peer chaincode query \
  -C medical-main-channel \
  -n anonymous-sharing \
  -c '{"function":"GetSharedRecord","Args":["REC-2024-001","AUTH-REC-2024-001-1234567890"]}' \
  --peerAddresses peer0.research.medical-network.com:10051 \
  --tlsRootCertFiles /path/to/tls/ca.crt
```

**Result**: 
- Returns `SharedRecordPayload` with encrypted pointer
- `RecordAccessed` event emitted
- Research org can decrypt and access the actual data using the encrypted pointer

#### Step 6: Verify Hash (Optional - Any Organization)

```bash
# As any organization (e.g., HospitalMSP2)
peer chaincode query \
  -C medical-main-channel \
  -n anonymous-sharing \
  -c '{"function":"VerifySharedRecord","Args":["REC-2024-001","AUTH-REC-2024-001-1234567890","a3f5b8c9..."]}' \
  --peerAddresses peer0.hospital2.medical-network.com:9051 \
  --tlsRootCertFiles /path/to/tls/ca.crt
```

**Result**: Returns `true` if hash matches, `false` otherwise

## Privacy Guarantees vs Academic Paper's Anonymous Sharing

### Alignment with Academic Paper Requirements

This implementation aligns with academic medical blockchain paper requirements for **supervised anonymous information sharing**:

#### 1. **Privacy Boundaries**
- ✅ **Private Data Collections**: Sensitive payloads stored in PDCs, invisible to unrelated organizations
- ✅ **Hash Commitments**: Only hash + metadata on public ledger
- ✅ **Access Control**: MSP-based access control enforced at chaincode level
- ✅ **Unrelated Organizations**: See only hash commitments, cannot view shared content

#### 2. **Supervision Model**
- ✅ **Authorization Enforcement**: Requires `GRANTED` authorization before sharing
- ✅ **Hospital Control**: Only HospitalMSP can initiate sharing
- ✅ **Regulatory Oversight**: RegulatorMSP can audit all transactions and access payloads
- ✅ **Multi-Party Authorization**: Integrates with authorization-consent chaincode

#### 3. **Anonymity Properties**
- ✅ **Content Anonymity**: Unrelated organizations cannot view shared content (only hash)
- ✅ **Metadata Privacy**: Authorization metadata visible only to authorized parties
- ✅ **Audit Trail**: Complete audit trail for regulatory compliance
- ✅ **Integrity Verification**: Hash commitments enable verification without revealing payload

#### 4. **Regulatory Compliance**
- ✅ **Full Auditability**: All share and access operations emit events
- ✅ **Regulatory Access**: RegulatorMSP can access all shared records
- ✅ **Compliance Tracking**: Events include invoker MSP, client ID, timestamp
- ✅ **Authorization Tracking**: Links sharing to authorization requests

### Differences from Pure Anonymous Sharing

This implementation provides **supervised anonymous sharing** rather than pure anonymous sharing:

1. **Supervision**: RegulatorMSP can access all shared records (regulatory oversight)
2. **Authorization**: Requires explicit authorization before sharing (not fully anonymous)
3. **Audit Trail**: Complete audit trail (trades some anonymity for compliance)
4. **Access Control**: MSP-based access control (identifies organizations, not individuals)

### Privacy Guarantees

#### What is Private?
- **Shared Record Payloads**: Stored in PDC, visible only to:
  - Owning HospitalMSP
  - Requesting ResearchOrgMSP
  - RegulatorMSP
- **Encrypted Pointers**: Encrypted by middleware before storage

#### What is Public?
- **Hash Commitments**: Visible to all channel members
- **Metadata**: recordID, authorizationRequestID, timestamp
- **Events**: Share and access events (for auditability)

#### What Unrelated Organizations See?
- **Hash Commitments**: Can verify integrity but cannot view content
- **Metadata**: recordID, authorizationRequestID, timestamp
- **Cannot Access**: Private data payloads, encrypted pointers

## Private Data Collection Configuration

### Collection Definition

The chaincode uses a single Private Data Collection: `shared_records_collection`

**Configuration** (`collections_config.json`):
```json
[
  {
    "name": "shared_records_collection",
    "policy": "OR('HospitalMSP1.member', 'HospitalMSP2.member', 'ResearchOrgMSP.member', 'RegulatorMSP.member')",
    "requiredPeerCount": 0,
    "maxPeerCount": 3,
    "blockToLive": 0,
    "memberOnlyRead": true,
    "memberOnlyWrite": true
  }
]
```

**Policy Explanation**:
- **Member Organizations**: HospitalMSP1, HospitalMSP2, ResearchOrgMSP, RegulatorMSP
- **Read/Write Access**: Restricted to member organizations only
- **BlockToLive**: 0 (permanent storage)
- **Member Only**: Only members can read/write

### Collection Name Pattern

**Requirement**: `collection_record_<recordID>`

**Implementation Note**: 
- Fabric requires collections to be statically defined
- Current implementation uses a single collection with recordID in keys
- To use pattern-based collections, collections must be pre-defined per record
- Both approaches provide equivalent privacy guarantees

**To Use Pattern-Based Collections**:
1. Pre-define collections for each recordID
2. Update `getCollectionName()` to return `fmt.Sprintf("collection_record_%s", recordID)`
3. Deploy collections configuration with all required collections

## Deployment

### Prerequisites

- Hyperledger Fabric v2.x network running
- Channel `medical-main-channel` created and peers joined
- `authorization-consent` chaincode deployed (anonymous-sharing queries it)
- `medical-records` chaincode deployed (authorization-consent queries it)
- Go 1.20+ installed
- Fabric chaincode dependencies

### Build Chaincode

```bash
cd chaincode/anonymous-sharing
go mod download
go mod vendor  # Optional: vendor dependencies
```

### Package Chaincode

```bash
peer lifecycle chaincode package anonymous-sharing.tar.gz \
  --path ./chaincode/anonymous-sharing \
  --lang golang \
  --label anonymous-sharing_1.0
```

### Install Chaincode

Install on all peer organizations:

```bash
# Install on RegulatorMSP peer
peer lifecycle chaincode install anonymous-sharing.tar.gz

# Install on HospitalMSP1 peer
peer lifecycle chaincode install anonymous-sharing.tar.gz

# Install on HospitalMSP2 peer
peer lifecycle chaincode install anonymous-sharing.tar.gz

# Install on ResearchOrgMSP peer
peer lifecycle chaincode install anonymous-sharing.tar.gz
```

### Approve Chaincode

Approve from each organization (example for RegulatorMSP):

```bash
peer lifecycle chaincode approveformyorg \
  -o orderer0.medical-network.com:7050 \
  --channelID medical-main-channel \
  --name anonymous-sharing \
  --version 1.0 \
  --package-id <PACKAGE_ID> \
  --sequence 1 \
  --collections-config collections_config.json \
  --tls \
  --cafile /path/to/orderer/tls/ca.crt
```

**Important**: Include `--collections-config collections_config.json` to deploy the Private Data Collection.

### Commit Chaincode

```bash
peer lifecycle chaincode commit \
  -o orderer0.medical-network.com:7050 \
  --channelID medical-main-channel \
  --name anonymous-sharing \
  --version 1.0 \
  --sequence 1 \
  --collections-config collections_config.json \
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

## Security Considerations

### Access Control
- ✅ MSP-based access control enforced at chaincode level
- ✅ Authorization verification before sharing
- ✅ Ownership verification (only owning hospital can share)
- ✅ Expiration checking for authorizations

### Privacy
- ✅ Private Data Collections for sensitive payloads
- ✅ Hash commitments on public ledger
- ✅ Encrypted pointers (encrypted by middleware)
- ✅ Access control restricts PDC visibility

### Auditability
- ✅ All share and access operations emit events
- ✅ Events include invoker MSP, client ID, timestamp
- ✅ Complete transaction history on ledger
- ✅ Regulatory access for compliance

### Integrity
- ✅ SHA-256 hash commitments
- ✅ Hash verification function
- ✅ Authorization status verification
- ✅ Duplicate sharing prevention

## Constraints and Assumptions

### Cryptographic Constraints
- **No Cryptography Implementation**: Chaincode does not implement encryption/decryption
- **Hash Comparison Only**: Uses SHA-256 for hash commitments
- **Encrypted Pointer Assumption**: `encryptedPointer` is assumed to be already encrypted by middleware

### Middleware Responsibilities
- **Encryption**: Middleware must encrypt pointers before calling `ShareRecord`
- **Decryption**: Middleware must decrypt pointers after retrieving from `GetSharedRecord`
- **Key Management**: Middleware responsible for key management

### Network Assumptions
- **Channel**: All chaincodes on same channel (`medical-main-channel`)
- **Chaincode Dependencies**: Requires `authorization-consent` chaincode to be deployed
- **MSP Configuration**: MSP names must match network configuration

## Troubleshooting

### Access Denied Errors

If you receive "access denied" errors:
1. Verify the client identity is correctly authenticated
2. Check that the MSP ID matches expected values
3. For `ShareRecord`, ensure invoker is HospitalMSP
4. For `GetSharedRecord`, ensure invoker is authorized (HospitalMSP, ResearchOrgMSP, or RegulatorMSP)

### Authorization Not GRANTED

If authorization is not GRANTED:
1. Verify authorization request exists
2. Check authorization status via `authorization-consent` chaincode
3. Ensure both hospital and patient have approved
4. Verify authorization has not expired

### Private Data Collection Not Found

If PDC access fails:
1. Verify collection configuration was included during deployment
2. Check collection name matches configuration
3. Verify invoker is a member of the collection policy
4. Ensure collection was properly deployed on all peers

### Hash Verification Fails

If hash verification fails:
1. Verify hash format (64-character hex string)
2. Check that record was shared (commitment exists)
3. Ensure provided hash matches the stored commitment
4. Verify hash computation matches chaincode implementation

## References

- [Hyperledger Fabric Chaincode Documentation](https://hyperledger-fabric.readthedocs.io/en/latest/chaincode4ade.html)
- [Fabric Contract API Go](https://github.com/hyperledger/fabric-contract-api-go)
- [Private Data Collections](https://hyperledger-fabric.readthedocs.io/en/latest/private-data/private-data.html)
- [Client Identity Library](https://github.com/hyperledger/fabric-chaincode-go)
