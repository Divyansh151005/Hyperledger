# End-to-End Request Lifecycle Examples

## Example 1: Complete Medical Record Sharing Flow

This example demonstrates the complete lifecycle from creating a medical record to sharing it with a research organization.

### Step 1: Hospital Creates Medical Record

**Actor:** Hospital (HospitalMSP1)

**Request:**
```bash
POST /api/v1/records
Headers:
  X-Certificate-ID: hospital-cert-123
Body:
{
  "patient_id": "patient-123",
  "data": {
    "diagnosis": "Type 2 Diabetes",
    "medications": ["Metformin 500mg", "Insulin"],
    "lab_results": {
      "glucose": 120,
      "hba1c": 7.2
    },
    "notes": "Patient responding well to treatment"
  }
}
```

**Middleware Processing:**
1. Validates identity (must be Hospital)
2. Encrypts medical data using hospital-specific key
3. Stores encrypted data in off-chain storage (S3/Database)
4. Computes SHA-256 hash of encrypted data
5. Invokes `CreateMedicalRecord` chaincode with metadata
6. Chaincode stores metadata on blockchain
7. Chaincode emits `MedicalRecordCreated` event

**Response:**
```json
{
  "record_id": "rec-001",
  "record_hash": "a1b2c3d4e5f6...",
  "status": "ACTIVE"
}
```

**Blockchain State:**
- Medical record metadata stored on-chain
- Hash commitment stored on-chain
- Full encrypted data stored off-chain

### Step 2: Research Organization Requests Access

**Actor:** Research Organization (ResearchOrgMSP)

**Request:**
```bash
POST /api/v1/authorizations/request
Headers:
  X-Certificate-ID: research-cert-456
Body:
{
  "record_id": "rec-001",
  "duration": 48
}
```

**Middleware Processing:**
1. Validates identity (must be Research Org)
2. Verifies medical record exists on blockchain
3. Invokes `RequestAccess` chaincode
4. Chaincode creates authorization request in REQUESTED status
5. Chaincode emits `AuthorizationRequested` event

**Response:**
```json
{
  "request_id": "AUTH-rec-001-1234567890",
  "status": "REQUESTED"
}
```

**Blockchain State:**
- Authorization request created with status REQUESTED
- Requires both hospital and patient approval

### Step 3: Hospital Approves Authorization

**Actor:** Hospital (HospitalMSP1)

**Request:**
```bash
POST /api/v1/authorizations/AUTH-rec-001-1234567890/approve/hospital
Headers:
  X-Certificate-ID: hospital-cert-123
```

**Middleware Processing:**
1. Validates identity (must be Hospital)
2. Verifies hospital owns the record
3. Invokes `ApproveByHospital` chaincode
4. Chaincode updates status to HOSPITAL_APPROVED
5. Chaincode emits `HospitalApproved` event

**Response:**
```json
{
  "status": "approved"
}
```

**Blockchain State:**
- Authorization request status: HOSPITAL_APPROVED
- Still requires patient approval

### Step 4: Patient Approves Authorization

**Actor:** Patient

**Request:**
```bash
POST /api/v1/authorizations/AUTH-rec-001-1234567890/approve/patient
Headers:
  X-Certificate-ID: patient-cert-789
```

**Middleware Processing:**
1. Validates identity (must be Patient)
2. Verifies patient matches authorization request
3. Invokes `ApproveByPatient` chaincode
4. Chaincode updates status to GRANTED
5. Chaincode emits `PatientApproved` event
6. **Middleware generates encryption key for authorizationRequestID**
7. Key stored encrypted in database

**Response:**
```json
{
  "status": "approved"
}
```

**Blockchain State:**
- Authorization request status: GRANTED
- Encryption key generated and stored (off-chain)

### Step 5: Hospital Shares Record

**Actor:** Hospital (HospitalMSP1)

**Request:**
```bash
POST /api/v1/sharing/share
Headers:
  X-Certificate-ID: hospital-cert-123
Body:
{
  "record_id": "rec-001",
  "authorization_request_id": "AUTH-rec-001-1234567890"
}
```

