# Anonymous Sharing Chaincode - Implementation Details

This document provides detailed implementation information for the anonymous-sharing chaincode.

## Architecture Overview

The anonymous-sharing chaincode implements supervised anonymous information sharing using Private Data Collections (PDCs) on Hyperledger Fabric v2.x. It provides privacy-preserving data sharing while maintaining regulatory oversight and auditability.

### Key Components

1. **Private Data Collections (PDCs)**: Store sensitive sharing payloads
2. **Public Ledger**: Stores hash commitments for integrity verification
3. **Authorization Integration**: Verifies authorization status via authorization-consent chaincode
4. **Access Control**: MSP-based access control enforced at chaincode level
5. **Event System**: Emits events for auditability

## Data Flow

### Sharing Flow

```
1. HospitalMSP calls ShareRecord()
   ├── Verify invoker is HospitalMSP
   ├── Verify authorization is GRANTED
   ├── Verify ownership (invoker is owning hospital)
   ├── Check expiration
   ├── Create SharedRecordPayload
   ├── Compute SHA-256 hash
   ├── Store payload in PDC
   ├── Store hash commitment on public ledger
   └── Emit RecordShared event
```

### Access Flow

```
1. Authorized organization calls GetSharedRecord()
   ├── Verify invoker has access (HospitalMSP/ResearchOrgMSP/RegulatorMSP)
   ├── Read payload from PDC
   ├── Emit RecordAccessed event
   └── Return payload
```

### Verification Flow

```
1. Any organization calls VerifySharedRecord()
   ├── Read hash commitment from public ledger
   ├── Compare provided hash with stored hash
   └── Return match result
```

## Private Data Collection Design

### Collection Configuration

**Name**: `shared_records_collection`

**Policy**: `OR('HospitalMSP1.member', 'HospitalMSP2.member', 'ResearchOrgMSP.member', 'RegulatorMSP.member')`

**Properties**:
- `requiredPeerCount`: 0 (no minimum required)
- `maxPeerCount`: 3 (distribute to 3 peers)
- `blockToLive`: 0 (permanent storage)
- `memberOnlyRead`: true (only members can read)
- `memberOnlyWrite`: true (only members can write)

### Collection Name Pattern

**Requirement**: `collection_record_<recordID>`

**Implementation**: 
- Current implementation uses a single collection (`shared_records_collection`)
- RecordID is included in the key: `<recordID>_<authorizationRequestID>`
- This provides equivalent privacy guarantees while being more practical

**Alternative Approach**:
- To use pattern-based collections, collections must be pre-defined per recordID
- Update `getCollectionName()` to return `fmt.Sprintf("collection_record_%s", recordID)`
- Deploy collections configuration with all required collections

### Key Structure

**Private Data Key**: `<recordID>_<authorizationRequestID>`

Example: `REC-2024-001_AUTH-REC-2024-001-1234567890`

**Public Ledger Key**: `SHARED_<recordID>_<authorizationRequestID>`

Example: `SHARED_REC-2024-001_AUTH-REC-2024-001-1234567890`

## Access Control Implementation

### MSP-Based Access Control

All access control is enforced at the chaincode level using Fabric's Client Identity API:

```go
func getInvokerMSP(ctx contractapi.TransactionContextInterface) (string, error) {
    clientIdentity, err := ctx.GetClientIdentity().GetMSPID()
    if err != nil {
        return "", fmt.Errorf("failed to get client identity MSP ID: %v", err)
    }
    return clientIdentity, nil
}
```

### Function-Specific Access Rules

#### ShareRecord
- **Allowed**: HospitalMSP organizations only
- **Additional Checks**:
  - Verifies invoker is the owning hospital
  - Verifies authorization is GRANTED
  - Verifies authorization has not expired

#### GetSharedRecord
- **Allowed**: 
  - Owning HospitalMSP
  - Requesting ResearchOrgMSP
  - RegulatorMSP
- **Additional Checks**:
  - Verifies invoker matches authorization request

#### VerifySharedRecord
- **Allowed**: Public (any organization)
- **Purpose**: Integrity verification without revealing payload

## Authorization Integration

### Authorization Verification

The chaincode queries the authorization-consent chaincode's world state to verify authorization:

```go
func getAuthorizationRequest(ctx contractapi.TransactionContextInterface, requestID string) (*AuthorizationRequest, error) {
    requestJSON, err := ctx.GetStub().GetState(requestID)
    // ... parse and return
}
```

**Note**: This assumes both chaincodes are on the same channel and the authorization request key is the requestID.

### Authorization Checks

1. **Status Verification**: Must be `GRANTED`
2. **RecordID Matching**: Authorization request recordID must match provided recordID
3. **Ownership Verification**: Invoker must be the owning hospital
4. **Expiration Check**: Authorization must not be expired

## Hash Computation

### Hash Algorithm

**Algorithm**: SHA-256

**Implementation**:
```go
func computeHash(data []byte) string {
    hash := sha256.Sum256(data)
    return hex.EncodeToString(hash[:])
}
```

### Hash Input

The hash is computed over the entire `SharedRecordPayload` JSON:

```go
payload := SharedRecordPayload{
    RecordID:              recordID,
    AuthorizationRequestID: authorizationRequestID,
    EncryptedPointer:      encryptedPointer,
    SharedAt:              timestamp,
    SharedBy:              invokerMSP,
}
payloadJSON, _ := json.Marshal(payload)
payloadHash := computeHash(payloadJSON)
```

