# Middleware Architecture for Hyperledger Fabric Medical Information Sharing Platform

## Overview

The middleware layer serves as a bridge between client applications and the Hyperledger Fabric blockchain network. It abstracts the complexity of Fabric SDK operations, manages encryption/decryption, handles off-chain storage, maps identities, and provides audit capabilities.

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           Client Applications                            │
│                    (Web Apps, Mobile Apps, APIs)                        │
└──────────────────────────────┬──────────────────────────────────────────┘
                                │
                                │ REST/gRPC API
                                │
┌───────────────────────────────▼──────────────────────────────────────────┐
│                         Middleware Layer                                  │
│                                                                           │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                    API Gateway (REST/gRPC)                       │   │
│  │  - Request validation                                            │   │
│  │  - Authentication/Authorization                                  │   │
│  │  - Rate limiting                                                │   │
│  └───────────────────────────┬─────────────────────────────────────┘   │
│                              │                                           │
│  ┌───────────────────────────▼─────────────────────────────────────┐   │
│  │              Business Logic Layer                                │   │
│  │  - Medical Record Service                                        │   │
│  │  - Authorization Service                                         │   │
│  │  - Sharing Service                                              │   │
│  └───────────────────────────┬─────────────────────────────────────┘   │
│                              │                                           │
│        ┌──────────────────────┼──────────────────────┐                  │
│        │                      │                      │                  │
│  ┌─────▼─────┐      ┌─────────▼─────────┐  ┌────────▼────────┐         │
│  │ Encryption│      │  Identity Mapping │  │  Audit Service │         │
│  │  Service  │      │     Service       │  │                │         │
│  └─────┬─────┘      └───────────────────┘  └────────┬───────┘         │
│        │                                             │                  │
│  ┌─────▼────────────────────────────────────────────▼───────┐         │
│  │              Fabric SDK Integration Layer                 │         │
│  │  - Chaincode Invocation                                  │         │
│  │  - Event Listening                                       │         │
│  │  - Identity Management                                  │         │
│  └───────────────────────────┬─────────────────────────────┘         │
│                              │                                           │
└──────────────────────────────┼──────────────────────────────────────────┘
                               │
                               │ Fabric SDK
                               │