**Middleware Processing:**
1. Validates identity (must be Hospital)
2. Verifies authorization status is GRANTED
3. Retrieves encrypted medical record from off-chain storage
4. Creates encrypted pointer using authorization-specific key
5. Invokes `ShareRecord` chaincode with encrypted pointer
6. Chaincode stores encrypted pointer in Private Data Collection
7. Chaincode stores hash commitment on public ledger
8. Chaincode emits `RecordShared` event

**Response:**
```json
{
  "record_id": "rec-001",
  "authorization_request_id": "AUTH-rec-001-1234567890",
  "status": "SHARED"
}
```

**Blockchain State:**
- Encrypted pointer stored in Private Data Collection (visible to Hospital, Research Org, Regulator)
- Hash commitment stored on public ledger (visible to all)

### Step 6: Research Organization Retrieves Shared Record

**Actor:** Research Organization (ResearchOrgMSP)

**Request:**
```bash
GET /api/v1/sharing/rec-001/AUTH-rec-001-1234567890
Headers:
  X-Certificate-ID: research-cert-456
```

**Middleware Processing:**
1. Validates identity (must be Research Org)
2. Invokes `GetSharedRecord` chaincode
3. Chaincode returns encrypted pointer from Private Data Collection
4. Middleware retrieves encryption key for authorizationRequestID
5. Middleware decrypts pointer
6. Middleware retrieves encrypted medical record from off-chain storage using pointer
7. Middleware decrypts medical record
8. Returns decrypted data to client

**Response:**
```json
{
  "record_id": "rec-001",
  "authorization_request_id": "AUTH-rec-001-1234567890",
  "data": {
    "diagnosis": "Type 2 Diabetes",
    "medications": ["Metformin 500mg", "Insulin"],
    "lab_results": {
      "glucose": 120,
      "hba1c": 7.2
    },
    "notes": "Patient responding well to treatment"
  },
  "shared_at": "2024-01-01T12:00:00Z"
}
```

**Blockchain State:**
- Access logged via `RecordAccessed` event
- Audit trail maintained

## Example 2: Encryption Flow Details

### Key Management Per Authorization Request

When an authorization request reaches GRANTED status:

1. **Key Generation:**
   - Middleware generates a unique AES-256 key
   - Key ID: `authorizationRequestID`
   - Key stored encrypted in database (encrypted with master key)

2. **Pointer Encryption:**
   - When sharing, pointer is encrypted with authorization-specific key
   - Only parties with the key can decrypt the pointer
   - Key is accessible only to authorized parties

3. **Data Decryption:**
   - Research org retrieves encrypted pointer from blockchain
   - Middleware retrieves key for authorizationRequestID
   - Middleware decrypts pointer
   - Middleware uses pointer to retrieve encrypted medical record
   - Medical record decrypted using hospital key
   - Decrypted data returned to client

### Key Isolation

- Each authorization request has its own encryption key
- Keys are isolated per authorizationRequestID
- Revocation invalidates access but keys remain for audit purposes
- Keys are encrypted at rest using master key

## Example 3: Audit Integration

### Event Flow

1. **Chaincode emits event:**
   ```go
   eventPayload := EventPayload{
       RecordID: "rec-001",
       Action: "MedicalRecordCreated",
       InvokerMSP: "HospitalMSP1",
       Timestamp: "2024-01-01T00:00:00Z"
   }
   ctx.GetStub().SetEvent("MedicalRecordCreated", eventJSON)
   ```

2. **Event listener receives event:**
   - Event listener subscribes to blockchain events
   - Receives event from Fabric network
   - Extracts event payload

3. **Audit service logs event:**
   ```sql
   INSERT INTO audit_logs (
       event_type, chaincode_name, transaction_id,
       block_number, invoker_msp, client_id, payload, timestamp
   ) VALUES (
       'MedicalRecordCreated', 'medical-records', 'tx-123',
       100, 'HospitalMSP1', 'User1@hospital1...', '{"recordID":"rec-001"}', NOW()
   )
   ```