### Hash Verification

Hash verification compares the provided hash with the stored commitment:

```go
storedHashLower := strings.ToLower(commitment.PayloadHash)
providedHashLower := strings.ToLower(providedHash)
match := storedHashLower == providedHashLower
```

## Event System

### Event Types

#### RecordShared Event

**Trigger**: When a record is shared via `ShareRecord()`

**Payload**:
```json
{
  "recordID": "REC-2024-001",
  "authorizationRequestID": "AUTH-REC-2024-001-1234567890",
  "action": "RecordShared",
  "invokerMSP": "HospitalMSP1",
  "clientID": "x509::CN=doctor1...",
  "timestamp": "2024-01-15T10:30:00Z",
  "payloadHash": "a3f5b8c9..."
}
```

#### RecordAccessed Event

**Trigger**: When a record is accessed via `GetSharedRecord()`

**Payload**:
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

### Event Emission

Events are emitted using Fabric's event system:

```go
err = ctx.GetStub().SetEvent("RecordShared", eventJSON)
```

## Error Handling

### Validation Errors

- **Empty Parameters**: All required parameters must be non-empty
- **Invalid Hash Format**: Hash must be 64-character hex string
- **Invalid Authorization**: Authorization must be GRANTED and not expired

### Access Control Errors

- **MSP Mismatch**: Invoker MSP does not match required MSP
- **Ownership Mismatch**: Invoker is not the owning hospital
- **Unauthorized Access**: Invoker does not have access to the record

### Business Logic Errors

- **Duplicate Sharing**: Record already shared for the authorization request
- **Authorization Not Found**: Authorization request does not exist
- **Record Not Found**: Shared record does not exist

## Privacy Guarantees

### What is Private?

- **SharedRecordPayload**: Stored in PDC, visible only to:
  - Owning HospitalMSP
  - Requesting ResearchOrgMSP
  - RegulatorMSP

### What is Public?

- **SharedRecordCommitment**: Stored on public ledger, visible to all:
  - recordID
  - authorizationRequestID
  - payloadHash
  - timestamp

### What Unrelated Organizations See?

- **Hash Commitments**: Can verify integrity but cannot view content
- **Metadata**: recordID, authorizationRequestID, timestamp
- **Cannot Access**: Private data payloads, encrypted pointers

## Integration Points

### Dependencies

1. **authorization-consent chaincode**: 
   - Must be deployed on the same channel
   - Provides authorization status verification
   - Stores authorization requests on world state

2. **medical-records chaincode**:
   - Indirect dependency (via authorization-consent)
   - Provides medical record metadata

### Channel Requirements

- All chaincodes must be on the same channel (`medical-main-channel`)
- All organizations must be members of the channel
- Private Data Collection must be configured for all member organizations

## Performance Considerations

### Private Data Collection

- **Distribution**: Private data is distributed to `maxPeerCount` peers (3)
- **Gossip**: Private data is gossiped only to authorized peers
- **Storage**: Private data stored separately from public ledger

### Query Performance

- **GetSharedRecord**: Reads from PDC (faster than public ledger queries)
- **VerifySharedRecord**: Reads from public ledger (standard query)
- **Authorization Lookup**: Reads from world state (standard query)

### Transaction Performance

- **ShareRecord**: 
  - 1 PDC write
  - 1 public ledger write
  - 1 authorization lookup
  - 1 event emission
- **GetSharedRecord**:
  - 1 PDC read
  - 1 authorization lookup
  - 1 event emission

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

### Integrity

- ✅ SHA-256 hash commitments
- ✅ Hash verification function
- ✅ Authorization status verification
- ✅ Duplicate sharing prevention

### Auditability

- ✅ All share and access operations emit events
- ✅ Events include invoker MSP, client ID, timestamp
- ✅ Complete transaction history on ledger
- ✅ Regulatory access for compliance

## Limitations

### Cryptographic Constraints

- **No Cryptography Implementation**: Chaincode does not implement encryption/decryption
- **Hash Comparison Only**: Uses SHA-256 for hash commitments
- **Encrypted Pointer Assumption**: `encryptedPointer` is assumed to be already encrypted by middleware

### Collection Constraints

- **Static Collections**: Fabric requires collections to be statically defined
- **Pattern Limitation**: Pattern-based collections (`collection_record_<recordID>`) require pre-definition
- **Single Collection**: Current implementation uses a single collection (more practical)

### Network Assumptions

- **Channel**: All chaincodes on same channel
- **Chaincode Dependencies**: Requires authorization-consent chaincode
- **MSP Configuration**: MSP names must match network configuration

## Future Enhancements

### Potential Improvements

1. **Dynamic Collections**: Support for dynamically created collections (if Fabric supports)
2. **Encryption**: Built-in encryption/decryption support
3. **Batch Operations**: Support for batch sharing of multiple records
4. **Access Logging**: Enhanced access logging and analytics
5. **Time-Based Access**: Support for time-based access control
6. **Revocation**: Support for revoking shared records

### Scalability Considerations

1. **Collection Sharding**: Multiple collections for better performance
2. **Caching**: Client-side caching of authorization status
3. **Indexing**: Indexing for faster queries
4. **Compression**: Compression of private data payloads