┌──────────────────────────────▼──────────────────────────────────────────┐
│                    Hyperledger Fabric Network                            │
│  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐     │
│  │ Medical Records  │  │ Authorization   │  │ Anonymous Sharing│     │
│  │   Chaincode      │  │   Chaincode     │  │    Chaincode     │     │
│  └──────────────────┘  └──────────────────┘  └──────────────────┘     │
└─────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────┐
│                        Off-Chain Storage                                 │
│  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐       │
│  │  Database        │  │  Object Storage  │  │  Key Store       │       │
│  │  (PostgreSQL)    │  │  (S3/IPFS)      │  │  (Encrypted Keys)│       │
│  └──────────────────┘  └──────────────────┘  └──────────────────┘       │
└─────────────────────────────────────────────────────────────────────────┘
```

## Component Details

### 1. API Gateway Layer

**Responsibilities:**
- Expose REST/gRPC endpoints
- Request/response validation
- Authentication middleware
- Rate limiting
- Error handling and formatting

**Endpoints:**
- `POST /api/v1/records` - Create medical record
- `GET /api/v1/records/:recordID` - Get medical record
- `POST /api/v1/authorizations/request` - Request access
- `POST /api/v1/authorizations/:requestID/approve` - Approve access
- `POST /api/v1/authorizations/:requestID/revoke` - Revoke access
- `POST /api/v1/sharing/share` - Share record
- `GET /api/v1/sharing/:recordID/:authRequestID` - Retrieve shared record
- `GET /api/v1/audit/logs` - Get audit logs

### 2. Business Logic Layer

**Medical Record Service:**
- Encrypts medical data before storage
- Computes hash for blockchain
- Stores encrypted data off-chain
- Invokes `CreateMedicalRecord` chaincode
- Retrieves and decrypts records

**Authorization Service:**
- Creates authorization requests
- Manages approval workflows
- Tracks authorization state
- Invokes authorization chaincode functions

**Sharing Service:**
- Encrypts pointers before sharing
- Manages encryption keys per authorizationRequestID
- Invokes `ShareRecord` chaincode
- Retrieves and decrypts shared records

### 3. Encryption Service

**Key Management:**
- Generates encryption keys per `authorizationRequestID`
- Uses AES-256-GCM for symmetric encryption
- Stores keys in secure key store (encrypted at rest)
- Key rotation support
- Key lifecycle management

**Encryption Flow:**
```
1. Generate/Retrieve key for authorizationRequestID
2. Encrypt medical record or pointer
3. Store encrypted data off-chain
4. Return encrypted data or pointer
```

**Decryption Flow:**
```
1. Retrieve encrypted data from off-chain storage
2. Retrieve key for authorizationRequestID
3. Decrypt data
4. Return decrypted data
```

### 4. Identity Mapping Service

**Responsibilities:**
- Maps Fabric X.509 certificate to user roles
- Maintains identity registry (Patient, Hospital, Research Org)
- Validates identity before blockchain operations
- Enforces role-based access control

**Identity Types:**
- **Patient**: X.509 CN = patientID
- **Hospital**: MSP ID = HospitalMSP1 or HospitalMSP2
- **Research Organization**: MSP ID = ResearchOrgMSP
- **Regulator**: MSP ID = RegulatorMSP

**Mapping Database Schema:**
```sql
CREATE TABLE identity_mappings (
    certificate_id VARCHAR(255) PRIMARY KEY,
    msp_id VARCHAR(100) NOT NULL,
    role VARCHAR(50) NOT NULL, -- PATIENT, HOSPITAL, RESEARCH, REGULATOR
    user_id VARCHAR(255), -- patientID, hospitalID, etc.
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### 5. Off-Chain Storage Service

**Storage Options:**
- **Database (PostgreSQL)**: Metadata, audit logs, identity mappings
- **Object Storage (S3/IPFS)**: Full medical records, encrypted pointers

**Data Storage Strategy:**
```
Medical Record:
  - Metadata (recordID, patientID, hospitalID, hash) → Blockchain
  - Full encrypted record → Off-chain storage (DB or Object Storage)
  - Storage location/pointer → Stored with metadata

Shared Record:
  - Encrypted pointer → Private Data Collection (on-chain)
  - Actual encrypted data → Off-chain storage
```

### 6. Fabric SDK Integration Layer

**Responsibilities:**
- Initialize Fabric SDK with connection profiles
- Manage gateway connections
- Invoke chaincode functions
- Query chaincode state
- Listen to blockchain events
- Handle transaction submission and endorsement

**Chaincode Interactions:**
- `medical-records`: CreateMedicalRecord, GetMedicalRecord, VerifyMedicalRecordHash
- `authorization-consent`: RequestAccess, ApproveByHospital, ApproveByPatient, RevokeAuthorization, GetAuthorization
- `anonymous-sharing`: ShareRecord, GetSharedRecord, VerifySharedRecord

### 7. Audit Service

**Event Listening:**
- Subscribes to Fabric events from all chaincodes
- Filters events by type (MedicalRecordCreated, AuthorizationRequested, etc.)
- Persists events to audit database

**Audit Log Schema:**
```sql
CREATE TABLE audit_logs (
    id SERIAL PRIMARY KEY,
    event_type VARCHAR(100) NOT NULL,
    chaincode_name VARCHAR(100) NOT NULL,
    transaction_id VARCHAR(255) NOT NULL,
    block_number BIGINT NOT NULL,
    invoker_msp VARCHAR(100),
    client_id VARCHAR(255),
    payload JSONB,
    timestamp TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## Data Flow Diagrams

### Create Medical Record Flow

```
1. Client → API: POST /api/v1/records
2. API → Identity Service: Validate identity (must be Hospital)
3. API → Encryption Service: Encrypt medical record
4. API → Storage Service: Store encrypted record off-chain
5. API → Encryption Service: Compute hash of encrypted record
6. API → Fabric SDK: Invoke CreateMedicalRecord(recordID, patientID, hash)
7. Fabric SDK → Chaincode: Transaction submitted
8. Chaincode → Event: Emit MedicalRecordCreated event
9. Event Listener → Audit Service: Persist event to audit log
10. API → Client: Return recordID and status
```

### Request Access Flow

```
1. Client → API: POST /api/v1/authorizations/request
2. API → Identity Service: Validate identity (must be Research Org)
3. API → Fabric SDK: Invoke RequestAccess(recordID, duration)
4. Fabric SDK → Chaincode: Transaction submitted
5. Chaincode → Event: Emit AuthorizationRequested event
6. Event Listener → Audit Service: Persist event
7. API → Client: Return requestID
```

### Approve Access Flow

```
1. Client → API: POST /api/v1/authorizations/:requestID/approve
2. API → Identity Service: Validate identity (Hospital or Patient)
3. API → Fabric SDK: Invoke ApproveByHospital/ApproveByPatient(requestID)
4. Fabric SDK → Chaincode: Transaction submitted
5. Chaincode → Event: Emit HospitalApproved/PatientApproved event
6. Event Listener → Audit Service: Persist event
7. If status = GRANTED:
   - Encryption Service: Generate encryption key for authorizationRequestID
   - Storage Service: Store key mapping
8. API → Client: Return success
```

### Share Record Flow

```
1. Client → API: POST /api/v1/sharing/share
2. API → Identity Service: Validate identity (must be Hospital)
3. API → Fabric SDK: Query GetAuthorization(authorizationRequestID)
4. API → Validation: Verify status = GRANTED
5. API → Encryption Service: Retrieve key for authorizationRequestID
6. API → Storage Service: Retrieve encrypted medical record
7. API → Encryption Service: Encrypt pointer to record (using authorization key)
8. API → Fabric SDK: Invoke ShareRecord(recordID, authorizationRequestID, encryptedPointer)
9. Fabric SDK → Chaincode: Transaction submitted
10. Chaincode → Event: Emit RecordShared event
11. Event Listener → Audit Service: Persist event
12. API → Client: Return success
```

### Retrieve Shared Record Flow

```
1. Client → API: GET /api/v1/sharing/:recordID/:authRequestID
2. API → Identity Service: Validate identity (Research Org or Hospital)
3. API → Fabric SDK: Invoke GetSharedRecord(recordID, authorizationRequestID)
4. Chaincode → Returns: Encrypted pointer from Private Data Collection
5. API → Encryption Service: Retrieve key for authorizationRequestID
6. API → Encryption Service: Decrypt pointer
7. API → Storage Service: Retrieve encrypted medical record using pointer
8. API → Encryption Service: Decrypt medical record
9. API → Client: Return decrypted medical record
```

## Security Considerations

### 1. Encryption
- **Symmetric Encryption**: AES-256-GCM for medical records
- **Key Management**: Keys stored encrypted at rest
- **Key Isolation**: Separate keys per authorizationRequestID
- **Key Rotation**: Support for key rotation without data re-encryption

### 2. Access Control
- **Identity Verification**: All requests verify X.509 certificate
- **Role-Based Access**: Enforced before blockchain invocation
- **Chaincode Enforcement**: Middleware does not bypass chaincode access control
- **Audit Trail**: All operations logged

### 3. Data Privacy
- **Off-Chain Storage**: Full medical data never on blockchain
- **Encrypted Pointers**: Pointers encrypted before sharing
- **Private Data Collections**: Used for sensitive sharing metadata
- **Access Logging**: All data access logged

### 4. Network Security
- **TLS**: All communications encrypted
- **Certificate Validation**: X.509 certificates validated
- **Rate Limiting**: API rate limiting to prevent abuse
- **Input Validation**: All inputs validated before processing

## Deployment Architecture

### Middleware Components
- **API Server**: REST/gRPC server (Go)
- **Event Listener**: Background service listening to Fabric events
- **Database**: PostgreSQL for metadata and audit logs
- **Object Storage**: S3-compatible storage or IPFS for medical records
- **Key Store**: Encrypted key storage (can use database or dedicated key management service)

### Scalability
- **Horizontal Scaling**: API servers can be scaled horizontally
- **Load Balancing**: Load balancer in front of API servers
- **Database Replication**: Read replicas for audit queries
- **Caching**: Redis for frequently accessed data

## Configuration

### Environment Variables
```bash
# Fabric Network
FABRIC_NETWORK_CONFIG_PATH=/path/to/connection-profile.json
FABRIC_CHANNEL_NAME=medical-main-channel
FABRIC_CHAINCODE_MEDICAL_RECORDS=medical-records
FABRIC_CHAINCODE_AUTHORIZATION=authorization-consent
FABRIC_CHAINCODE_ANONYMOUS_SHARING=anonymous-sharing

# Database
DATABASE_URL=postgresql://user:pass@localhost:5432/medical_middleware
DATABASE_MAX_CONNECTIONS=25

# Object Storage (S3)
S3_ENDPOINT=https://s3.amazonaws.com
S3_BUCKET=medical-records
S3_ACCESS_KEY=xxx
S3_SECRET_KEY=xxx

# Encryption
ENCRYPTION_KEY_STORE_PATH=/secure/path/to/keystore
ENCRYPTION_ALGORITHM=AES-256-GCM

# API Server
API_PORT=8080
API_HOST=0.0.0.0
API_LOG_LEVEL=info

# Identity Mapping
IDENTITY_DB_PATH=/path/to/identity-db
```

## Error Handling

### Error Types
1. **Validation Errors**: Invalid input parameters (400)
2. **Authentication Errors**: Invalid or missing credentials (401)
3. **Authorization Errors**: Insufficient permissions (403)
4. **Not Found Errors**: Resource not found (404)
5. **Fabric Errors**: Blockchain transaction failures (500)
6. **Storage Errors**: Off-chain storage failures (500)

### Error Response Format
```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable error message",
    "details": {},
    "timestamp": "2024-01-01T00:00:00Z"
  }
}
```

## Monitoring and Observability

### Metrics
- API request rate and latency
- Blockchain transaction success/failure rate
- Storage operation metrics
- Encryption/decryption operation metrics
- Event processing metrics

### Logging
- Structured logging (JSON format)
- Log levels: DEBUG, INFO, WARN, ERROR
- Request/response logging
- Blockchain transaction logging
- Error logging with stack traces

### Health Checks
- `/health` endpoint for API server
- Database connectivity check
- Fabric network connectivity check
- Storage connectivity check