4. **Regulator queries audit logs:**
   ```bash
   GET /api/v1/audit/logs?event_type=MedicalRecordCreated&limit=100
   ```

## Example 4: Identity Mapping

### Identity Registration

**Register Hospital Identity:**
```sql
INSERT INTO identity_mappings (
    certificate_id, msp_id, role, user_id
) VALUES (
    'x509::CN=User1@hospital1...', 'HospitalMSP1', 'HOSPITAL', 'hospital-001'
)
```

**Register Patient Identity:**
```sql
INSERT INTO identity_mappings (
    certificate_id, msp_id, role, user_id
) VALUES (
    'x509::CN=patient-123...', 'HospitalMSP1', 'PATIENT', 'patient-123'
)
```

### Identity Validation Flow

1. Client sends request with `X-Certificate-ID` header
2. Middleware extracts certificate ID
3. Middleware queries identity_mappings table
4. Middleware validates role and MSP
5. Middleware enforces access control before blockchain invocation
6. If valid, request proceeds; otherwise, returns 403 Forbidden

## Example 5: Error Scenarios

### Scenario 1: Unauthorized Access Attempt

**Request:**
```bash
GET /api/v1/records/rec-001
Headers:
  X-Certificate-ID: research-cert-456  # Research org trying to access hospital record
```

**Response:**
```json
{
  "error": "access denied: only regulator or owning hospital can access this record"
}
```

**Status Code:** 403 Forbidden

### Scenario 2: Invalid Authorization Status

**Request:**
```bash
POST /api/v1/sharing/share
Body:
{
  "record_id": "rec-001",
  "authorization_request_id": "AUTH-rec-001-1234567890"  # Status is REQUESTED, not GRANTED
}
```

**Response:**
```json
{
  "error": "authorization request must be GRANTED, current status: REQUESTED"
}
```

**Status Code:** 400 Bad Request

### Scenario 3: Missing Certificate

**Request:**
```bash
GET /api/v1/records/rec-001
# Missing X-Certificate-ID header
```

**Response:**
```json
{
  "error": "missing certificate ID"
}
```

**Status Code:** 401 Unauthorized

## Sequence Diagrams

### Create Medical Record Sequence

```
Client          Middleware        Encryption      Storage      Blockchain
  |                  |                 |            |              |
  |--POST /records-->|                 |            |              |
  |                  |--Validate ID----|            |              |
  |                  |<--Identity OK---|            |              |
  |                  |--Encrypt Data-->|            |              |
  |                  |<--Encrypted-----|            |              |
  |                  |--Store--------->|            |              |
  |                  |<--Stored--------|            |              |
  |                  |--Compute Hash-->|            |              |
  |                  |<--Hash----------|            |              |
  |                  |--Invoke Chaincode---------------->|
  |                  |<--Success-------------------------|
  |<--201 Created----|                 |            |              |
```

### Share Record Sequence

```
Client          Middleware        Encryption      Storage      Blockchain
  |                  |                 |            |              |
  |--POST /share---->|                 |            |              |
  |                  |--Get Auth------>|            |              |
  |                  |<--Status=GRANTED|            |              |
  |                  |--Get Record---->|            |              |
  |                  |<--Encrypted----|            |              |
  |                  |--Encrypt Pointer->|          |              |
  |                  |<--Encrypted Ptr--|            |              |
  |                  |--Share Chaincode---------------->|
  |                  |<--Success-------------------------|
  |<--201 Created----|                 |            |              |
```

## Security Considerations

1. **Encryption Keys:**
   - Keys generated per authorization request
   - Keys encrypted at rest with master key
   - Keys never exposed to clients

2. **Access Control:**
   - Identity validated before every operation
   - Role-based access enforced in middleware
   - Chaincode access control as final enforcement

3. **Audit Trail:**
   - All operations logged
   - Events captured from blockchain
   - Immutable audit log in database

4. **Data Privacy:**
   - Full medical data never on blockchain
   - Only hashes and metadata on-chain
   - Encrypted pointers in Private Data Collections
   - Decryption only for authorized parties
